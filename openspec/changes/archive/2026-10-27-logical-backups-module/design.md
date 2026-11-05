## Context

Why the change is needed is in proposal.md, section Why. This document covers what shapes the approach. Paths are under `backend/internal/` unless they start with another top-level directory. `BB` stands for `features/backups/backups` and `FR` for `features/restores`. The change starts from the state that `engines-dump-restore` and `backup-configs-module` leave: dump and restore run through engine sources and one pipeline per direction, and the configs live in the `backups` module.

**Logical backups.**
- The model is `BB/core/logical/model.go:13-42`, table `logical_backups`, with status `IN_PROGRESS`, `COMPLETED`, `FAILED` or `CANCELED`, sizes in megabytes as floats (`backup_size_mb`, `backup_raw_db_size_mb`), the duration `backup_duration_ms`, the encryption salt and IV, the TimescaleDB version, the verification status and `is_skip_retry`. No production code sets `is_skip_retry`, but the scheduler reads it (`BB/backuping/logical/scheduler.go:203`, in `GetRemainedBackupTryCount`).
- The reference to `databases` is `CASCADE` from `migrations/20250605090323_init.sql:201-204`, and the one to `storages` is `RESTRICT`.
- The repository saves whole rows for the first insert, every progress tick, the publication and every terminal state (`BB/core/logical/repository.go:15-34`). It reads on the pool through `storage.GetDb()`.

**Running a backup.**
- The scheduler (`BB/backuping/logical/scheduler.go:57-101`) is a ticker loop of 15 seconds with a `hasRun` guard. At its start it fails the backups left in progress and panics when that fails.
- `StartBackup` (`:108-190`) skips with a log line when a backup runs or the config has no storage, inserts the row and starts a bare goroutine. The HTTP handler answers 200 either way.
- `GetRemainedBackupTryCount` (`:192-253`) counts the failed backups among the newest ones.
- The backuper (`BB/backuping/logical/backuper.go:47-230`) registers the run in `tasks_cancellation.Registry` and calls the pipeline. On success it writes the `.metadata` sidecar and publishes in one transaction: confirm the write receipts, save the row, record the success on the database. On failure it records the failure on the database, discards the files in one transaction and sends the notification. Its `isCallNotifier` check (`:220`) can never take effect.
- The completion listeners, only the verification scheduler's, run each in a bare goroutine (`scheduler.go:358`).

**Retention.** The cleaner (`BB/backuping/logical/cleaner.go`) ticks every 3 seconds and picks the time period, count or GFS policy per config (`:105-131`). It keeps backups younger than 60 minutes (`:25`, `:335`). `DeleteBackup` (`:75-99`) runs the restores listener, which refuses while a restore runs, then requests the file deletions and deletes the row. GFS is the pure function `BB/backuping/shared/gfs/gfs.go:17`, which the physical cleaner also calls.

**Restores.**
- The model is `FR/core/model.go:11-27`, table `restores`, with the duration `restore_duration_ms` and a GORM relation to the backup. The repository saves whole rows, preloads the backup and joins `logical_backups` (`FR/core/repository.go:11-89`).
- The service (`FR/service.go:96-190`) checks access, the target connection and version, the disk space for a file-based PostgreSQL restore, and refuses a parallel restore of the same database. It then saves the row, puts the credentials into a cache with a one-hour lifetime, starts the restorer in a bare goroutine and answers 200.
- The restorer (`FR/restoring/restorer.go:44-234`) returns without a status on six early failures (`:51-119`) and detects the target version again without the restore context (`:148`).
- The restore scheduler (`FR/restoring/scheduler.go:37-124`) fails the rows left in progress at startup, then only writes `lastCheckTime`.

