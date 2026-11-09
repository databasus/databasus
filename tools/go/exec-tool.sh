#!/usr/bin/env bash
set -euo pipefail

root="$(cd "${BASH_SOURCE[0]%/*}/../.." && pwd)"
tool="${0##*/}"
case "$tool" in
  gopls|govulncheck) ;;
  *) echo "unsupported Go development tool: $tool" >&2; exit 2 ;;
esac

tool_path="$(mise -C "$root" run --quiet tools:go "$tool")"
if [[ "$tool_path" != /* || ! -x "$tool_path" ]]; then
  echo "the locked Go development tool is unavailable" >&2
  exit 1
fi

exec "$tool_path" "$@"
