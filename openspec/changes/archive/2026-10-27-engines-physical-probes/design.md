## Context

See proposal.md, "Why", for the motivation. Paths are under `backend/internal/` unless they start with another top-level directory. `UP` stands for `features/backups/backups/usecases/physical/postgresql`, and `EP` for `engines/postgresql/physical`.

**How the old code reaches the source.** Every probe opens a connection through `UP/source_connection.go:14`, which wraps `EP/source.go:41` `OpenInspectionConn`, runs SQL on the returned `*pgx.Conn` and closes it. The callers are:

- the timeline checks before a backup (`UP/executor.go:218`, `UP/executor.go:377`, `UP/create_incremental_uc.go:160`);
- the summarizer pre-check (`UP/create_incremental_uc.go:187`);
- the history upload (`UP/create_full_uc.go:146`, `UP/wal_watch_dir.go:108`);
- the per-backup slot (`UP/backup_slot.go:45`, `:123`, `:224`) and the slot cleanup after a database deletion (`UP/slot_cleanup_listener.go:98`);
- the WAL stream supervisor's slot probe (`UP/slot_lsn_watcher.go:112`), the rebuild (`UP/slot_rebuild.go:99`, `:148`) and the forced rotation (`UP/wal_rotation.go:129`, `:155`).

**The probes and their engine counterparts.**

| Old probe | SQL | Engine counterpart |
|---|---|---|
| `InspectSlot` (`UP/slot.go:38`) | `pg_replication_slots` joined with `pg_stat_replication`, plus `pg_current_wal_lsn()` and `safe_wal_size` | `readSlot` (`EP/replication_slot.go:156`): type, active, active PID |
| backup slot create (`UP/backup_slot.go:59`) | `pg_create_physical_replication_slot($1, true)` | the same statement in `VerifySlot` (`EP/replication_slot.go:55`) |
| `dropBackupSlotIfExists` (`UP/backup_slot.go:132`) | `pg_drop_replication_slot(slot_name) FROM pg_replication_slots WHERE slot_name = $1`, whatever the slot's type; an active slot makes it fail | `dropSlot` (`EP/replication_slot.go:213`), called only after a type and activity check |
| `terminateOwnedSlotBackend` (`UP/slot_rebuild.go:98`) | `pg_terminate_backend` only for a backend whose `application_name` starts with the receiver prefix | `waitForSlotDetach` (`EP/replication_slot.go:177`), which terminates any consumer and polls |
| `readClusterIdentitySources` (`UP/timeline.go:338`) | WAL file name unless in recovery, control file timeline, `system_identifier` | `InspectCluster` (`EP/cluster_identity.go:18`): version, `system_identifier`, `wal_segment_size` |
| `readHistoryFromCluster` (`UP/timeline.go:445`) | `pg_read_binary_file('pg_wal/' \|\| $1)` | none |
| `readCurrentWalLSN`, `forceWalRotation` (`UP/wal_rotation.go:145`, `:154`) | `pg_current_wal_lsn()`, `pg_switch_wal()` | none |
| `isInsufficientPrivilegeError` (`UP/wal_rotation.go:168`) | SQLSTATE `42501` or "permission denied" in the text | `classifyServerError` (`EP/errors.go:44`) |
| `isSummarizerEnabled`, `readSummaryWindow`, `isSummarizerRunning` (`UP/summarizer.go:190`, `:230`, `:210`) | `summarize_wal`, `MIN(start_lsn)` of `pg_available_wal_summaries()`, `pg_get_wal_summarizer_state()` | none |
| liveness in the slot probe (`UP/slot_lsn_watcher.go:134`) | `SELECT 1` after the slot read, so a source that accepts connections but stops answering counts as unreachable | `ProbeLiveness` (`EP/source.go:49`), which opens its own connection |

**Slot names.** `pgphysical.Settings` carries one `ReplicationSlotName` (`EP/settings.go:24`). The `databases` module fills it with the WAL stream's slot (`modules/databases/internal/providers/postgresqlphysical/provider.go:223`) and keeps the per-backup slot apart in `PostgresqlPhysicalConnectionSettings.BackupSlotName` (`modules/databases/internal/providers/connection_settings.go:60`). To drop the backup slot through the engine, `UP/backup_slot.go:110-113` copies the settings and swaps the name.

