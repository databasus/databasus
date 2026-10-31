#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
source "$root/scripts/fedora.sh"

lock_directory=${TMPDIR:-/tmp}
# Full runs take turns; a single package run never takes this lock.
full_run_lock="$lock_directory/databasus-full-test.lock"
# The containers helpers and the upgrade test label every container, network and volume they create
# with this ID, so the trap removes only what this run started.
DATABASUS_TEST_RUN_ID="$(date +%s)-$$-$RANDOM"
export DATABASUS_TEST_RUN_ID

# Ryuk is disabled, so nothing else removes the containers an interrupted run leaves behind.
# The shared compose fixtures stay up for the next run and for other worktrees.
cleanup() {
    trap - EXIT INT TERM
    pkill -TERM -P $$ 2>/dev/null || true
    local run_filter="label=databasus.test-run=$DATABASUS_TEST_RUN_ID"
    local ids
    ids=$(docker ps -aq --filter "$run_filter")
    if [[ -n "$ids" ]]; then
        # shellcheck disable=SC2086 # one container ID per word
        docker rm -f $ids >/dev/null
    fi
    ids=$(docker network ls -q --filter "$run_filter")
    if [[ -n "$ids" ]]; then
        # shellcheck disable=SC2086 # one network ID per word
        docker network rm $ids >/dev/null
    fi
    ids=$(docker volume ls -q --filter "$run_filter")
    if [[ -n "$ids" ]]; then
        # shellcheck disable=SC2086 # one volume name per word
        docker volume rm -f $ids >/dev/null
    fi
}

exec {full_run_lock_fd}>"$full_run_lock"
if ! flock -n "$full_run_lock_fd"; then
    echo "waiting for another full test run to finish"
    flock "$full_run_lock_fd"
fi

trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

if [[ -f "$root/.env" ]]; then
    set -a
    # shellcheck source=/dev/null # .env is local and untracked, so CI has nothing to follow
    source "$root/.env"
    set +a
fi

apply_libedit_shim

cd "$root"
docker compose up -d
timeout 60 bash -c 'until docker exec backend-db-test pg_isready -U postgres >/dev/null; do sleep 2; done'

mise run backend:pull-testcontainers

cd "$root/backend"
discovered_packages=$(go list ./internal/... ./cmd/...)
mapfile -t backend_packages < <(grep -Ev '/internal/app/browsertests(/|$)' <<<"$discovered_packages")
TESTCONTAINERS_RYUK_DISABLED=true \
    go test -race -p=8 -count=1 -failfast -timeout 30m "${backend_packages[@]}"

# The upgrade test builds the image unless DATABASUS_IMAGE names one, so it runs only after every
# package passed.
mise run backend:test-upgrade
