## Context

Why the change is needed is in proposal.md. This change starts from the state `physical-tests-on-apptest` leaves:
- every non-pure physical test is an API test in `internal/modules/backups/physicalapitests` or an end-to-end scenario;
- the pure tests sit next to their code;
- the old loops' startup work is one step, `RunStartupWork`, which `app.StartModules` calls;
- `RunJobPass` already answers `backups_physical_scheduler` and `backups_physical_cleaner`;
- the WAL stream timings arrive through `backups.WalStreamTimings`.

Paths are under `backend/internal/` unless they start with another top-level directory. Abbreviations:
- `FB` is `features/backups/backups`;
- `BP` is `FB/backuping/physical`, `CP` is `FB/core/physical`, `UP` is `FB/usecases/physical/postgresql`;
- `SV` is `FB/services`, `CT` is `FB/controllers/physical`;
- `BM` is `modules/backups`.

**The shape of the old code.** It has about 19,000 production lines. Its package graph, lowest first:
- `CP/models`, `CP/enums`, `CP/dto`;
- `CP/repositories`;
- `CP/chain_view`;
- `CP/wal_health` and `CP/service`;
- `UP`;
- `BP`;
- `SV`, `FB/download` and `FB/dto/physical`;
- `CT`.

Two files serve both the backuper and the WAL streaming:
- `UP/timeline.go`: `UploadHistoryFile` is called from `UP/create_full_uc.go:157` and `UP/wal_watch_dir.go:106`;
- `UP/constants.go`: the metadata sidecar suffix.

`wal_health` imports `chain_view` only for `LSNRange` (`CP/wal_health/dto.go:30`).

**What breaks the target rules.**
- **Database access.**
  - No repository method takes a context, 0 of 60, and every repository reads `storage.GetDb()`.
  - SQL also runs outside repositories, in the scheduler (`BP/scheduler.go:195`, `:403-466`, `:527-600`, `:778-833`), the backuper (`BP/backuper.go:598-652`), `CP/chain_view/service.go:204-214`, `CP/service/service.go` and `CP/service/database_deletion_handler.go:18-46`.
  - `storage.GetTransactor()` is read in 14 places.
- **Context and goroutines.** There are 21 `context.Background()` calls, 5 bare `go` statements, 2 `WaitGroup.Go` calls and 1 manual `WaitGroup`.
- **Time.** There are 67 `time.Now` reads and two inline `clock.NewSystem()` calls (`UP/executor.go:89`, `UP/receivewal_process.go:47`). SQL `NOW()` appears in `CP/repositories/in_flight_backup_repository.go:37` and eleven times in `wal_streamer_repository.go`, whose comment at `:78-82` prefers the database clock to avoid skew between instances.
- **Getters and singletons.** `config.GetEnv()` is read 5 times and `logger.GetLogger()` 11 times. There are 18 `sync.OnceValue` singletons and 10 positional struct literals.
- **Bridges.** Every bridge use has a replacement on a neighbour's `module.go`. The exceptions are the WAL queue root (`config.GetEnv().DataFolder`, `BP/wal_supervisor.go:291`, `:315`), which `lib/config` already holds (`lib/config/load.go:147`), and telemetry's read of the latest completed full backup.

**The logical side, the pattern to follow.**
- `BM/module.go:240` builds one `background.Group` for backups, restores and download heartbeats.
- `BM/internal/pipeline/run_context.go:16-50` derives a run's context from the application's context. The context ends with the shutdown cause when the application stops, with the timeout cause after 23 hours, and with the cancel cause when the registry cancels it.
- The config service reaches the logical side through a function type passed in `Deps` (`BM/module.go:233`), which breaks the import cycle.

**The rules that constrain the approach.** `backend/AGENTS.md` fixes how dependencies arrive ("Dependencies"), the roles of services and repositories ("Handlers, services, repositories"), how background work runs ("Long-running work"), errors and logging. `docs/architecture.md` adds the module boundary and the event rules ("Modules"), and the shutdown and time rules ("Long-running work").

**The temporary seams this change removes.**
- `config.Service` publishes `PhysicalBackupConfigSaved` after the commit (`BM/internal/config/service.go:379-449`). It stops at the first handler error, and the config stays saved.
- `internal/app/wiring.go:51-86` subscribes the old handlers.
- `internal/app/server.go:86` waits for the shared registry.
- `BM/module.go:210-212` carries that registry in `Deps`.

