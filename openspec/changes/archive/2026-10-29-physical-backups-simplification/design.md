## Context

Why the change is needed is in proposal.md, section Why. Paths are under `backend/` unless they start with another top-level directory, and `P/` stands for `internal/modules/backups/internal/physical`.

**The tables.** `backups_physical_full` and `backups_physical_incremental` have the same columns, except that an incremental also has `root_full_backup_id` (`NOT NULL`, `RESTRICT` to the full) and `parent_incremental_backup_id` (`NULL` when the parent is the full, `RESTRICT` to the incremental) (`migrations/20260523120000_add_physical_pg_17_backups.sql:135-233`). `migrations/20261027214332_move_physical_backups_into_module.sql` gave them the module prefix, `RESTRICT` to `databases` and to `storages`, and `CHECK` constraints on the status, the compression, the encryption and the error reason. The WAL segments, the history files, the streamers and the samples reference the database, not a backup, so nothing outside the two tables references a backup row. `backups_physical_in_flight` holds one row per database with `backup_type` and `backup_id` and no reference to the backup row (`20260523120000:236-257`).

**Who writes the status.**

- `Scheduler.claimAndInsert` inserts the claim and the `IN_PROGRESS` row in one transaction under the advisory lock `AcquireBackupAndOrphanCleanupLock` (`P/backuping/scheduler.go:274-358`).
- `Backuper.saveTerminalResultIfInProgress` records the database's backup success, reads the status with `SELECT ... FOR UPDATE`, confirms the file writes or requests their deletion, writes the result with an `UPDATE` by ID without a status condition and releases the claim (`P/backuping/backuper.go:637-685`). It is reached from `persistFullResult` and `persistIncrResult` (`:500-596`), which fill the same fields of two types.
- `Scheduler.failBackupAndReleaseClaim` fails an orphan at startup with a conditional `UPDATE` (`P/catalog/catalog_repository.go:262-271`), requests its file deletions and releases the claim (`P/backuping/scheduler.go:525-562`). The startup sweep runs it twice: once for every claim and once more, in `failClaimlessInProgressBackups`, for running rows without a claim (`:45-70`, `:389-449`).
- `Canceller.cancelClaim` cancels the run's context and releases the claim, but leaves the row to the run (`P/backuping/canceller.go:57-72`).
- `UpdateStatus` and `UpdateFailure` of both repositories have no callers.

**The cancel and delete paths.** The cancel operation resolves the backup in both tables (`P/service.go:104-124`) and calls the canceller (`P/backup_cancellation_service.go:14-48`). Turning backups off calls `CancelInFlightForDatabase` (`P/config_reaction_service.go:21`). Deleting a running backup cancels it before the deletion runs and then deletes the rows (`P/backup_deletion_service.go:58-126`). A database deletion cancels after commit (`P/database_events_service.go:10-27`). `pg_basebackup` gets SIGINT and, 10 seconds later, SIGKILL (`internal/engines/postgresql/physical/base_backup.go:22`, `:99`). All backups of a database share one slot name: `WithBackupSlot` drops the slot by name before it creates its own and again in a deferred cleanup (`P/backuping/backup_slot.go:33-89`), and dropping an active slot fails (`internal/engines/postgresql/physical/slot.go:68-76`).

**The scheduler.** `decidePhysicalBackup(state, config, now)` is already a pure function with table tests (`P/backuping/scheduler_decision.go:16-83`). `loadSchedulingState` repeats the order of its conditions to read the chain only when an incremental is due (`P/backuping/scheduler_state.go:25-80`), because the spec `physical-backups/background-load` forbids reading chain history on a pass with nothing due. Under the lock, `claimAndInsert` checks the system identifier mismatch again and calls `isImmediateReplacementTriggerCurrent` (`P/backuping/scheduler.go:360-387`), a second copy of `decideImmediateReplacementFull` (`:224-258`).

**Who reads the claim.** The cancel and delete paths above. The statistics read whether a full backup is pending (`P/wal_stream_statistics_service.go:74-87`). The WAL orphan cleanup skips a database whose full backup is running (`P/chains/service.go:133-175`). The two lookups `HasInFlightFullBackup` and `HasCompletedFullBackupCoveringWal` live in the claim's repository, although the second one reads full backups (`P/catalog/in_flight_backup_repository.go:105-143`).