**Downloads.**
- `download_tokens` (`migrations/20260108111730_add_download_tokens_table.sql`) references the backup and the user with `CASCADE`, and marks a consumed token in the boolean column `used`.
- `download_token.Service.Generate` issues a 256-bit token that lives 5 minutes. `ValidateAndConsume` checks, acquires the user's stream, then saves `used` with a whole-row `Save` (`BB/download/download_token/service.go:32-96`).
- `stream_guard` keeps one heavy stream per user in the in-memory cache with a 5-second lock and a 3-second heartbeat. The physical restore stream shares it (`BB/download/di.go:26-45`).
- A ticker loop deletes expired tokens.

**Couplings of the old code that stays.**
- The verification runs call `GetBackup`, `GetBackupFileWithoutAuth`, `GetLatestVerifiableBackup` and `SetRestoreVerificationStatus` (`features/verification/runs/service.go:48,389,496,516,619,655`; `scheduler.go:133`), and listen for completions (`di.go:82`).
- Their repository joins `logical_backups` in raw SQL (`features/verification/runs/repository.go:91`) and queries the backup model in its own transaction (`:136`). `restore_verifications` references the backup table with `CASCADE` (`migrations/20260520120000_add_restore_verification.sql:107-111`).
- Telemetry reads `HasSuccessfulBackupSince` and `GetLatestCompletedBackup`.
- The system health check reads the scheduler's `IsSchedulerRunning` (`features/system/healthcheck/service.go:57`).
- The physical code uses `stream_guard`, `tasks_cancellation` and `gfs`.

**Loops in `internal/app`.** `internal/app/background_loops.go:20-31` starts the logical scheduler, the cleaner, the restore scheduler and the token cleanup as old loops through `background.Group`.

**Tests.**
- `BB/backuping/logical/*_test.go` call private methods on the DI singletons, with 47 `time.Sleep`.
- `BB/controllers/logical/controller_test.go` and `FR/controller_test.go` go through the old router.
- `features/tests/logical/**` back up and restore real servers through the old API.

**Rules in force.**
- `docs/architecture.md`, "Modules": module boundary, events, after-commit hooks.
- "Data": prefixes, `RESTRICT` between modules, `CHECK`, one `UPDATE` per column, `(T, bool, error)`.
- "Stored files": write receipts, pending deletions, every attempt its own name.
- "Long-running work": `background.Runner`, `lib/process`, goroutines through `lib`, cancellation by ID, shutdown to `FAILED`, time from the clock.
- "HTTP API": 202 for long operations, cursors, `Problem`.
- `backend/AGENTS.md` and `frontend/AGENTS.md`, and the `api-design` and `testing` skills.
- The change answers to the root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`.

## Goals / Non-Goals

**Goals:**

- Everything a logical backup, its restore and its download need lives in the `backups` module and passes every rule of the target layout.
- The behavior the specs state stays, and the defects listed in proposal.md are gone.
- The end-to-end safety net keeps covering every engine, now through the new API.

**Non-Goals:**

- Physical backups, their restore stream and the recovery script.
- The verification runs and the agent, apart from the module methods they call instead of the old service and raw SQL.
- Changing retention, notifications or roles.

## Decisions

### 1. The packages

```
internal/modules/backups/internal/
  config/, transfer/   from backup-configs-module
  logical/             logical backups: model, repository, service (start, cancel, delete,
                       list, read), runner of one backup, publication, startup recovery,
                       scheduler job and its due function, retention job
  restore/             restores: model, repository, service, runner of one restore,
                       startup recovery
  download/            download tokens, their cleanup job, the user streams
  pipeline/            the backup and restore pipelines, the logical engine registry and
                       the restore target types
  httpapi/             generated server, handlers, conversions
internal/lib/background/  + the cancellation registry
internal/lib/gfs/         the grandfather-father-son selection
internal/lib/diskusage/   the disk usage of the data volume, moved from features/disk
```

`logical` and `restore` are separate units: a restore has its own row, target and lifecycle, and reaches backups only through the methods `logical` offers inside the module. `pipeline` holds the code `engines-dump-restore` built, now with the registry instead of the type switch.

The imports inside the module run one way, and `internal/module` wires the packages together:

```
restore --> logical --> config
   |           |
   +-----> pipeline <--+     download --> logical
