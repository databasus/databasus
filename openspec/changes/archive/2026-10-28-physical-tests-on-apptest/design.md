## Context

Why the change is needed is in proposal.md. Paths are under `backend/internal/` unless they start with another top-level directory. Abbreviations:
- `FB` is `features/backups/backups`;
- `BP` is `FB/backuping/physical`, `CP` is `FB/core/physical`, `UP` is `FB/usecases/physical/postgresql`;
- `SV` is `FB/services`, `CT` is `FB/controllers/physical`.

**The old tests.** The physical code has about 17,000 lines of tests and tagged helpers. The inventory for this change went through every top-level test and subtest, 564 cases in all, and sorted each into one of four classes:

| Class | Meaning | Cases |
|---|---|---|
| U | pure logic, no database, no getter | 162 |
| A | can become an API test of the running application | 267 |
| E | already checked by one of the 14 end-to-end scenarios | 32 |
| X | reachable only from inside the code as written | 103 |

What the A, E and X tests depend on:
- `UP/testing.go` (1,158 lines) builds sources, fixtures and streamers through package getters and the old repositories.
- WAL streaming and upload tests replace the storages module with `FakeFileStore` (`UP/testing.go:764`) over `mockWalStorage` (`UP/mocks_test.go:26`).
- Backuper tests replace our own use cases with `fakeFull/IncrementalExecutor` and the notifiers module with `recordingNotificationSender` (`BP/mocks_test.go`).
- `CT/controller_test.go` (2,706 lines) builds its own router (`CT/setup_test.go`).
- Several `setup_test.go` files run a `TestMain` that subscribes the old event handlers without the application.

`docs/testing.md:7-28` allows module API tests on the running application, and unit tests only for pure logic. It never mocks our own modules and services.

**The test application.**
- `apptest.New` allows one application per process (`app/apptest/apptest.go:50`).
- It starts the live physical loops in `New` when asked (`apptest.go:181-195`), so the loops run before a test can seed anything.
- `RunJobPass` reaches only jobs of `background.Runner` (`app/app.go:306`).
- `RestartModules` restarts the modules and waits for their runs, but knows nothing of the physical loops (`apptest.go:260-276`).
- The cleaner runs only through `RunPhysicalBackupCleanerPass` (`apptest.go:343`).
- The module's cleanup waits only for the module's runs (`apptest.go:289-298`). A physical backup started outside the live loops would outlive it.

**The old loops.**
- **Startup work runs inside each `Run`.** The scheduler fails the backups left in progress (`BP/scheduler.go:69-114`, `:611-693`). The supervisor marks stale streamer rows failed (`BP/wal_supervisor.go:108-166`). The cleaner sweeps orphan backup slots (`BP/cleaner.go:43-67`, `:96-105`). The scheduler and the supervisor panic when their recovery fails. The slot sweep is best effort: it logs a warning and leaves a slot it cannot drop to the next orphan sweep (`BP/cleaner.go:100-104`).
- **A tick that dispatches a backup.** The scheduler's tick is `runPendingBackups` (`BP/scheduler.go:116`), which starts each backup with a bare `go` (`:385`) on a context derived from `context.Background()` (`BP/backuper.go:108`, `:214`).
- **The live state route never sees a running streamer in a test.** The route reads the singleton supervisor (`SV/di.go:31`). `StartPhysicalBackupLoops` runs a fresh instance (`app/background_loops.go:45`).
- **Timings are constants.** The forced rotation interval is in `UP/wal_rotation.go:19`, the archive staleness threshold in `CP/wal_health/constants.go:28`, and the local queue pause size in `UP/wal_backpressure.go:20`, at least 512 MiB.
- **Time is wall-clock time.** The scheduler, the cleaner and the alert throttle read `time.Now` (`BP/scheduler.go:122`, `BP/cleaner.go:73`, `:571`, `BP/wal_supervisor.go:499`). The test application's manual clock does not reach them.

**The rules that constrain the approach.** The testing skill and `docs/testing.md` fix the layers, the waiting rules and the naming of tests. `backend/AGENTS.md` keeps dependencies in `Deps` and forbids package-level state, which shapes decisions 6 and 7. The linter rejects `time.Sleep`, `Eventually` and `context.Background` in the target layout (`backend/.golangci.yml:67-87`), which is where the new tests live.