**Defects the inventory found, both caused by the old wiring.**
- **A cancel can be lost.** The scheduler starts `go MakeBackup` (`BP/scheduler.go:385`) before the backup registers for cancellation (`BP/backuper.go:109`). A cancel or a delete in that window releases the claim and finds nothing to cancel, so the backup runs on.
- **A streamer row can stay `RUNNING` after a stop.** `stopAllStreamers` (`BP/wal_supervisor.go:584-595`) marks only the streamers still in its map. A streamer whose goroutine already returned keeps its row `RUNNING` until the 90-second staleness passes.

## Goals / Non-Goals

**Goals:**
- The physical code lives in the module, follows `docs/architecture.md` and `backend/AGENTS.md`, and passes the linters and `archlint` with no exception.
- What a user sees stays the same, apart from the shutdown outcome in the `application-shutdown` delta.
- The API tests and end-to-end scenarios from `physical-tests-on-apptest` pass without edits to their test code. The two defects above are gone as a consequence of the new wiring.

**Non-Goals:**
- Any simplification milestone 11 owns. The status is still written from several places, full and incremental backups keep two tables, the scheduler keeps both immediate-replacement checks, and the stream health is still computed in the live trackers and from the samples. The codec that worked is not remembered either. Four rules of "Physical backups" in `docs/architecture.md` therefore do not hold for the moved code yet: one owner of the status with conditional transitions, one table for full and incremental backups, one stream health verdict, and the codec remembered per database. Task 6.1 records them as a temporary relaxation in the rewrite plan, lifted by milestone 11.
- Changing what a cancel records.
- The physical operations in the contract, which belong to `physical-backups-api`.

## Decisions

### 1. Subpackages by meaning, with one import direction

```
internal/modules/backups/internal/physical/
  catalog/        models, enums, repositories, LSNRange, sidecar naming
  timelines/      timeline checks, history file upload, start LSN validation
  chains/         chain view, restore set, restore coverage, list, delete, cascade,
                  the database deletion handler   (CP/chain_view, CP/service)
  streamhealth/   verdict, keeping up, statistics          (CP/wal_health)
  backuping/      scheduler, backuper, canceller, full and incremental runs,
                  backup slot, summarizer, manifest, encryption   (BP, UP)
  walstreaming/   Manager, Streamer, upload, lag monitor,
                  rotation, slots, back pressure                  (BP, UP)
  retention/      the cleaner                                     (BP)
  restorestream/  restore tokens, restore stream writer
  service.go      Service, the facade module.go calls             (SV)
```

`catalog` imports nothing of `physical`. `timelines`, `chains` and `streamhealth` import `catalog`. `backuping`, `walstreaming`, `retention` and `restorestream` import those. The facade imports all of them. `LSNRange` and `LSNMax` move from `CP/chain_view/view.go:25` to `catalog`, so `streamhealth` no longer imports the chain view. The validation result moves from `chain_view` to `timelines`, the only package that produces it. The startup slot sweep moves from the cleaner to `backuping`, the owner of backup slots, so `retention` does not import `backuping`. Each package has its `errors.go` and role files, as `backend/AGENTS.md` says.

Names:
- The supervisor of all streams becomes `walstreaming.Manager`. In prose and in `docs/glossary.md` it is the "WAL stream manager", a new term.
- The per-database supervisor becomes `walstreaming.Streamer`. The glossary already has the term "WAL streamer" for it (`docs/glossary.md:55`); the type name joins that row's code column, and no synonym is added.
- The facade `SV.PhysicalBackupService` becomes `physical.Service`.
- The catalog service `CP/service.PhysicalBackupService` becomes `chains.Service`.
- Type names drop the stutter: `walstreaming.Manager`, not `walstreaming.WalStreamManager`.

Alternatives rejected:
- *One `physical` package.* About 19,000 lines in one package hide the boundaries between catalog, backups and streaming that milestone 11 needs to cut along.
- *The old split one-to-one (`core`, `usecases`, `backuping`).* `usecases` mixes backups and streaming, and the old split is what made `timeline.go` and `constants.go` cross-cutting.

### 2. Repositories own every query, with the context and the transaction in it

