<img src="assets/icon.png" alt="" width="88" align="right">

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

## Download

Every release ships pre-built binaries — there is nothing to compile. Open the
[**latest release**](https://github.com/pcpl2/github-actions-versions-mcp/releases/latest)
and download the file matching your system:

| System | Chip | File |
|--------|------|------|
| **Windows** | Intel / AMD | **`gha-mcp_<version>_windows_amd64_setup.exe`** ← installer |
| **Windows** | ARM | **`gha-mcp_<version>_windows_arm64_setup.exe`** ← installer |
| Windows (no installer) | Intel / AMD | `gha-mcp_<version>_windows_amd64.zip` |
| Windows (no installer) | ARM | `gha-mcp_<version>_windows_arm64.zip` |
| Linux | Intel / AMD | `gha-mcp_<version>_linux_amd64.tar.gz` |
| Linux | ARM (Raspberry Pi, Ampere) | `gha-mcp_<version>_linux_arm64.tar.gz` |
| macOS | Apple Silicon (M1–M4) | `gha-mcp_<version>_darwin_arm64.tar.gz` |
| macOS | Intel | `gha-mcp_<version>_darwin_amd64.tar.gz` |

Not sure which chip you have? On macOS run `uname -m` (`arm64` or `x86_64`); on
Linux run `uname -m` (`aarch64` or `x86_64`); on Windows check
Settings → System → About → System type.

Debian, Fedora and Alpine users can skip the archive and install a system
package instead — see [Linux packages](#linux-packages) below.

## Install

### Windows (installer)

Run `gha-mcp_<version>_windows_<arch>_setup.exe`. It installs per-user, so it
never asks for administrator rights, and offers two optional steps:

- **Add gha-mcp to my PATH** — so `gha-mcp` works in any terminal.
- **Configure my AI tools automatically** — registers the server with Claude
  Desktop, Claude Code, Cursor and VS Code, whichever are installed. Existing MCP
  servers are left alone, and each config is backed up first.

Uninstalling from Settings → Apps reverses both: the MCP entries are removed and
the folder comes back out of `PATH`.

The installer is unsigned, so SmartScreen may warn on first run — choose
**More info → Run anyway**.

### Linux and macOS

```bash
# 1. extract (adjust the filename to what you downloaded)
tar -xzf gha-mcp_*_linux_amd64.tar.gz

# 2. install onto your PATH
sudo install -m 755 gha-mcp /usr/local/bin/gha-mcp

# 3. check it works
gha-mcp --version
```

On macOS the binary is not code-signed, so Gatekeeper quarantines it. Clear that
once, after installing:

```bash
xattr -dr com.apple.quarantine /usr/local/bin/gha-mcp
```

No root access? Extract anywhere and use the absolute path in your MCP client
config — the server never needs to be on your `PATH`.

### Windows (zip)

```powershell
# 1. extract into a folder you control
Expand-Archive .\gha-mcp_*_windows_amd64.zip -DestinationPath "$env:LOCALAPPDATA\Programs\gha-mcp"

# 2. add that folder to your user PATH (new terminals pick it up)
[Environment]::SetEnvironmentVariable(
  "Path",
  [Environment]::GetEnvironmentVariable("Path", "User") + ";$env:LOCALAPPDATA\Programs\gha-mcp",
  "User")

# 3. check it works (in a NEW terminal)
gha-mcp --version
```

The binary is unsigned, so SmartScreen may warn on first run — choose
**More info → Run anyway**, or skip `PATH` entirely and point your MCP client at
the full path to `gha-mcp.exe`.

### Linux packages

Prefer your package manager? Download the `.deb`, `.rpm` or `.apk` for your
architecture from the
[latest release](https://github.com/pcpl2/github-actions-versions-mcp/releases/latest):

```bash
sudo dpkg -i gha-mcp_*_linux_amd64.deb                     # Debian / Ubuntu
sudo rpm -i gha-mcp_*_linux_amd64.rpm                      # Fedora / RHEL / openSUSE
sudo apk add --allow-untrusted gha-mcp_*_linux_amd64.apk   # Alpine
```

These install `gha-mcp` to `/usr/bin`, already on your `PATH`.

### Verifying your download

Every release includes `checksums.txt` and an SBOM per archive. Verifying is
optional but takes a second — download `checksums.txt` into the same folder:

```bash
sha256sum -c checksums.txt --ignore-missing        # Linux
shasum -a 256 -c checksums.txt --ignore-missing    # macOS
```

```powershell
# Windows — compare the printed hash against the matching line in checksums.txt
Get-FileHash .\gha-mcp_1.0.0_windows_amd64.zip -Algorithm SHA256
```

### Homebrew and Scoop

Not available yet. Both are configured in
[`.goreleaser.yaml`](.goreleaser.yaml) but disabled, because publishing to a tap
or bucket needs a personal access token this project does not use today.

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

### Automatic setup (recommended)

Every release archive contains a script that registers the server with whichever
supported tools it finds — **Claude Desktop, Claude Code, Cursor and VS Code**.
Servers you already have configured are kept, and each file is backed up to
`*.bak` before it is touched.

```bash
# Linux / macOS — run it from the extracted archive
./configure-ai-clients.sh

# with a token, to raise the rate limit
./configure-ai-clients.sh --token ghp_xxx
```

```powershell
# Windows — the installer offers to do this for you; to run it by hand:
powershell -ExecutionPolicy Bypass -File configure-ai-clients.ps1
powershell -ExecutionPolicy Bypass -File configure-ai-clients.ps1 -GithubToken ghp_xxx
```

Restart the tool afterwards. To undo it, pass `--uninstall` (or `-Uninstall` on
Windows). The Linux/macOS script needs `python3`, which both platforms ship.

Prefer doing it yourself? The per-client instructions below do the same thing.

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

## Changelog

Every release is documented in [CHANGELOG.md](CHANGELOG.md).

## Contributing

Issues and pull requests are welcome. Please keep the test suite offline, run
`golangci-lint run ./...` before opening a PR, use
[Conventional Commits](https://www.conventionalcommits.org) (`feat:`, `fix:`, …),
and add an entry to [CHANGELOG.md](CHANGELOG.md) under `## [Unreleased]`.

[CLAUDE.md](CLAUDE.md) documents the architecture, the conventions and how
releases are cut.

Found a security problem? Please do not open a public issue — follow
[SECURITY.md](SECURITY.md) instead.

## License

[BSD 2-Clause](LICENSE) © Patryk Ławicki