**CI.** `mise-tasks/backend/test-group-packages:16` runs `UP` as the `wal-usecases` group, and `.github/workflows/ci-release.yml:316-320` runs it in three shards. The `ci-pipeline` spec names that group.

## Goals / Non-Goals

**Goals:**
- Every old test of the physical code that is not pure becomes:
  - an API test on `apptest` that checks the same outcome;
  - a unit test of a pure function extracted from the old code; or
  - an entry in the list of drops, with the coverage it loses.
- The API tests use only what `physical-backups-module` keeps: the old routes, `apptest` methods and options, job names and SQL on tables that do not change. That change can then move the code without editing them.
- After this change, no physical test needs the `testhelpers` tag, a package getter or a fake of our own code.

**Non-Goals:**
- Changing what any physical operation does. The production edits below are seams and extractions that keep behavior.
- Testing the live state of a running streamer through the routes beyond what decision 7 enables. The full check of `GetWalStreamLive` waits for `physical-backups-module`, which owns the streamers.

## Decisions

### 1. The tests run against the old code through the old routes

The new tests call the routes of `CT` through `apptest.CallOldAPI`, `CallOldPublicAPI` and `DownloadOldPublicAPI`, as the end-to-end scenarios from `physical-e2e-on-apptest` do. They check status codes and bodies, the files in the local storage, the replication slots on the source, webhook deliveries, and, through `QuerySQL`, the columns no route returns: `error_reason`, `raw_size_mb`, the encryption columns, `backups_physical_in_flight`, `backups_physical_wal_streamers.status`, uncommitted WAL claims and stored samples. They seed with `ExecuteSQL` only catalog states the API cannot reach. The table names already match the module since `physical-backups-schema`.

Alternatives rejected:
- *Rewrite the tests together with the move.* The move would then be checked by tests written in its own diff, which the rewrite plan forbids.
- *Package tests built through `Deps` with a real internal database.* `docs/testing.md` has no such layer, and the next change would rewrite them again against the module's `Deps`.
- *The generated client.* The physical operations are not in the contract until `physical-backups-api`.

### 2. Five test-only packages under the module

The API tests live in `internal/modules/backups/physicalapitests/`, one test-only package per area. Each is its own binary, so the packages run in parallel even though `apptest` allows one application per process.

| Package | What it tests | Main sources |
|---|---|---|
| `catalog` | the list with its cursor, filters and WAL ranges; trigger, cancel and delete; access and audit; the reactions to a saved config; retention and the orphan sweep; deleting a database | `CT/controller_test.go`, `CT/wal_ranges_test.go`, `BP/cleaner_test.go`, `BP/cancellation_listener_test.go`, `CP/service/service_test.go`, `CP/chain_view/service_test.go` |
| `restoring` | restore tokens, the restore stream and its layout, the recovery script, the restore set | `CT/controller_test.go`, `CP/chain_view/restore_set_test.go`, `FB/download/restore_stream/writer_test.go` |
| `streamstatistics` | the statistics and the live state routes, the restore coverage they report, the health codes | `CT/controller_test.go`, `CP/chain_view/restore_coverage_test.go` |
| `backuping` | scheduler passes, full and incremental outcomes, backup slots, the summarizer, mTLS, notifications, restart recovery | `BP/scheduler_test.go`, `BP/backuper_test.go`, `BP/client_processes_test.go`, `UP/backup_slot_test.go`, `UP/create_*_test.go`, `UP/summarizer_test.go`, `UP/stream_test.go`, `UP/mtls_test.go` |
| `walstreaming` | streamer claims and sessions, uploads, rotation, the lag monitor, back pressure, chain-risk alerts, the slot sweep, deleting a streaming database | `BP/wal_supervisor_test.go`, `UP/wal_*_test.go`, `UP/lag_monitor_test.go`, `UP/receivewal_process_test.go`, `UP/slot_*_test.go`, `modules/databases/deletion_test.go:104-150` |

Each top-level test function names what it tests, and each subtest names its result and condition, as the testing skill says.

Alternatives rejected:
- *The module's root test package `internal/modules/backups`.* Its tests run one application at a time. Hundreds of tests with source containers and 30-second samples would outlast the 15-minute run limit of `backend:test-package` (`mise.toml:92-104`).
- *One package of physical API tests.* It has the same limit.
- *Under `internal/modules/backups/e2e`.* That directory holds the backup and restore cycle layer of `docs/testing.md`, which these tests are not.

