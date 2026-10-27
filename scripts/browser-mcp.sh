#!/usr/bin/env bash
# Starts the pinned Chrome DevTools MCP server registered for Claude Code in .mcp.json
# and for Codex in .codex/config.toml. BROWSER_HEADLESS chooses a headless browser (true,
# the default) or a visible window (false); a value in the client's environment wins over .env.
# --isolated gives every session a temporary profile, so parallel sessions never share a lock.
set -euo pipefail

# mise picks the pinned version from the mise.toml of the current directory, and the client may
# start this script from anywhere.
cd "$(dirname "$0")/.."

# .env holds quoted connection strings, so the value is read out of it instead of sourcing it.
read_env_file_value() {
    local name=$1
    [[ -f .env ]] || return 0
    sed -n "s/^${name}=//p" .env | tail -n 1 | tr -d "\"'\r"
}

headless=${BROWSER_HEADLESS:-$(read_env_file_value BROWSER_HEADLESS)}
headless=${headless:-true}

case "$headless" in
    true) headless_flags=(--headless) ;;
    false) headless_flags=() ;;
    *)
        echo "BROWSER_HEADLESS must be true or false, got '$headless'" >&2
        exit 1
        ;;
esac

exec mise exec -- chrome-devtools-mcp --isolated --viewport 1440x900 "${headless_flags[@]}"
