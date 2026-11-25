#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)

usage="usage: scripts/test.sh [--change <change>]"
# Without arguments the run selects every test; with --change it runs the selection of that
# OpenSpec change's test plan (docs/testing.md, "Test plans") on the same schedule.
change_name=""
if [[ "${1:-}" == --change ]]; then
    if (($# != 2)); then
        echo "$usage" >&2
        exit 2
    fi
    change_name=$2
elif (($# > 0)); then
    echo "$usage" >&2
    exit 2
fi

is_change_run=false
if [[ -n "$change_name" ]]; then
    is_change_run=true
fi

# The run restarts itself with its output copied to temp/, so a long run can be read after the
# terminal scrolled past it. The copying tee processes stay outside the run, whose cleanup kills every
# child it has, and they ignore Ctrl+C, so the cleanup output of an interrupted run reaches the file.
if [[ ! -v DATABASUS_TEST_LOG_PATH ]]; then
    log_name="test"
    if [[ "$is_change_run" == true ]]; then
        log_name="test-change-${change_name//[^a-z0-9-]/_}"
    fi
    mkdir -p "$root/temp"
    DATABASUS_TEST_LOG_PATH="$root/temp/$log_name-$(date +%Y%m%d-%H%M%S).txt"
    export DATABASUS_TEST_LOG_PATH
    echo "writing the output to $DATABASUS_TEST_LOG_PATH"
    : >"$DATABASUS_TEST_LOG_PATH"

    # Each stream gets its own tee, so the console keeps stdout and stderr apart.
    exec {stdout_copy_fd}> >(tee -i -a --output-error=warn "$DATABASUS_TEST_LOG_PATH")
    stdout_copy_process_id=$!
    exec {stderr_copy_fd}> >(tee -i -a --output-error=warn "$DATABASUS_TEST_LOG_PATH" >&2)
    stderr_copy_process_id=$!

    # A background command ignores SIGINT, so Ctrl+C reaches the run as SIGTERM; either signal
    # then runs its cleanup.
    bash "${BASH_SOURCE[0]}" "$@" >&"$stdout_copy_fd" 2>&"$stderr_copy_fd" \
        {stdout_copy_fd}>&- {stderr_copy_fd}>&- &
    run_process_id=$!
    trap 'kill -TERM "$run_process_id" 2>/dev/null || true' INT TERM

    run_status=0
    wait "$run_process_id" || run_status=$?
    while kill -0 "$run_process_id" 2>/dev/null; do
        run_status=0
        wait "$run_process_id" || run_status=$?
    done

    exec {stdout_copy_fd}>&- {stderr_copy_fd}>&-
    wait "$stdout_copy_process_id" "$stderr_copy_process_id" || true
    exit "$run_status"
fi

all_suite_names=(
    browser-ui e2e-browser e2e-unit lifecycle lifecycle-interruption-checks upgrade scripts
    review-hooks
)
declare -A selected_suites=()
backend_patterns=()
backend_filters=()
frontend_paths=()
frontend_test_names=()

if [[ "$is_change_run" == true ]]; then
    if [[ ! "$change_name" =~ ^[a-z0-9][a-z0-9-]*$ ]]; then
        echo "no OpenSpec change is named \`$change_name\`; no test ran" >&2
        exit 1
    fi
    if ! plan_lines=$(python3 "$root/scripts/test_plan.py" list "$root/openspec/changes/$change_name"); then
        echo "no test ran" >&2
        exit 1
    fi
    if [[ "$plan_lines" == none ]]; then
        echo "the test plan of $change_name selects no tests"
        exit 0
    fi
    while IFS=$'\t' read -r kind first_field second_field; do
        case "$kind" in
            backend)
                backend_patterns+=("$first_field")
                backend_filters+=("$second_field")
                ;;
            frontend)
                frontend_paths+=("$first_field")
                frontend_test_names+=("$second_field")
                ;;
            suite) selected_suites[$first_field]=1 ;;
        esac
    done <<<"$plan_lines"
else
    for suite_name in "${all_suite_names[@]}"; do
        selected_suites[$suite_name]=1
    done
fi

is_backend_selected=false
if [[ "$is_change_run" == false || ${#backend_patterns[@]} -gt 0 ]]; then
    is_backend_selected=true
fi
is_frontend_selected=false
if [[ "$is_change_run" == false || ${#frontend_paths[@]} -gt 0 ]]; then
    is_frontend_selected=true
fi

is_suite_selected() {
    [[ -v "selected_suites[$1]" ]]
}

lock_directory=${TMPDIR:-/tmp}
# `mise run test` and `mise run test-change` take turns; a single package run never takes this lock.
test_run_lock="$lock_directory/databasus-test-run.lock"
# The containers helpers and the upgrade test label every container, network and volume they create
# with this ID, so the trap removes only what this run started.
DATABASUS_TEST_RUN_ID="$(date +%s)-$$-$RANDOM"
export DATABASUS_TEST_RUN_ID
# Runs take turns through the lock, so one fixed tag cannot collide with another run, and the next
# run's build replaces an image a killed run left behind.
test_run_image=databasus-test-run:local
is_test_run_image_built=false

suite_names=()
declare -A suite_process_ids=()
declare -A suite_outcomes=()
work_directory=""
selector_failures=()
selector_notes=()

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
    if [[ "$is_test_run_image_built" == true ]]; then
        docker image rm "$test_run_image" >/dev/null 2>&1 || true
    fi
    if [[ -n "$work_directory" ]]; then
        rm -rf "$work_directory"
    fi
}

exec {test_run_lock_fd}>"$test_run_lock"
if ! flock -n "$test_run_lock_fd"; then
    echo "waiting for another test run to finish"
    flock "$test_run_lock_fd"
fi
SECONDS=0

trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

DATABASUS_TEST_ROOT_RUN_ID=$DATABASUS_TEST_RUN_ID
export DATABASUS_TEST_ROOT_RUN_ID

work_directory=$(mktemp -d "$lock_directory/databasus-test-run.XXXXXX")
# Unfiltered selectors share one record, because they run every test of their packages. A filtered
# selector gets its own, so tests another selector ran in the same package do not count for it.
unfiltered_ran_tests_file="$work_directory/ran-tests-unfiltered"

# Runs one suite in the background with its lines marked by the suite's name, and records how many
# seconds it took. The process substitution keeps $! on the suite itself, whose exit status decides
# the run. Its own session lets an interruption reach the test binaries and workers the suite started.
start_suite() {
    local suite_name=$1
    shift
    # shellcheck disable=SC2016 # expanded by the suite's own shell
    setsid bash -c 'SECONDS=0; "${@:2}"; status=$?; echo "$SECONDS" >"$1"; exit "$status"' \
        suite "$work_directory/duration-$suite_name" "$@" > >(sed -u "s/^/[$suite_name] /") 2>&1 &
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

# Go's own text output hides which tests ran, so the backend runs with -json and the report turns
# the stream back into text while it records each test that passed or failed.
run_reported_go_test() {
    local ran_tests_file=$1
    shift
    set -o pipefail
    "$@" 2>&1 | python3 "$DATABASUS_TEST_ROOT/scripts/go_test_report.py" "$ran_tests_file"
}
export -f run_reported_go_test
DATABASUS_TEST_ROOT=$root
export DATABASUS_TEST_ROOT

print_summary() {
    local failed_suites=()
    local not_started_suites=()
    echo "summary:"
    for suite_name in "${suite_names[@]}"; do
        local outcome=${suite_outcomes[$suite_name]}
        local duration=""
        if [[ -f "$work_directory/duration-$suite_name" ]]; then
            duration="$(<"$work_directory/duration-$suite_name")s"
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
    local selector_note
    for selector_note in "${selector_notes[@]}"; do
        echo "selector $selector_note"
    done
    local selector_failure
    for selector_failure in "${selector_failures[@]}"; do
        echo "selector $selector_failure" >&2
    done
    ((${#failed_suites[@]} == 0 && ${#not_started_suites[@]} == 0 && ${#selector_failures[@]} == 0))
}

# A filter's depth is the number of its slash-separated parts, the level of test it selects.
get_filter_depth() {
    local filter=$1
    if [[ -z "$filter" ]]; then
        echo 1
        return
    fi
    local slashes=${filter//[^\/]/}
    echo $((${#slashes} + 1))
}

# Prints `ran` when a test at the depth passed or failed, `skipped` when tests there only skipped,
# such as the Neon tests without their settings, and `none` otherwise.
get_selector_outcome() {
    local packages_file=$1
    local depth=$2
    local ran_tests_file=$3
    if [[ ! -f "$ran_tests_file" ]]; then
        echo none
        return
    fi
    awk -F '\t' -v depth="$depth" '
        NR == FNR { is_selected[$0] = 1; next }
        ($1 in is_selected) && split($2, parts, "/") >= depth {
            if ($3 == "skip") { is_skipped = 1 } else { is_ran = 1 }
        }
        END { print is_ran ? "ran" : (is_skipped ? "skipped" : "none") }
    ' "$packages_file" "$ran_tests_file"
}

# Vitest matches a path argument as a substring of a file's path, so a directory is passed with its
# trailing slash; otherwise `src/features/edit` would also run `src/features/edit-notifier`.
get_frontend_filter() {
    local path=$1
    if [[ -d "$root/frontend/$path" ]]; then
        echo "$path/"
    else
        echo "$path"
    fi
}

cd "$root"
if [[ "$is_backend_selected" == true ]]; then
    docker compose up -d --no-recreate backend-db-test
    timeout 60 bash -c 'until docker exec backend-db-test pg_isready -U postgres >/dev/null; do sleep 2; done'

    mise run backend:pull-testcontainers
fi

upgrade_packages='^databasus-backend/e2e/upgrade(/|$)'
unfiltered_packages=()
filtered_selector_indexes=()
if [[ "$is_backend_selected" == true ]]; then
    cd "$root/backend"
    if [[ "$is_change_run" == true ]]; then
        declare -A is_unfiltered_package=()
        for index in "${!backend_patterns[@]}"; do
            pattern=${backend_patterns[$index]}
            packages_file="$work_directory/selector-$index.packages"
            listed_packages_file="$work_directory/selector-$index.listed"
            if ! go list "$pattern" >"$listed_packages_file" 2>"$work_directory/go-list-error"; then
                cat "$work_directory/go-list-error" >&2
                selector_failures+=("backend $pattern could not be listed; go list failed")
                continue
            fi
            grep -Ev "$upgrade_packages" "$listed_packages_file" >"$packages_file" || true
            if [[ ! -s "$packages_file" ]]; then
                selector_failures+=("backend $pattern matched no package")
                continue
            fi
            if [[ -n "${backend_filters[$index]}" ]]; then
                filtered_selector_indexes+=("$index")
                continue
            fi
            while IFS= read -r package; do
                if [[ ! -v "is_unfiltered_package[$package]" ]]; then
                    is_unfiltered_package[$package]=1
                    unfiltered_packages+=("$package")
                fi
            done <"$packages_file"
        done
    else
        discovered_packages=$(go list ./...)
        mapfile -t unfiltered_packages < <(grep -Ev "$upgrade_packages" <<<"$discovered_packages")
    fi
fi
mapfile -t listed_slow_packages < <(grep -Ev '^(#|$)' "$root/backend/test-slow-packages.txt")

# An entry this run does not select is skipped. scripts/test_sh_test.py names one that the full run
# does not select, because here it would only stop a whole run.
declare -A is_listed_slow_package=()
for package in "${listed_slow_packages[@]}"; do
    is_listed_slow_package[$package]=1
done
declare -A is_unfiltered_selected=()
for package in "${unfiltered_packages[@]}"; do
    is_unfiltered_selected[$package]=1
done

slow_packages=()
for package in "${listed_slow_packages[@]}"; do
    if [[ -v "is_unfiltered_selected[$package]" ]]; then
        slow_packages+=("$package")
    fi
done
other_packages=()
for package in "${unfiltered_packages[@]}"; do
    if [[ ! -v "is_listed_slow_package[$package]" ]]; then
        other_packages+=("$package")
    fi
done

if [[ "$is_frontend_selected" == true ]] || is_suite_selected browser-ui; then
    pnpm --dir "$root/frontend" install --frozen-lockfile
fi
if is_suite_selected scripts; then
    # The script tests import the browser project's locked dependencies (docs/testing.md,
    # "Python quality checks").
    uv sync --locked --project "$root/e2e/browser"
fi

# Slow packages mostly wait on containers, and the Docker daemon is their shared bottleneck. Measured
# with the end-to-end packages on the reference machine, -parallel=8 finishes the group in about 8
# minutes with no failure, while Go's default of one test per logical processor runs dozens of Patroni
# clusters at once: container stops and network removals then outlast their deadlines, the group
# fails and leaks clusters, and it takes longer. -parallel=16 is no faster and failed a recovery
# scenario. Eight packages at once keep the longest package on the critical path; a higher package
# limit adds memory without shortening the group. The others mostly compute.
#
# The end-to-end packages make the limit queue the CNPG scenarios: that package took 918 seconds
# alone under it, against 391 with Go's default, and beside the other suites the Patroni and agent
# packages take 13 to 15 minutes. The group therefore gets 30 minutes. A test plan can select the
# same packages beside the same suites, so a plan run keeps these limits. Running CNPG with the
# default limit beside the group instead makes Docker cleanups of both packages outlast their
# deadlines.
slow_package_limits=(-p=8 -parallel=8 -timeout 30m)
other_package_limits=(-p=16 -timeout 15m)
if ((${#slow_packages[@]} > 0)); then
    start_suite backend-slow run_reported_go_test "$unfiltered_ran_tests_file" env TESTCONTAINERS_RYUK_DISABLED=true \
        go test ${CI:+-race} -json "${slow_package_limits[@]}" -count=1 "${slow_packages[@]}"
fi
if ((${#other_packages[@]} > 0)); then
    start_suite backend-other run_reported_go_test "$unfiltered_ran_tests_file" env TESTCONTAINERS_RYUK_DISABLED=true \
        go test ${CI:+-race} -json "${other_package_limits[@]}" -count=1 "${other_packages[@]}"
fi
for index in "${filtered_selector_indexes[@]}"; do
    mapfile -t selector_packages <"$work_directory/selector-$index.packages"
    selector_limits=("${other_package_limits[@]}")
    for package in "${selector_packages[@]}"; do
        if [[ -v "is_listed_slow_package[$package]" ]]; then
            selector_limits=("${slow_package_limits[@]}")
        fi
    done
    start_suite "backend-selector-$((index + 1))" run_reported_go_test \
        "$work_directory/ran-tests-selector-$index" env TESTCONTAINERS_RYUK_DISABLED=true \
        go test ${CI:+-race} -json "${selector_limits[@]}" -count=1 -run "${backend_filters[$index]}" \
        "${selector_packages[@]}"
done
cd "$root"

if [[ "$is_change_run" == false ]]; then
    start_suite frontend pnpm --dir "$root/frontend" test
else
    unnamed_frontend_paths=()
    for index in "${!frontend_paths[@]}"; do
        if [[ ! -e "$root/frontend/${frontend_paths[$index]}" ]]; then
            selector_failures+=("frontend ${frontend_paths[$index]} matched no file")
            continue
        fi
        if [[ -z "${frontend_test_names[$index]}" ]]; then
            unnamed_frontend_paths+=("$(get_frontend_filter "${frontend_paths[$index]}")")
        fi
    done
    if ((${#unnamed_frontend_paths[@]} > 0)); then
        start_suite frontend pnpm --dir "$root/frontend" test --reporter=default --reporter=json \
            "--outputFile.json=$work_directory/frontend.json" "${unnamed_frontend_paths[@]}"
    fi
    for index in "${!frontend_paths[@]}"; do
        if [[ -n "${frontend_test_names[$index]}" && -e "$root/frontend/${frontend_paths[$index]}" ]]; then
            report_file="$work_directory/frontend-selector-$((index + 1)).json"
            start_suite "frontend-selector-$((index + 1))" pnpm --dir "$root/frontend" test \
                --reporter=default --reporter=json "--outputFile.json=$report_file" \
                "$(get_frontend_filter "${frontend_paths[$index]}")" -t "${frontend_test_names[$index]}"
        fi
    done
fi

# A crashed-run probe expects its own later run to remove the run it crashed, and the interruption
# checks expect the same of the lifecycle command they kill. Any browser run starting meanwhile
# would remove it first, so these run while no other browser run of this command is active. They
# run one probe after another and take longer than every browser run that may not overlap them,
# so they start first. Measured on the reference machine, the full run then took about 14 minutes,
# and 14 to 15.5 minutes when they waited for the other browser runs instead.
if is_suite_selected lifecycle; then
    start_suite lifecycle-crashed-runs mise run e2e:lifecycle --only-crashed-run-probes
fi
if is_suite_selected scripts; then
    start_suite scripts python3 -m unittest discover -s scripts -p '*_test.py'
fi
if is_suite_selected review-hooks; then
    start_suite review-hooks python3 -m unittest discover -s .agent-hooks -p '*_test.py'
fi
if is_suite_selected e2e-unit; then
    start_suite e2e-unit mise run e2e:unit
fi
if is_suite_selected browser-ui; then
    start_suite browser-ui pnpm --dir frontend test:browser --project=ui
fi

is_image_ready=false
if is_suite_selected upgrade || is_suite_selected e2e-browser; then
    tested_image=${DATABASUS_IMAGE:-}
    if [[ -z "$tested_image" ]]; then
        tested_image=$test_run_image
        is_test_run_image_built=true
        start_suite image-build docker build --tag "$tested_image" "$root"
        finish_suite image-build
    else
        mark_suite image-build skipped
    fi

    is_image_ready=true
    if [[ "${suite_outcomes[image-build]}" == failed ]]; then
        is_image_ready=false
    fi
fi

if is_suite_selected upgrade; then
    if [[ "$is_image_ready" == true ]]; then
        start_suite upgrade env DATABASUS_IMAGE="$tested_image" mise run backend:test-upgrade
    else
        mark_suite upgrade "not started"
    fi
fi

if is_suite_selected lifecycle; then
    finish_suite lifecycle-crashed-runs
fi
if is_suite_selected lifecycle-interruption-checks; then
    start_suite lifecycle-interruption-checks env PYTHONDONTWRITEBYTECODE=1 UV_PYTHON_PREFERENCE=only-system \
        uv run --locked --project e2e/browser pytest e2e/browser/lifecycle_interruption
    finish_suite lifecycle-interruption-checks
fi

if is_suite_selected e2e-browser; then
    if [[ "$is_image_ready" == true ]]; then
        start_suite e2e-browser env DATABASUS_IMAGE="$tested_image" mise run e2e:browser
    else
        mark_suite e2e-browser "not started"
    fi
fi
if is_suite_selected lifecycle; then
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

if [[ "$is_change_run" == true ]]; then
    for index in "${!backend_patterns[@]}"; do
        packages_file="$work_directory/selector-$index.packages"
        if [[ ! -s "$packages_file" ]]; then
            continue
        fi
        filter=${backend_filters[$index]}
        ran_tests_file=$unfiltered_ran_tests_file
        if [[ -n "$filter" ]]; then
            ran_tests_file="$work_directory/ran-tests-selector-$index"
        fi
        selector="backend ${backend_patterns[$index]}${filter:+ -run $filter}"
        case "$(get_selector_outcome "$packages_file" "$(get_filter_depth "$filter")" "$ran_tests_file")" in
            skipped) selector_notes+=("$selector skipped every test it selects") ;;
            none) selector_failures+=("$selector ran no test") ;;
        esac
    done
    for index in "${!frontend_paths[@]}"; do
        path=${frontend_paths[$index]}
        if [[ ! -e "$root/frontend/$path" ]]; then
            continue
        fi
        report_file="$work_directory/frontend.json"
        if [[ -n "${frontend_test_names[$index]}" ]]; then
            report_file="$work_directory/frontend-selector-$((index + 1)).json"
        fi
        # Vitest reports a test that -t filtered out as skipped, so only a passed or failed test counts.
        if [[ ! -f "$report_file" ]] || ! jq -e --arg prefix "$root/frontend/$(get_frontend_filter "$path")" '
            [.testResults[] | select(.name == $prefix or (.name | startswith($prefix) and ($prefix | endswith("/"))))
                | .assertionResults[] | select(.status == "passed" or .status == "failed")]
            | length > 0' "$report_file" >/dev/null; then
            name=${frontend_test_names[$index]}
            selector_failures+=("frontend $path${name:+ -t $name} ran no test")
        fi
    done
fi

print_summary