`docs/testing.md` gets one sentence in the layers table: a module whose API tests are too slow for one package may split them into test-only packages under its directory.

### 3. Every old test gets a recorded destination

`coverage.md` in this change's directory lists every old top-level test and subtest with its destination: the new test and subtest name, a unit test, an end-to-end scenario, or a drop with its reason and the coverage it loses. Review compares old and new against that table, as it did for the end-to-end scenarios.

`coverage.md` names at its top the commit the change starts from. Each task of section 3 first lists every top-level test and every subtest of its source files at that commit, then fills in the destination of each. A check confirms the table is complete. For every source file, it takes the names that `git show <base>:<file> | rg -o --replace '$1$2' 'func (Test\w+)|t\.Run\(\s*"([^"]+)"'` prints, and confirms that `rg -F` finds each name in `coverage.md`. Subtests whose names come from a table field are listed by the case names in that table.

The classes follow these rules:

- **U.** The test stays in its file in the old package and moves with its code in `physical-backups-module`. Its file loses the `testhelpers` tag and any `TestMain` it no longer needs.
- **A.** The test becomes an API test in one of the five packages.
- **E.** The test is deleted, and `coverage.md` names the scenario that covers it. Where the old test checked something the scenario does not, the missing check joins a test in one of the five packages. Examples are the slot sweep of an orphan slot before a database deletion, an incremental that completes after a WAL gap, and an encrypted manifest.
- **X with a pure core.** Decision 10 extracts the logic, and the test becomes a unit test of it.
- **X without one.** The test is dropped. The list is fixed in decision 11.

Alternatives rejected:
- *A summary per old file.* Review could not check that a given old check survived, which is the point of moving tests before the code.

### 4. One pass of the scheduler and the cleaner by the module's job names

`app.RunJobPass` answers two more names until `physical-backups-module` makes them jobs of `background.Runner`:
- `backups_physical_scheduler` runs one scheduler pass;
- `backups_physical_cleaner` runs one cleaner pass.

The names follow the existing jobs `backups_logical_scheduler` and `backups_logical_retention`. The scheduler gains `RunPass(ctx) error`, the body of today's tick (`BP/scheduler.go:116`). `Run` calls it on each tick and logs its error, as the cleaner already does with its own `RunPass` (`BP/cleaner.go:69`). A pass returns once it has dispatched its backups. Tests wait for the outcome on the list route with `testwait.Until`, as they will once the backups run in the module's group of runs.

A scheduler pass is the only way to check that the scheduler starts nothing: a test seeds the state, runs the pass, and checks that the list has no new row. Live loops cannot prove an absence without waiting. `apptest.RunPhysicalBackupCleanerPass` is deleted, and the two end-to-end scenarios that use it call `RunJobPass` with the cleaner's name.

`apptest` must know when the backups a pass dispatched have ended, and the shared cancellation registry cannot tell it. The scheduler dispatches a backup with a bare `go` (`BP/scheduler.go:385`), and the backup registers only inside `MakeBackup` (`BP/backuper.go:108-109`, `:214-215`). Right after a pass, the registry can be empty while a backup is about to start. In-flight claims cannot tell it either, because tests seed claims with SQL that no backup ever releases. So the old backuper gains `Dispatch(ctx, backupID, isCallNotifier)`, which counts the run in a `sync.WaitGroup` of the backuper and starts `MakeBackup` as the scheduler does today. It also gains `WaitForDispatchedBackups(ctx) error`. The scheduler calls `Dispatch` where it now writes `go s.backuper.MakeBackup(...)`. That keeps behavior: the same goroutine starts at the same moment with the same context, and only a counter is added. `Serve` never waits on the counter, so a stop of the server behaves as today. `app.WaitForPhysicalBackups(ctx)` exposes the wait. `apptest`'s module cleanup and `RestartModules` call it, with or without the live loops. The wait covers every backup that a pass or the live scheduler started, and nothing that a test only seeded. Notification checks then wait through `app.WaitForNotificationDeliveries`, which delegates to the notifiers module. Cleanup and restart also drain deliveries after stopping their emitters, before closing the database or restarting modules.