**WAL arithmetic.** `util/walmath/constants.go:5-8` holds `WalSegmentSize` and `xLogSegmentsPerXLogID` as package variables, and `SetWalSize` (`:24`) changes both. The production code already passes the cluster's size: `ParseWALFilenameWithSize` (`UP/wal_resume.go:34`, `UP/wal_upload.go:555`), `segmentNoAtLSN` (`UP/wal_resume.go:104`) and `WalUploadDeps.WalSegmentSizeBytes` (`UP/wal_upload.go:50`). The fallback in `UP/wal_stream.go:578-584` reads the global directly. Filename recognition in the queue and backlog counting reads it indirectly through `IsWalFilename`. That fallback applies when the database row has no detected size, which the `databases` module fills at the first connection (`modules/databases/internal/providers/postgresqlphysical/connected_database.go:76`). `GetFilename` and `FirstLSN`, which use the global, are called only by tests. Several parts of the package have no caller at all: `timeline_history.go`, `WalSegmentDescription`, `GetNextWalFilename`, `TryFetchTimelineAndLogSegNo`, `GetSegmentNoFromLsn`, `NewWalSegmentNo` with the global size, `Previous` and the global `ParseWALFilename`. `LSN` is a column type of the GORM models in `features/backups/backups/core/physical/models` through its `Value` and `Scan`.

**The `pg_receivewal` directory.** `UP/wal_watch_dir.go:22` decides that a segment is complete by its size, `:139` removes `.partial` files and `:156` parses the timeline from a `.history` name. `UP/wal_resume.go:18` mirrors `pg_receivewal`'s own choice of where to resume. `UP/wal_local_queue.go:25` and `UP/wal_resume.go:60` build the Databasus queue on top, with its `pending-upload` subdirectory.

**Constraints.** `docs/architecture.md`, "Backend layout": `engines` know nothing of modules, tables or HTTP. `backend/AGENTS.md`, "Where code goes": code that knows a database engine but no tables goes into `internal/engines/<engine>`. Code moved into the engine falls under the full lint of the target layout: attribute-only `...Context` log calls, every error in `errors.go`, no `context.Background()`, no `time.Sleep` in tests and no bare `go` (`docs/architecture.md`, "Automatic checks").

## Goals / Non-Goals

**Goals:**

- After this change, the old physical code contains no SQL against the source cluster and does not import `pgx`.
- `walmath` has no package-level state, and every function that depends on the segment size says so in its signature.
- One probe per question about the source, wherever two probes really answer the same question.

**Non-Goals:**

- Any change to what a probe returns, to the decisions built on it or to the number of connections opened per operation.
- The processes, the manifest and the codec order, which `engines-physical-processes` moves.

## Decisions

### 1. An inspection session instead of a raw connection

`Source.OpenInspection(ctx)` returns an `*Inspection` that holds one connection to the source and offers the probes as methods. The caller closes it, as it closes the connection now. `OpenInspectionConn` is deleted. The old test helpers that run their own SQL on the source build a connection from the fixture's settings with `pgshared.BuildConnConfig`. That helper lives in the old package's `testing.go`, which `physical-backups-module` deletes.

The old callers keep their shape: open, probe one or more times, close. Each operation therefore opens exactly as many connections as today. That matters for `WithBackupSlot` (`UP/backup_slot.go:39`), which holds its connection for the whole backup, and for the timeline check, which runs several reads on one connection.

Alternatives considered:

- *Each probe a method of `Source` that opens its own connection.* This is the shape of `InspectCluster` and `VerifySlot` today. It would change how many connections a backup and the WAL stream supervisor open against a source sized for its replicas, and the held connection of `WithBackupSlot` would no longer exist.
- *Free functions taking `*pgx.Conn`, as the old code has.* They keep `pgx` in the old code and in the module that inherits it in `physical-backups-module`. They also leave the connection's lifetime to code that knows nothing of the engine.

`Source.InspectCluster`, `VerifySlot`, `DropSlot` and `DropSlotForRemoval` stay methods of `Source` that open their own session, because each of their callers makes one such call and holds no session of its own. `InspectCluster` is called by the `databases` module (`modules/databases/internal/providers/postgresqlphysical/connected_database.go:64`). The slot methods are called by the old physical code: `UP/deleted_database_slot.go:27`, `:44`, `UP/slot_rebuild.go:68`, `UP/backup_slot.go:113` and `UP/slot_cleanup_listener.go:122`.

The session also offers `Ping`, the `SELECT 1` that the slot probe runs after reading the slot (`UP/slot_lsn_watcher.go:134`). It runs on the session's connection and under the caller's deadline, unlike `ProbeLiveness`, which opens a connection of its own for the source health check.

### 2. Merge probes that answer the same question; keep apart the ones that only look alike

Merged:

- **Slot reading.** One `GetSlot(name)` returns the union of `InspectSlot` and `readSlot`: type, activity, active PID, the consumer's application name, WAL status, restart and confirmed flush LSN, lag, the source's current position and `safe_wal_size`. A missing slot is reported as absent, not as an error. On a source in recovery `pg_current_wal_lsn()` fails. `readSlot` must keep working there, because the slot drops call it, and `InspectSlot` fails there today. The merged query therefore reads the position, the lag and the allowance only when `pg_is_in_recovery()` is false and returns them as absent otherwise. The WAL stream callers of the old `InspectSlot` treat an absent position as the error they get today, so the slot probe still reports a source in recovery as unreachable.
- **Slot creation.** `VerifySlot` and the per-backup slot share one statement and one classification of a taken name. `VerifySlot` treats a taken name as success, as now. The per-backup slot is created right after it was dropped, so it still reports a taken name as an error.
- **Cluster identity.** One query reads the version string, the system identifier, the segment size, the control file's timeline and the current WAL file name. Two session methods read it. `InspectCluster` parses the version and refuses one that is not 17 or 18, as `Source.InspectCluster` does today, and returns `ClusterIdentity` (`EP/cluster_identity.go:12`). `ReadLiveTimeline` returns the system identifier and the live timeline and never looks at the version, so the timeline checks before a backup fail or pass exactly as now. The rule that prefers the WAL file name over the control file (`UP/timeline.go:381`) moves with it.
- **Insufficient privilege.** `SwitchWal` returns the engine's `ErrWalSwitchDenied` when `classifyServerError` reports a permission refusal. `isInsufficientPrivilegeError` is deleted. A unit test pins that a refused `pg_switch_wal` (SQLSTATE `42501`, "permission denied for function pg_switch_wal") is still told apart from an unreachable source.

Kept apart, each with the reason:

- **Dropping the per-backup slot** keeps its own statement, which drops a slot of that name whatever its type, without the type and activity check of `DropSlot`. Routing it through `dropSlot` would refuse a same-named logical slot that the per-backup drop removes today.
- **Terminating a consumer.** The rebuild terminates a consumer only when its application name marks it as Databasus's receiver, and then waits with its own deadline (`UP/slot_rebuild.go:94-166`). `DropSlotForRemoval` terminates any consumer, because the slot belongs to a deleted database. The spec `physical-backups/wal-slot-rebuild` forbids dropping a slot that another consumer holds. So the engine shares only the probes, `GetSlot` and `TerminateBackend`, while each loop and its ownership rule stay with their caller.

Alternative considered: *merge everything the plan named.* Two of those merges would change what Databasus may drop or terminate on a user's server, and this change promises no behavior change.

### 3. Slot names are arguments

`VerifySlot`, `DropSlot`, `DropSlotForRemoval` and every slot probe take the slot name. `pgphysical.Settings` loses `ReplicationSlotName`. `PostgresqlPhysicalConnectionSettings` gains `WalStreamSlotName` beside `BackupSlotName`, and the provider fills it from its stored `ReplicationSlotName`. Both slots are replication slots, so the field that callers read names which one it is, while the stored settings and their column keep their name. The copy in `UP/backup_slot.go:110-113` disappears.

Alternative considered: *keep the field and keep the copy.* A database has two slots, and settings that hold one name make every operation on the other a copy-and-swap. The change touches the `databases` provider and its tests in a handful of lines, and the stored settings and the API keep their shape.

### 4. `walmath` becomes a subpackage of the physical engine

`util/walmath` moves to `engines/postgresql/physical/walmath`. The attribution header of the code adapted from wal-g stays on its files.

- `WalSegmentNo.FirstLSN` and `WalSegmentNo.GetFilename` take the segment size.
- `ParseWALFilenameWithSize` becomes `ParseWALFilename(name, segmentSizeBytes)`, replacing the global variant.
- `segmentNoAtLSN` (`UP/wal_resume.go:104`) becomes `walmath.NewWalSegmentNo(lsn, segmentSizeBytes)`: the unused global-size `NewWalSegmentNo` is re-signed rather than deleted.
- `parseHistoryTimeline` (`UP/wal_watch_dir.go:156`) becomes `walmath.ParseHistoryFilename`, the inverse of `FormatHistoryFilename`.
- The package variables, `SetWalSize` and the rest of the code without callers listed in Context are deleted.

The fallback in `UP/wal_stream.go:578` uses `walmath.DefaultSegmentSizeBytes`, 16 MiB, PostgreSQL's compiled-in default. Tests that built file names with the old default pass that constant. `IsWalFilename` also takes the segment size; its existing queue and backlog callers pass the default explicitly. The queue keeps that filename filter after reading segment metadata from the engine, so this extraction preserves the set of filenames it accepts.

Alternatives considered:

- *Merge it into `pgphysical`.* Every one of the ~60 importing files changes either way, so the diff is the same size. But the GORM models and `chain_view` need only the `LSN` type, and they would import the package that holds the source, `pgx` and, after `engines-physical-processes`, the processes. The package would also grow into a grab bag of source access and value types.
- *Keep it in `util` until `physical-backups-module`.* `util` is the legacy home that the rewrite empties. The plan places it in the engine in this step.
- *Refuse to stream, or detect the size at the streamer's start, when the row has no size.* Either changes what happens to a database saved before detection existed. The plan leaves that to milestone 11.

The new package name is unique in the backend, so `.golangci.yml` needs no import alias for it.

### 5. The `pg_receivewal` directory format is engine code; the queue is not

The engine gets one unit that reads a `pg_receivewal` directory. It tells whether a segment file is complete for a segment size, recognizes `.partial` and `.history` files and finds the segment `pg_receivewal` resumes from: the one after the highest complete segment, as its `FindStreamingStart` does. `GetResumeSegmentNo`, `isCompleteSegmentFile` and the `.partial` and `.history` recognition move there. The `pending-upload` subdirectory, the move of segments out of the resume path, the queue listing and its order are Databasus's own queue, so they stay in `UP` and call the engine.

Alternative considered: *move the whole local queue.* `pending-upload` is a decision of the uploader, not a format of PostgreSQL, and `physical-backups-module` moves it with the uploader.

### 6. The summarizer probe returns a state; the decision stays

`GetWalSummarizerState` returns whether `summarize_wal` is on, the oldest summary's start LSN or none, and whether the summarizer process runs, in one query. `CheckSummarizerReadiness` (`UP/summarizer.go:62`) keeps its decisions and its order of checks, and reads them from the state. Reading the process state even when the summarizer is off costs one more column in a query that runs once per incremental backup.

Alternative considered: *three probes, as now.* The three answers are only ever read together.

### 7. Tests follow the code

- The probe tests move to the engine and run on its servers through `runOnEverySupportedVersion` (`EP/postgres_server_test.go:61`): the slot reading with and without a WAL limit (`UP/slot_test.go:18`), the summarizer process state (`UP/summarizer_test.go:182`, `:192`), the live timeline after a promotion (`UP/timeline_test.go:277`), the timeline preference rule (`:227`, `:260`) and the `pg_receivewal` directory reading (`UP/wal_resume_test.go:17`, `:33`). The promotion test waits for the replayed relation with `testwait.Until` instead of `time.Sleep`.
- New engine tests cover what the merge changes: the slot reading on a source in recovery reports the slot without a position; the per-backup slot statement drops a slot of any type; `ReadLiveTimeline` passes on a source whose version `InspectCluster` would refuse; `Ping` fails on a source that stops answering; and a refused `pg_switch_wal` returns `ErrWalSwitchDenied`.
- Tests in `UP` that call a moved or deleted helper are rewritten on the engine and stay where they are: the three `Test_CheckSummarizerReadiness_*` tests and the wait tests that replace `summarizerCheck` (`UP/summarizer_test.go:25-233`) read through an `Inspection`; the slot reads in `UP/wal_stream_test.go:497`, `:719`, `:743` and `UP/lag_monitor_test.go:757` use `GetSlot`; `UP/lag_monitor_test.go:627` uses `GetCurrentWalLSN`; and `UP/wal_stream_test.go:373`, `:479`, `:501` use the engine's resume segment and `walmath.NewWalSegmentNo`. `Test_SegmentNoAtLsn_WithNonDefaultSegmentSize_UsesTheClusterSize` (`UP/wal_resume_test.go:134`) moves to `walmath` as the test of `NewWalSegmentNo`, and the two `Test_IsInsufficientPrivilegeError_*` tests (`UP/wal_rotation_test.go:70`, `:80`) become the engine test of `ErrWalSwitchDenied`.
- The tests of decisions, trackers and the supervisor stay in `UP`: the stall tracker and the slot probe outcomes, the rebuild cap, the backup slot lifecycle and the startup cleanup.
- The 14 end-to-end scenarios run unchanged.

## Risks / Trade-offs

- [The merged slot query reads differently on a source in recovery] → The position columns are guarded by `pg_is_in_recovery()`. The callers that read the position treat its absence as today's error, and an engine test runs the query on a standby.
- [A merged probe drifts from what its old caller read] → Each merged probe keeps the old query's columns and their `COALESCE` defaults, and the engine tests pin them on every supported version.
- [The `walmath` import change touches about 60 files and hides a real change] → The move lands as its own task before the probe work, and `rg -n 'util/walmath'` printing nothing is its check.
- [A renamed or moved test silently stops running] → The tasks list each moved test by name, and the review compares the old and new test names.

## Migration Plan

Nothing to deploy: no schema, configuration or stored data changes. Rollback is a revert of the change.
