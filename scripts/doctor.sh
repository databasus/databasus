#!/usr/bin/env bash
# Runs the hook tool check, then the development environment check. Without set -e, so a failure of
# the first part or of the libedit shim still lets the environment report print.

set -uo pipefail

scripts_dir="$(cd "${BASH_SOURCE[0]%/*}" && pwd)"
root="${scripts_dir%/*}"

"$root/.agent-hooks/run.sh" doctor "$@"
hook_check_exit_code=$?

echo

# The check looks up the bundles' libraries the way mise run run and mise run test start them,
# with the shim in a directory of its own, so the check leaves nothing behind.
LIBEDIT_SHIM_DIR="$(mktemp -d)"
export LIBEDIT_SHIM_DIR
trap 'rm -rf "$LIBEDIT_SHIM_DIR"' EXIT
source "$scripts_dir/fedora.sh"
apply_libedit_shim

python3 -I "$scripts_dir/development_environment_check.py"
environment_check_exit_code=$?

if [[ $hook_check_exit_code -ne 0 ]]; then
    exit "$hook_check_exit_code"
fi
exit "$environment_check_exit_code"