Alternatives rejected:
- *`apptest` methods named after the old code, such as `RunPhysicalSchedulerPass`.* The tests would change again when the scheduler becomes a job.
- *Making the old loops jobs of `background.Runner` now.* The streamers derive their context from the supervisor's `Run`. Moving them is the heart of `physical-backups-module`, and doing it here would move half the change into the test change.

No pass is offered for the supervisor's reconciliation, for the same reason. WAL streaming tests use the live loops.

### 5. The physical startup work runs when the modules start

The startup work of the three loops leaves their `Run`. `BP` gets one function, `RunStartupWork(ctx) error`, which runs in this order:
1. the scheduler's recovery of backups left in progress;
2. the supervisor's recovery of stale streamers;
3. the cleaner's sweep of orphan backup slots.

`app.StartModules` calls it after `backupsModule.Start` (`app/app.go:296`). An error of step 1 or 2 now stops the start of the application instead of panicking a loop, which has the same effect for the user. Step 3 stays best effort: it logs a warning and returns no error, so a source that cannot be reached at boot does not block the start, as today. On the server it runs before the loops start (`app/server.go:55`). That also closes an ordering gap the inventory found: the slot sweep no longer runs at the same moment as the scheduler's first tick.

`apptest.RestartModules` uses this:
- In both branches, the restart first waits for the dispatched physical backups through `app.WaitForPhysicalBackups` (decision 4), as the `api-contract` requirement says, so the startup work never fails a backup that is still running.
- Without the live loops, it then restarts the modules, which runs the startup work as the server does.
- With the live loops, it stops the scheduler and the supervisor as a shutdown does, before that wait, so no new backup starts during it. The streamers stop and keep their slots and their local queue. Stopping the loops waits for their group, which ends only after every streamer has stopped. The restart then waits for the dispatched backups, restarts the modules, which runs the startup work, and starts the loops again.

Tests use the restart for the startup checks: restart recovery, the boot slot sweep, stale streamer recovery and a new streaming session. `physical-backups-module` puts the same work into `Module.Start`, so the restart keeps meaning the same thing.

A restart while a physical backup runs therefore waits for the backup to end, as the requirement says. No test of this change relies on that wait for its outcome. After `physical-backups-module`, a restart interrupts such a backup and fails it with the shutdown message instead, and that change states and tests the new behavior on its own.

Alternatives rejected:
- *One `apptest` method per startup step.* The steps merge into `Module.Start` in the next change, and the tests would change with them.
- *Keep the work inside `Run` and restart only the loops.* A restart without the live loops would then skip it, and the meaning would differ between the two changes.

### 6. WAL stream timings through `Deps`, shortened only by tests

`module.go` of `backups` gains `WalStreamTimings` and `ProductionWalStreamTimings()`, the way `storages` declares `FileTimings` and `ProductionFileTimings` (`modules/storages/module.go:77`). It has three fields:
- `ForcedRotationInterval`: five minutes in production;
- `ArchiveStalenessThreshold`: thirty minutes;
- `LocalQueuePauseBytes`: zero in production, which keeps today's size derived from the segment size (`UP/wal_backpressure.go:43`).

`app.Environment` gets the field, and the server fills it with the production values. `NewPhysicalWalStreamSupervisor` takes it and passes it on where the constants are read today. The server no longer runs the supervisor, the scheduler and the cleaner as `sync.OnceValue` singletons (`BP/di.go:45-105`): `app.New` builds the three from `Environment` with their constructors, and `Serve` runs those instances. `StartPhysicalBackupLoops` builds its own instances the same way. The getters `GetPhysicalBackupsScheduler`, `GetPhysicalBackupCleaner` and `GetPhysicalWalStreamSupervisor` go, together with their checks in `app/getters_test.go`. The timings reach the code at:
- `BP/wal_supervisor.go:317`, `:318`;
- `UP/wal_stream.go:288`.

`apptest.WithWalStreamTimings(timings)` sets it for one application. In `physical-backups-module` the same field reaches the module's `Deps`.