Every repository holds `*gorm.DB`, takes `ctx` first and runs through `database.Conn(ctx, r.db)`. A lookup that finds nothing returns `false`, not `nil, nil`. The five methods that take a transaction as a parameter take it from the context instead. The SQL now outside repositories moves into them:
- the mismatch query and the immediate-replacement lookups of the scheduler;
- the claim and insert;
- the conditional failure update;
- the backuper's status read under lock and its result persist;
- the chain successor query;
- the list union, usage sum, anchor lookup and cascade statements of `CP/service/service.go`;
- the deletion handler's reads and deletes;
- `CP/repositories/coordination.go`, folded into `in_flight_backup_repository.go`.

The whole-row saves (`BP/backuper.go:479`, `:533`, `:690`, `:739`, `UP/create_full_uc.go:47`, `UP/create_incremental_uc.go:80`) become `UPDATE`s of exactly the columns each one changes today.

Services open transactions with `Transaction(ctx, fn)` of the module's `Transactor`. Inside `fn` every call takes the `ctx` that `fn` receives.

Alternatives rejected:
- *Keep the transaction parameters and pass `*gorm.DB` around.* `forbidigo` allows `*gorm.DB` only in repositories (`backend/.golangci.yml:79-84`).

### 3. Every transaction keeps its statements and its lock order

`physical-tests-on-apptest` could not keep tests for some races (its decision 11), so this change does not leave their correctness to tests. Each transaction of the old code keeps the same statements, in the same order, with the same locks:

| Transaction | Old code | Locks, in order |
|---|---|---|
| claim and insert of a scheduled backup | `BP/scheduler.go:403-466` | orphan cleanup advisory lock, in-flight claim, mismatch recheck, insert |
| failure and claim release | `BP/scheduler.go:778-833` | conditional update, file deletion request, claim delete |
| result persist of a backup | `BP/backuper.go:598-652` | the database row through `RecordBackupSuccess`, the backup row `FOR UPDATE`, claim delete |
| cascade delete of a full backup | `CP/service/service.go:625-730` | advisory lock, root full `FOR UPDATE`, in-flight check |
| cascade delete of an incremental | `CP/service/service.go:229-260` | root full `FOR UPDATE` |
| orphan and span WAL deletion | `CP/service/service.go:145`, `:168` | advisory lock |
| deletion of a database's physical rows | `CP/service/database_deletion_handler.go:18-46` | inside the publisher's transaction |
| history file insert | `UP/timeline.go:282-306` | insert, write receipt |
| WAL segment commit | `UP/wal_upload.go:467-491` | the database row through `RecordBackupSuccess`, segment update, write receipt |
| full backup publish | `UP/executor.go:133`, `UP/create_full_uc.go:201` | as today |

Review compares each row against the new code.

Alternatives rejected:
- *Turn the transactions into the conditional updates of milestone 11 now.* That mixes the simplification into the move, which the rewrite plan keeps apart so a failure can be bisected.

### 4. Injected clocks for timestamps and execution time

Every time read takes an injected clock from `Deps`. `PhysicalExecutionClock` is a separate source for elapsed tool execution, progress and stall timers, and the bounded wait for the source summarizer. The composition root supplies a system clock, so these measurements advance while a test holds its schedule clock fixed. Scheduling, retention and persisted timestamps use the module's main `Clock`. Together they cover cadence, retention, deadlines, persisted timestamps, durations and the alert throttle. The two inline system clocks go too. SQL receives the time as a parameter, including the streamer heartbeats and staleness checks that use `NOW()` today, and the claim time of `in_flight_backup_repository.go:37`. The comment in the streamer repository justified the database clock by skew between instances. The `single-instance-runtime` spec runs one application process per installation, so one process clock has no skew to avoid. The engine's own timers, such as the stall watchdog, stay in the engine.

Alternatives rejected:
- *Use the schedule clock for execution duration and watchdogs.* A real PostgreSQL process keeps running while a test holds its schedule clock fixed; duration becomes zero and stall detection never expires.
- *Introduce a new stopwatch API in the engines during this move.* The engine already accepts an injected `clock.Source`; a separate execution source fixes the frozen-clock defect without changing that API.
- *Keep `NOW()` for heartbeats.* It breaks the rule in "Long-running work" of `docs/architecture.md` for a skew that one process cannot have. It also mixes two clocks in one comparison as soon as a test runs on a clock it moves.

### 5. Three jobs of `background.Runner`

`Module.GetJobs` adds three jobs. `physical-tests-on-apptest` already made `RunJobPass` answer the first two names; this change adds the third:

| Job | Interval | Pass |
|---|---|---|
| `backups_physical_scheduler` | 1 s | decide per enabled config, claim and insert, start the backup in the group of runs |
| `backups_physical_cleaner` | 1 min | retention, orphans, abandoned WAL claims, expired samples |
| `backups_physical_wal_stream_manager` | 15 s | start the WAL streamers that should run, stop those that should not, heartbeat the running ones |

The `hasRun` guards, the tickers, `isReady`, `lastTickTime` and the health methods go. No production code reads them (`internal/features/system/healthcheck/service.go:58` reads only the logical scheduler), and the runner reports each job's last result. `internal/app/background_loops.go` loses the three loops and `StartPhysicalBackupLoops`.

### 6. Backups and streamers run in the module's group of runs

`Module.Start` stores its context, as the logical service does. Everything below starts from that context through the module's `background.Group`, the one `BM/module.go:240` already builds.

- **A scheduled backup.** The scheduler's pass derives the run context and registers it under the backup ID before the claim-and-insert transaction can commit. It unregisters the run when no backup is claimed; otherwise it starts the run in the group after the commit. A cancel or a delete that comes right after the claim therefore always finds the backup.
- **The physical run context.** `run_context.go` moves from `pipeline` to `BM/internal/runcontext`, which the logical backups, the restores and the physical backups share. Its constructor takes the shutdown cause and an optional time limit. Logical backups and restores keep their 23 hours. Physical backups and streamers get no limit, as today. A full backup of a terabyte database over a WAN can take days, which is the reason the server-side compression exists (`docs/architecture.md`, "Physical backups").
- **A shutdown while a backup runs.** The interrupted run is recorded the way the startup recovery records it today, with three differences:
  - the fail message is `backup interrupted by application shutdown`;
  - the reason is the existing `APPLICATION_RESTART`;
  - it happens at once.

  As in the recovery, the backup's files go to the deletion of stored files and the claim is released. Then the failure notification goes out as for any other failed backup. The write uses `context.WithoutCancel`, as the logical side does. A user cancel while `pg_basebackup` streams keeps `CANCELED`, the reason `CANCELED_BY_USER` and the fail message `context canceled`. The explicit `ctx.Err()` branch in `UP/stream.go:69-77` takes precedence over the interrupted tool error. The existing `TestCancelRunningPhysicalBackup` and the full and incremental stream tests pin this outcome. Shutdown is distinguished by `context.Cause` and overrides this result before it is persisted.
- **A WAL streamer.** Each streamer runs in the group on a context from `Module.Start`, registered under the database ID. When its context ends, whether by shutdown, by a cancel or by the manager stopping it, the streamer records itself stopped with `context.WithoutCancel` before it returns. The slot and the local queue stay, as `stopAllStreamers` keeps them today. Because each streamer records its own stop, no row stays `RUNNING` after a shutdown. The manager keeps its map of running streamers only to start, stop and heartbeat them.
- **The remaining goroutines.** The backup slot drop retries (`UP/backup_slot.go:169`), the upload worker (`UP/wal_upload.go:643`), the streamer's inner loops (`UP/wal_stream.go:340`) and the upload dispatcher (`UP/wal_upload_dispatcher.go:61`) start through `lib/background`. Each is tied to the context of the run that owns it.

`WaitForRuns` therefore waits for physical backups, streamers and restore stream heartbeats, within the same 8-second shutdown limit as the logical runs, as the modified `application-shutdown` requirement says. The "running tasks" step of `internal/app/server.go:86` goes.

Alternatives rejected:
- *Reuse `pipeline.NewBackupRunContext` as is.* Its 23-hour limit would end long full backups and stop every streamer once a day.
- *Run streamers as long passes of the manager's job.* A pass that never returns blocks the job, and the runner never overlaps passes.

### 7. The startup work runs in `Module.Start`

`Module.Start` runs, in order:
1. the restores' recovery;
2. the logical backups' recovery;
3. `backuping.FailBackupsLeftInProgress`, which fails the backups the previous run left in progress;
4. `walstreaming.FailStaleStreamers`, which records as failed the streamers whose heartbeat went stale;
5. `backuping.DropOrphanBackupSlots`, which drops the backup replication slots no backup holds, best effort as today.

`RunStartupWork` from `physical-tests-on-apptest` splits into these three steps in the packages that own them, so each name says what it does.

An error stops the application, as now. `app.StartModules` no longer calls the physical step itself.

