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
# Full runs take turns through the lock, so one fixed tag cannot collide with another full run, and
# the next full run's build replaces an image a killed run left behind.
full_run_image=databasus-full-test:local
is_full_run_image_built=false

suite_names=()
declare -A suite_process_ids=()
declare -A suite_outcomes=()
durations_directory=""

get_run_resource_ids() {
    local run_label
    for run_label in databasus.test-run databasus.test-root-run; do
        docker "$@" --filter "label=$run_label=$DATABASUS_TEST_RUN_ID" || return
    done | LC_ALL=C sort -u
}

# Ryuk is disabled, so nothing else removes the containers an interrupted run leaves behind.
# The shared compose fixtures stay up for the next run and for other worktrees.
cleanup() {
    trap - EXIT INT TERM
    pkill -TERM -P $$ 2>/dev/null || true
    # Containers are removed only after their tests stopped, so none appears after the removal.
    for suite_name in "${suite_names[@]}"; do
        if [[ ! -v "suite_process_ids[$suite_name]" ]]; then
            continue
        fi
        local process_id=${suite_process_ids[$suite_name]}
        kill -TERM -- "-$process_id" 2>/dev/null || true
        wait "$process_id" 2>/dev/null || true
    done
    local ids
    ids=$(get_run_resource_ids ps -aq)
    if [[ -n "$ids" ]]; then
        # shellcheck disable=SC2086 # one container ID per word
        docker rm -f $ids >/dev/null
    fi
    ids=$(get_run_resource_ids network ls -q)
    if [[ -n "$ids" ]]; then
        # shellcheck disable=SC2086 # one network ID per word
        docker network rm $ids >/dev/null
    fi
    ids=$(get_run_resource_ids volume ls -q)
    if [[ -n "$ids" ]]; then
        # shellcheck disable=SC2086 # one volume name per word
        docker volume rm -f $ids >/dev/null
    fi
    if [[ "$is_full_run_image_built" == true ]]; then
        docker image rm "$full_run_image" >/dev/null 2>&1 || true
    fi
    if [[ -n "$durations_directory" ]]; then
        rm -rf "$durations_directory"
    fi
}

exec {full_run_lock_fd}>"$full_run_lock"
if ! flock -n "$full_run_lock_fd"; then
    echo "waiting for another test run to finish"
    flock "$full_run_lock_fd"
fi
SECONDS=0

trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

if [[ -f "$root/.env" ]]; then
    set -a
    # shellcheck source=/dev/null # .env is local and untracked, so CI has nothing to follow
    source "$root/.env"
    set +a
fi

DATABASUS_TEST_ROOT_RUN_ID=$DATABASUS_TEST_RUN_ID
export DATABASUS_TEST_ROOT_RUN_ID

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

durations_directory=$(mktemp -d "$lock_directory/databasus-test-durations.XXXXXX")

# Runs one suite in the background with its lines marked by the suite's name, and records how many
# seconds it took. The process substitution keeps $! on the suite itself, whose exit status decides
# the run. Its own session lets an interruption reach the test binaries and workers the suite started.
start_suite() {
    local suite_name=$1
    shift
    # shellcheck disable=SC2016 # expanded by the suite's own shell
    setsid bash -c 'SECONDS=0; "${@:2}"; status=$?; echo "$SECONDS" >"$1"; exit "$status"' \
        suite "$durations_directory/$suite_name" "$@" > >(sed -u "s/^/[$suite_name] /") 2>&1 &
    suite_process_ids[$suite_name]=$!
    suite_names+=("$suite_name")
}

finish_suite() {
    local suite_name=$1
    if [[ -v "suite_outcomes[$suite_name]" ]]; then
        return
    fi
    if wait "${suite_process_ids[$suite_name]}"; then
        suite_outcomes[$suite_name]=passed
    else
        suite_outcomes[$suite_name]=failed
    fi
}

mark_suite() {
    local suite_name=$1
    local outcome=$2
    suite_names+=("$suite_name")
    suite_outcomes[$suite_name]=$outcome
}