The three fields bring three terms into a `module.go`, so `docs/glossary.md` gains them in its Product table, with the wording of the main specs:
- "Forced WAL rotation": a WAL switch Databasus asks the source for, so a segment completes even when the source writes rarely;
- "Archive staleness threshold": how long the archive progress point may stay still while the source writes before archiving counts as fallen behind (`physical-backups/wal-archiving`);
- "Local WAL queue": the directory where `pg_receivewal` writes segments before they are uploaded.

The sample interval of the lag monitor stays fixed. The health verdict counts in samples (`CP/wal_health/constants.go:9`), so a shorter interval would test different arithmetic from the server's. Tests that need two samples wait about a minute.

Alternatives rejected:
- *Package variables that tests overwrite.* That is process-wide state, which `backend/AGENTS.md` forbids.
- *Dropping the tests that need the timings.* They pin requirements of `physical-backups/wal-archiving` and `physical-backups/wal-stream-monitoring`: the alert on a refused rotation, the stale archive alert, and the pause and restart counters.

### 7. The live state route reads the supervisor that runs

`BP` keeps the running supervisor in an atomic pointer. `Run` sets it on entry and clears it on return. `SV` reads it on each call to the live route instead of holding the singleton from `SV/di.go:31`. The pointer covers the server's supervisor and the test application's alike, because both now come from the constructor (decision 6) and both go through `Run`. With the live loops, the route reports the streamer the test application runs. Without them, it reports that no streamer runs, as the server does before its loops start. The pointer is process-wide state, tolerated in the old code until `physical-backups-module` deletes it. `apptest` already allows one application per process.

Alternatives rejected:
- *Let `StartPhysicalBackupLoops` run the singleton.* Its `Run` panics on a second call, and every test builds a new application.
- *Keep `RegisterTestRunningStreamer` (`BP/testing.go:116`).* It injects a never-started streamer, a fake of our own code.

### 8. Real failures instead of fakes of our own code

| Old fake or setup | Real failure used instead |
|---|---|
| `FakeFileStore` failing writes | an S3 storage pointing at `127.0.0.1:1`, created through the storages API |
| `FakeFileStore` failing, then recovering | the same storage, then an update through the storages API that points it at a MinIO from `containers.StartMinio` |
| a fake executor returning an error | a database row whose host is changed to `127.0.0.1:1` with `ExecuteSQL` |
| a fake executor returning a server rejection | removing the physical-replication `pg_hba.conf` entry on the source after API registration and reloading PostgreSQL, so the real `pg_basebackup` receives the server rejection. Revoking `REPLICATION` stops earlier at slot creation, so it cannot cover the tool exit text |
| the no-summary container | `ALTER SYSTEM SET summarize_wal = off` and a reload, because the API refuses to create an incremental database on a source without summaries |
| a corrupted parent manifest | overwriting the stored manifest file in the local storage |
| a tool failing after writing output | terminating the `pg_basebackup` walsender on the source mid-run |
| `recordingNotificationSender` | a webhook notifier attached to the database, with the notification triggers the test needs |
| `registerTestTask` in the registry | a real backup and a real streamer |

Tests that need a stub `pg_basebackup` written into the client tools directory become real-failure tests as above. Where only the stub could reach the case, the test is dropped (decision 11). No override of the client tools directory is added.

Alternatives rejected:
- *Keep `FakeFileStore` and the fake executors.* They replace our own storages module and use cases, which `docs/testing.md` forbids, and they rely on the old package's seams that the next change deletes.
- *An option that points the application at stub client tools.* It would add a production seam for a handful of cases that a real failure reaches as well.

### 9. Races through a lock the test holds

`apptest` gets `BeginSQLTransaction(tb)`, a transaction on its own connection that the test commits or rolls back when it chooses. It generalizes `LockStorageRow` (`app/apptest/storage_locks.go:20`). Two races keep their tests this way:
- **A claim waits for the orphan sweep.** The test holds the orphan cleanup advisory lock, runs a scheduler pass in a goroutine, and checks that no claim appears until it releases the lock (`BP/scheduler_test.go:655`).
- **A cascade delete keeps WAL for a full backup that claims meanwhile.** The test holds the root full's row lock while it starts a delete and a claim (`CP/service/service_test.go:326`).

Races that need a gate inside our code have no such handle. They are dropped and listed in decision 11.

Alternatives rejected:
- *Drop every race test.* Two of them reach their race through a lock the test can hold, and they cover the claim and cascade paths `physical-backups-module` rewires.
- *Gates inside our code that tests open and close.* They would be production seams that exist only for tests.

