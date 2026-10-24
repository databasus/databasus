#!/usr/bin/env bash
# Starts review_gate.py with the Python and OpenSpec versions pinned in mise.toml, so a Claude Code
# started from an IDE without the mise shims on PATH runs the same hook as a terminal one.
# A hook that cannot run stops the agent with the reason instead of letting the turn pass unreviewed.
# Only bash builtins and coreutils run before mise, because nothing else is guaranteed on PATH yet.

set -uo pipefail

hooks_dir="$(cd "${BASH_SOURCE[0]%/*}" && pwd)"
root="${hooks_dir%/*}"
action="${1:-}"
environment_check_hint="Run \`mise install\` to install the pinned tools and \`mise run doctor\` to check the environment."

find_mise() {
    if command -v mise; then
        return 0
    fi
    if [[ -x "$HOME/.local/bin/mise" ]]; then
        printf '%s\n' "$HOME/.local/bin/mise"
        return 0
    fi
    return 1
}

escape_json() {
    local value="$1"
    value="${value//\\/\\\\}"
    value="${value//\"/\\\"}"
    value="${value//$'\n'/\\n}"
    value="${value//$'\t'/\\t}"
    printf '%s' "$value"
}

stop_agent() {
    local message
    message="$(escape_json "$1")"
    printf '{"continue": false, "stopReason": "%s", "systemMessage": "%s"}\n' "$message" "$message"
    exit 0
}

if ! mise_path="$(find_mise)"; then
    reason="The review hooks could not find the version manager mise on PATH or in \$HOME/.local/bin."
    if [[ "$action" == "doctor" ]]; then
        printf '%s\n' "$reason" >&2
        exit 1
    fi
    stop_agent "$reason Install mise first. $environment_check_hint"
fi

# mise exec runs a command even when a pinned version is missing: python3 and openspec then resolve
# to whatever else PATH holds. So the launcher refuses to start until both are installed.
missing_tools="$("$mise_path" -C "$root" ls --current --missing --no-truncate python npm:@fission-ai/openspec 2>&1)"
missing_check_exit_code=$?
if [[ $missing_check_exit_code -ne 0 || -n "$missing_tools" ]]; then
    if [[ $missing_check_exit_code -ne 0 ]]; then
        reason="The review hooks could not check the pinned tools: mise exited with code $missing_check_exit_code."
    else
        reason="The pinned tools the review hooks need are not installed."
    fi
    reason="$reason
$missing_tools"
    if [[ "$action" == "doctor" ]]; then
        printf '%s\n' "$reason" >&2
        exit 1
    fi
    stop_agent "$reason
$environment_check_hint"
fi

# An installation would not fit the hook timeout.
export MISE_EXEC_AUTO_INSTALL=false
command=("$mise_path" -C "$root" exec -- python3 "$hooks_dir/review_gate.py" "$@")

if [[ "$action" == "doctor" ]]; then
    exec "${command[@]}"
fi

error_log="$(mktemp)"
trap 'rm -f "$error_log"' EXIT

output="$("${command[@]}" 2>"$error_log")"
exit_code=$?
if [[ $exit_code -eq 0 ]]; then
    printf '%s\n' "$output"
    exit 0
fi

error_tail="$(tail -n 20 "$error_log" | tr -d '\000-\010\013-\037')"
stop_agent "The review hook \`${action:-(none)}\` failed with exit code $exit_code, so the review did not run. $environment_check_hint

End of its error output:
${error_tail:-(no output)}"
