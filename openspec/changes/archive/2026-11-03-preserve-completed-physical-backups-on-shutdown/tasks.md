## 1. Reproduce the shutdown race

- [x] 1.1 Extend `TestPhysicalBackupShutdown` with a completed-backup regression for FULL and INCREMENTAL. Hold only metadata publication, start shutdown, observe cancellation of the original backup run context, and then release the write before draining and restarting modules. Add only the test-application support needed for this ordering, with unconditional barrier cleanup. Verify the API outcome and completed details, retained archive/manifest/metadata after the deletion job runs beyond the file commit window, and one success notification with no failure for the target backup. Before changing production code, run `TESTCONTAINERS_RYUK_DISABLED=true go test -count=1 -run '^TestPhysicalBackupShutdown$' ./internal/modules/backups/physicalapitests/backuping` from `backend/`, using the Fedora preparation below, and confirm the new scenarios fail because shutdown replaces success with `ERROR / APPLICATION_RESTART`; existing interrupted-stream and user-cancellation scenarios must still pass.

The regression also covers a required metadata write that fails after archive/manifest completion during shutdown, for both FULL and INCREMENTAL. It verifies `ERROR / APPLICATION_RESTART`, file deletion after the commit window, and failure notification without success notification, using the same targeted command.

## 2. Preserve completed execution results

- [x] 2.1 Restrict `classifyShutdown` to results whose status is not `BackupStatusCompleted`. Preserve completed results in full and leave detached persistence, lifecycle publication checks, and interrupted execution handling intact. Verify by rerunning the same targeted Go command: completed FULL and INCREMENTAL cases must pass, while interrupted streams must still report `APPLICATION_RESTART`, delete their files, and send failure notifications; user cancellation must retain its canceled outcome.

## 3. Verify the change

- [x] 3.1 Run `mise run backend:format`, `mise run backend:lint`, and the full `mise run test` command in Change verification. Resolve failures and record the targeted regression's before/after evidence and full-run outcome before mandatory implementation review. Finish with the implementation and artifacts ready for review and commit, without committing.

The targeted test commands above use the pinned toolchain. In the same shell, prepare the bundled PostgreSQL clients as described in `docs/testing.md`:

```sh
cd backend
source ../scripts/fedora.sh
apply_libedit_shim
TESTCONTAINERS_RYUK_DISABLED=true go test -count=1 -run '^TestPhysicalBackupShutdown$' ./internal/modules/backups/physicalapitests/backuping
```

## Verification evidence

- Before the fix, the targeted `TestPhysicalBackupShutdown` run failed only in the two new completed-backup scenarios: both reported `ERROR` instead of `COMPLETED`. Interrupted-stream and user-cancellation scenarios passed.
- After the classifier change, the same targeted command passed all scenarios (11.284 seconds).
- The targeted command also passed after strengthening file-retention assertions to run deletion beyond the two-second commit window (12.331 seconds).
- The targeted command passed with additional FULL and INCREMENTAL cases that cancel the metadata write after archive/manifest completion and shutdown cancellation (29.785 seconds). Both cases retain `ERROR / APPLICATION_RESTART`, delete all accompanying files after the commit window, send a failure notification, and send no success notification.
- `mise run backend:format` and `mise run backend:lint` passed; the latter reported zero issues and passed API generation, architecture, and module checks.
- `gopls check` reported no diagnostics for the changed backuper, options, shutdown test, and test helpers.
- The final full `mise run test`, including the metadata-failure cases, passed all backend packages, 2,363 frontend tests across 183 files, and the Docker upgrade test from v3.64.0. The physical backup API package passed in 28.733 seconds; the upgrade scenario passed in 82.648 seconds.
- `openspec validate preserve-completed-physical-backups-on-shutdown --strict` passed.

## Change verification

```sh
mise run test
```