### 10. Pure logic is extracted, then unit-tested

Each extraction keeps behavior. The old test passes against the extracted code first. Then it is rewritten as a unit test of the pure function, and the integration version is deleted.

| Extraction | Old code | Old tests |
|---|---|---|
| The scheduler's decision as a pure function of a loaded state, the config and the time. A loader does the reads, including the mismatch query. | `BP/scheduler.go:174-269` | `BP/scheduler_test.go:124` and `:332-579` (14) |
| The backup notification without the workspace lookup | `BP/backuper.go:828` | `BP/backuper_test.go:377`, `:412`, `:427`, `:652` |
| The chain alert throttle as a type with an injected time | `BP/wal_supervisor.go:495-502`, `:552` | `BP/wal_supervisor_test.go:276`, `:303`, `:332` |
| `buildChainRiskAlert`, already pure | `BP/wal_supervisor.go:432` | `BP/wal_supervisor_test.go:362` |
| The receiver restart latch | `UP/wal_stream.go:541-556`, `UP/receivewal_process.go:28` | `UP/receivewal_process_test.go:100` |
| The slot probe outcome | `UP/slot_lsn_watcher.go:109` | `UP/slot_lsn_watcher_test.go:145` |
| The summarizer wait, taking a probe and a clock | `UP/summarizer.go:120`, `:143` | `UP/summarizer_test.go:155`, `:169`, `:194`, `:211`, `:223` |
| The back pressure size and backlog over a directory | `UP/wal_backpressure.go:43`, `:85-89` | `UP/wal_backpressure_test.go:16`, `:34` |
| The live state built from its inputs | `UP/wal_live_state.go:46` | `UP/wal_live_state_test.go` (4) |
| The resume path plan, apart from the probe | `UP/wal_resume.go:62` | `UP/wal_resume_test.go:73` |
| The decision for an existing WAL claim | `UP/wal_upload.go:130`, `:202` | `UP/wal_upload_test.go:165`, `:190`, `:331`, `:373` |
| The WAL gap decision | `UP/wal_upload.go:341`, `:391`, `:399` | `UP/wal_upload_test.go:270`, `:298`, `:544-628` (10) |
| The archiving statistics | `UP/wal_upload_stats.go:47-71` | `UP/wal_upload_test.go:471` |
| The upload dispatcher with an injected upload step, and the local queue listing | `UP/wal_upload_dispatcher.go:19-98`, `UP/wal_local_queue.go:25-55`, `UP/wal_staleness.go:124` | `UP/wal_watch_dir_test.go` (10) |
| Segment bounds for a 64 MiB segment, a new table case | `UP/wal_upload.go:550` | `UP/wal_stream_test.go:257` |
| `buildSlotBreakSample` | `UP/lag_monitor.go:273` | `UP/wal_stream_test.go:967` |

The dispatcher test runs the dispatcher with a function in place of one upload. It checks the dispatcher's own concurrency limit, order and recovery. That is pure logic too slow to enumerate through HTTP, not a fake of the storages module.

The scheduler's decision function keeps both of today's checks for an immediate replacement. Merging them stays in milestone 11.

### 11. What is dropped, and what it costs

Dead code goes together with its tests, after a reference search confirms nothing calls it:
- the dependents summary preview `GetDependentsSummary` (`CP/service/service.go:308`), with `CP/service/service_test.go:498` and `:537`;
- `CheckHistoryFilePresence`, with `CP/chain_view/service_test.go:281` and `:294`;
- the protected-slot branch of the startup slot sweep, whose only caller passes "never protected" (`BP/cleaner.go:97`), with `UP/backup_slot_test.go:167`.

Dropped tests, with the coverage lost:

