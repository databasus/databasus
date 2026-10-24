#!/bin/bash
# Podman subset of the e2e suite, run by `mise run agent:e2e-podman` with the runner's
# engine socket pointed at Podman's Docker-compatible API. It covers the
# engine-dependent paths: the privileges the kernel applied, a successful
# restore and a failed one, each with its leak check. The other tests do not
# depend on the engine and run on Docker only, through run-all.sh.
set -uo pipefail

SCRIPT_DIR="$(dirname "$0")"

TESTS=(
  "test-container-privileges.sh"
  "test-restore-success.sh 18"
  "test-restore-broken.sh"
)

echo "========================================"
echo "  Verification agent e2e (Podman)"
echo "========================================"

PASSED=0
FAILED=0

for test in "${TESTS[@]}"; do
  read -r -a args <<< "$test"
  echo ""
  echo "---- $test ----"
  if bash "$SCRIPT_DIR/${args[0]}" "${args[@]:1}"; then
    PASSED=$((PASSED + 1))
  else
    FAILED=$((FAILED + 1))
  fi
done

echo ""
echo "========================================"
echo "  Results: $PASSED passed, $FAILED failed"
echo "========================================"

if [ "$FAILED" -ne 0 ]; then
  exit 1
fi
