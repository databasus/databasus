## Why

The old physical backup code in `features/backups/backups/usecases/physical/postgresql` talks to the source cluster in SQL of its own, beside the physical engine that already does the same. It reads replication slots in `slot.go`, creates and drops the per-backup slot in `backup_slot.go`, terminates a slot's consumer in `slot_rebuild.go`, and reads the cluster identity and the live timeline in `timeline.go`. The engine already reads slots in `readSlot`, creates and drops them in `VerifySlot` and `dropSlot`, waits for a slot to detach in `waitForSlotDetach` and reads the cluster identity in `InspectCluster`. The WAL arithmetic in `util/walmath` keeps the WAL segment size in a package-level variable that `SetWalSize` can change, although segment bounds and resume calculations already pass the size of their cluster explicitly. Filename recognition through `IsWalFilename` also reads that global indirectly.

Milestone 10 of the backend rewrite moves physical backups into the `backups` module. Its first step, `physical-e2e-on-apptest`, put the 14 physical scenarios on `apptest`, and they stay unchanged until the API moves. The planned step `engines-physical` was split in two because it mixed a large mechanical move with a change to how external processes live and die. This change is the first half: it moves everything that only reads from or writes to the source cluster, and the WAL arithmetic, into the engine without changing behavior. `engines-physical-processes` follows with `pg_basebackup` and `pg_receivewal`. Kept apart, the second diff shows only the process changes, and a bisect tells the two apart.

## What Changes

- `util/walmath` moves to `engines/postgresql/physical/walmath`:
  - Every function that needs the WAL segment size takes it as a parameter.
  - The package-level `WalSegmentSize`, `xLogSegmentsPerXLogID` and `SetWalSize` are deleted, together with the functions that read them and the functions no caller uses.
  - The fallback in `wal_stream.go` for a database whose segment size was never detected uses a named 16 MiB default of the engine instead. Filename recognition in the queue and backlog counting also passes that default explicitly, preserving the filenames those callers accept.
  - Every importer switches to the new path. **BREAKING** for importers of `util/walmath`; nothing outside the backend imports it.
- The engine gains an inspection session: one connection to the source with a method for each probe the old code runs. It replaces `Source.OpenInspectionConn`, so the old physical code no longer imports `pgx`. The probes are:
  - reading a replication slot, with its activity, consumer, WAL status and position;
  - creating and dropping a slot, and terminating a backend;
  - the liveness check that a source still answers on the session;
  - the live timeline and the system identifier, read by the same query as the cluster identity;
  - reading a timeline history file;
  - the current WAL position and `pg_switch_wal`;
  - the state of the WAL summarizer.
- Duplicated probes merge where they do the same thing:
  - `InspectSlot` and `readSlot` become one slot reading;
  - the slot creation of `backup_slot.go` and `VerifySlot` share one statement;
  - `readClusterIdentity` and `InspectCluster` share one query, and only `InspectCluster` refuses an unsupported version, as today;
  - `isInsufficientPrivilegeError` gives way to the engine's `classifyServerError`.

  The design records each pair that looks alike but differs in behavior, and those stay apart.
- The slot operations of the engine take the slot name as an argument. `pgphysical.Settings` loses `ReplicationSlotName`. The connection settings of the `databases` module carry the WAL stream's slot name as `WalStreamSlotName` beside `BackupSlotName`; the stored column keeps its name. This ends the copy of the settings in `backup_slot.go` that only swapped one slot name for the other. **BREAKING** for callers of `VerifySlot`, `DropSlot` and `DropSlotForRemoval`, all inside the backend.
- How a `pg_receivewal` directory is read moves to the engine: when a segment file is complete, which files are `.partial` and `.history`, the timeline of a history file and the segment `pg_receivewal` resumes from. The `pending-upload` staging directory is a Databasus queue, not a PostgreSQL format, so it stays in the old code.
- The probe tests move to the engine and run against its own servers for every supported version. The tests of decisions and of the WAL stream supervisor stay with the old code.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. Backups, WAL streaming, slot rebuilds and their outcomes behave as before, so no spec changes and `.openspec.yaml` sets `skip_specs: true`. The 14 physical end-to-end scenarios run unchanged as the proof.

## Out of scope

- Starting `pg_basebackup` and `pg_receivewal`, the byte stall watcher, the progress poller, the codec fallback order, the classification of their error output and the rebuilt `backup_manifest`. They move in `engines-physical-processes`.
- Moving state, loops or repositories of physical backups into the `backups` module, the table renames and the schema. That is `physical-backups-module`.
- The physical API and its frontend. That is `physical-backups-api`.
- Detecting a missing WAL segment size from the source when the database row has none. The 16 MiB fallback stays as it is.
- Changing what a slot rebuild, a backup slot or a slot cleanup does, or which slots they may terminate or drop.
- Moving the decisions built on the probes: timeline compatibility, summarizer readiness, the slot stall tracker and the forced rotation tracker stay in the old code.
- The `pending-upload` directory, the upload queue and the WAL uploader.
- The end-to-end scenarios under `internal/modules/backups/e2e/postgresqlphysical` and `internal/app/apptest/physicaltest`, which must not change.

## Impact

- **Backend, new and changed:**
  - `backend/internal/engines/postgresql/physical` (the inspection session, the slot, identity, history, WAL position and summarizer probes, the `pg_receivewal` directory reading, the slot name as an argument);
  - `backend/internal/engines/postgresql/physical/walmath` (moved from `util/walmath`).
- **Backend, reduced:** `backend/internal/features/backups/backups/usecases/physical/postgresql` loses `slot.go`, `source_connection.go` and the SQL of `backup_slot.go`, `slot_rebuild.go`, `timeline.go`, `wal_rotation.go`, `slot_lsn_watcher.go`, `summarizer.go` and `wal_watch_dir.go`.
- **Backend, adjusted:** `backend/internal/modules/databases` (the WAL stream slot name in the physical connection settings), and every importer of `util/walmath` across `features/backups/backups` (models, repositories, `chain_view`, `wal_health`, the controller, the services and their tests).
- **Deleted:** `backend/internal/util/walmath`.
- **Not affected:** the schema, the contract, the frontend and every behavior a user or the end-to-end tests can observe.
- **Documents:** `docs/architecture.md`, "Engines and providers", where the list of what an engine knows gains reading a physical source's slots, identity, timeline history and WAL position. `temp/refactor/plan.md` (git-ignored, main checkout only) splits 10b in two.
- **Rules the change answers to:** the root `AGENTS.md` and `backend/AGENTS.md`.
