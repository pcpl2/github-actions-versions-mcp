// Package workflow parses GitHub Actions workflow YAML to extract the external
// actions it references (the `uses:` clauses), without depending on a full YAML
// library — a focused regexp is sufficient and avoids a heavy dependency.
package workflow

import (
	"regexp"
)

// ActionRef is a single `uses:` reference to a Marketplace action.
type ActionRef struct {
	Owner   string // e.g. "actions"
	Repo    string // e.g. "checkout"
	Subpath string // e.g. "sub/path" for actions/checkout/sub/path@v3, else ""
	Ref     string // the pinned ref after '@': a tag, branch, or commit SHA
}

// usesPattern matches `uses: owner/repo[/subpath]@ref`, allowing optional
// surrounding quotes. Local (`./...`) and Docker (`docker://...`) references do
// not match because they lack the owner/repo@ref shape.
var usesPattern = regexp.MustCompile(`(?m)uses:\s*["']?([\w.-]+)/([\w.-]+)((?:/[\w./-]+?)?)@([\w.-]+)["']?`)

// shaPattern matches a 7-to-40 character lowercase hex commit SHA.
var shaPattern = regexp.MustCompile(`^[0-9a-f]{7,40}$`)

// ParseActions extracts every Marketplace action reference from workflow YAML,
// in document order, with duplicates removed.
func ParseActions(content string) []ActionRef {
	matches := usesPattern.FindAllStringSubmatch(content, -1)
	refs := make([]ActionRef, 0, len(matches))
	seen := map[string]bool{}
	for _, m := range matches {
		ref := ActionRef{
			Owner:   m[1],
			Repo:    m[2],
			Subpath: trimSlash(m[3]),
			Ref:     m[4],
		}
		key := ref.Owner + "/" + ref.Repo + "/" + ref.Subpath + "@" + ref.Ref
		if seen[key] {
			continue
		}
		seen[key] = true
		refs = append(refs, ref)
	}
	return refs
}

// IsSHA reports whether ref looks like a Git commit SHA rather than a tag or
// branch name.
func IsSHA(ref string) bool {
	return shaPattern.MatchString(ref)
}

func trimSlash(s string) string {
	if len(s) > 0 && s[0] == '/' {
		return s[1:]
	}
	return s
}