```

- `logical`, `restore` and `download` import `pipeline`, and `pipeline` imports none of them. `RestoreTarget` and its variants therefore live in `pipeline`, beside `OpenRestoreTarget`, which takes them. `restore` builds a `pipeline.RestoreTarget` from the request and hands it to the registry. Putting the target types into `restore` would make `pipeline` import `restore` while `restore` imports the registry.
- `logical` imports `config`: the start reads the config, and the scheduler and retention list the enabled configs. `config` needs the reverse once, to delete a database's logical backups when its logical storage changes. It declares the interface it needs, `logicalBackupRemover` with `RemoveBackupsForStorageChange(ctx, databaseID)`, and `internal/module` passes the `logical` service into `config.New`.
- `restore` and `download` import `logical`, to read a backup and open its file. `logical` needs the reverse once, to know whether a restore runs: deleting a backup, retention and the database deletion skip or refuse a backup that a running restore reads. It declares `runningRestores` with `IsRestoreRunningForBackups(ctx, backupIDs)` and `IsRestoreRunningForDatabase(ctx, databaseID)`, and `internal/module` passes the `restore` repository into `logical.New`.

Each of these interfaces sits beside its one consumer, as `docs/architecture.md` asks of an interface, and Go accepts the graph because no package imports a package that imports it back.

The GFS selection moves to `lib/gfs`. It is a pure function over IDs and times that knows nothing of backups, and both the module's retention and the old physical cleaner call it until milestone 10. "Shared code" in `docs/architecture.md` lists every `lib` package, so task 7.1 adds the row `gfs`, the grandfather-father-son selection over times.

### 2. The logical engine registry

```go
type logicalEngine interface {
	OpenDumpSource(ctx context.Context, databaseID uuid.UUID) (logicalDumpSource, error)
	OpenRestoreTarget(ctx context.Context, target RestoreTarget) (logicalRestoreSink, error)
	GetStoredCompression() storedCompression
}
```

One entry per logical database type sits in a map keyed by `databases.Type` in `pipeline/registry.go`, the only place in the module that names the four types. An entry reads the decrypted settings through the typed getter of `databases`, such as `GetMysqlConnectionSettings`, and opens the engine source through its tunnel. A physical database is not in the registry, so a logical operation on it answers 400 `invalid_request`. `logicalRestoreSink` is the opened handle a restore writes into, named apart from the glossary's restore target, which is the settings the user sends.

This is a second registry keyed by the database type, beside the provider registry of the `databases` module. "Engines and providers" in `docs/architecture.md` and `backend/AGENTS.md` speak of one registry per family, so task 7.1 changes both: a consumer that serves several engines through an interface of its own maps each database type to its implementation in one registry, and only there. `databases` cannot hold these entries, because they open dump and restore pipelines that `databases` knows nothing about, and a restore target is not a database of `databases` at all.

*Rejected: keep the switch in the pipeline.* `backend/AGENTS.md` keeps type switches out of everything but a registry, and the restore needs the same four entries for its target.

*Rejected: let the `databases` module hand out the engine source of a database.* It would serve the backup but not the restore target, which has settings and no database row, so the restore would still need its own mapping.

### 3. Background work

Three jobs go to `background.Runner`:

| Job | Interval | Pass |
|---|---|---|
| `backups_logical_scheduler` | 15 s | list enabled configs with their last backups, compute the due set, start each |
| `backups_logical_retention` | 1 min | per config, compute the backups the policy drops, delete them with their files and metadata files |
| `backups_download_tokens_cleanup` | 1 min | delete expired tokens |

The due set is the pure function `getDueBackups(configs, lastBackups, now)`. For each config it says whether a backup is due by the schedule or by a retry, using today's rule of `GetRemainedBackupTryCount`. Unit tests cover the schedules, the retries and the running-backup skip without a database. The retention pass also uses a pure function per policy, so the three policies, the one-hour shield and the GFS tiers are tested the same way. The pass then leaves out the backups the policy drops but that are still `IN_PROGRESS` or that a running restore reads, as the old cleaner skips the backups that `DeleteBackup` refuses. A later pass removes them once the work has ended.

The retention interval grows from 3 seconds to 1 minute. A backup older than its period now waits up to a minute longer for its deletion. A pass reads every config and its backups, and running it twenty times a minute bought nothing.

Running backups and restores start through a `background.Group` that `New` builds with `background.New`, as `storages.New` builds its own. The group recovers a panic into an error and lets the shutdown wait for the runs. Each run registers its cancel function in `background.CancellationRegistry`, which moves from `features/tasks/cancellation`: `Register`, `Cancel`, `Unregister`, `WaitUntilEmpty`. A cancel request calls `Cancel` directly. The pub/sub hop of the old requester goes away, because the single process holds both ends. The physical code switches to the same registry instance.

`Module.Start(ctx)` runs the startup recovery before `internal/app` starts the runner:
- backups left `IN_PROGRESS` become `FAILED` with "Backup failed due to application restart", their files are handed to the deletion worker and the notification is sent;
- restores left `IN_PROGRESS` become `FAILED` with "Restore failed due to application restart".

An error stops `Serve`. `Start` also keeps the application's context, which ends on the stop signal, and each run derives its own context from it. That replaces `NewBackupContext` and `DescribeBackupInterruption` of `BB/core/logical/backup_interruption.go`, which watched the shutdown signal themselves. Their successors live in `pipeline`, beside the restore's 23-hour storage-read timeout:
- `newRunContext(appCtx)` adds the 23-hour time limit of a backup, ending with `errBackupTimeout`, and returns the cancel function that the registry calls with `errCancelled`;
- `describeInterruption(appCtx, runCtx)` tells the causes apart in today's order. The time limit gives today's timeout message, and an ended application context gives "backup interrupted by application shutdown" or "restore interrupted by application shutdown". `errCancelled` gives `CANCELED`, and any other cause is the storage write's error, reported as `save to storage: ...`.

The terminal write of an interrupted run uses `context.WithoutCancel`, so it still reaches the database while the application context has ended, and the shutdown drain waits for it through the group. The `logical-backup-outcome` scenario "Backup exceeds the time limit" keeps its test against the module's clock.

`Module.IsLogicalBackupSchedulerRunning(ctx)` answers from the last pass time the scheduler job records on the module clock, with today's five-minute threshold, for the old health check until milestone 13 moves it onto the runner's results.

*Rejected: keep `tasks_cancellation` with its pub/sub requester.* A module may not import old code, and the spec `single-instance-runtime` rules out a second process that would need the hop.

### 4. Starting a backup

`logical.Service.StartBackup(ctx, actor, databaseID)` runs these steps:
1. Look the database up, call `Authorize` with `ActionView` (today's rule), and read the config.
2. Answer 400 when the config names no storage.
3. In a transaction, insert the row `IN_PROGRESS` with its own file name. A partial unique index on `(database_id) WHERE status = 'IN_PROGRESS'` turns a second start into a unique violation, which answers 409 `already_running`. Write the audit event in the same transaction.
4. After the commit, hand the run to the group and answer 202 with the row.

The scheduler calls the same insert without an actor, so a scheduled and a manual start can never both run.

The runner of one backup keeps the order of the old backuper: pipeline, metadata sidecar, publication in one transaction, `LogicalBackupCompleted` after the commit. It changes these writes:
- progress becomes `UPDATE backups_logical SET backup_size_mb = ?, backup_duration_ms = ? WHERE id = ? AND status = 'IN_PROGRESS'`;
- publication, failure and cancellation become conditional `UPDATE`s of the status and its fields, each `WHERE status = 'IN_PROGRESS'`, so a late progress tick cannot resurrect a finished row.

The failure notification goes out on every failure, as today. The dead `isCallNotifier` flag and the size-limit branch (`BB/backuping/logical/backuper.go:101-121`) are deleted.

*Rejected: refuse a second start by reading first.* Two requests that both read "nothing running" would both insert. The partial index decides it in the database.

### 5. Restores

`restore.Service.StartRestore(ctx, actor, request)` runs these steps:
1. Look the backup up, call `Authorize` with `ActionView` (today's rule), and check that the backup is completed and that the target's type matches the backup's engine, answering 400 `invalid_request` otherwise.
2. Open the target through the registry, detect its version and check that the backup's version can be restored into it. A connection failure answers 400 `database_connection_failed`, and a version that cannot take the backup answers 400 `invalid_request`.
3. For a PostgreSQL restore through a file, which is one with parallel jobs, the extension filter or a source database that skips user mappings, as `engines-dump-restore` set the conditions, check the disk, answering 409 `insufficient_disk_space` with the needed and the free bytes in `detail`. The check reads the free space through `lib/diskusage`, the package `docs/architecture.md` names for it. The 29 lines of `features/disk/service.go` move there, and the old disk controller and the system health check call it.
4. In a transaction, insert the restore `IN_PROGRESS` and write the audit event. A partial unique index on the backup's database, kept in a `database_id` column copied from the backup, turns a parallel restore into 409 `already_running`. The schema keeps the copy equal to the backup's database through a composite reference (decision 10).
5. After the commit, hand the run to the group with the open target and answer 202.

Cancelling answers 202 and calls the registry's cancel function of the run. A restore that is no longer `IN_PROGRESS` answers 409 `not_running`, a new code, because a finished restore is a state that forbids the action. The run is cancellable in each of its phases: while it opens the storage reader and downloads the archive into a file, through the run's context that the storage reader and the engine's copy observe, and while the tool runs, through `lib/process`.

The run holds the target, its credentials and its tunnel in memory only. The cache, its key prefix `restore_db` and the restore scheduler are deleted. Every exit of the run writes a terminal status with a conditional `UPDATE`. That covers the early exits of `FR/restoring/restorer.go:51-119`, such as a storage reader that fails to open, and records `FAILED` with the error. The version is detected once, in step 2, inside the request. The run does not detect it again, so the uncancellable detection at `:148` is gone.

*Rejected: store the target encrypted to survive a restart.* A restore restarts from the beginning anyway, and a stored production password is a secret that `docs/architecture.md` "Data" would then have to protect for no gain.

*Rejected: answer 202 before checking the target.* The user would learn about a wrong password from a failed row instead of the dialog, and today's dialog shows that error inline.

### 6. Downloads

A user stream is the one heavy stream a user may run at a time: a backup file download, or the restore set of a physical backup. It is held in the in-memory cache with a 5-second lock and a 3-second heartbeat, as the old `stream_guard` holds it, and the term "User stream" joins `docs/glossary.md` in task 7.1.

- `POST /download-tokens` with `{backupId}` checks `ActionView`, refuses with 409 `already_running` while the user holds a user stream, inserts a token that lives 5 minutes and writes the audit event.
- `GET /logical-backups/{backupId}/file?token=` is `x-access: public`. Its handler calls one service method, `download.Service.OpenDownload(ctx, backupID, token)`, and maps the result to the strict server's `io.Reader` response with `Content-Disposition`.

`OpenDownload` runs these steps:
1. Look the unused, unexpired token up for that backup. No row answers 401 `unauthenticated`.
2. Acquire the user stream of the token's user. When it is held, answer 409 `already_running` and leave the token unused, so the user can download with it once the other stream ends.
3. Consume the token with `UPDATE backups_download_tokens SET is_used = TRUE WHERE token = ? AND backup_id = ? AND NOT is_used AND expires_at > ?`. No row means another request consumed it first: release the user stream and answer 401 `unauthenticated`.
4. Open the decrypted file and start the heartbeat through a `lib/background` goroutine. Return a reader whose `Close` stops the heartbeat and releases the user stream.

Two requests with one token both pass step 1, and the user stream lets only one of them reach step 3; even without it, the conditional `UPDATE` lets only one consume the token. A request with a second token of the same user waits for nothing and gets 409 at step 2.

The token's text never reaches a log: the access log of `lib/httpserver` records the path without the query string (`lib/httpserver/access_log.go:34`), and an attribute named `token` is masked by the log handler.

The user stream code moves from `stream_guard` into `download` unchanged. `module.go` offers `AcquireUserStream(ctx, userID) (UserStream, error)`, with `Refresh` and `Release`, for the old physical restore stream until milestone 10.

### 7. Deleting a database

The `DatabaseDeleting` handler of the module replaces `BB/services/logical_service.go:79-86,434-462`:
1. Refuse with 409 `in_use` when a backup of the database is `IN_PROGRESS`, or when `runningRestores` reports a running restore of the database.
2. Run `DELETE FROM backups_logical WHERE database_id = ? RETURNING storage_id, file_name`.
3. Call `storages.RequestFileDeletions` once with each file and its `.metadata` sidecar.

The `databases` spec lists the refusals of a database deletion, so this change adds the running restore there through a MODIFIED delta of its deletion requirement.

Restores and download tokens go with their backups through `CASCADE` inside the module. The handler runs on the deletion's transaction through `database.Conn`, so a refused deletion keeps everything. The reference from `backups_logical` to `databases` becomes `RESTRICT`, so a backup row that a handler missed stops the deletion loudly instead of vanishing without its files. Deleting a single backup and retention use the same repository delete with `RETURNING`, followed by one `RequestFileDeletions` with the files and their `.metadata` sidecars. `logical.Service.DeleteBackup` refuses a backup that is `IN_PROGRESS` with 409 `already_running`, and a backup that `runningRestores` reports as read by a running restore with 409 `in_use`. Both refusals change nothing. The delete locks the backup row with `SELECT ... FOR UPDATE`, checks its status, asks `runningRestores` and deletes the row, all in one transaction. A restore whose insert comes in that gap waits on the lock, because the foreign key check of its insert takes a share lock on the backup row, and then finds the backup gone. Retention takes the same lock per backup it removes.

The module's handler, which runs the logical removal and then the config's, registers after the physical handlers in `internal/app/wiring.go`, because those handlers still read the database's config. A refusal from the module rolls back the physical handlers' work in the same transaction and drops their after-commit hooks.

### 8. The contract

Eleven operations under the tag `backups`:

| Operation | Answer |
|---|---|
| `GET /logical-backups?databaseId=&status=&createdBefore=&limit=&after=` `listLogicalBackups` | page of `LogicalBackup`, newest first by `(createdAt, id)` |
| `GET /logical-backups/{backupId}` `getLogicalBackup` | `LogicalBackup` |
| `POST /logical-backups` `startLogicalBackup` with `{databaseId}` | 202 `LogicalBackup` |
| `POST /logical-backups/{backupId}/cancel` `cancelLogicalBackup` | 202 `LogicalBackup`; 409 `not_running` for a backup that is no longer running |
| `DELETE /logical-backups/{backupId}` `deleteLogicalBackup` | 204 |
| `POST /download-tokens` `createDownloadToken` with `{backupId}` | 201 `{token, fileName, expiresAt}` |
| `GET /logical-backups/{backupId}/file?token=` `downloadLogicalBackup`, `x-access: public` | the file |
| `GET /restores?backupId=&limit=&after=` `listRestores` | page of `Restore`, newest first |
| `GET /restores/{restoreId}` `getRestore` | `Restore` |
| `POST /restores` `startRestore` with `{backupId, target}` | 202 `Restore` |
| `POST /restores/{restoreId}/cancel` `cancelRestore` | 202 `Restore`; 409 `not_running` for a restore that is no longer running |

- `LogicalBackup` holds the ID, database, storage, status, fail message, `sizeMb`, `rawDatabaseSizeMb`, `durationMs`, encryption, verification status and `createdAt`. The sizes stay in megabytes, as the columns and the physical backups keep them; milestone 10 changes the unit of both kinds at once, if at all. The salt, the IV and the file name stay internal.
- `Restore` holds the ID, backup, status, fail message, `durationMs` and `createdAt`, and no field of the target.
- `RestoreTarget`, the database a restore writes into, is a `oneOf` with the discriminator `type`, whose values are those of the database type in the contract: `POSTGRES_LOGICAL`, `MYSQL`, `MARIADB` and `MONGODB`. One name per type keeps the glossary's rule, and the SPA sends the backup's database type as it reads it. Each variant holds the connection settings of its engine, the shared TLS and SSH tunnel schemas of the `databases` operations, and that engine's restore options: `cpuCount`, `isExcludeExtensions`, `isRestoreOwnership` and `isRestorePrivileges` for PostgreSQL, `isSkipGaleraDisable` for MariaDB, `cpuCount` for MongoDB.
- Every secret is `writeOnly`. The version is not sent, because the backend detects it.
- The paths are `/logical-backups`, not `/backups`, because milestone 10 adds `/physical-backups` beside them, and the old physical routes under `/backups/physical/...` stay mounted until then.
- The codes `already_running`, `insufficient_disk_space` and `not_running` join `ProblemCode`, `lib/apierror` and the six dictionaries. A cancel of finished work is a state conflict, which the `api-design` skill answers with 409, and no existing code names it.
- Both lists page by `(createdAt, id)`, newest first. `lib/database` holds only `NameCursor` today, so task 1.3 adds `TimeCursor` beside it: an opaque encoding of a time and an ID, with a decode error that the handler answers with 400 `invalid_request`. "Shared code" in `docs/architecture.md` already places keyset cursors in `lib/database`. The physical list keeps its own cursor until milestone 10.

*Rejected: keep `/backups` for logical backups.* The glossary names two kinds of backup with different fields, and one path would have to grow a `oneOf` in milestone 10 or change again.

### 9. `module.go` for the old code

- `GetLogicalBackups(ctx, backupIDs)`, `GetLatestCompletedLogicalBackup(ctx, databaseID)`, which the verification runs and telemetry both use, `OpenLogicalBackupFile(ctx, backupID)` (decrypted) and `SetLogicalBackupVerificationStatus(ctx, backupID, status)` replace the old service calls and the raw `JOIN` of the verification runs. The claim query of `features/verification/runs/repository.go:136` reads its own table and then calls `GetLogicalBackups`.
- `LogicalBackupCompleted` is published after the publication commits, and the old verification scheduler subscribes to it in `internal/app/wiring.go`.
- `HasSuccessfulLogicalBackupSince` and `GetLatestCompletedLogicalBackup` for telemetry, and `IsLogicalBackupSchedulerRunning` for the health check.
- `AcquireUserStream` and `CancellationRegistry` access for the physical code.
- `LogicalBackupConfigStorageChanging` from `backup-configs-module` is deleted. The config service now calls the deletion of `logical` directly, inside the same save transaction, so the behavior of the `backup-storage-assignment` requirement stays.

### 10. The migration

One migration, in its transaction:
1. Rename `logical_backups` to `backups_logical`, `restores` to `backups_restores` and `download_tokens` to `backups_download_tokens`, with their indexes and constraints.
2. Rename `used` in `backups_download_tokens` to `is_used`, which the boolean prefix rule of the root `AGENTS.md` asks for. The size and duration columns keep their names and units: the physical tables use the same ones, and milestone 10 can change both kinds together.
3. Drop `is_skip_retry`.
4. Add `database_id` to `backups_restores`, filled from its backup. Add `UNIQUE (id, database_id)` to `backups_logical`, and replace the restore's reference to its backup with the composite `(backup_id, database_id) REFERENCES backups_logical (id, database_id) ON DELETE CASCADE`. The copy can then never differ from its backup's database, and through the backup the restore reaches `databases` with `RESTRICT`.
5. Repair a status, encryption or verification status that the new `CHECK` would reject: an unknown status becomes `FAILED` with a fail message that names the stored value, and an unknown encryption becomes `NONE` only when the row has no salt.
6. Add the `CHECK` constraints and the two partial unique indexes on `IN_PROGRESS`, failing all but the newest in-progress row per database first.
7. Replace the references to `databases` and to `users` with `RESTRICT`.

Users are never deleted, so `RESTRICT` on the token's user changes nothing a user can do. The upgrade test restores the previous release's backup through `startRestore`, which proves that the renamed rows and the encryption metadata survive.

### 11. Tests

- **Engine end-to-end:** `features/tests/logical/{postgresql,mysql,mariadb,mongodb}` become API tests in one package per engine under `internal/modules/backups/e2e/<engine>` through `apptest`, so the four run in parallel with the module's own tests. They keep their scenarios: every supported version, SSL, mTLS, SSH tunnels, TimescaleDB and the PostgreSQL dump scope. They start servers through `util/testing/containers` and wait with `testwait.Until` on `getLogicalBackup` and `getRestore`.
- **Scheduler and retention:** the pure functions get unit tests, and API tests advance the `apptest` clock and run one pass with `Runner.RunPass`. That replaces the private calls and the 47 sleeps.
- **Outcomes:** the `logical-backup-outcome`, `database-tool-failure-output` and `backup-encryption` scenarios move with the pipeline tests from `engines-dump-restore` and run against the module.
- **API:** every requirement of the `logical-backups`, `restores`, `backup-list-paging` and `databases` deltas and the `workspace-access` scenarios get API tests, the token race and the second start through concurrent calls among them. A scenario that needs a running restore lands with the restore operations in task 5.2.
- **Startup recovery:** a test inserts an `IN_PROGRESS` restore with `ExecuteSQL`, restarts the modules through `apptest`'s `RestartModules`, which stops them as a shutdown does and starts them on the same database, and checks the failed row and a new restore of the same database. The same restart covers a restore that runs at shutdown. The refusal to start when the recovery fails stays out of the spec and out of the tests: `Start` returns the repository's error, and `Serve` stops on it.
- **Upgrade:** the upgrade test's restore goes through the new operation.

### 12. Frontend

- `entity/backups/logical` and `entity/restores` lose their clients and types.
- The logical backup list follows the cursor with the existing infinite scroll, and keeps its refresh of the first page per `backup-list-paging`.
- The manual backup shows the returned backup at once.
- The download asks for a token and opens the file link.
- The restore dialog builds the `RestoreTarget` variant from the database's engine, shows `database_connection_failed`, `already_running` and `insufficient_disk_space` inline through `translateApiError`, and follows the returned restore.
- The six dictionaries get the three new codes.

## Risks / Trade-offs

- [The end-to-end tests are rewritten in the same change as the code they guard] → `engines-dump-restore` kept them unchanged across the dump move. This change rewrites only their HTTP calls, and keeps every scenario and every assertion on restored rows.
- [The change is the largest of milestone 9] → Tasks land per unit (registry, backups, retention, downloads, restores, old code, operations, frontend) with each unit's tests passing before the next.
- [A partial unique index could reject a row an old start left behind] → The migration and the startup recovery fail stale `IN_PROGRESS` rows first.
- [The retention pass runs less often] → Decision 3 states the cost: a deletion can come up to one minute later.
- [The old physical code depends on two module methods for one more milestone] → The relaxation table lists them, and milestone 10 moves that code into the module.

## Migration Plan

The migration runs at startup. Backups and restores left in progress by the previous run are failed by `Start` before the jobs begin, as today.
