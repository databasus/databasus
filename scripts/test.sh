#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
source "$root/scripts/fedora.sh"

lock_directory=${TMPDIR:-/tmp}
# Full runs take turns; a single package run never takes this lock.
full_run_lock="$lock_directory/databasus-full-test.lock"
# Every test binary holds this lock shared while it runs, so the exclusive side waits for running
# tests before the shared test PostgreSQL servers are reset.
fixtures_lock="$lock_directory/databasus-test-fixtures.lock"
fixture_reset_wait_seconds=${DATABASUS_FIXTURE_RESET_WAIT_SECONDS:-600}
fixture_reset_timeout_status=75

# The containers helpers label every container and network with this ID, so the trap removes only
# what this run started.
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
}

wait_for_port() {
    local port=$1
    timeout 60 bash -c "until (exec 3<>/dev/tcp/127.0.0.1/$port) 2>/dev/null; do sleep 2; done"
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
wait_for_port "${TEST_LOGICAL_POSTGRES_16_PORT:-5004}"
wait_for_port "${TEST_PHYSICAL_POSTGRES_17_PORT:-5007}"
wait_for_port "${TEST_PHYSICAL_POSTGRES_18_PORT:-5008}"

just --justfile "$root/backend/Justfile" pull-testcontainers

cd "$root/backend"
reset_status=0
flock -x -w "$fixture_reset_wait_seconds" -E "$fixture_reset_timeout_status" "$fixtures_lock" \
    go run -tags testhelpers ./cmd/cleanup_test_db || reset_status=$?
if ((reset_status == fixture_reset_timeout_status)); then
    echo "other test runs are using the shared test PostgreSQL servers for over ${fixture_reset_wait_seconds}s;" \
        "they were not reset, rerun after those runs finish" >&2
    exit 1
fi
if ((reset_status != 0)); then
    exit "$reset_status"
fi

TESTCONTAINERS_RYUK_DISABLED=true \
    go test -race -tags testhelpers -p=8 -count=1 -failfast -timeout 15m ./internal/... ./cmd/...
