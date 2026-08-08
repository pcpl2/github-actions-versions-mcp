# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.1.0] - 2026-08-08

### Added

- **Windows installer**, built with [Inno Setup](https://jrsoftware.org/isinfo.php).
  Installs `gha-mcp.exe`, optionally adds it to `PATH`, and can configure your AI
  tools during setup. Published for `amd64` and `arm64`.
- **Automatic AI client setup.** `configure-ai-clients.ps1` (Windows) and
  `configure-ai-clients.sh` (Linux/macOS) register the server with Claude Desktop,
  Claude Code, Cursor and VS Code, preserving any MCP servers already configured.
  Both scripts ship inside every release archive.
- This changelog.

### Changed

- The release workflow takes a version **without** the leading `v` (`1.1.0`), and
  adds it when creating the tag, so tags stay SemVer-compatible.
- The release run is titled with the version, so it is identifiable in the Actions
  list without opening it.
- Release notes are now taken from this changelog instead of being generated only
  from commit messages. A release fails if the changelog has no matching entry.
- README no longer documents the release process — that is contributor material and
  now lives in CLAUDE.md.

## [1.0.0] - 2026-08-08

### Added

- MCP server, over stdio, exposing three tools: `latest_release`, `list_releases`
  and `check_workflow_actions`.
- Version status classification: `up-to-date`, `major-pinned`, `pinned-sha`,
  `outdated` and `unknown`.
- Anonymous GitHub API access, with `GITHUB_TOKEN` support to raise the rate limit
  from 60 to 5000 requests per hour.
- Release pipeline producing archives for linux/darwin/windows on amd64/arm64,
  `.deb`/`.rpm`/`.apk` packages, SBOMs and checksums.
- CI across Linux, macOS and Windows: tests, vet, golangci-lint, tidy check,
  cross-compilation and a GoReleaser dry run.

[Unreleased]: https://github.com/pcpl2/github-actions-versions-mcp/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/pcpl2/github-actions-versions-mcp/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/pcpl2/github-actions-versions-mcp/releases/tag/v1.0.0
