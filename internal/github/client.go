// Package github provides a thin REST client for the GitHub API, scoped to the
// endpoints needed to discover the latest versions of GitHub Actions.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	defaultBaseURL = "https://api.github.com"
	apiVersion     = "2026-03-10"
)

// Release mirrors the fields we consume from a GitHub release object.
type Release struct {
	TagName     string    `json:"tag_name"`
	PublishedAt time.Time `json:"published_at"`
	HTMLURL     string    `json:"html_url"`
	Prerelease  bool      `json:"prerelease"`
	Draft       bool      `json:"draft"`
}

// Client talks to the GitHub REST API.
type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL overrides the API base URL (used in tests).
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = url }
}

// WithHTTPClient overrides the underlying HTTP client.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.httpClient = h }
}

// WithToken sets the bearer token explicitly.
func WithToken(token string) Option {
	return func(c *Client) { c.token = token }
}

// NewClient builds a Client. If no token is supplied via WithToken, it falls
// back to the GITHUB_TOKEN environment variable, which raises the rate limit
// from 60 to 5000 requests per hour.
func NewClient(opts ...Option) *Client {
	c := &Client{
		baseURL:    defaultBaseURL,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		token:      os.Getenv("GITHUB_TOKEN"),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// NotFoundError is returned when the GitHub API responds with 404.
type NotFoundError struct {
	Resource string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("not found: %s", e.Resource)
}

// RateLimitError is returned when the GitHub API rate limit is exhausted.
type RateLimitError struct {
	Authenticated bool
}

func (e *RateLimitError) Error() string {
	if e.Authenticated {
		return "GitHub API rate limit exceeded for the authenticated token"
	}
	return "GitHub API rate limit exceeded (unauthenticated, 60 req/h); set GITHUB_TOKEN to raise it to 5000 req/h"
}

// APIError captures any other non-success response.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("github api error: %d %s", e.StatusCode, e.Message)
}

// LatestRelease returns the latest published, non-draft release.
func (c *Client) LatestRelease(ctx context.Context, owner, repo string) (*Release, error) {
	path := fmt.Sprintf("/repos/%s/%s/releases/latest", owner, repo)
	var rel Release
	if err := c.get(ctx, path, nil, &rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

// ListReleases returns up to limit releases, newest first. A limit <= 0 means
// "use the GitHub default page size".
func (c *Client) ListReleases(ctx context.Context, owner, repo string, limit int) ([]Release, error) {
	path := fmt.Sprintf("/repos/%s/%s/releases", owner, repo)
	query := map[string]string{}
	if limit > 0 {
		query["per_page"] = strconv.Itoa(limit)
	}
	var rels []Release
	if err := c.get(ctx, path, query, &rels); err != nil {
		return nil, err
	}
	if limit > 0 && len(rels) > limit {
		rels = rels[:limit]
	}
	return rels, nil
}

func (c *Client) get(ctx context.Context, path string, query map[string]string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	if len(query) > 0 {
		q := req.URL.Query()
		for k, v := range query {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", apiVersion)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		if err := json.Unmarshal(body, out); err != nil {
			return fmt.Errorf("decoding response from %s: %w", path, err)
		}
		return nil
	case resp.StatusCode == http.StatusNotFound:
		return &NotFoundError{Resource: path}
	case resp.StatusCode == http.StatusForbidden && resp.Header.Get("X-RateLimit-Remaining") == "0":
		return &RateLimitError{Authenticated: c.token != ""}
	default:
		return &APIError{StatusCode: resp.StatusCode, Message: extractMessage(body)}
	}
}

func extractMessage(body []byte) string {
	var payload struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &payload) == nil && payload.Message != "" {
		return payload.Message
	}
	return http.StatusText(0)
}