**Who reads the two tables.** `GetTotalUsageMBByDatabase`, `backupListItemsQuery`, `GetNextAnchorLSNAfter` and `HasUnresolvedSystemIdentifierMismatch` join them with `UNION ALL` (`P/catalog/catalog_repository.go:18-260`). A WAL query reads full backups (`P/catalog/wal_segment_repository.go:129-160`). The database deletion deletes from both tables and the claims (`P/catalog/database_deletion_repository.go:14`). Outside the package, `apptest` seeds rows into all three tables and waits on both (`internal/app/apptest/physical_{full,incremental,in_flight}_backup.go`, `apptest.go:327`), `physicaltest/promotion.go:110` reads incrementals, `internal/features/workspaces/testing/testing.go:161-171` deletes from all three, and 22 API test files under `internal/modules/backups` read or write the tables with SQL.

**The contract.** The list item carries `rootFullBackupId` and `parentIncrementalBackupId` (`api/openapi.yaml:4917-4921`). The module contract exposes `PhysicalFullBackup = catalog.FullBackup` and `GetLatestCompletedPhysicalFullBackup` to telemetry (`internal/modules/backups/module.go:124`, `:350`; `internal/features/telemetry/di.go:123`).

**The compression.** Each backup tries `zstd`, `gzip` and none in order inside one slot, and only the codec of the attempt the source accepted is stored (`P/backuping/executor.go:59-129`, `internal/engines/postgresql/physical/codec.go:20`). Nothing reads a codec back for the next backup.

**`docs/architecture.md`, "Physical backups"** (`docs/architecture.md:206-224`) already describes the catalog as the only writer of the status and the one table with `kind`. It also says that the codec that worked is remembered and that the notifications read the sample-based verdict. Neither is true of the code, and the second contradicts the spec `physical-backups/wal-stream-monitoring`.

## Goals / Non-Goals

**Goals:**

- One backup type, one repository and one table for full and incremental backups, with the chain and the running-backup rule enforced by the schema.
- One service that writes the status of a backup, and no other writer.
- No new backup of a database while the process of an earlier one is still alive.
- One decision function for the scheduler, used before and under the lock, that still reads no chain history on a pass with nothing due.
- `docs/architecture.md` and the specs describe the code as it is.

**Non-Goals:**

- Changing what the scheduler chooses in any state, the retry and escalation rules, the notifications or their text.
- Changing the chains, WAL ownership and retention logic beyond replacing two tables with one in their queries.
- Changing the API contract or the frontend.

## Decisions

### 1. One table `backups_physical`

The table holds every column the two tables share, plus:

- `kind TEXT NOT NULL`, `FULL` or `INCREMENTAL`;
- `parent_backup_id UUID`, the direct parent: the full backup for the first incremental of a chain, otherwise the previous incremental;
- `root_full_backup_id UUID`, the full backup that heads the chain. The name stays as it is in the incremental table and in the contract.

Both references point to `backups_physical (id)` with `RESTRICT`. The references to `databases` and `storages` stay `RESTRICT`. The constraints are:

- `ck_backups_physical_kind`;
- `ck_backups_physical_chain`: a full backup has neither reference, an incremental has both;
- `ck_backups_physical_completed`: a `COMPLETED` row has `file_name`, `start_lsn`, `stop_lsn` and `completed_at`;
- the status, compression, encryption and error reason checks of `20261027214332`, unchanged in their values;
- `uq_backups_physical_database_in_progress`: a unique index on `database_id WHERE status = 'IN_PROGRESS'`.

The indexes are those of both tables, under the names of `20261027214332`'s convention: `idx_backups_physical_database_id_status_created_at`, `idx_backups_physical_database_id_timeline_id`, `idx_backups_physical_root_full_start_lsn` and `idx_backups_physical_parent_backup_id`. A new `idx_backups_physical_database_id_kind_created_at` serves the newest full and the newest incremental, which the scheduler reads every second.

