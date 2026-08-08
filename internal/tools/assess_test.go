package tools

import (
	"context"
	"errors"
	"testing"

	"github.com/pcpl2/github-actions-versions-mcp/internal/github"
	"github.com/pcpl2/github-actions-versions-mcp/internal/workflow"
)

func TestAssess(t *testing.T) {
	tests := []struct {
		name         string
		ref          string
		latest       string
		wantStatus   string
		wantOutdated bool
	}{
		{"exact match", "v4.2.2", "v4.2.2", "up-to-date", false},
		{"major pin still current", "v4", "v4.2.2", "major-pinned", false},
		{"outdated major", "v3", "v4.2.2", "outdated", true},
		{"outdated exact", "v4.1.0", "v4.2.2", "outdated", true},
		{"sha pin", "1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b", "v4.2.2", "pinned-sha", false},
		{"branch ref", "main", "v4.2.2", "outdated", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, outdated, _ := assess(tt.ref, tt.latest)
			if status != tt.wantStatus {
				t.Errorf("status = %q, want %q", status, tt.wantStatus)
			}
			if outdated != tt.wantOutdated {
				t.Errorf("outdated = %v, want %v", outdated, tt.wantOutdated)
			}
		})
	}
}

// fakeFetcher implements releaseFetcher for tests.
type fakeFetcher struct {
	latest map[string]*github.Release
}

func (f *fakeFetcher) LatestRelease(_ context.Context, owner, repo string) (*github.Release, error) {
	if rel, ok := f.latest[owner+"/"+repo]; ok {
		return rel, nil
	}
	return nil, &github.NotFoundError{Resource: owner + "/" + repo}
}

func (f *fakeFetcher) ListReleases(_ context.Context, owner, repo string, limit int) ([]github.Release, error) {
	return nil, errors.New("not used in this test")
}

func TestBuildWorkflowReport(t *testing.T) {
	f := &fakeFetcher{latest: map[string]*github.Release{
		"actions/checkout":   {TagName: "v4.2.2"},
		"actions/setup-node": {TagName: "v4.2.2"},
	}}
	refs := []workflow.ActionRef{
		{Owner: "actions", Repo: "checkout", Ref: "v3"},   // outdated
		{Owner: "actions", Repo: "setup-node", Ref: "v4"}, // major-pinned
		{Owner: "ghost", Repo: "missing", Ref: "v1"},      // not found
	}

	report := buildWorkflowReport(context.Background(), f, refs)

	if len(report) != 3 {
		t.Fatalf("got %d entries, want 3", len(report))
	}
	if report[0].Status != "outdated" || !report[0].Outdated {
		t.Errorf("checkout entry = %+v", report[0])
	}
	if report[0].LatestRelease != "v4.2.2" {
		t.Errorf("checkout latest = %q", report[0].LatestRelease)
	}
	if report[1].Status != "major-pinned" {
		t.Errorf("setup-node entry = %+v", report[1])
	}
	if report[2].Status != "unknown" || report[2].Note == "" {
		t.Errorf("missing entry should report unknown with a note, got %+v", report[2])
	}
}