Alternatives rejected:
- *Keep the physical step as a separate call in `app.StartModules`.* The module's startup would then live in two places, and `Module.Start` would no longer describe what the module does at start.

### 8. The config save calls the physical side directly

`config.Deps` gains `PhysicalConfigSavedReactionFunc`, a function type with the method `ReactToSavedConfig(ctx, previous, saved) error`, the way `LogicalBackupRemoverFunc` gives the config service the logical side (`BM/module.go:233`). `savePhysicalBackupConfig` calls it after the commit with the previous and the saved config. It returns its error as the save's error, as the event's handlers do today. The reaction keeps the old order and conditions: first the cancellation listener's part (`BP/cancellation_listener.go:41-60`), then the removal of backups kept outside the new storage (`SV/physical_service.go:412-463`). `PhysicalBackupConfigSaved`, its handler type, `SubscribeToPhysicalBackupConfigSaved` and the paragraph of `docs/architecture.md` that allows it are deleted.

Alternatives rejected:
- *Call the physical side inside the save's transaction.* That would change what the event exists for. A full backup or a streamer starting meanwhile must already see the new storage committed.
- *Keep the event inside the module.* `docs/architecture.md` forbids any new after-commit event. The event existed only because the old code sat outside the module.

### 9. The database events keep their order inside the module's handlers

`Module.OnDatabaseDeleting` runs:
1. the physical cancellation, after commit;
2. the physical rows and the request to delete their files, in the transaction;
3. the physical slot drop, after commit;
4. the logical backups;
5. the config.

This is the order `internal/app/wiring.go:51-74` gives the separate handlers today. The after-commit hooks keep the same order, so the streamer stops before its slot is dropped. `Module.OnDatabaseBackupTypeChanged` runs the config's fitting first and then the physical cancellation, also as today. `wiring.go` loses the physical subscriptions.

Alternatives rejected:
- *Merge the physical handlers into one now.* Milestone 11 owns that merge, and the order is what makes the streamer stop before its slot goes.

The server cancels the module and job contexts, waits for the job passes to return, then drains the module runs and notification deliveries. Waiting for the jobs first prevents a scheduler pass from adding a backup after the module run group has already drained.

### 10. One cancellation registry, owned by the module

`backups.New` creates the registry. Logical backups, restores and physical backups and streamers share it inside the module, under keys that are random UUIDs with distinct meanings: backup ID, restore ID, database ID for a streamer. `Deps.Cancellations`, `backups_bridge.GetCancellationRegistry` and its uses in `internal/app` (`app.go:251`, `server.go:86`) go.

Alternatives rejected:
- *A second registry for the physical side.* Nothing gains from it, and every cancel path of a database deletion would have to know which registry holds which work.

### 11. What `Deps` gains

- **`WalQueueDir`**: the root of the local WAL queues. The glossary term "Local WAL queue" that `physical-tests-on-apptest` adds gets `WalQueueDir` in its code column. `internal/app` sets it to `filepath.Join(config.DataFolder, "wal-queue")`, the directory the old code builds (`BP/wal_supervisor.go:291`), so a queue on disk survives the upgrade.
- **`WalStreamTimings`**: the type `physical-tests-on-apptest` added to `module.go`.
- **`PhysicalExecutionClock`**: a system clock for real tool durations, watchdogs, progress and summarizer waits, separate from the schedule and persistence clock as decision 4 explains.

`ClientToolsDir`, `Clock`, `Notifiers`, `Databases`, `Storages`, `Workspaces`, `Users` and `AuditLog` already exist.

Alternatives rejected:
- *A new directory for the queues.* Segments already received but not yet uploaded would stay behind in the old directory after an upgrade, and the streamer would not find them.
- *Read the data folder inside the module.* `backend/AGENTS.md` lets configuration reach a module only through `Deps`.

### 12. Temporary `module.go` methods, one per old route

