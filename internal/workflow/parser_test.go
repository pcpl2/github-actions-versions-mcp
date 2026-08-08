package workflow

import "testing"

func TestParseActions(t *testing.T) {
	content := `
name: CI
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4.0.1
      - name: cached action
        uses: docker/build-push-action@1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b
      - uses: ./.github/actions/local
      - uses: docker://alpine:3.20
      - uses: actions/checkout/sub/path@v3
`
	refs := ParseActions(content)

	if len(refs) != 4 {
		t.Fatalf("got %d refs, want 4 (local and docker:// ignored)\n%+v", len(refs), refs)
	}

	want := []ActionRef{
		{Owner: "actions", Repo: "checkout", Ref: "v4"},
		{Owner: "actions", Repo: "setup-node", Ref: "v4.0.1"},
		{Owner: "docker", Repo: "build-push-action", Ref: "1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b"},
		{Owner: "actions", Repo: "checkout", Subpath: "sub/path", Ref: "v3"},
	}
	for i, w := range want {
		g := refs[i]
		if g.Owner != w.Owner || g.Repo != w.Repo || g.Ref != w.Ref || g.Subpath != w.Subpath {
			t.Errorf("ref[%d] = %+v, want %+v", i, g, w)
		}
	}
}

func TestParseActionsDeduplicates(t *testing.T) {
	content := `
      - uses: actions/checkout@v4
      - uses: actions/checkout@v4
`
	refs := ParseActions(content)
	if len(refs) != 1 {
		t.Fatalf("got %d, want 1 (identical refs deduplicated)", len(refs))
	}
}

func TestIsSHA(t *testing.T) {
	cases := map[string]bool{
		"1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b": true, // 40-char full SHA
		"1a2b3c4": true, // 7-char short SHA
		"v4":      false,
		"v4.2.2":  false,
		"main":    false,
	}
	for ref, want := range cases {
		if got := IsSHA(ref); got != want {
			t.Errorf("IsSHA(%q) = %v, want %v", ref, got, want)
		}
	}
}