The code gets one type `catalog.Backup` with `Kind`, `ParentBackupID` and `RootFullBackupID`, and one `catalog.BackupRepository`. `Kind` has its own type `catalog.BackupKind` with `FULL` and `INCREMENTAL`, the values `ck_backups_physical_kind` allows. `BackupType`, which also holds `WAL`, stays the discriminator of the list, whose rows include WAL ranges. Every pair collapses: the two `persist*Result` and `finalize*` families of the backuper, the two notification builders, the lookups by ID that try both tables, and the queries that join the full and incremental tables with `UNION ALL`. Unions with WAL rows remain because the list also contains WAL ranges. The list keeps its contract: `rootFullBackupId` is the column, and `parentIncrementalBackupId` is `parent_backup_id` when it differs from `root_full_backup_id`.

Alternatives rejected:

- *Keep two tables and share code through helpers.* The pairs leave the code but stay in the schema, `UNION ALL` stays, and the running-backup rule still needs a third table, because a unique index cannot span two tables.
- *Only `parent_backup_id`, without the root.* The chain of a backup would then need a recursive query. The scheduler, the restore set, the chain view and the cleaner read "all incrementals of this full" on every pass, and `physical-backups/background-load` bounds that work.
- *Keep `parent_incremental_backup_id` with its `NULL` for "the parent is the full".* Every reader of the parent would have to treat `NULL` as the chain's full backup. A direct parent reference says the same thing without a special case.
- *A `CHECK` that a failed backup has an error reason.* `20261027214332` deliberately cleared unknown reasons, so a failed row without a reason is a valid state on upgraded instances.
- *A `CHECK` that the chain root is a full backup.* A `CHECK` cannot read another row. Only the scheduler creates incrementals, and it takes the root from an extendable chain.

### 2. The running row is the claim

`backups_physical_in_flight` is dropped. The partial unique index of decision 1 holds the rule "at most one running backup per database", which `docs/architecture.md`, "Data", asks to express as a partial unique index. Starting a backup is one `INSERT ... ON CONFLICT (database_id) WHERE status = 'IN_PROGRESS' DO NOTHING`, and a finished transition releases the database by leaving `IN_PROGRESS`. Nobody releases a claim, and no state with a claim but no row, or a row but no claim, can exist. The startup sweep therefore has one pass over the running rows.

The advisory lock that orders a backup start against the WAL orphan cleanup stays, with its callers. It moves from the claim's repository to `BackupRepository`, together with `HasCompletedFullBackupCoveringWal` and the lookup of a running full row, which replaces `HasInFlightFullBackup`. The statistics read the running row instead of the claim.

The glossary term becomes "running backup" (decision 12), so the code drops "in-flight" for backups, as the glossary allows one term per concept: `HasInFlightFullBackup` becomes `HasRunningFullBackup`, `CancelInFlightForDatabase` becomes `CancelRunningForDatabase`, `CancelInFlightBackup` and `CancelInFlightBackupSpec` become `CancelRunningBackup` and `CancelRunningBackupSpec`, `stopInFlightForDelete` becomes `stopRunningForDelete`, and the locals and fields `hasInFlightFull`, `inFlightBackup`, `inFlightBackupID`, `inFlightRepo`, `inFlightBackupRepository` and `InFlightBackups` take the same word or go away with the repository. "In flight" stays only in the WAL upload dispatcher (`P/walstreaming/wal_upload_dispatcher.go`), its caller `P/walstreaming/wal_queue_recovery.go` and its test, where it means segment uploads under way, not backups.

Alternatives rejected:

- *Keep the claim table and release it only from terminal transitions.* The two records could still disagree after a crash, and the startup sweep would keep its second pass.
- *A foreign key from the claim to the backup row.* It would make the claim row-shaped, which is what the running row already is.

### 3. One owner of status changes

The SQL of every status change lives in `BackupRepository`, as `docs/architecture.md`, "Modules", and `backend/AGENTS.md`, "Handlers, services, repositories", require: `InsertRunning` runs the `INSERT ... ON CONFLICT` of decision 2, `MarkCompleted`, `MarkFailed`, `MarkChainBroken` and `MarkCanceled` each run one `UPDATE ... WHERE id = ? AND status = 'IN_PROGRESS'` and return `(bool, error)`, and `FindAllRunning` lists the rows the startup sweep fails. `catalog.BackupLifecycle` is the only caller of these methods. It writes no SQL and holds the transaction, the order of the steps and the file obligations of each transition:

- `Start` inserts the running row;
- `Complete` records the database's backup success first, so the database row is locked before the backup row as commit `88390c19` requires, then moves the row to `COMPLETED` with its result and confirms the write receipts;
- `Fail`, `MarkChainBroken` and `Cancel` move the row to `ERROR`, `CHAIN_BROKEN` and `CANCELED` and request the deletion of the files the row names: the artifact, its metadata and its manifest;
- `FailLeftInProgress` fails every running row at startup with the reason `APPLICATION_RESTART` and the message the sweep writes today.

Each runs in one transaction with its file obligations. When the repository's `UPDATE` moves nothing, the method requests the deletion of the files of the attempt the caller passed and returns `false`, which is the path `errBackupRowSuperseded` takes today (`P/backuping/backuper.go:671-684`). The backuper, the scheduler, the cancel operation, the reaction to turning backups off and the startup sweep call these methods. `GetStatusForUpdate`, `UpdateTerminalResult`, `FailInProgressBackup`, `ReleaseOwned` and `Release` go away. The progress and attempt file name writes stay where they are: they change no status, and the progress write is already guarded by the running status (`P/catalog/full_backup_repository.go:85-95`).

The service lives in `catalog/backup_lifecycle_service.go`. It sits in `catalog` because `docs/architecture.md` names the catalog as the only writer of the status, and because `backuping`, `chains` and the facade all reach `catalog` already. It takes the storages and databases modules through `Deps`, as `backend/AGENTS.md`, "Dependencies", requires.

Alternatives rejected:

- *The callers call the repository's transition methods themselves and add the file obligations.* Every caller would again pair the `UPDATE` with the right file calls in the right transaction, which is the duplication this change removes.
- *The owner in `backuping`.* The facade's cancel operation would reach a status writer through the package that runs processes. `docs/architecture.md` would have to move the rule as well.
- *One `Finish(result)` method instead of four transitions.* The backuper's executors do produce one result type with a status, but a completed result needs receipts and a failure needs file names. Separate methods make each transition's obligations visible in its signature, and their names match `docs/architecture.md`.

### 4. Cancel and delete record the outcome, then stop the process

The cancel operation calls `Cancel` and, after the commit, cancels the run's context through the cancellation registry. The canceller locks and reloads the backup row before calling `Cancel`, so a backup that completed while cancellation waited keeps its committed files. When the row is no longer running, the answer is `409 not_running`. A row that is running without a live process, which a failed terminal write can leave behind, is cancelled the same way, because the transition does not depend on the process. Turning backups off does the same for the database's running row. Deleting a running backup deletes its rows first and cancels its process afterwards. Today the process is cancelled before the deletion, so a deletion that fails leaves a cancelled backup in the list.

The run then finds its row no longer running. Its terminal transition moves nothing and only hands its attempt files to the deletion worker, as decision 3 says.

Recording the outcome at once frees the database in the catalog while `pg_basebackup` may still need up to 10 seconds to stop. The backuper therefore keeps the IDs of the databases that have a live backup process: `registerRun` takes the database ID and refuses a second run for it, and the run's `Finish` removes the database. The scheduler already registers the run before it inserts the row (`P/backuping/scheduler.go:168-216`), so a refusal skips the database on that pass without a query. The rule fits the single-instance runtime of the spec `single-instance-runtime`: every live process of an instance is in its own memory.

Alternatives rejected:

- *Keep the database busy until the process stops: cancel only cancels the context, and the run writes `CANCELED`.* A running row without a live process could never be cancelled until the next restart. A deleted running backup would still need the in-memory rule, because its row is gone at once. The spec's `202` would also leave the backup `IN_PROGRESS` in the list for up to 10 seconds.
- *Wait in the request until the process stops.* An HTTP request would block on a process for up to 10 seconds plus its cleanup, and a stuck cleanup would block it longer.
- *Give each backup its own slot name.* The overlap would no longer break the slot, but two `pg_basebackup` processes would still run against one source, and the slot name is part of the stored connection settings.

