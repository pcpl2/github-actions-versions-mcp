package github

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestServer(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewClient(WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
}

func TestLatestRelease(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/actions/checkout/releases/latest" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("X-GitHub-Api-Version"); got != "2026-03-10" {
			t.Errorf("API version header = %q, want 2026-03-10", got)
		}
		if got := r.Header.Get("Accept"); got != "application/vnd.github+json" {
			t.Errorf("Accept header = %q", got)
		}
		w.Write([]byte(`{
			"tag_name": "v4.2.2",
			"published_at": "2024-10-23T14:30:00Z",
			"html_url": "https://github.com/actions/checkout/releases/tag/v4.2.2",
			"prerelease": false,
			"draft": false
		}`))
	})

	rel, err := c.LatestRelease(context.Background(), "actions", "checkout")
	if err != nil {
		t.Fatalf("LatestRelease: %v", err)
	}
	if rel.TagName != "v4.2.2" {
		t.Errorf("TagName = %q, want v4.2.2", rel.TagName)
	}
	if rel.HTMLURL != "https://github.com/actions/checkout/releases/tag/v4.2.2" {
		t.Errorf("HTMLURL = %q", rel.HTMLURL)
	}
	if rel.Prerelease {
		t.Errorf("Prerelease = true, want false")
	}
	if rel.PublishedAt.Year() != 2024 || rel.PublishedAt.Month() != 10 {
		t.Errorf("PublishedAt parsed wrong: %v", rel.PublishedAt)
	}
}

func TestLatestReleaseWithToken(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret-token" {
			t.Errorf("Authorization = %q, want Bearer secret-token", got)
		}
		w.Write([]byte(`{"tag_name": "v1.0.0"}`))
	})
	c.token = "secret-token"

	if _, err := c.LatestRelease(context.Background(), "o", "r"); err != nil {
		t.Fatalf("LatestRelease: %v", err)
	}
}

func TestLatestReleaseNotFound(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "Not Found"}`))
	})

	_, err := c.LatestRelease(context.Background(), "nope", "nope")
	var nf *NotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("expected NotFoundError, got %v", err)
	}
}

func TestRateLimit(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message": "API rate limit exceeded"}`))
	})

	_, err := c.LatestRelease(context.Background(), "o", "r")
	var rl *RateLimitError
	if !errors.As(err, &rl) {
		t.Fatalf("expected RateLimitError, got %v", err)
	}
}

func TestListReleasesLimit(t *testing.T) {
	c := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("per_page"); got != "3" {
			t.Errorf("per_page = %q, want 3", got)
		}
		w.Write([]byte(`[
			{"tag_name": "v4.2.2"},
			{"tag_name": "v4.2.1"},
			{"tag_name": "v4.2.0"},
			{"tag_name": "v4.1.0"}
		]`))
	})

	rels, err := c.ListReleases(context.Background(), "actions", "checkout", 3)
	if err != nil {
		t.Fatalf("ListReleases: %v", err)
	}
	if len(rels) != 3 {
		t.Fatalf("len = %d, want 3 (limit applied client-side)", len(rels))
	}
	if rels[0].TagName != "v4.2.2" {
		t.Errorf("first tag = %q", rels[0].TagName)
	}
}
