## 1. Reproduce cancellation and shutdown failures

- [x] 1.1 Add a real TimescaleDB engine regression that blocks archive input after pre-restore, observes restoring mode, cancels execution and queries the target after draining. Run `mise run backend:test-package ./internal/engines/postgresql/logical/...` on unfixed code and record restoring mode remaining enabled.
- [x] 1.2 Add application shutdown regressions for cleanup that cooperates promptly, stalls after pre-restore, or is already detached when the signal arrives. Assert the unchanged eight-second exit bound, target state and recorded outcome/warning or next-start recovery. Use the real target and controlled external connection failure; run `mise run backend:test-package ./internal/app/... ./internal/modules/backups/e2e/postgresql` to record the unfixed cleanup/budget failures.

## 2. Share the remaining shutdown budget

- [x] 2.1 Add an instance-owned absolute-deadline controller in lib/shutdown and publish the six-second drain deadline before module cancellation. Reuse that absolute deadline in Drain, reserve one second inside draining for restore outcome persistence and retain the separate two-second log flush. Cover normal 30-second contexts, shutdown shortening an active context, cleanup starting after cutoff and disposal of registrations. Verify with `mise run backend:test-package ./internal/lib/shutdown ./internal/app/...`.
- [x] 2.2 Pass the cleanup-context factory through backups module wiring, restore/pipeline and the engine spec without app imports in the engine. Run post-restore synchronously with the earlier ordinary/shutdown cutoff, keep tunnels alive through cleanup, and bound final outcome persistence by the remaining drain time. Preserve user cancellation that stopped the restore before shutdown begins during cleanup, ordinary cancel versus shutdown-failure classification and original-error precedence. Verify with `mise run backend:test-package ./internal/engines/postgresql/logical/... ./internal/modules/backups/internal/restore ./internal/modules/backups/internal/pipeline ./internal/modules/backups/internal/module ./internal/modules/backups`.

## 3. Verify target state, forced exit and documentation

- [x] 3.1 Complete the real TimescaleDB API/process scenarios for normal cancellation, success/archive failure, unreachable cleanup, cooperative shutdown cutoff, shutdown during an existing cleanup and forced drain expiration. Assert restoring mode is cleared when cleanup succeeds, may remain when interrupted, no false success, prompt failure/warning where possible, and next-start failure when the hard deadline prevents persistence. Verify with `mise run backend:test-package ./internal/app/... ./internal/modules/backups ./internal/modules/backups/e2e/postgresql`.
- [x] 3.2 Document the cleanup budget and unchanged supervision limit in docs/architecture.md. Check the shared-statement consumers without changing the agent protocol. Run `mise run agent:test`, `mise run backend:format`, `mise run backend:lint` and all Change verification commands. Resolve mandatory review findings and leave the change ready to commit.
- [x] 3.3 Preserve an archive failure established before detached cleanup through the engine adapter and restore pipeline when cancellation, timeout or shutdown arrives later. Add real TimescaleDB API regressions for cancellation and shutdown during stalled cleanup. Verify with `mise run backend:test-package ./internal/engines/postgresql/logical/... ./internal/modules/backups/internal/pipeline ./internal/modules/backups ./internal/modules/backups/e2e/postgresql`.

## Verification evidence

- Before the fix, `mise run backend:test-package ./internal/engines/postgresql/logical/...` failed in `TestTimescaleRestore/leaves_restoring_mode_after_cancellation`: the post-restore connection was canceled and a fresh connection still observed restoring mode enabled.
- With a Go source overlay restoring the original use of the canceled restore context for post-restore, the SIGTERM regression failed because a fresh target connection still observed restoring mode enabled. The same process test passes with the fix. The overlay stayed outside the repository.
- Final package verification, `mise run agent:test`, `mise run backend:format`, `mise run backend:lint`, gopls diagnostics and strict OpenSpec validation passed. After changing the successful-restore check to use a fresh target connection, `TestBackupAndRestoreTimescaledb` passed again.
- The archive-error regression failed before repair: cancellation changed `FAILED` to `CANCELED`, and shutdown replaced the original archive error with its interruption message. After repair, both API regressions, existing cancellation/shutdown scenarios, the complete PostgreSQL logical engine, pipeline, backups API and PostgreSQL backup/restore packages passed. Backend formatting, linting, gopls diagnostics and strict OpenSpec validation passed again.

## Change verification

```sh
mise run backend:test-package ./internal/lib/shutdown ./internal/app/...
mise run backend:test-package ./internal/engines/postgresql/logical/... ./internal/modules/backups/internal/restore ./internal/modules/backups/internal/pipeline ./internal/modules/backups/internal/module ./internal/modules/backups ./internal/modules/backups/e2e/postgresql
mise run agent:test
```
