// Package tools registers the MCP tools exposed by the server and contains the
// version-comparison logic that backs them.
package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/pcpl2/github-actions-versions-mcp/internal/github"
	"github.com/pcpl2/github-actions-versions-mcp/internal/workflow"
)

// Register wires all tools onto the given MCP server, backed by f.
func Register(server *mcp.Server, f releaseFetcher) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "latest_release",
		Description: "Get the latest published release of a GitHub Action repository (e.g. owner=actions, repo=checkout). Returns the tag, publish date, URL and pinning advice.",
	}, latestReleaseHandler(f))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_releases",
		Description: "List the most recent releases of a GitHub Action repository, newest first.",
	}, listReleasesHandler(f))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "check_workflow_actions",
		Description: "Parse a GitHub Actions workflow YAML and report, for every `uses:` action, whether the pinned version is the latest release.",
	}, checkWorkflowHandler(f))
}

// ---- latest_release ----

type latestReleaseInput struct {
	Owner string `json:"owner" jsonschema:"the repository owner, e.g. actions"`
	Repo  string `json:"repo" jsonschema:"the repository name, e.g. checkout"`
}

type latestReleaseOutput struct {
	Action        string `json:"action"`
	TagName       string `json:"tag_name"`
	PublishedAt   string `json:"published_at"`
	HTMLURL       string `json:"html_url"`
	Prerelease    bool   `json:"prerelease"`
	PinningAdvice string `json:"pinning_advice"`
}

func latestReleaseHandler(f releaseFetcher) mcp.ToolHandlerFor[latestReleaseInput, latestReleaseOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in latestReleaseInput) (*mcp.CallToolResult, latestReleaseOutput, error) {
		if in.Owner == "" || in.Repo == "" {
			return errorResult("both 'owner' and 'repo' are required"), latestReleaseOutput{}, nil
		}
		rel, err := f.LatestRelease(ctx, in.Owner, in.Repo)
		if err != nil {
			return errorResult(err.Error()), latestReleaseOutput{}, nil
		}
		out := latestReleaseOutput{
			Action:        in.Owner + "/" + in.Repo,
			TagName:       rel.TagName,
			PublishedAt:   formatTime(rel),
			HTMLURL:       rel.HTMLURL,
			Prerelease:    rel.Prerelease,
			PinningAdvice: pinningAdvice(rel.TagName),
		}
		return nil, out, nil
	}
}

// ---- list_releases ----

type listReleasesInput struct {
	Owner string `json:"owner" jsonschema:"the repository owner, e.g. actions"`
	Repo  string `json:"repo" jsonschema:"the repository name, e.g. checkout"`
	Limit int    `json:"limit,omitempty" jsonschema:"max number of releases to return (default 10)"`
}

type releaseInfo struct {
	TagName     string `json:"tag_name"`
	PublishedAt string `json:"published_at"`
	HTMLURL     string `json:"html_url"`
	Prerelease  bool   `json:"prerelease"`
}

type listReleasesOutput struct {
	Action   string        `json:"action"`
	Releases []releaseInfo `json:"releases"`
}

func listReleasesHandler(f releaseFetcher) mcp.ToolHandlerFor[listReleasesInput, listReleasesOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in listReleasesInput) (*mcp.CallToolResult, listReleasesOutput, error) {
		if in.Owner == "" || in.Repo == "" {
			return errorResult("both 'owner' and 'repo' are required"), listReleasesOutput{}, nil
		}
		limit := in.Limit
		if limit <= 0 {
			limit = 10
		}
		rels, err := f.ListReleases(ctx, in.Owner, in.Repo, limit)
		if err != nil {
			return errorResult(err.Error()), listReleasesOutput{}, nil
		}
		out := listReleasesOutput{Action: in.Owner + "/" + in.Repo}
		for _, r := range rels {
			out.Releases = append(out.Releases, releaseInfo{
				TagName:     r.TagName,
				PublishedAt: formatTime(&r),
				HTMLURL:     r.HTMLURL,
				Prerelease:  r.Prerelease,
			})
		}
		return nil, out, nil
	}
}

// ---- check_workflow_actions ----

type checkWorkflowInput struct {
	Content string `json:"content" jsonschema:"the full text of a GitHub Actions workflow YAML file"`
}

type checkWorkflowOutput struct {
	Summary string             `json:"summary"`
	Actions []actionAssessment `json:"actions"`
}

func checkWorkflowHandler(f releaseFetcher) mcp.ToolHandlerFor[checkWorkflowInput, checkWorkflowOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in checkWorkflowInput) (*mcp.CallToolResult, checkWorkflowOutput, error) {
		if strings.TrimSpace(in.Content) == "" {
			return errorResult("'content' (workflow YAML) is required"), checkWorkflowOutput{}, nil
		}
		refs := workflow.ParseActions(in.Content)
		report := buildWorkflowReport(ctx, f, refs)
		out := checkWorkflowOutput{
			Actions: report,
			Summary: summarize(report),
		}
		return nil, out, nil
	}
}

func summarize(report []actionAssessment) string {
	outdated := 0
	for _, a := range report {
		if a.Outdated {
			outdated++
		}
	}
	return fmt.Sprintf("%d action(s) found, %d outdated.", len(report), outdated)
}

// ---- helpers ----

func pinningAdvice(tag string) string {
	major := tag
	if i := strings.Index(tag, "."); i > 0 {
		major = tag[:i]
	}
	return fmt.Sprintf(
		"Pin to the major tag (@%s) for automatic patch/minor updates, or to a full commit SHA for maximum supply-chain security. Avoid pinning to a branch like @main.", major)
}

func formatTime(rel *github.Release) string {
	if rel.PublishedAt.IsZero() {
		return ""
	}
	return rel.PublishedAt.UTC().Format("2006-01-02T15:04:05Z")
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}
}