### 5. The scheduler decides with one function, before and under the lock

The decision function becomes two pure steps in `P/backuping/scheduler_decision.go`. The first takes the newest full backup, the newest incremental, the mismatch flag, the config and the time, and returns a decision, nothing, or that it needs the chain. The second also takes the extendable chain, its parent and the recent full backups, and returns a decision or nothing. `loadSchedulingState` reads the cheap facts, calls the first step and reads the chain facts only when the first step asks for them. The order of the conditions then lives only in the decision functions, and a pass with nothing due still reads only what `physical-backups/background-load` allows.

A pass first decides without a transaction. When that decision is nothing, the pass ends, as today. Otherwise it opens the transaction, takes the advisory lock, loads the state again, decides again with the same functions and inserts what the second decision says. `isImmediateReplacementTriggerCurrent`, the second mismatch check in `claimAndInsert`, the `immediateTrigger` field, `backupTrigger` and `errBackupDecisionSuperseded` are deleted. Clearing a forced request stays after the commit, with the compare-and-clear it has now.

Alternatives rejected:

- *Decide only under the lock.* Every pass of every database would open a transaction and wait for the advisory lock, which the WAL orphan cleanup can hold while it deletes segments.
- *Decide before the lock and compare the second decision with the first.* The comparison would need an equality over decisions and would drop a fresher decision for no gain. The second decision is the one made on locked state.
- *Load the chain on every pass.* `physical-backups/background-load` forbids it.

### 6. The compression falls back on every backup

The code keeps its fixed order. The spec `physical-backups/backup-method` and `docs/architecture.md` stop requiring a remembered codec. A source rejects an unsupported codec at the start of the attempt, while a remembered weaker codec would keep a source rebuilt with zstd on gzip or no compression, which roughly triples the traffic of a large database. The engine tests `TestCodecPreference` and `TestBaseBackupCodecRejection` (`internal/engines/postgresql/physical/codec_test.go`) pin the order and the rejection classification. The API test `TestPhysicalBackupCompression` replaces only the external `pg_basebackup` command: it rejects zstd, then delegates gzip to the real client against PostgreSQL 17. Both successive backups must complete with `GZIP` stored, and each backup must first request `server-zstd:5`. The test application accepts a client tool directory so the command replacement stays isolated to that application.

Alternatives rejected:

- *Start from the codec of the newest completed backup.* That would add a read to every backup and keep a rebuilt source on the weaker codec until someone noticed.
- *A remembered codec that is retried periodically.* That would add state and a timer for a cost of about one second per backup.

### 7. The migration

One migration, created with `goose -dir backend/migrations create merge_physical_full_and_incremental_backups sql`:

1. Create `backups_physical` with its columns, primary key and references, without the new checks and the partial unique index.
2. Copy the full backups with `kind = 'FULL'` and both references `NULL`. Then copy the incrementals with `kind = 'INCREMENTAL'`, `root_full_backup_id` as it is, and `parent_backup_id = COALESCE(parent_incremental_backup_id, root_full_backup_id)`. IDs and every other column are copied as they are. Parents are inserted before their children because full backups go first and the incremental references are `RESTRICT`; the references are checked at the end of each statement, so incrementals copied in one statement may reference each other.
3. Repair the rows the new constraints would reject:
   - a `COMPLETED` row without a completion time takes its creation time and stays `COMPLETED`, because its files and LSN range are intact;
   - a `COMPLETED` row without a file name, a start LSN or a stop LSN becomes `ERROR`, with a fail message that names what was missing, because no restore can read it;
   - of several running rows of one database, every row but the newest becomes `ERROR` with `APPLICATION_RESTART` and the startup sweep's message. The old cancel path allows this state. Their files are already tracked by pending deletions, which `WriteFile` records before the first byte, so the deletion worker removes them without a request.
4. Add the checks and the partial unique index.
5. Drop `backups_physical_incremental`, `backups_physical_full` and `backups_physical_in_flight`, in that order.

