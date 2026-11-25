# Test plan

## Impact

| Changed | Tests it can break | Why |
|---|---|---|
| Test-image catalog record of `databasus/verification-postgres:19.0` | `./internal/lib/testimages/...` | The package parses the test-image catalog and rejects a malformed record. |
| Verification image helper of the agent end-to-end tests, and the single `TestVerifyBackup` loop | `TestVerifyBackup/completes_with_an_encrypted_backup_on_postgres_*` for 12 and 19 | 19 moves from the local build to the catalog image. 12 is the oldest major and takes the test-image catalog path the helper keeps. |
| `scripts/image_pins.py` | `suite scripts` (`dependency_verification_test.py`) | The check now reads its majors from the pinned versions and allows a pinned version without a record. |
| Podman subset task and its scenario check | `suite scripts` (`agent_e2e_podman_test.py`, `go_test_report_test.py`) | The scenario names, the pattern and the missing-scenario failure are tested with a stub `go`. The task now reads the ran-tests file that `go_test_report.py` writes. |
| `scripts/retry-on-registry-rate-limit.sh` | `suite scripts` (`retry_on_registry_rate_limit_test.py`) | The retry and the immediate failure are tested with shortened waits. |
| Release push and manifest steps in `.github/workflows/ci-release.yml` | `suite scripts` (`artifact_transfer_test.py`, `release_gate_test.py`, `build_inputs_test.py` and the other workflow tests, with `release_image_push_test.py` asserting `max-parallel: 8`, the retry script around every push and `imagetools create` call, and the failure on a missing digest) | These tests read the push jobs, their artifacts and their pins, and now pin the concurrency limit and the retry wrapper. |

## Not selected

- The other majors of `TestVerifyBackup` (13 to 18) and its other scenarios: they take the same test-image catalog path as 12, and the change does not touch their scenario code.
- `TestRestoreContainer`: it keeps PostgreSQL 18 and the test-image catalog path, and only its name in the Podman pattern changes.
- The real Podman run of the subset: it needs a rootful Podman socket, which `mise run test-change` does not configure. The script test covers the task's selection and its check, and CI runs the subset on Podman.
- The release push jobs themselves: they run only on a release in CI. The script test covers the retry, and the workflow tests and `actionlint` cover the steps.
- `docs/how-to-upgrade-deps.md`: documentation only.

## Selection

```tests
backend ./internal/lib/testimages/...
backend ./internal/agent/e2e/... -run TestVerifyBackup/completes_with_an_encrypted_backup_on_postgres_(12|19)$
suite scripts
```
