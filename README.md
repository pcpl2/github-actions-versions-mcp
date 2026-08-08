# github-actions-versions-mcp

[![CI](https://github.com/pcpl2/github-actions-versions-mcp/actions/workflows/ci.yml/badge.svg)](https://github.com/pcpl2/github-actions-versions-mcp/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/pcpl2/github-actions-versions-mcp?sort=semver)](https://github.com/pcpl2/github-actions-versions-mcp/releases/latest)
[![Go Reference](https://pkg.go.dev/badge/github.com/pcpl2/github-actions-versions-mcp.svg)](https://pkg.go.dev/github.com/pcpl2/github-actions-versions-mcp)
[![Go Report Card](https://goreportcard.com/badge/github.com/pcpl2/github-actions-versions-mcp)](https://goreportcard.com/report/github.com/pcpl2/github-actions-versions-mcp)
[![License](https://img.shields.io/badge/license-BSD--2--Clause-blue.svg)](LICENSE)

A [Model Context Protocol](https://modelcontextprotocol.io) server that reports the
**latest released versions of GitHub Actions**, using the GitHub REST API. Ask your
MCP client "is `actions/checkout@v3` up to date?" and it will tell you the newest
release and whether your pin is stale.

Written in Go, speaks MCP over **stdio**, so it can be launched directly by clients
such as Claude Code and Claude Desktop.

```
> Check the actions in .github/workflows/ci.yml

  actions/checkout          v7  → v7.0.1   major-pinned
  actions/setup-go          v7  → v7.0.0   major-pinned
  golangci/golangci-lint-…  v9  → v9.3.0   major-pinned
  5 action(s) found, 0 outdated.
```

## Tools

| Tool | Arguments | Returns |
|------|-----------|---------|
| `latest_release` | `owner`, `repo` | Latest release: `tag_name`, `published_at`, `html_url`, `prerelease`, plus pinning advice. Uses `GET /repos/{owner}/{repo}/releases/latest`. |
| `list_releases` | `owner`, `repo`, `limit` (default 10) | The most recent releases, newest first. Uses `GET /repos/{owner}/{repo}/releases`. |
| `check_workflow_actions` | `content` (workflow YAML) | For every `uses:` action, whether the pinned ref is the latest release, with a per-action status. |

### Version status semantics

`check_workflow_actions` classifies each pinned ref:

- **`up-to-date`** — pinned to the exact latest release tag (e.g. `@v4.2.2`).
- **`major-pinned`** — pinned to a major tag (e.g. `@v4`) that still resolves to the
  latest release. Recommended: you keep receiving patch/minor updates automatically.
- **`pinned-sha`** — pinned to a commit SHA (most secure against supply-chain attacks).
- **`outdated`** — a newer release exists (includes branch pins like `@main`).
- **`unknown`** — no releases found, or the repo does not exist. Some actions ship
  tags without GitHub releases.

## Install

### Homebrew (macOS / Linux)

```bash
brew install pcpl2/tap/gha-mcp
```

### Scoop (Windows)

```powershell
scoop bucket add pcpl2 https://github.com/pcpl2/scoop-bucket
scoop install gha-mcp
```

### Linux packages

Download the `.deb`, `.rpm` or `.apk` for your architecture from the
[latest release](https://github.com/pcpl2/github-actions-versions-mcp/releases/latest):

```bash
sudo dpkg -i gha-mcp_*_linux_amd64.deb     # Debian / Ubuntu
sudo rpm -i gha-mcp_*_linux_amd64.rpm      # Fedora / RHEL / openSUSE
sudo apk add --allow-untrusted gha-mcp_*_linux_amd64.apk   # Alpine
```

### Pre-built archives

Grab a `tar.gz` (Linux/macOS) or `zip` (Windows) for `amd64` or `arm64` from the
[releases page](https://github.com/pcpl2/github-actions-versions-mcp/releases/latest),
extract it, and put `gha-mcp` on your `PATH`. Every release ships `checksums.txt`
and a per-archive SBOM:

```bash
sha256sum -c checksums.txt --ignore-missing
```

### From source

Requires the Go version declared in [`go.mod`](go.mod).

```bash
go install github.com/pcpl2/github-actions-versions-mcp@latest   # installs as github-actions-versions-mcp
# or
git clone https://github.com/pcpl2/github-actions-versions-mcp
cd github-actions-versions-mcp
go build -o gha-mcp .
```

Verify the install:

```bash
gha-mcp --version
```

## Configure your MCP client

### Claude Code

```bash
claude mcp add github-actions -- gha-mcp
```

Use an absolute path instead of `gha-mcp` if the binary is not on your `PATH`.
To raise the API rate limit, pass a token through:

```bash
claude mcp add github-actions --env GITHUB_TOKEN=ghp_xxx -- gha-mcp
```

### Claude Desktop

Add to `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "github-actions": {
      "command": "/absolute/path/to/gha-mcp",
      "env": {
        "GITHUB_TOKEN": "ghp_xxx"
      }
    }
  }
}
```

## Authentication & rate limits

The server works **anonymously** (60 requests/hour). Set the `GITHUB_TOKEN`
environment variable to a personal access token (no scopes needed for public repos)
to raise the limit to **5000 requests/hour**. When the limit is hit, the tool returns
a clear error explaining how to raise it.

`check_workflow_actions` makes one API call per distinct action, so a large workflow
can exhaust the anonymous budget in a few runs.

## Pinning recommendations

In a real workflow, prefer:

1. **A commit SHA** (`@1a2b3c…`) for maximum supply-chain security, or
2. **A major tag** (`@v4`) for automatic patch/minor updates.

Avoid pinning to a branch like `@main` — it is not reproducible and can change under you.

## Development

```bash
go test ./...            # unit + in-memory integration tests (no network required)
go vet ./...
golangci-lint run ./...  # config in .golangci.yml
```

Tests use `httptest` for the GitHub client and an in-memory MCP transport for the
tools, so the suite is fully offline and deterministic.

See [CLAUDE.md](CLAUDE.md) for the architecture notes and project conventions.

## Releasing

Releases are cut by [GoReleaser](https://goreleaser.com) from a pushed tag:

```bash
git tag -a v1.0.0 -m "v1.0.0"
git push origin v1.0.0
```

[`.github/workflows/release.yml`](.github/workflows/release.yml) then builds
`linux`/`darwin`/`windows` × `amd64`/`arm64`, and publishes archives, Linux
packages, checksums and SBOMs to the GitHub Release.

Publishing the Homebrew cask and the Scoop manifest additionally requires two
public repositories — `pcpl2/homebrew-tap` and `pcpl2/scoop-bucket` — and a
`TAP_GITHUB_TOKEN` repository secret holding a PAT with `contents: write` on them.
Without that secret those two steps are skipped and the rest of the release still
publishes. Validate config changes locally with:

```bash
goreleaser check
goreleaser release --snapshot --clean --skip=publish,announce
```

## Contributing

Issues and pull requests are welcome. Please keep the test suite offline, run
`golangci-lint run ./...` before opening a PR, and use
[Conventional Commits](https://www.conventionalcommits.org) (`feat:`, `fix:`, …) —
release notes are generated from them.

For anything you would rather not discuss in a public issue — including suspected
security problems — mail <open-source@pcpl2.ovh>.

## License

[BSD 2-Clause](LICENSE) © Patryk Ławicki