| Old route | Method | Errors the adapter maps |
|---|---|---|
| `GET /database/:id/backups` | `ListPhysicalBackups(ctx, actor, databaseID, query)` | any → 400 |
| `POST /database/:id/trigger` | `RequestPhysicalBackup(ctx, actor, databaseID, kind)` | `ErrNoExtendablePhysicalChain` → 409, else 400 |
| `POST /backups/:id/cancel` | `CancelPhysicalBackup(ctx, actor, backupID)` | `ErrPhysicalBackupNotFound` → 404, else 400 |
| `DELETE /backups/:id` | `DeletePhysicalBackup(ctx, actor, backupID)` | `ErrPhysicalBackupNotFound` → 404, else 400 |
| `POST /database/:id/restore-token` | `IssuePhysicalRestoreToken(ctx, actor, databaseID, targetTime)` | `ErrUserStreamHeld` → 409, `PhysicalWalGapBeforeTargetError` and `ErrPhysicalTargetBeforeEarliest` → 422, `ErrNoPhysicalChainForRestore` → 404, else 400 |
| `POST /backups/:id/restore-token` | `IssuePhysicalBackupRestoreToken(ctx, actor, backupID)` | as above, plus `ErrPhysicalBackupNotFound` → 404 |
| `GET /restore-stream` | `OpenPhysicalRestoreStream(ctx, token)` | an invalid token → 401, `ErrUserStreamHeld` → 409, the restore set errors → 422 or 404 before the first byte |
| `GET /database/:id/wal-stream/statistics` | `GetWalStreamStatistics(ctx, actor, databaseID, period)` | any → 400 |
| `GET /database/:id/wal-stream/live` | `GetWalStreamLiveState(ctx, actor, databaseID)` | any → 400 |

The recovery script is static text and stays in the controller until `physical-backups-api` moves it with the routes. The request and response types the controller encodes today move from `FB/dto/physical` and `SV` into `physical` with their JSON tags unchanged, and `module.go` aliases them. The errors the adapter maps are aliased too. `GetLatestCompletedPhysicalFullBackup(ctx, databaseID)` serves telemetry. Every method, alias and error carries the comment that `physical-backups-api` deletes it.

**Access.** The facade looks the database or backup up and answers "not found" when it is missing. It then calls `workspaces.Authorize`, as `backend/AGENTS.md` says. A refused access comes back as an `apierror.Problem`, and the adapter's default arm still answers 400 with the error's text. Status codes therefore stay as the API tests from `physical-tests-on-apptest` expect them. Only the message text of a refused access changes, and no test or screen matches on it.

**Audit.** Audit events go through `BM/internal/auditing.Recorder`, with the same event texts the old `audit_logs` service wrote (`SV/physical_service.go:605`). Backup requests record the config update and audit in one transaction. Deletion keeps the cascade transaction boundaries fixed by decision 3, and cancellation signals a running process that cannot be rolled back. Those two actions therefore keep their separate audit transaction after the action; an audit error is logged, as before. This temporary departure from the same-transaction audit rule is recorded in the rewrite plan and ends in milestone 11.

**The restore stream.** It follows the logical download (`BM/internal/download/service.go:132-158`, `:203-224`). `OpenPhysicalRestoreStream` consumes the token and takes the user's stream. It resolves the restore set before any byte, starts the heartbeat in the group of runs, and returns the database ID for the file name, a write function and a close that releases the stream. `AcquireUserStream` and `IsUserStreamHeld` leave `module.go`.

Alternatives rejected:
- *Answer `apierror.Problem` statuses now.* The adapter would change its codes twice, here and in `physical-backups-api`, and the API tests would change in a change meant to move code under them.
- *Keep the response types in `FB/dto/physical`.* They embed stream health types that move into the module's `internal`.

### 13. `apptest` keeps its surface and changes what is behind it

- **`backups_physical_wal_stream_manager` runs only with the live jobs.** A WAL stream measured on a clock the test moves would report a false health, so without the option `RunJobPass` refuses that name and fails the test with a message saying so. With the option, a pass of it goes to the live runner.
- **`WithPhysicalBackupLoops()`** keeps its name. It builds the application on the system clock and starts a runner over two physical jobs, `backups_physical_scheduler` and `backups_physical_wal_stream_manager`, for the application's life. The cleaner stays out of it and runs only through `RunJobPass`, as the `api-contract` requirement says, so retention and the orphan sweep never run under a test on their own. `App.Clock` is nil for such an application. While the runner runs, `RunJobPass` sends a pass of one of its two jobs to it, so a test pass never overlaps a live pass of the same job.
- **Without the option**, the application keeps the manual clock, no physical job runs on its own, and `RunJobPass` runs the module's jobs.
- **`RestartModules` and cleanup** stop physical jobs and cancel module contexts, then wait through the module's `WaitForRuns`. `RestartModules` uses the production shutdown drain budget for stopping jobs, completing runs and delivering notifications, so shutdown tests prove that outcomes are persisted before the application would exit. The production `app.App.WaitForPhysicalBackups` goes with the old backuper. The test helper `apptest.App.WaitForPhysicalBackups` keeps polling the catalog for terminal backup outcomes during normal operation: the shared run group also owns long-lived WAL streamers, so waiting for the entire group would block while those streams are running. That catalog poll is not a substitute for draining runs on restart or cleanup.
- **`apptest_test.go`.** Its own tests of the loop cleanup switch from the registry to the module's runs. They test the harness, not physical behavior.