| Old test | Why it cannot be an API or unit test | Lost |
|---|---|---|
| `BP/scheduler_test.go:140`, `:168`, `:188` | a decision going stale between the decision and the claim inside one pass | the stale-decision guard (`BP/scheduler.go:562-600`), which milestone 11 replaces |
| `BP/scheduler_test.go:624`, `:876`, `BP/wal_supervisor_test.go:104`, `UP/wal_upload_test.go:421` | concurrent claimants, which a pass of one job and the trigger route never produce | the repositories' answer under concurrent claims; the unique keys stay in the schema |
| `BP/backuper_test.go:808` | the row stops being in progress while the backup persists its result | the guard that keeps a finished backup from being revived |
| `BP/backuper_test.go:1153`, `UP/wal_upload_test.go:640` | the database deletion holding its lock at an exact point | the deadlock-free lock order against a deletion |
| `UP/wal_upload_test.go:227` | a storage write held open while a cascade runs | the null claim left by a cascade during an upload |
| `BP/backuper_test.go:116`, `:303`, `:543`, `:556`, `:606`, `BP/scheduler_test.go:927` | branches no caller reaches: a missing root full, an enabled config without a storage, a row absent from both tables, an internal write failure | defensive branches |
| `BP/backuper_test.go:789` | a claim held by another backup ID, which the scheduler never creates | the backup-scoped claim release |
| `BP/scheduler_test.go:941`, `BP/wal_supervisor_test.go:148` | the guard against a second `Run` | nothing; `physical-backups-module` deletes the guard |
| `UP/backup_slot_test.go:566` | it pins a known limitation, not a behavior | nothing |
| `UP/deleted_database_slot_test.go:21` | a streamer starting after its database is deleted | `createSlotUnlessDatabaseDeleted` (`UP/deleted_database_slot.go:24`) |
| `UP/lag_monitor_test.go:833` | five minutes of receiver running time before a rebuild | that a sample is stored before the rebuild |
| `UP/slot_cleanup_listener_test.go:135` | duplicates the engine's `replication_slot_test.go:107` | nothing |
| `UP/slot_lsn_watcher_test.go:180` | a source that accepts but never answers the handshake | the handshake timeout of the slot probe |
| `UP/stream_test.go:313`, `:337` | the pure timeout is covered by `UP/stream_test.go:291` | the call site in `UP/manifest_sidecar.go:45` |
| `UP/wal_upload_test.go:129`, `:146` | covered by the engine's `receivewal_directory_test.go:88` | the call sites |
| `FB/download/restore_token/store_test.go:59` | it fakes our own cache store | an error passed through unchanged |

### 12. Time stays real, and tests leave the clock alone

The physical tests never move `App.Clock`. They seed timestamps relative to the real current time, with margins of minutes, never seconds. Examples are a claim older than the one-hour grace period, a sample older than the retention, and a full backup outside the cadence. The same tests then hold in `physical-backups-module`, where the scheduler and the cleaner take their time from the module's clock. That clock starts at the real time and does not move in these tests.

A check that something does not happen runs a fixed number of explicit passes and then checks the state. The linter rejects `time.Sleep` and the `Eventually` family in the target layout (`backend/.golangci.yml:67-72`). This change adds its own rule against `require.Never`, which task 4.4 checks, because a bounded wait for an absence passes on a slow machine by luck. Live-loop tests check only positive outcomes with `testwait.Until`.

Alternatives rejected:
- *Move the manual clock in the tests.* The old code reads `time.Now`, so a moved clock changes nothing it decides, and the same test would behave differently after the move.

### 13. Seeders in `apptest`, scenario helpers in the packages

`apptest` gets the physical catalog seeders next to `InsertDatabase` and `InsertPhysicalBackupConfigUsingStorage`:
- `InsertPhysicalDatabase`, which writes a physical database row without a source server, for tests that need no server;
- inserts for full and incremental backups, WAL segments, history files, in-flight claims, streamers and stream samples;
- `PublishStorageFile`, a published file that the storage deletion job keeps;
- `OpenOldPublicAPI`, a streaming call that the test closes when it chooses, so a test can hold a user's restore stream through the real route;
- `IsWalQueueDirectoryPresent`.

Helpers that only one package needs stay in that package's `*_test.go` files. `physicaltest` keeps the end-to-end bodies.

`InsertPhysicalDatabase` writes the row with SQL, as `InsertDatabase` does for logical databases (`app/apptest/databases.go:42`). A test that needs a reachable source creates the database through the API instead.

Alternatives rejected:
- *The seeders in `physicaltest`.* That package holds the end-to-end bodies and their source helpers. `apptest` already holds the SQL seeders of the other modules.
- *Create every database through the API.* Each catalog test would need a source container just to pass the connection test.

### 14. CI runs the new packages as `physical-api`

