## Context

The PostgreSQL engine pairs TimescaleDB pre/post hooks in backend/internal/engines/postgresql/logical/timescaledb_restore.go:14. Both hooks open target connections through execStatements at timescaledb_restore.go:67. The target database outlives the restore, so an interrupted archive still needs a cleanup attempt. Real TimescaleDB backup/restore fixtures exist at backend/internal/modules/backups/e2e/postgresql/engine_postgresql_test.go:130.

backend/internal/app/server.go:20 allocates six seconds to drain and two seconds to flush logs. App publishes the shared absolute drain deadline before canceling module work at server.go:74. The restore service records its outcome only after OpenedRestoreTarget.Restore returns at backend/internal/modules/backups/internal/restore/service.go:383. Cleanup must therefore release part of the drain budget for that terminal write.

Follow backend/AGENTS.md, the testing skill, docs/architecture.md supervision and cancellation rules, and the existing application-shutdown limit. The agent's separate pre/post flow in backend/internal/agent/runner/runner.go:316 shares statement definitions, not this engine Restore call.

## Goals / Non-Goals

**Goals:** give normal cancellation a real cleanup attempt, shorten active cleanup when shutdown arrives, reserve time for outcome/logging, and report any remaining target risk.

**Non-Goals:** extend process lifetime, add persistent target recovery, or guarantee cleanup when the target is unreachable or execution ignores its context.

## Decisions

### Give ordinary cancellation a separate 30-second cleanup context

After a successful pre-restore, run the complete post-restore connection/SQL sequence with a timeout over context.WithoutCancel(ctx). Ordinary user cancellation, timeout and archive failure do not cancel that cleanup. Keep request logging values. Pre-restore and archive execution remain cancelable.

Rejected: reuse the canceled restore context, which prevents even connecting. Rejected: detach without a limit or use context.Background(), losing the existing context and bound.

### Share the absolute shutdown deadline before canceling module work

Introduce an instance-owned shutdown deadline controller in lib/shutdown, initialized by app wiring and passed through Deps to the restore pipeline. At the first shutdown signal, app publishes the absolute six-second drain deadline before cancelLoops. shutdown.Drain uses that same deadline rather than starting another six-second window. A cleanup cutoff is one second before that deadline; the final two-second flush budget is unchanged.

The controller creates cleanup contexts with the earlier of the ordinary 30-second deadline and that cutoff. If shutdown begins after an ordinary detached cleanup has already started, it shortens that active context to the same cutoff. Dispose timer/listener registrations when cleanup ends. Use lib context/lifecycle helpers; no unmanaged goroutine or global deadline state.

The engine receives a cleanup-context factory through its RestoreSpec or an equivalent explicit typed dependency, not an app import or environment access. A standalone engine call without an application shutdown owner retains the normal 30-second limit. Propagate the factory through the existing pipeline, preserving the target tunnel until synchronous cleanup returns.

Rejected: start a new short timeout only when cleanup starts after shutdown, which misses an already-running detached attempt and can reset the budget. Rejected: give cleanup all six seconds, leaving no intended window for terminal-state persistence. Rejected: extend shutdown to accommodate 30 seconds.

### Stop or skip cleanup when its shutdown cutoff is exhausted

Run cleanup synchronously as part of the supervised restore execution. If no cleanup budget remains, skip new provider/SQL work and immediately report incomplete cleanup. When the cutoff cancels an attempt, return promptly, log that the target may remain in restoring mode and let the restore service record its shutdown failure using a context bounded by the remaining drain deadline. Preserve the first interruption cause. When user cancellation was the first cause of the restore stopping before shutdown, keep the canceled outcome even if shutdown shortens cleanup. When shutdown causes the interruption, use the existing shutdown-failure classification.

If a target call ignores cancellation, supervision still stops waiting at the unchanged six-second drain deadline, flushes logs within two seconds and exits. The shutdown supervisor reports that drain did not finish; the target may remain in restoring mode. A restore whose outcome could not be written is failed by the existing next-start sweep. Do not promise that a post-restore warning or a final row write always happens when the hard budget wins.

Rejected: fire-and-forget cleanup, which can lose its tunnel and outlive supervision. Rejected: report cleanup success after cutoff or block exit waiting for an uncooperative call.

### Preserve error precedence

If archive restoration already failed or was canceled, retain that error and log cleanup failure separately. Before cleanup starts, the PostgreSQL engine marks an archive failure only when the restore context is still active. Its adapter translates that classification into the pipeline-owned archive failure. The pipeline preserves this established failure when cancellation, timeout or shutdown arrives during cleanup, while retaining storage-read error precedence. A cleanup-only failure is not marked as an archive failure. If only cleanup failed, return it as the restore failure. Include a clear warning about possible restoring mode and keep all secrets redacted. When shutdown causes the interruption, preserve the existing shutdown-failure message. A restore stopped first by user cancellation remains canceled; the shutdown controller changes its cleanup budget without replacing its cause.

Rejected: classify the result only from the context after cleanup, because a later interruption would erase an established archive failure. Rejected: replace the original cause with cleanup error, or mark an archive-only success as fully successful after cleanup failed.

### Test target state and process deadlines

For normal cancellation, hold archive input after real TimescaleDB pre-restore, wait for restoring mode, cancel, drain and query the target through a fresh connection. Also cover normal success/failure and unreachable-target cleanup.

Test the shared deadline controller with controlled time, including shutdown arriving during an existing cleanup context and exhausted budget before cleanup starts. Through real application shutdown, use a controlled external connection/proxy stall around post-restore after pre-restore has executed. Verify prompt cleanup when possible; otherwise observe the warning/drain-timeout result, process exit within eight seconds, shutdown failure before exit when persistence remains possible or next-start failure when it does not, and the honest possibility that restoring mode remains on. Preserve the one-second state-recording reserve and two-second flush allocation. Use condition waits and absolute elapsed deadlines, never fixed sleeps or mocked engine SQL. A real FIFO replacing a test-owned local archive exercises a storage read that ignores cancellation: draining expires before cleanup starts, the process exits, and the next start recovers the unfinished restore. The proxy scenarios separately exercise post-restore connection stalls and cutoff cancellation.

Rejected: only check the context passed to a helper, or only test user cancellation without the application shutdown path.

## Risks / Trade-offs

- Shutdown can interrupt mandatory cleanup -> explicitly report possible restoring mode; do not extend process lifetime.
- A target ignores context cancellation -> the existing hard supervision bound wins and startup recovery handles an unrecorded restore.
- Deadline publication races module cancellation -> publish before cancelLoops and exercise both cleanup-start/signal orderings.
- Tunnel teardown races cleanup -> retain the tunnel until the synchronous engine call returns or supervision reaches the hard deadline.
- Outcome persistence is slow -> bound it by the remaining drain deadline, then use the existing next-start sweep.

## Migration Plan

No schema or HTTP changes. Ship engine cleanup, shared absolute-deadline wiring, caller propagation and regression tests together. Update docs/architecture.md with the normal-versus-shutdown cleanup policy and unchanged global bound. Already-stranded targets still require operator repair; startup recovery records restore failure but cannot reconnect using discarded target credentials.
