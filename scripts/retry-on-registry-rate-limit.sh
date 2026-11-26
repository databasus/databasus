#!/usr/bin/env bash
# Usage: scripts/retry-on-registry-rate-limit.sh <command> [<argument>...]
#
# Runs the command and passes its output through. When the command fails and its output names the
# registry's rate limit, the script waits and runs it again; Docker Hub refuses bursts of pushes and
# manifest updates with 429 Too Many Requests. Any other failure, or a failure of the last attempt,
# returns the command's exit status at once.
set -uo pipefail

if (($# == 0)); then
    echo "usage: $0 <command> [<argument>...]" >&2
    exit 2
fi

# The script's test shortens the waits; a release uses the defaults.
read -r -a waits <<<"${REGISTRY_RATE_LIMIT_WAITS:-60 120 240}"
output_file=$(mktemp)
trap 'rm -f "$output_file"' EXIT

attempt=1
while true; do
    "$@" 2>&1 | tee "$output_file"
    status=${PIPESTATUS[0]}
    if ((status == 0)); then
        exit 0
    fi
    if ! grep -qiE '429 Too Many Requests|toomanyrequests' "$output_file"; then
        exit "$status"
    fi
    if ((attempt > ${#waits[@]})); then
        echo "the registry still refuses with its rate limit after $attempt attempts" >&2
        exit "$status"
    fi

    wait_seconds=${waits[attempt - 1]}
    echo "the registry refused with its rate limit; attempt $((attempt + 1)) in ${wait_seconds}s" >&2
    sleep "$wait_seconds"
    attempt=$((attempt + 1))
done