print_summary() {
    local failed_suites=()
    local not_started_suites=()
    echo "summary:"
    for suite_name in "${suite_names[@]}"; do
        local outcome=${suite_outcomes[$suite_name]}
        local duration=""
        if [[ -f "$durations_directory/$suite_name" ]]; then
            duration="$(<"$durations_directory/$suite_name")s"
        fi
        printf '  %-32s %-12s %s\n' "$suite_name" "$outcome" "$duration"
        case "$outcome" in
            failed) failed_suites+=("$suite_name") ;;
            "not started") not_started_suites+=("$suite_name") ;;
        esac
    done
    echo "total: ${SECONDS}s"

    if ((${#not_started_suites[@]} > 0)); then
        echo "not started: ${not_started_suites[*]}" >&2
    fi
    if ((${#failed_suites[@]} > 0)); then
        echo "failed: ${failed_suites[*]}" >&2
    fi
    ((${#failed_suites[@]} == 0 && ${#not_started_suites[@]} == 0))
}

pnpm --dir "$root/frontend" install --frozen-lockfile

# Slow packages mostly wait on containers, and the Docker daemon is their shared bottleneck. Measured
# with the end-to-end packages on the reference machine, -parallel=8 finishes the group in about 8
# minutes with no failure, while Go's default of one test per logical processor runs dozens of Patroni
# clusters at once: container stops and network removals then outlast their deadlines, the group
# fails and leaks clusters, and it takes longer. -parallel=16 is no faster and failed a recovery
# scenario. Eight packages at once keep the longest package on the critical path; a higher package
# limit adds memory without shortening the group. The others mostly compute.
if ((${#slow_packages[@]} > 0)); then
    start_suite backend-slow env TESTCONTAINERS_RYUK_DISABLED=true \
        go test ${CI:+-race} -p=8 -parallel=8 -count=1 -timeout 15m "${slow_packages[@]}"
fi

if ((${#other_packages[@]} > 0)); then
    start_suite backend-other env TESTCONTAINERS_RYUK_DISABLED=true \
        go test ${CI:+-race} -p=16 -count=1 -timeout 15m "${other_packages[@]}"
fi

start_suite frontend pnpm --dir "$root/frontend" test

if [[ "$is_full_run" == true ]]; then
    cd "$root"
    # A crashed-run probe expects its own later run to remove the run it crashed, and the interruption
    # checks expect the same of the lifecycle command they kill. Any browser run starting meanwhile
    # would remove it first, so these run while no other browser run of this command is active. They
    # run one probe after another and take longer than every browser run that may not overlap them,
    # so they start first. Measured on the reference machine, the run then took about 14 minutes,
    # and 14 to 15.5 minutes when they waited for the other browser runs instead.
    start_suite lifecycle-crashed-runs mise run e2e:lifecycle --only-crashed-run-probes

    start_suite scripts python3 -m unittest discover -s scripts -p '*_test.py'
    start_suite review-hooks python3 -m unittest discover -s .agent-hooks -p '*_test.py'
    start_suite e2e-unit mise run e2e:unit
    start_suite browser-ui pnpm --dir frontend test:browser --project=ui

    tested_image=${DATABASUS_IMAGE:-}
    if [[ -z "$tested_image" ]]; then
        tested_image=$full_run_image
        is_full_run_image_built=true
        start_suite image-build docker build --tag "$tested_image" "$root"
        finish_suite image-build
    else
        mark_suite image-build skipped
    fi

    is_image_ready=true
    if [[ "${suite_outcomes[image-build]}" == failed ]]; then
        is_image_ready=false
        mark_suite upgrade "not started"
    else
        start_suite upgrade env DATABASUS_IMAGE="$tested_image" mise run backend:test-upgrade
    fi

    finish_suite lifecycle-crashed-runs
    start_suite lifecycle-interruption-checks env PYTHONDONTWRITEBYTECODE=1 UV_PYTHON_PREFERENCE=only-system \
        uv run --locked --project e2e/browser pytest e2e/browser/lifecycle_interruption
    finish_suite lifecycle-interruption-checks

    if [[ "$is_image_ready" == true ]]; then
        start_suite e2e-browser env DATABASUS_IMAGE="$tested_image" mise run e2e:browser
    else
        mark_suite e2e-browser "not started"
    fi
    # One process runs its lifecycle probes one after another, so the probes without crashed runs
    # are split into parts that run at once. Measured on the reference machine, seven parts take
    # about two minutes together, near the single longest probe, where the three lifecycle groups
    # took four.
    lifecycle_shard_count=7
    for ((shard = 1; shard <= lifecycle_shard_count; shard++)); do
        start_suite "lifecycle-$shard-of-$lifecycle_shard_count" mise run e2e:lifecycle \
            --without-crashed-run-probes --shard "$shard/$lifecycle_shard_count"
    done
fi

for suite_name in "${suite_names[@]}"; do
    finish_suite "$suite_name"
done

print_summary