The `Down` section raises an error that says the merge is one-way and that a rollback restores a dump taken before the upgrade. `docs/architecture.md`, "Migrations", makes an upgrade one-way, and no user path runs `Down`, so recreating the old tables and copying the rows back would be code and tests nobody uses.

The running rows that stay are failed at startup by `FailLeftInProgress`, as today. Migrations run before the modules start, so no backup is running while they run.

The existing test of `20261027214332` applies every migration and then checks the renamed tables, and its `Down` step rolls back only the newest migration (`migrations/physical_backups_migration_test.go:18-280`). The filter that `openDatabaseMigratedBefore` already applies moves into a helper that applies every migration before a named one, and that test calls it with the new migration's name instead of `applyAllMigrations`. It then keeps checking `20261027214332` alone, and its `Down` step still rolls back that migration. The storage reference migration test, which checks the latest schema, reads the merged table and its storage reference. The new test seeds the old tables before the new migration: a full backup with two chained incrementals, a second chain, a `COMPLETED` row without a completion time, a `COMPLETED` row without a stop LSN, two running rows of one database with a claim, and a running row of another database. It checks the copied kinds, parents and roots with the IDs kept, the repairs, every new check, the partial unique index, the index names, that the three old tables are gone, and that `Down` fails without changing the schema.

Alternatives rejected:

- *Fail every running row in the migration.* That would duplicate the startup sweep's message in SQL for rows the sweep fails anyway. Only the duplicates block the index.
- *Keep the old tables as views.* That would be backward compatibility nobody asked for. The proposal marks the schema break.

### 8. Tests

The seed helpers of `apptest` keep their names and fields, so the API tests that seed full and incremental rows keep their bodies: `InsertPhysicalFullBackup` and `InsertPhysicalIncrementalBackup` write into `backups_physical`, and the incremental's parent is derived as in the migration. `InsertPhysicalInFlightBackup` and its seed type are deleted. A test that seeded a running row and a claim seeds only the running row, and a test that reads the claim reads the running row. Tests that build a state the schema now forbids are deleted, with their names listed in tasks.md: a claim without a row, a row without a claim, a claim naming a finished backup. `WaitForPhysicalBackups` counts the running rows of one table. `physicaltest/promotion.go` and `internal/features/workspaces/testing/testing.go` switch to the new table.

New tests pin the new behavior, each in the package that already tests the feature:

- `physicalapitests/catalog`: disabling backups preserves committed files when a full completes while cancellation waits on its row; a cancel answers after the row is `CANCELED`; a cancel of a running row without a live process; a deleted running backup is gone before the answer; turning backups off answers after the running row is `CANCELED`;
- `physicalapitests/backuping`: after startup fails an interrupted row, a newly requested full must complete while the interrupted row stays `ERROR`; after a cancel of a live backup with an immediate request for a full backup, the next backup starts only after the cancelled run has ended, and it completes. A scheduler pass in that window inserts nothing. When the cancelled run ends, the backup is still `CANCELED`, and the files of its attempt are pending deletion, read from `storages_pending_deletions` because the API does not return them. Two more tests follow the same steps: one deletes a live running full backup with the full cadence due, sees it gone from the list right after the `204`, sees a scheduler pass insert nothing while the deleted run is alive, and sees the next full start and complete after the run has ended; the other turns backups off during a live full backup, sees it `CANCELED` right after the save, turns backups back on and sees the next full start only after the old run has ended;
- `P/backuping`: table tests of the two decision steps, including the case where the state changes between the decision before the lock and the decision under it. They are pure functions, which the testing skill allows to test without the application.

The transitions themselves are covered through these operations rather than by tests of `BackupLifecycle`, as the testing skill requires: the existing API tests reach `Complete`, `Fail` and `MarkChainBroken` through real backups, and the new ones reach `Cancel` and the superseded path.

The 14 end-to-end scenarios and the `physical-api` shards keep their scenarios. Their SQL uses the merged table. The wait on an advisory lock in `TestPhysicalBackupTimeAfterWal` filters by its test database OID, so another parallel test cannot satisfy it before this scheduler chooses its timestamp.

### 9. The upgrade test gains an incremental backup

