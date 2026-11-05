#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
source "$root/scripts/fedora.sh"

is_full_run=false
if [[ "${1:-}" == --full ]]; then
    is_full_run=true
fi

lock_directory=${TMPDIR:-/tmp}
# `mise run test` and `mise run test-full` take turns; a single package run never takes this lock.
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
    # Containers are removed only after their tests stopped, so none appears after the removal.
    for process_id in "${process_ids[@]}"; do
        kill -TERM -- "-$process_id" 2>/dev/null || true
        wait "$process_id" 2>/dev/null || true
    done
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

process_ids=()
suite_names=()

exec {full_run_lock_fd}>"$full_run_lock"
if ! flock -n "$full_run_lock_fd"; then
    echo "waiting for another test run to finish"
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
docker compose up -d --no-recreate backend-db-test
timeout 60 bash -c 'until docker exec backend-db-test pg_isready -U postgres >/dev/null; do sleep 2; done'

mise run backend:pull-testcontainers

cd "$root/backend"
discovered_packages=$(go list ./...)
excluded_packages='^databasus-backend/(e2e/upgrade|internal/agent/e2e|internal/modules/backups/e2e)(/|$)'
if [[ "$is_full_run" == true ]]; then
    excluded_packages='^databasus-backend/e2e/upgrade(/|$)'
fi
mapfile -t backend_packages < <(grep -Ev "$excluded_packages" <<<"$discovered_packages")
mapfile -t listed_slow_packages < <(grep -Ev '^(#|$)' test-slow-packages.txt)

# An entry this run does not select is skipped. scripts/test_sh_test.py names one that neither command
# selects, because here it would only stop a whole run.
declare -A is_backend_package=()
for package in "${backend_packages[@]}"; do
    is_backend_package[$package]=1
done

declare -A is_slow_package=()
slow_packages=()
for package in "${listed_slow_packages[@]}"; do
    if [[ -v "is_backend_package[$package]" ]]; then
        is_slow_package[$package]=1
        slow_packages+=("$package")
    fi
done

other_packages=()
for package in "${backend_packages[@]}"; do
    if [[ ! -v "is_slow_package[$package]" ]]; then
        other_packages+=("$package")
    fi
done

# Runs one suite in the background with its lines marked by the suite's name. The process
# substitution keeps $! on the suite itself, whose exit status decides the run. Its own session lets
# an interruption reach the test binaries and workers the suite started.
start_suite() {
    local suite_name=$1
    shift
    setsid "$@" > >(sed -u "s/^/[$suite_name] /") 2>&1 &
    process_ids+=("$!")
    suite_names+=("$suite_name")
}

pnpm --dir "$root/frontend" install --frozen-lockfile

# Slow packages mostly wait on containers. Their limit stays low, because above it the run gets no
# faster and Docker runs out of network address pools. The others mostly compute.
if ((${#slow_packages[@]} > 0)); then
    start_suite backend-slow env TESTCONTAINERS_RYUK_DISABLED=true \
        go test ${CI:+-race} -p=8 -count=1 -timeout 15m "${slow_packages[@]}"
fi

if ((${#other_packages[@]} > 0)); then
    start_suite backend-other env TESTCONTAINERS_RYUK_DISABLED=true \
        go test ${CI:+-race} -p=16 -count=1 -timeout 15m "${other_packages[@]}"
fi

start_suite frontend pnpm --dir "$root/frontend" test

failed_suites=()
for index in "${!process_ids[@]}"; do
    if ! wait "${process_ids[$index]}"; then
        failed_suites+=("${suite_names[$index]}")
    fi
done

process_ids=()

if ((${#failed_suites[@]} > 0)); then
    echo "failed: ${failed_suites[*]}" >&2
    exit 1
fi

if [[ "$is_full_run" == true ]]; then
    cd "$root"
    python3 -m unittest discover -s scripts -p '*_test.py'
    python3 -m unittest discover -s .agent-hooks -p '*_test.py'
    mise run e2e:unit

    # Image and browser suites run sequentially because they disturb each other's timing under load.
    pnpm --dir frontend test:browser --project=ui
    mise run backend:test-upgrade
    mise run e2e:browser
    mise run e2e:lifecycle
    PYTHONDONTWRITEBYTECODE=1 UV_PYTHON_PREFERENCE=only-system \
        uv run --locked --project e2e/browser pytest e2e/browser/lifecycle_interruption
fi
