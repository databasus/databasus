#!/usr/bin/env bash
# Runs the hook tool check, then the development environment check. Without set -e, so a failure of
# the first part still lets the environment report print.

set -uo pipefail

scripts_dir="$(cd "${BASH_SOURCE[0]%/*}" && pwd)"
root="${scripts_dir%/*}"

"$root/.agent-hooks/run.sh" doctor "$@"
hook_check_exit_code=$?

echo

python3 -I "$scripts_dir/development_environment_check.py"
environment_check_exit_code=$?

doctor_exit_code=$hook_check_exit_code
if [[ $doctor_exit_code -eq 0 ]]; then
    doctor_exit_code=$environment_check_exit_code
fi

print_verdict() {
    local color=$1 text=$2
    echo
    if [[ -t 1 && -z ${NO_COLOR:-} ]]; then
        printf '\033[1;%sm%s\033[0m\n' "$color" "$text"
    else
        printf '%s\n' "$text"
    fi
}

if [[ $doctor_exit_code -eq 0 ]]; then
    print_verdict 32 "All checks passed"
else
    print_verdict 31 "Some checks failed, see the lines above"
fi
exit "$doctor_exit_code"
