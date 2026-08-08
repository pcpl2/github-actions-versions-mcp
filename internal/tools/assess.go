package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/pcpl2/github-actions-versions-mcp/internal/github"
	"github.com/pcpl2/github-actions-versions-mcp/internal/workflow"
)

// releaseFetcher is the subset of *github.Client that the tools depend on,
// declared as an interface so handlers can be tested without network access.
type releaseFetcher interface {
	LatestRelease(ctx context.Context, owner, repo string) (*github.Release, error)
	ListReleases(ctx context.Context, owner, repo string, limit int) ([]github.Release, error)
}

// actionAssessment is one row of a check_workflow_actions report.
type actionAssessment struct {
	Action        string `json:"action"`
	UsedRef       string `json:"used_ref"`
	LatestRelease string `json:"latest_release"`
	Outdated      bool   `json:"outdated"`
	Status        string `json:"status"`
	Note          string `json:"note,omitempty"`
}

// assess compares the pinned ref against the latest release tag and classifies
// the result. Statuses: "up-to-date", "major-pinned", "outdated", "pinned-sha".
func assess(ref, latestTag string) (status string, outdated bool, note string) {
	switch {
	case workflow.IsSHA(ref):
		return "pinned-sha", false, fmt.Sprintf(
			"Pinned to a commit SHA (most secure). Latest release is %s.", latestTag)
	case ref == latestTag:
		return "up-to-date", false, ""
	case isSameMajor(ref, latestTag):
		return "major-pinned", false, fmt.Sprintf(
			"Pinned to major %s, which currently resolves to %s. Fine for receiving patch/minor updates.", ref, latestTag)
	default:
		return "outdated", true, fmt.Sprintf("Newer release available: %s.", latestTag)
	}
}

// isSameMajor reports whether ref is a major-only tag (e.g. "v4") that the
// latest tag (e.g. "v4.2.2") falls under.
func isSameMajor(ref, latestTag string) bool {
	return latestTag == ref || strings.HasPrefix(latestTag, ref+".")
}

// buildWorkflowReport assesses every referenced action against its latest
// release. Lookup failures become an "unknown" row rather than aborting the
// whole report.
func buildWorkflowReport(ctx context.Context, f releaseFetcher, refs []workflow.ActionRef) []actionAssessment {
	report := make([]actionAssessment, 0, len(refs))
	for _, ref := range refs {
		entry := actionAssessment{
			Action:  actionName(ref),
			UsedRef: ref.Ref,
		}
		rel, err := f.LatestRelease(ctx, ref.Owner, ref.Repo)
		if err != nil {
			entry.Status = "unknown"
			entry.Note = describeLookupError(err)
			report = append(report, entry)
			continue
		}
		entry.LatestRelease = rel.TagName
		entry.Status, entry.Outdated, entry.Note = assess(ref.Ref, rel.TagName)
		report = append(report, entry)
	}
	return report
}

func actionName(ref workflow.ActionRef) string {
	name := ref.Owner + "/" + ref.Repo
	if ref.Subpath != "" {
		name += "/" + ref.Subpath
	}
	return name
}

func describeLookupError(err error) string {
	var nf *github.NotFoundError
	if errors.As(err, &nf) {
		return "No releases found, or repository does not exist. The action may use tags without GitHub releases."
	}
	return "Could not determine latest release: " + err.Error()
}
