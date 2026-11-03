## Context

See `proposal.md` for the motivation. Each decision below was checked against the code at `2afd5b9d` and the main specs. Several of them differ from the wording of the review that found the defect, and each such decision says why.

## Goals / Non-Goals

**Goals:** fix each defect at the place that causes it, with a test that fails first, and keep the fixes independent so any one can be reverted alone.

**Non-Goals:** no shared abstraction across the fixes, and no change to the HTTP contract, the schema or the frontend.

The approach is constrained by `backend/AGENTS.md`: services keep the audit event rule of its handlers, services and repositories section, and each role stays in its role file. The one change to the audit rule is named below.

## Decisions

### Client tools are checked in `main.serve`, before `app.New`

`app.New` migrates at `backend/internal/app/app.go:76`, and `Serve` checks the tools at `backend/internal/app/server.go:31`. A broken PostgreSQL bundle therefore migrates and then exits, and the old image stops at `ErrSchemaNewerThanBinary`.

The check moves to a method on `Environment`, which `main.serve` calls before `app.New`, and `Serve` no longer runs it. `Environment` builds the one `clienttools.Checker` on first use, from the final `ClientToolsDir` that `apptest` may have overridden, and `app.New` takes it from there, because the checker caches its verdicts for the health check (`client-tool-bundles`: the health check does not run the clients again).

The server runs in `binary_test.go` that assert on migration output use an operator host, which has no `assets/tools/<arch>` and now stops at this check. Those runs move to a host whose PostgreSQL clients are stubs that report the right version, so they still reach migrations. The real clients are not used, because some need libraries a machine may lack: `psql` 12 needs `libedit.so.2`.

- Rejected: checking inside `app.New`. Console commands share `New` and run without client tools (`environment.go:28`).
- Rejected: a second checker in `main`. The health check would then run every client again on its first call.

### The Helm chart gets a startup probe, and slot cleanup stays synchronous

The review proposed running `DropOrphanBackupSlots` in the background after the port opens. That creates a race. `WithBackupSlot` (`backup_slot.go:33`) creates the slot and only then starts `pg_basebackup`. A background drop arriving between them succeeds, because `pg_drop_replication_slot` only refuses an active slot, and the backup then fails. The scheduler's first pass runs right after the port opens, so the race falls on exactly the sources that are reachable. Today the cleanup finishes before the scheduler starts, so the race does not exist.

The restart loop comes from liveness firing during startup: `initialDelaySeconds: 30` with 3 failures every 10 seconds restarts the pod about 50 seconds after the container starts. The cleanup can take sources / 10 x 5 s, so 120 sources need 60 s, and this only counts physical PostgreSQL sources that hang rather than refuse. Migrations run before the port opens too, so a long migration hits the same loop.

A `startupProbe` on `/api/v1/system/version`, every 10 seconds with 60 allowed failures, covers every slow-start cause. Kubernetes holds liveness and readiness back until it succeeds.

- Rejected: background cleanup with a guard that skips running backups. It needs new coordination between the scheduler and the cleanup, and the only gain is opening the port earlier in a rare case, by about a minute at 120 such sources.
- Rejected: a total time cap on the cleanup. It could skip a reachable source whose orphan slot then keeps WAL pinned until the next backup.
- Trade-off: on such a start the port stays closed for that time, and a hung first start takes up to 10 minutes to be detected.

### A stale publication falls through to the file deletion

`transition` lets `ErrStaleStoragePublication` pass at `backup_lifecycle_service.go:150` and then returns it at line 163, which rolls back the file deletion that the first check meant to reach. Line 163 now lets it pass as well, through one small predicate both checks share. The transaction commits `RequestFileDeletions`, which is an insert-or-take and safe to repeat, and `Complete` returns `isChanged=false`. `Backuper.persistResult` (`backuper.go:543`) turns that into `errBackupRowSuperseded`, and the run logs it and returns before its final log and `sendBackupNotification`, so nothing else changes.

The row is handled by what the storage change already does. A switch with an operation deletes the running row in its own transaction (`storage_change_test.go:81`). Clearing the storage runs `CancelRunningForDatabase` after commit (`config_reaction_service.go:21`), which records `CANCELED`. The row stays `IN_PROGRESS` only until then, or until restart recovery if the canceller itself fails, as it does today.

- Rejected: recording the row as failed in this branch, as the review's test expected. A backup that succeeded would send a failure email, while one the switch stops earlier ends as `CANCELED` with no notification (`stream.go:79`).
- Rejected: recording the row as `CANCELED` in this branch. It covers only the canceller failing, a case restart recovery already ends, at the cost of a new branch in `transition` and a change to the final log.

### Interrupted logical backups use `DescribeBackupInterruption`

`RunBackup`'s docstring (`pipeline/backup.go:53`) already promises this, but `OpenDumpSource` and the config read return their own errors. The review proposed mapping any error to `ErrBackupCancelled`, which would record a shutdown as cancelled and break `application-shutdown`.

The normalisation goes inside the function passed to `RunRecovering`, so a panic stays a `PanicError`: when `runBackup` returns an error and `ctx` has ended, it returns `DescribeBackupInterruption(ctx, ctx)`. `RunBackup` then reaches the existing branches at `backup_run_service.go:56`, which record the outcome.

`logical.Service.runBackup` reads the config with `context.WithoutCancel` (`backup_run_service.go:37`). `failBackup` sends no notification without a config, so a stop that cut this read short would lose the failure notification the shutdown and time-limit outcomes promise. The read is one short query, and the stop reaches the very next step. A read that fails on its own still records a failed backup without a notification, as it does today.