The previous release already makes a full backup through `/backups/physical/database/{id}/trigger` with `{"type": "full"}`. It then requests `{"type": "incremental"}` the same way and waits for it to complete, before it forces the WAL switch (`e2e/upgrade/previous_release_physical_test.go:62-140`). The source already runs `summarize_wal=on`. After the upgrade the test lists the incremental with its `rootFullBackupId` equal to the full backup's ID and no `parentIncrementalBackupId`, and the restore stream holds an `incr-1/` entry beside `full/` (`P/restorestream/writer.go:19`, `:71`). This is the first check of the merged table against data written by a real release, which `docs/architecture.md`, "Migrations", requires for a migration that moves rows.

### 10. The module contract

`PhysicalFullBackup` becomes `PhysicalBackup = catalog.Backup`, and `GetLatestCompletedPhysicalFullBackup` keeps its name and returns the newest completed row of kind `FULL`. Telemetry's reader and its test fake switch to the new type. Nothing else outside the module uses the physical types.

### 11. Dead code

Deleted with their tests, each confirmed with the LSP references before deletion: `PhysicalSlotCleanupListener` (`P/walstreaming/slot_cleanup_listener.go`), `FindAllForGFS`, `FindByIncrementalParent`, `UpdateStatus`, `UpdateFailure`, `GetNextAnchorLSNAfter`, `FindWalGapsInChain`, `GetSlotRebuildTimestamps`, `DeleteWalSegmentsInSpan`, `IsUnbounded` and `ParentBackupID`. `ProcessSegment`, `RecoverSegment` and `RecordConcurrentUploads` of the WAL uploader stay: `P/walstreaming/wal_watch_dir.go:21` and `:44` pass them as method values. A method on this list that turns out to implement an interface, or to be passed as a value, stays.

### 12. Documents

- `docs/architecture.md`, "Physical backups": the table bullet describes `kind`, the parent, the chain root and the partial unique index for the running backup; the status bullet names `BackupLifecycle`, its transitions and that cancel and delete record the outcome before the process stops; the scheduler bullet says the same functions decide before and under the lock; the compression bullet drops the remembered codec; the health bullet says that the API and the Monitoring tab read the sample-based verdict, while the notifications come from the WAL streamer's live tracking, which also pauses the receiver, rebuilds the slot and rotates WAL.
- `docs/glossary.md`: "In-flight claim" is replaced by "Running backup" (`IN_PROGRESS`, `uq_backups_physical_database_in_progress`), the one `IN_PROGRESS` row a database may have.
- `temp/refactor/plan.md` lives only in the main checkout and is git-ignored. Its milestone 11 line is checked off when the change is done.

## Risks / Trade-offs

- [A backup process that ignores its cancellation keeps its database blocked until the restart.] → `lib/process` sends SIGKILL 10 seconds after SIGINT and kills the process group. The storage writes take the run's context. The block lasts as long as the process stays alive, which is also when a second backup would fail on the shared slot.
- [The list shows `CANCELED` while `pg_basebackup` is still stopping.] → The spec delta says so. The next backup waits for the process, so nothing else can observe the overlap.
- [The repair turns a `COMPLETED` backup into `ERROR`, and a restore set that listed it disappears.] → Such a row has no file or LSN range, so its restore set could not be streamed anyway. The fail message says what was missing.
- [A missed caller writes the status outside `BackupLifecycle`.] → Only `BackupRepository` holds status updates, and task 3.1's verification checks with LSP references that only `BackupLifecycle` calls its transition methods.
- [A deadlock between a terminal transition and a database deletion.] → `Complete` keeps the order database row, then backup row, of `88390c19`. `Fail`, `MarkChainBroken` and `Cancel` lock only the backup row, as `FailInProgressBackup` does today.
- [The diff is large: about 30 production files and 25 test files.] → The table merge and the transitions are separate task groups. Each group ends with the physical API shards and the end-to-end scenarios green.

## Migration Plan

The migration runs at startup before the modules start. An upgraded instance keeps every backup with its ID, chain, files and status, apart from the repaired rows of decision 7. The API and the frontend do not change. The binary refuses to start on an older schema, so a rollback restores the internal database from a dump taken before the upgrade, as for every release. The `Down` section refuses to run, for the same reason.