Alternatives rejected:
- *A manual clock with live jobs.* Samples taken by a frozen clock all carry one time, and the health verdict cannot tell a running streamer from a stalled one.
- *Start the application's single runner with every job.* Logical and storage jobs would then run on their own in a test that asked only for physical ones.

### 14. Tests

Unsharded package runs (`backend:test-package` and `scripts/test.sh`) allow 30 minutes per package. The WAL streaming API package exceeds the former 15-minute budget; its condition waits otherwise exhaust that shared deadline near the end of the package. CI shards and `backend:test-group` keep their 15-minute limits because they select only part of the package.

- The unit tests left in the old packages move with their code into the new subpackages. Their bodies stay; their package clauses and imports change.
- The API tests in `physicalapitests` and the end-to-end scenarios keep their code.
- New API tests in `physicalapitests/backuping` and `physicalapitests/walstreaming` pin the `application-shutdown` delta: a full and an incremental backup interrupted by `RestartModules`, a WAL stream that stops, keeps its slot and resumes, and a cancel that still records today's outcome.
- A scheduler pass followed at once by a cancel pins the end of the lost-cancel defect.
- The `application-shutdown` scenario of a client process that ignores the interrupt is covered by two existing tests. `TestDrain/finishes at the drain limit when a step ignores its context` in `lib/shutdown` shows that the drain gives up at its limit whatever the step, `WaitForRuns` included. The `api-contract` scenario "A restart fails a backup left in progress" shows that the next start fails what was left. Task 5.1 runs both packages.

The physical code then falls under `exhaustruct`, `forbidigo`, `sloglint` and the other target-layout linters, with their findings fixed rather than suppressed.

Alternatives rejected:
- *Rewrite the moved unit tests against new seams.* They test pure functions whose signatures do not change, so a rewrite would only add a diff to review.
- *Suppress the linter findings in the moved code for now.* That is the exception list this change exists to empty.

### 15. Callers outside the module

- `internal/app`: `background_loops.go`, `wiring.go`, `server.go`, `app.go` and `getters_test.go` lose the physical parts.
- `features/telemetry` reads the latest completed full backup through the temporary method instead of `CP/models` and `CP/service`.
- `cmd/seed_dev/main_test.go` and `features/test_once_protection_test.go` stop calling the old subscriptions, and the last tagged physical helpers go.

## Risks / Trade-offs

- **[The diff is the largest of the milestone.]** About 19,000 lines move, with their structure. → The tests under it do not change. The tasks move the code area by area, so each step builds, and review reads the transaction table of decision 3 against the new code.
- **[A transaction loses its context.]** A repository call inside `Transaction` that takes the outer `ctx` runs outside the transaction and breaks a lock order that has no test any more. → Decision 3 lists every transaction. `exhaustruct`, the linters and review check the calls. The held-lock tests and the end-to-end scenarios cover the claim and cascade paths.
- **[The stop takes longer than the drain.]** `pg_basebackup` and `pg_receivewal` get `SIGINT` and are killed after 10 seconds (`engines/postgresql/physical/base_backup.go:22`), while the drain gives up after 6 (`internal/app/server.go:23`). → A backup that outlives the drain is recorded at the next start, as the `application-shutdown` spec already allows. The tools usually exit on `SIGINT` at once, and the API tests check that the outcome is recorded.
- **[Heartbeats switch from the database clock to the process clock.]** → One process per installation, as `single-instance-runtime` requires. Staleness compares a time the same process wrote.
- **[A queue directory moves by mistake.]** → `WalQueueDir` is the same path the old code builds, and the end-to-end scenarios restart with a queue in place.

## Migration Plan

The schema does not change. On upgrade:
- a physical backup that ran at the stop is failed by the startup work, as today;
- streamers resume from their slots and their queue directory, which keeps its path.

Rollback is the previous image; nothing in the data depends on this change.

## Open Questions

None.