- Rejected: checking in `logical.Service` alone. It would leave the docstring false for every other caller.
- Rejected: an explicit exception for the config read. The outcome would then differ by the stage, which is the defect being fixed.

The stop must also reach the tunnel. `dialAndCacheBastionLocked` (`lib/sshtunnel/forwarder.go:275`) uses `ctx` only for the TCP dial, so a bastion that never handshakes holds the run for the 30-second handshake timeout, longer than the 8-second shutdown limit. A `context.AfterFunc(ctx, ...)` closes the connection during the handshake, and its stop function is called once the handshake ends. If that stop finds the function already ran, the new client connection is closed and the context's error is returned. Every engine's tunnel gains this, which is the same promise for restores.

- Rejected: a shorter handshake timeout. It would fail slow but healthy bastions and would still not end when a user cancels.

`publishBackup` stays out: a cancel arriving after the dump deliberately no longer fails the backup, as `metadataWriteTimeout`'s comment says.

### The archive failure is detected with `IsArchiveFileNeeded`

The review proposed skipping the rewrite for "the path with the archive file", and the code seems to offer `archiveRestoreError` for that. That type means something else: `restoreWithTimescaleDBHooks` wraps every archive failure in it, and an ordinary parallel restore never produces it.

`restoreFromStorage` already has the sink, and `IsArchiveFileNeeded` says that every read of the stream happens before the tool starts. When it is true, a stream failure returns a new `errBackupDownloadFailed` instead of `errBackupStreamFailed`. The message makes no claim about the target as a whole, because TimescaleDB's pre-restore step has already run at that point. It only says the restore tool did not start.

### Retention logs a failed removal and continues

`applyRetention` logs the failure with the backup's ID and continues, and returns `ctx.Err()` when the context has ended, so a stop does not log one error per backup.

- Rejected: `errors.Join`. The caller only logs the result, so it adds a type of error with the same outcome and loses the per-backup log line.

### Comparison errors are returned as they are

The three engines split `if err != nil || isDowngrade` and return `err` after closing the tunnel. The restore service's `default` branch (`restore/service.go:272`) answers `500` with `code` `internal`, which fits a state the user cannot fix.

- Rejected: a dedicated `4xx`. An unreadable recorded version is damaged data, not a request the user can correct, and this would add a code to the contract.

### Slack declares a longer delivery limit

The review proposed a per-attempt deadline. Each request is already limited by `sendTimeout` (`providers/http_client.go:9`). The 30-second limit that defeats the wait is the outer one at `delivery.go:119`.

A global increase is ruled out, because `notifiers` promises that a hung webhook times out after 30 seconds. Slack therefore implements an optional `GetDeliveryTimeout()`, which `Delivery.send` consults. `TestSendService` ignores it, because a person is waiting for a test send. `waitForRetry` fails at once with `errRateLimited` when the wait does not fit the context's deadline, so `Retry-After: 3600` does not hold a delivery for 3 minutes. A hung Slack endpoint still ends after 30 seconds through `sendTimeout`.

- Rejected: adding the method to `Provider`. Six other providers would implement an identical answer.
- Rejected: ignoring the outer deadline in Slack with no budget. `Retry-After` is unbounded.

Shutdown follows the existing `notifiers` rule: a delivery still waiting when the drain ends is abandoned with the process.

### The reset audit event follows a successful send and has no author

The review proposed recording after `issueResetCode`, which would claim a send that may not have happened. The event is written after `SendEmail` succeeds, so it cannot share a transaction with the operation. The rule in `docs/architecture.md`, `docs/glossary.md` and `backend/AGENTS.md` says an audit event is written in the transaction of the action, so this change adds that an event about an effect outside the database is written right after that effect. `users/internal/settings/test_email_service.go:80` already works this way.

- The author is nil, as for other actions without a signed-in user (`audit/event.go:10`), because a stranger can trigger this for any address. The message names the address, so the event identifies the account.
- A write failure is logged and not returned. Returning it would answer differently for existing accounts, which `RequestReset`'s comment forbids.
- Volume is bounded by `maxResetCodesPerHour`.

`passwordreset.Deps` gains `AuditLog`.

### `flag.ErrHelp` maps to exit code 0

`run` checks `errors.Is(err, flag.ErrHelp)` before its `return 2`, which matches what Go's own `flag.ExitOnError` does. The usage text stays on the error output.

## Risks / Trade-offs

- [Stubs do not prove the real clients start] → These tests pin the order of the check and the migrations. The real clients stay covered by the checker's own tests and the image smoke run in CI.
- [The stale race cannot be timed through the API] → The test holds a physical stream as `shutdown_test.go` does, moves `storage_generation` by SQL, which is the "state the API cannot create" case in the testing skill, and releases the stream. Going through `Backuper` also lets it observe the notifications.
- [An unreadable recorded version cannot be created through the API] → The test sets it by SQL. The MySQL `version` column is `TEXT` with no check, so the database accepts such a value.
- [Retention and audit failures need a real failing write] → Each test installs a trigger in its own database that refuses the write, so nothing of ours is mocked.
- [A hung Slack request is limited by a fixed 30-second client timeout] → That one delivery test waits the real 30 seconds for the last send error to appear. The test send case uses a short timeout through `WithNotificationDeliveryTimeout`.

## Migration Plan

Nothing to migrate. The chart's new values have defaults, and an operator who overrides the existing probes keeps those settings. Each fix can be reverted alone.
