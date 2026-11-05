## Why

A canceled PostgreSQL restore uses its canceled context for mandatory TimescaleDB post-restore cleanup, leaving the target in restoring mode. The repair must preserve the application's existing eight-second shutdown limit rather than detach cleanup for longer than shutdown can wait.

## What Changes

- After successful pre-restore, run mandatory cleanup independently of normal restore cancellation, timeout or archive failure.
- Allow up to 30 seconds during ordinary operation; during shutdown shorten cleanup to the remaining drain budget with one second reserved for recording the restore outcome.
- Preserve the six-second drain and two-second log-flush budgets, including when shutdown begins during an already-detached cleanup attempt.
- Preserve `CANCELED` when user cancellation is the first cause of the restore stopping before shutdown; shutdown only shortens that cleanup attempt.
- Report interrupted/failed cleanup honestly: the target may remain in restoring mode. Do not extend shutdown or claim cleanup completed when it did not.
- Test normal cancellation and both successful and interrupted cleanup during graceful shutdown against a real TimescaleDB target.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `restores`: TimescaleDB cleanup survives normal restore cancellation and obeys the remaining application shutdown budget, with explicit incomplete-cleanup outcomes.
- `application-shutdown`: a restore stopped first by user cancellation before the stop signal keeps its canceled outcome while cleanup observes the shutdown budget.

## Impact

Constrained by root `AGENTS.md`, `backend/AGENTS.md`, the testing skill and `docs/architecture.md` supervision/context rules. Affects the PostgreSQL logical engine and its caller/pipeline, shared shutdown deadline handling and app wiring, with engine/API/process-shutdown tests. Document the cleanup deadline in the architecture. No schema, HTTP shape or runtime dependency changes are required.

### Out of scope

- Extending the existing shutdown limit or guaranteeing target cleanup after a hard kill.
- Retrying forever or persisting target credentials for later recovery.
- Changing restore concurrency, backup formats, TimescaleDB versions or the agent's separate container cleanup protocol.