`mise-tasks/backend/test-group-packages` maps `physical-api` to `./internal/modules/backups/physicalapitests/...`. The `other` group leaves that subtree out and takes `UP`, which now holds only unit tests. `.github/workflows/ci-release.yml` replaces the three `wal-usecases` shards with `physical-api` shards. `backend/test-shard-weights.txt` drops the old names and records the new functions with the durations of one local run.

The WAL scenarios together exceed one test binary's 15-minute limit. `backend:test-group` accepts one or more existing CI shard names, validates every selection before starting work, and runs the selected shards concurrently as separate `go test` processes. It preserves the package task's race checking, build tag and timeout. Each test process stops after its first failed test, as CI does, so the failure stays visible in review output. It waits for every shard and reports a failure if any shard fails. Interrupted runs terminate their own test processes.

The implementation review accepts this command only for shards of `physical`, `logical` and `physical-api`, with positive shard numbers within their group. It rejects unsharded groups, `other` and shell syntax. Change verification runs the three physical-api shards together, the three end-to-end physical shards together, and the other affected packages together. This keeps all checks under the existing review time budget without shortening tests or raising global timeouts.

Alternatives rejected:
- *Add the packages to the `physical` group.* Its three shards already carry the end-to-end scenarios at about seven minutes per version.
- *Leave them in `other`.* That group runs whole, and it would wait for the slowest streaming tests.

### 15. What stays of the old test code

After this change, the old physical packages keep only the U tests and the helpers those tests use, such as `capturedLog`, which moves to the file of the tests that use it. These tagged files stay, because code outside physical backups still uses them until `physical-backups-module`:
- `BP/database_deleting_subscription_testhelpers.go`, used by `cmd/seed_dev/main_test.go:350-351` and `features/test_once_protection_test.go:14-15`.

`SV/database_events_subscription_testhelpers.go` goes: its only caller is `CT/controller_test.go`, which this change deletes.

`internal/modules/databases/deletion_test.go:104-150` stops registering a fake streamer in `backups_bridge.GetCancellationRegistry()`. Its scenario, "stops the WAL streamer and drops its replication slots after the database is gone", moves to `walstreaming` with a real streamer.

## Risks / Trade-offs

- **[Run time grows.]** About 270 API tests replace in-process tests. Streaming tests wait on real samples and source containers. → Five packages run in parallel, CI shards `physical-api` three ways, and tests in one package share one source container per top-level test where their states do not interfere.
- **[Race coverage drops.]** Decision 11 lists races no request can reach. The largest losses are the deadlock-free lock order against a database deletion and the no-revive guard. `physical-backups-module` rewires the transactions these depend on. → Its design lists every transaction of the old code with the statements inside it, and its review compares them one by one. The two races that a held lock can reach keep their tests.
- **[A fast backup hides its slot.]** A backup can finish before a test sees its slot on the source. → Slot tests write enough data first that a full backup takes seconds, and they observe the slot through `testwait.Until` while the backup is listed as in progress.
- **[Statistics show only the newest sample.]** Per-sample history is visible only through the series buckets. → Tests that checked one sample's fields check `current` after the sample they need. Ordering checks inside one tick are dropped (decision 11).
- **[The inventory was produced by reading, not by running.]** Line numbers and classes can drift. → Building `coverage.md` is the first task of each package, and it lists any test whose class changes on closer reading, with the reason.
- **[Webhook alerts are throttled.]** "Alerts once" cannot tell the incident tracker from the throttle. → The throttle has its own unit tests (decision 10). API tests wait for completed deliveries and check exact counts across repeated failures in one process; a process restart starts a new throttle window.

## Migration Plan

The schema and installation steps stay unchanged. Verification also repairs the failures it exposes: stream cancellation classification, malformed parent manifest classification, and terminal timestamps. The failed-upload recovery test observes the removal of one recorded claim ID, so an immediate retry cannot hide that the failed attempt released its claim. The slot-advance recovery test waits for a failed upload of the selected segment before interrupting its receiver, so it cannot mistake an earlier queued segment for the one being tested. The backpressure test generates segments only until its queue reaches the pause threshold and waits for source-side flush confirmation before each subsequent rotation. This keeps a second legitimate queue overflow from changing the intended single-pause scenario.

## Open Questions

None.
