#!/usr/bin/env bash
#
# Registers the gha-mcp MCP server with the AI tools installed on this machine.
#
# Adds a "github-actions" server to Claude Desktop, Cursor and VS Code, and to
# Claude Code via its CLI. Existing servers are preserved: the JSON is read, one
# key is added or replaced, and written back. Each file is backed up to *.bak.
#
# Only tools that are actually installed are touched — a client is detected by
# the presence of its configuration directory.
#
# Usage:
#   ./configure-ai-clients.sh [--binary PATH] [--token TOKEN] [--name NAME] [--uninstall]

set -euo pipefail

BINARY=""
TOKEN=""
SERVER_NAME="github-actions"
UNINSTALL=0

while [ $# -gt 0 ]; do
  case "$1" in
    --binary) BINARY="${2:?--binary needs a path}"; shift 2 ;;
    --token) TOKEN="${2:?--token needs a value}"; shift 2 ;;
    --name) SERVER_NAME="${2:?--name needs a value}"; shift 2 ;;
    --uninstall) UNINSTALL=1; shift ;;
    -h|--help) sed -n '2,14p' "$0"; exit 0 ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
done

ok()   { printf '  \033[32m[ok]\033[0m %s\n' "$1"; }
skip() { printf '  \033[90m[--]\033[0m %s\n' "$1"; }
warn() { printf '  \033[33m[!!]\033[0m %s\n' "$1"; }

PYTHON="$(command -v python3 || command -v python || true)"
if [ -z "$PYTHON" ]; then
  echo "error: python3 is required to edit the JSON config files safely." >&2
  exit 1
fi

resolve_binary() {
  if [ -n "$BINARY" ]; then
    [ -x "$BINARY" ] || { echo "error: '$BINARY' is not executable" >&2; exit 1; }
    ( cd "$(dirname "$BINARY")" && printf '%s/%s\n' "$(pwd)" "$(basename "$BINARY")" )
    return
  fi

  local sibling
  sibling="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/gha-mcp"
  if [ -x "$sibling" ]; then printf '%s\n' "$sibling"; return; fi

  if command -v gha-mcp >/dev/null 2>&1; then command -v gha-mcp; return; fi

  echo "error: could not find gha-mcp. Pass --binary with its full path." >&2
  exit 1
}

case "$(uname -s)" in
  Darwin)
    CLAUDE_DIR="$HOME/Library/Application Support/Claude"
    VSCODE_DIR="$HOME/Library/Application Support/Code/User"
    ;;
  *)
    CLAUDE_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/Claude"
    VSCODE_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/Code/User"
    ;;
esac
CURSOR_DIR="$HOME/.cursor"

# update_config <label> <config file> <root key> <detect dir> <with type: 0|1>
update_config() {
  local label="$1" file="$2" root="$3" detect="$4" with_type="$5"

  if [ ! -d "$detect" ]; then
    skip "$label not installed"
    return 0
  fi

  [ -f "$file" ] && cp -f "$file" "$file.bak"
  mkdir -p "$(dirname "$file")"

  if SERVER_NAME="$SERVER_NAME" BINARY="$RESOLVED_BINARY" TOKEN="$TOKEN" \
     ROOT="$root" WITH_TYPE="$with_type" UNINSTALL="$UNINSTALL" CONFIG="$file" \
     "$PYTHON" - <<'PY'
import json, os, sys

path = os.environ["CONFIG"]
root = os.environ["ROOT"]
name = os.environ["SERVER_NAME"]
uninstall = os.environ["UNINSTALL"] == "1"

try:
    with open(path, encoding="utf-8") as fh:
        text = fh.read().strip()
    data = json.loads(text) if text else {}
except FileNotFoundError:
    data = {}
except json.JSONDecodeError:
    sys.exit(3)  # comments or malformed JSON — leave it alone

if not isinstance(data, dict):
    sys.exit(3)

servers = data.setdefault(root, {})
if not isinstance(servers, dict):
    sys.exit(3)

if uninstall:
    if name not in servers:
        sys.exit(4)  # nothing to do
    del servers[name]
else:
    entry = {}
    if os.environ["WITH_TYPE"] == "1":
        entry["type"] = "stdio"
    entry["command"] = os.environ["BINARY"]
    if os.environ["TOKEN"]:
        entry["env"] = {"GITHUB_TOKEN": os.environ["TOKEN"]}
    servers[name] = entry

with open(path, "w", encoding="utf-8") as fh:
    json.dump(data, fh, indent=2)
    fh.write("\n")
PY
  then
    if [ "$UNINSTALL" = "1" ]; then ok "$label - removed '$SERVER_NAME'"; else ok "$label - configured ($file)"; fi
    CHANGED=$((CHANGED + 1))
  else
    case $? in
      3) warn "$label - could not parse $file (comments in the file?). Configure it by hand." ;;
      4) skip "$label - nothing to remove" ;;
      *) warn "$label - failed to update $file" ;;
    esac
  fi
}

update_claude_code() {
  if ! command -v claude >/dev/null 2>&1; then
    skip "Claude Code not installed"
    return 0
  fi

  claude mcp remove "$SERVER_NAME" >/dev/null 2>&1 || true
  if [ "$UNINSTALL" = "1" ]; then
    ok "Claude Code - removed '$SERVER_NAME'"
    CHANGED=$((CHANGED + 1))
    return 0
  fi

  local args=(mcp add "$SERVER_NAME")
  [ -n "$TOKEN" ] && args+=(--env "GITHUB_TOKEN=$TOKEN")
  args+=(-- "$RESOLVED_BINARY")

  if claude "${args[@]}" >/dev/null 2>&1; then
    ok "Claude Code - configured"
    CHANGED=$((CHANGED + 1))
  else
    warn "Claude Code - 'claude mcp add' failed"
  fi
}

CHANGED=0
RESOLVED_BINARY=""

if [ "$UNINSTALL" = "1" ]; then
  printf '\nRemoving the '\''%s'\'' MCP server from your AI tools...\n' "$SERVER_NAME"
else
  RESOLVED_BINARY="$(resolve_binary)"
  printf '\nRegistering gha-mcp with your AI tools...\n'
  printf '  binary: %s\n' "$RESOLVED_BINARY"
  [ -z "$TOKEN" ] && printf '  no token given - the server will use the anonymous 60 req/h limit\n'
fi

update_config "Claude Desktop" "$CLAUDE_DIR/claude_desktop_config.json" "mcpServers" "$CLAUDE_DIR" 0
update_config "Cursor"         "$CURSOR_DIR/mcp.json"                   "mcpServers" "$CURSOR_DIR" 0
update_config "VS Code"        "$VSCODE_DIR/mcp.json"                   "servers"    "$VSCODE_DIR" 1
update_claude_code

printf '\n'
if [ "$CHANGED" -eq 0 ]; then
  warn "No supported AI tools were found. Nothing was changed."
else
  printf 'Done - %d tool(s) updated. Restart them to pick up the change.\n' "$CHANGED"
fi
