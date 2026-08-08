# Security Policy

## Supported versions

This project follows semantic versioning, and only the latest release receives
fixes. If you are running an older tag, upgrade before reporting a problem.

| Version | Supported |
|---------|-----------|
| Latest release | ✅ |
| Older releases | ❌ |
| `main` branch | Best effort |

## Reporting a vulnerability

**Please do not open a public issue for a security problem.**

Report it privately in one of two ways:

1. **GitHub private vulnerability reporting** (preferred) — go to the
   [Security tab](https://github.com/pcpl2/github-actions-versions-mcp/security)
   of this repository and choose **Report a vulnerability**. The report stays
   private between you and the maintainer until a fix is published.
2. **Email** — <open-source@pcpl2.ovh>. Please include "security" in the subject.

Whichever route you pick, a useful report contains:

- the version (`gha-mcp --version`) and platform,
- what an attacker can achieve, and how you verified it,
- a minimal reproduction — for parser issues, the exact workflow YAML.

### What to expect

- **Acknowledgement** within 7 days.
- An assessment, and a fix or an explanation of why the behaviour is intended,
  within 30 days for confirmed issues.
- Credit in the release notes, unless you prefer to stay anonymous.

This is a small, unfunded open-source project maintained in spare time. There is
no bug bounty, and timelines are targets rather than guarantees.

## Scope

In scope — anything that lets untrusted input compromise the host or leak
credentials, for example:

- **Token exposure.** The server reads `GITHUB_TOKEN` from the environment. It
  must never appear in tool output, logs, or error messages.
- **Untrusted workflow YAML.** `check_workflow_actions` parses YAML supplied by
  whoever is talking to the MCP client. Crashes, unbounded resource use, or
  anything worse from crafted input is in scope.
- **Request forgery.** `owner`/`repo` arguments are interpolated into GitHub API
  paths; input that redirects requests to another host or endpoint is in scope.
- **Supply chain.** Problems with the released artifacts — archives, `.deb`/
  `.rpm`/`.apk` packages, the Homebrew cask, or the Scoop manifest.

Out of scope:

- GitHub API rate limiting (60 requests/hour anonymously) — a documented limit,
  not a denial of service.
- Vulnerabilities in the GitHub API itself; report those to
  [GitHub](https://bounty.github.com).
- Anything that requires an attacker to already control the machine running the
  server, or the MCP client's configuration.

## A note on the binaries

Released macOS and Windows binaries are **not** code-signed or notarized. The
Homebrew cask therefore clears the macOS quarantine attribute on install, and
Windows SmartScreen may warn on first run. Verify downloads against the
`checksums.txt` published with every release:

```bash
sha256sum -c checksums.txt --ignore-missing
```

Each release also ships an SBOM per archive, generated with
[syft](https://github.com/anchore/syft).
