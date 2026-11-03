## Context

See `proposal.md` for the motivation. The defect is present in the current checkout:

- `backend/internal/modules/backups/internal/physical/backuping/backuper.go:729` replaces any result with a shutdown failure when the context cause is `ErrBackupShutdown`, including `COMPLETED`.
- The FULL and INCREMENTAL paths call it before detaching persistence from cancellation (`backuper.go:210` and `backuper.go:347`). Failure finalization also calls it (`backuper.go:602`).
- Successful executors include the metadata write receipt before returning their result (`create_full_executor.go:139` and `create_incremental_executor.go:136`, in the same package). `metadata.go:131` detaches that final write from cancellation and gives it a separate timeout. Shutdown can therefore arrive before execution returns a valid success.
- `backend/internal/modules/backups/internal/physical/catalog/backup_lifecycle_service.go:108` checks publication eligibility, updates the completed backup, and confirms its file writes in a transaction. The failure path requests file deletions instead (`:167`).
- `backend/internal/modules/backups/physicalapitests/backuping/shutdown_test.go:19` already covers interrupted FULL and INCREMENTAL streams, application-restart reasons, failure notifications, file deletion, and user cancellation. It does not cover successful execution overtaken by shutdown before result classification.

The change follows root `AGENTS.md`, `backend/AGENTS.md`, and `.agents/skills/testing/SKILL.md`. The "Stored files", "Physical backups", and "Long-running work" sections of `docs/architecture.md` require file confirmation to remain atomic with publication and shutdown to retain its existing drain limit. No architecture change is needed.

## Goals / Non-Goals

**Goals:** Preserve the executor's complete successful result through shutdown classification and verify that the real publication path retains its files.

**Non-Goals:** Do not infer success from file existence, bypass publication checks, extend shutdown deadlines, or change the outcomes of user cancellation and interrupted streams. The full scope is in `proposal.md`.

## Decisions

### Preserve completed results in the shared shutdown classifier

Add `backupResult.Status != catalog.BackupStatusCompleted` to the shutdown condition. A completed result passes through unchanged, including its receipts and metadata. Every other result keeps the existing shutdown classification.

Changing only the FULL caller would leave INCREMENTAL exposed. Changing both callers duplicates the rule and leaves the shared classifier misleading. Reconstructing success from files would bypass executor validation and could preserve an incomplete backup. The shared condition is the smallest change that applies the rule consistently.

### Keep persistence and file ownership in the existing lifecycle

Continue using the detached persistence context and `BackupLifecycle.Complete`. Its existing checks still reject a deleted, canceled, or otherwise superseded backup and an invalid storage publication. Receipt confirmation remains in the transaction that publishes success, as `docs/architecture.md` and `backend/AGENTS.md` require.

Confirming receipts in the classifier or disabling file deletion during shutdown would separate file ownership from publication and could retain files for failed backups. Changing transaction cancellation is unnecessary because the callers already use `context.WithoutCancel`.

### Reproduce the boundary through real API tests with controlled progress

Extend `TestPhysicalBackupShutdown` for both backup kinds. Keep real application modules, PostgreSQL, and local storage. Use a test-side database barrier restricted to the `.metadata` write, following the existing SQL-barrier approach in `physicalapitests/backuping/storage_write_barrier_test.go:9` and `internal/app/apptest/storage_locks.go:32`. Archive and manifest writes must finish before the barrier holds metadata. The barrier uses a transaction-level advisory lock in the test database, and the test observes the waiting writer through `pg_blocking_pids`.

Initiate module shutdown while metadata is held, explicitly wait for the active backup's run context to be canceled by shutdown, then release the write and let execution and publication drain before restarting modules. Any support needed to observe that cancellation belongs in test-application helpers; do not add production workflow hooks or fake an internal module. Cancellation observation must use the backup's original run context, because the metadata write's context is detached. A test log sink captures that original context from the existing backup-start record through `apptest.WithLogSink`; `apptest.StopModules` starts cancellation without waiting for the held write. Register cleanup that releases the barrier even on failure, and do not wait for shutdown to drain while the barrier is still held.

This produces a completed executor result under an already canceled run context. The unfixed classifier deterministically turns it into an error; the fixed classifier allows publication. A sleep or file-existence poll alone cannot establish that ordering. A classifier-only test would miss publication, file ownership, and notifications.

For each kind, assert the API reports `COMPLETED`, has no error reason or fail message, and has the normal completed-backup details. After restart, advance the manual clock beyond the file commit window and run an explicit `storage_file_deletion` pass. Assert the archive, reconstructed manifest, and metadata remain after unconfirmed writes would have become eligible for deletion. Configure success and failure notifications so the test can verify one success and no failure for the target backup. Finish any prerequisite FULL and drain its notifications before observing the INCREMENTAL under test.

Retain and run the existing interrupted-stream and user-cancellation scenarios rather than duplicating them. The new regression must first fail on the unfixed code because shutdown replaces success, then pass after the one-condition repair.

For each kind, also exercise failure of the required metadata write after the archive and manifest have been stored. At the same barrier, start shutdown and observe the original context's cancellation, then use `pg_cancel_backend` to cancel only the statement blocked by this test's advisory lock in its isolated database. Wait for that statement to leave the barrier before releasing the lock. After restart, require `ERROR / APPLICATION_RESTART`, advance beyond the file commit window, and verify deletion of the archive and sidecars, one failure notification, and no success notification. This proves that an archive alone cannot publish an incomplete backup without replacing a real module or provider.

### Clarify the existing shutdown requirement

Modify the physical-backup requirement in `application-shutdown` to distinguish successful execution awaiting publication from interrupted execution. Preserve its WAL and user-cancellation scenarios. Include required accompanying writes that finish during shutdown, since the metadata barrier exercises this existing supported path.

A separate capability would split one shutdown rule across specs. A blanket promise to preserve every copied archive would hide required-file failures. The delta instead requires successful execution and publication within the existing shutdown limit.

## Risks / Trade-offs

- A test that releases metadata before run-context cancellation can pass on unfixed code. Mitigation: observe cancellation explicitly before release and require the regression to fail before applying the fix.
- Metadata uses a detached context, so blocking it while synchronously waiting for restart can exhaust the drain timeout. Mitigation: separate shutdown initiation from draining in test helpers, release the barrier before waiting for completion, and register unconditional cleanup.
- File existence immediately after completion does not prove durable ownership. Mitigation: restart modules, run file deletion explicitly, and check all required files afterward.
- A process forced to exit before publication can still lose the outcome. Mitigation: retain the existing shutdown deadline and next-start recovery; this change only preserves results that can be recorded within that budget.

## Migration Plan

No data migration or configuration change is required. Deploy the backend normally. Rolling back restores the narrow shutdown race but requires no schema rollback. Already failed or deleted backups are not recovered by this change.
