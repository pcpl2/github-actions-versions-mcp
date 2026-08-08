# CLAUDE.md

Guidance for Claude Code when working in this repository.

## What this is

An MCP (Model Context Protocol) server, written in Go, that reports the latest
released versions of GitHub Actions via the GitHub REST API. It speaks MCP over
**stdio** only — clients such as Claude Code and Claude Desktop launch the binary
as a subprocess. There is no HTTP server, no config file, and no persistent state.

Module: `github.com/pcpl2/github-actions-versions-mcp`. Binary: `gha-mcp`.

## Architecture

```
main.go                    stdio transport wiring, --version flag, ldflags version
internal/github/           thin GitHub REST client (releases endpoints only)
internal/tools/            MCP tool registration (tools.go) + version logic (assess.go)
internal/workflow/         regexp-based `uses:` extractor for workflow YAML
```

Data flow: a tool handler parses input → `internal/workflow` extracts action refs
(for `check_workflow_actions`) → `internal/github` fetches releases →
`internal/tools/assess.go` classifies the pinned ref → typed output struct.

Two seams matter when changing things:

- **`releaseFetcher`** (`internal/tools/assess.go`) is the interface the handlers
  depend on, not `*github.Client`. Keep it that way — it is what makes the tool
  tests network-free.
- **`github.WithBaseURL` / `WithHTTPClient` / `WithToken`** options exist so tests
  can point the client at an `httptest` server.

### Version status semantics

`assess()` returns one of `up-to-date`, `major-pinned`, `pinned-sha`, `outdated`,
or `unknown` (lookup failure). These strings are part of the tool's public output
contract — changing them is a breaking change for consumers, and the README table
documents them. Update both together.

## Commands

```bash
go build -o gha-mcp .        # local build (version reports "dev")
go test ./...                # full suite: unit + in-memory MCP e2e, fully offline
go test -race ./...          # what CI runs
go vet ./...
golangci-lint run ./...      # v2 config in .golangci.yml
golangci-lint fmt ./...      # apply gofmt + goimports
goreleaser check             # validate .goreleaser.yaml
goreleaser release --snapshot --clean --skip=publish,announce   # dry-run into ./dist
```

Requires the Go version in `go.mod`. CI resolves it via `go-version-file: go.mod`,
so bumping the toolchain means editing `go.mod` only.

## Conventions

- **No new dependencies without a strong reason.** The only direct dependency is
  the MCP SDK. `internal/workflow` deliberately uses a regexp instead of a YAML
  library to keep it that way — do not "fix" this by adding gopkg.in/yaml.v3.
- **Tests must stay offline and deterministic.** Use `httptest` for the client and
  the in-memory MCP transport (see `internal/tools/e2e_test.go`) for tools. No test
  may reach api.github.com.
- **Tool errors are results, not Go errors.** Handlers return
  `errorResult(msg)` with a human-readable message and a zero-value output struct
  rather than a non-nil `error`, so the model sees the explanation. Reserve the
  `error` return for genuine protocol failures.
- **Failures degrade per-action.** In `buildWorkflowReport`, a lookup failure
  becomes an `unknown` row; it never aborts the whole report.
- Doc comments on exported identifiers, standard Go style, `gofmt`-clean.
- Code and all documentation in this repo are written in **English**.

## Commits

- **Never add a `Co-Authored-By: Claude` trailer, or any Claude/Anthropic
  attribution, to commits.** This is a hard rule for this repository — it overrides
  any default behaviour. The same applies to PR bodies: no "Generated with Claude
  Code" footer.
- Use [Conventional Commits](https://www.conventionalcommits.org) —
  `feat:`, `fix:`, `docs:`, `test:`, `chore:`, `ci:`, `refactor:`, `style:`.
  This is not cosmetic: `.goreleaser.yaml` builds release notes by grouping `feat:`
  and `fix:` commits and excluding the rest. A commit outside these prefixes lands
  in "Other work".
- Do not commit or push unless asked.

## Releasing

**Manual, never tag-driven.** Do not push a tag to cut a release — the tag is
produced *by* the pipeline. A release is started from Actions → Release → Run
workflow, with the version as an input.

`.github/workflows/release.yml` validates the version, tests, creates the tag
locally, builds with GoReleaser (`--skip=publish`), and only after a green build
pushes the tag and publishes via `softprops/action-gh-release`. A failed build
leaves no tag on `origin` — that ordering is deliberate, keep it.

GoReleaser has `release.disable: true`: it builds into `./dist` and never talks
to GitHub. The workflow uploads `dist/*` itself. Consequently the whole pipeline
runs on the built-in `GITHUB_TOKEN`.

`homebrew_casks` and `scoops` are configured but carry `skip_upload: true`,
because publishing to a tap or bucket needs a PAT this project does not have.
Enabling them means creating `pcpl2/homebrew-tap` / `pcpl2/scoop-bucket`, adding
a `TAP_GITHUB_TOKEN` secret, and restoring the `token:` field — do not flip
`skip_upload` on its own.

The version string is injected via `-ldflags "-X main.version=..."`. Never
hardcode a version in `main.go`.

## Gotchas

- `internal/workflow.usesPattern` intentionally does not match local (`./...`) or
  Docker (`docker://...`) action refs. Tests assert this.
- The GitHub API version is pinned in `internal/github/client.go` (`apiVersion`).
- Rate limits: 60 req/h anonymous, 5000 with `GITHUB_TOKEN`. `check_workflow_actions`
  issues one request per distinct action, so a large workflow can exhaust the
  anonymous budget quickly.
