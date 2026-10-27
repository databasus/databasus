## Context

Why the change is needed is in proposal.md, section Why. This document covers what shapes the approach. Paths are under `backend/` unless they start with another top-level directory. `PH` stands for `internal/features/backups/backups/core/physical`, `BP` for `internal/features/backups/backups/backuping/physical` and `UP` for `internal/features/backups/backups/usecases/physical/postgresql`.

**The tables.** `migrations/20260523120000_add_physical_pg_17_backups.sql:135-345` creates six of the seven tables, and `migrations/20261018195439_add_physical_wal_stream_samples.sql:8-55` creates the samples and adds `failing_since` to the streamers. Later migrations add `fail_message` to full and incremental backups (`20260614221026`), drop `node_id` from the in-flight claims (`20260613095248`), and make every reference to `storages` `RESTRICT` (`20261026090000_restrict_storage_references.sql:45-130`). Every reference to `databases` is still `ON DELETE CASCADE`. Inside the group, an incremental references its root full and its parent incremental with `RESTRICT` (`20260523120000:214-224`). The only `CHECK` is `chk_physical_in_flight_backups_backup_type` (`:248-251`).

**The enums.** `PH/enums` defines every value the code writes:
- backup status: `IN_PROGRESS`, `COMPLETED`, `ERROR`, `CHAIN_BROKEN`, `CANCELED`;
- error reason: the 17 values of `PhysicalBackupErrorReason` (`PH/enums/enums.go:34-66`), from `PG_BASEBACKUP_FAILED` to `SUMMARIZER_FALLING_BEHIND`;
- compression: `ZSTD`, `GZIP`, `NONE`;
- streamer status: `RUNNING`, `FAILED`;
- source state: `REACHABLE`, `UNREACHABLE`, `SLOT_MISSING`.

Encryption is `NONE` or `ENCRYPTED` (`internal/features/backups/backups/core/enums`). The slot's WAL status in a sample is a plain string copied from `pg_replication_slots.wal_status` (`UP/wal_stream_sample.go:98`, `internal/engines/postgresql/physical/slot.go`).

**The deletion.** `DeleteDatabase` locks the database row with `FOR UPDATE`, runs every `DatabaseDeleting` handler in its transaction and then deletes the row (`internal/modules/databases/internal/userdatabase/deletion_service.go:17-37`). The physical handler collects the file names of the full and incremental backups, the WAL segments and the history files on that transaction and requests their deletion (`PH/service/database_deletion_handler.go:17-90`). The rows themselves disappear through the cascade. The WAL uploader and the backuper lock the database row before the backup row when they publish (`UP/wal_upload.go:483-491`, commit `88390c19`), and they treat a database that vanished mid-publication as a dropped segment or backup.

**Who names the tables.** The GORM models declare the table names (`PH/models/*.go`, one `TableName` each). Raw SQL names them in `PH/service/service.go` (eight places), `PH/repositories/{wal_segment,wal_streamer,wal_stream_sample,in_flight_backup}_repository.go`, `BP/scheduler.go`, `BP/wal_supervisor.go` and `UP/wal_stream.go`. The test code that names them is `PH/testing/testing.go`, `BP/wal_supervisor_test.go`, `internal/features/backups/backups/controllers/physical/controller_test.go`, the end-to-end helpers `internal/app/apptest/physicaltest/restore_wal_gap.go:29-38` and `promotion.go:50-110`, and `internal/modules/backups/{physical_config,storage_change_reactions}_test.go`. Three old test helpers delete database rows with SQL and rely on the cascade: `removeWorkspace` (`internal/features/workspaces/testing/testing.go:161-180`), `TestDatabaseDeletion.Finish` (`internal/features/databases/test_database_deletion.go:69-81`) and a slot test (`UP/backup_slot_test.go:601-605`).

**The upgrade test.** `e2e/upgrade` starts the newest release, `v3.63.1`, writes data through its API, stops it and starts the current image on the same volume (`e2e/upgrade/flow_test.go:60-101`). Its only PostgreSQL source is `postgres:16` (`source_database_test.go:14`), and physical backups need PostgreSQL 17. Containers are described by `containerSpec`, whose entrypoint a test may replace (`docker_test.go:75-86`). `v3.63.1` serves the same physical routes as the current code: `/backup-configs/physical/save`, `/backups/physical/database/:id/{backups,trigger,restore-token}` and the public `/backups/physical/restore-stream`. Its physical backup types are `FULL`, `FULL_INCREMENTAL` and `FULL_INCREMENTAL_WAL_STREAM`. The test helpers for physical sources configure the source with a `host replication` line in `pg_hba.conf` and `summarize_wal=on` (`internal/util/testing/containers/postgres_physical.go:87-140`).

**The checks.** `archlint` fails a migration newer than its cutoff that declares `ON DELETE CASCADE` from a table of one module to a table of another; a table belongs to a module by its name prefix (`tools/archlint/cascades.go:12-60`).

## Goals / Non-Goals

**Goals:**
- After the migration, every physical table carries the `backups_physical_` prefix, references `databases` with `RESTRICT` and rejects values the code never writes.
- The old physical code behaves exactly as before on the renamed tables, and the 14 physical end-to-end scenarios pass with only their table names changed.
- The upgrade test checks the physical tables against data written by the previous release.

**Non-Goals:**
- Moving any physical code, which `physical-backups-module` does.
- Changing what a deletion cancels, refuses or drops on the source, or the order of the deletion's handlers.

## Decisions

### 1. The schema goes first, as its own change

Milestone 10c is split into this change and `physical-backups-module`. This one changes the schema on the old code. The next one moves the code.

The schema goes first for two reasons. The upgrade test then checks the renamed tables against a real release before the large move begins, so the move runs under that check too. And the move becomes a diff without any schema change, which makes it easier to review and to bisect.

Alternatives rejected:
- *One change for the whole of 10c.* About 17 thousand lines of production code and 17 thousand lines of tests would land in one diff with a migration in the middle. The plan already named this risk.
- *The code first, the schema second.* The moved code would be written against the old names and changed again one change later, and the move would run without the upgrade test covering physical data.
- *Splitting the move into the catalog and the streaming.* Code outside the module cannot import the module's `internal` packages, so the streaming left in `features` would need dozens of temporary methods in `module.go` to reach the catalog.

### 2. Table names leave `backups_physical` free

The tables become `backups_physical_full`, `backups_physical_incremental`, `backups_physical_in_flight`, `backups_physical_wal_segments`, `backups_physical_wal_history_files`, `backups_physical_wal_streamers` and `backups_physical_wal_stream_samples`. Milestone 11 merges full and incremental backups into one table, and that table takes the name `backups_physical`.

Constraints and indexes follow the convention of `20261027140000_move_logical_backups_into_module.sql` and of the module's config tables: `<table>_pkey`, `fk_<table>_<target>`, `ck_<table>_<column>`, `uq_<table>_<columns>` and `idx_<table>_<columns>`. The two unique constraints `uk_physical_wal_segments_database_id_timeline_id_start_lsn` and `uk_physical_wal_history_files_database_id_timeline_id` therefore become `uq_backups_physical_wal_segments_database_timeline_start_lsn` and `uq_backups_physical_wal_history_files_database_timeline`. A constraint whose action does not change is renamed in place. A reference to `databases` is dropped and added again, because its action changes.

Alternatives rejected:
- *`backups_physical` for the full backups now.* Milestone 11 would have to rename it a second time, and the name would mean "full backups" for one release.
- *`backups_physical_full_backups`.* The word "backups" repeats the module prefix.
- *Keeping the old constraint and index names.* PostgreSQL names the constraint in a violation's error message. A name that disagrees with its table makes such an error harder to trace, and the next migration that touches the constraint has to look the old name up.

### 3. The deletion handler deletes the rows on the deletion's transaction

The physical `DatabaseDeleting` handler keeps reading the file names first and requesting their deletion. Then, on the same transaction, it deletes the database's rows in an order that respects the references inside the module: samples, streamer, in-flight claim, WAL segments, history files, incrementals, fulls. Each table takes one `DELETE ... WHERE database_id = ?`.

All incrementals of a database go in one statement, although one incremental can reference another with `RESTRICT`. PostgreSQL 17 checks a non-deferrable `RESTRICT` at the end of the statement, so a statement that removes the whole chain passes. I checked this on the test server: three chained incrementals deleted by one `DELETE` leave no violation.

The lock order stays safe. `DeleteDatabase` locks the database row before any handler runs. A WAL segment publication or a backup's terminal write also locks the database row first. Either it finishes before the deletion takes the lock, and its row is deleted with the others, or it waits for the deletion and then finds the database gone, which it already handles. A new claim or segment row needs a key share lock on the database row for its reference, so it waits for the deletion and then fails its reference, as it does under the cascade today.

Alternatives rejected:
- *Deleting incrementals leaves first, one row at a time, as `cascadeDelete` does (`PH/service/service.go:910-955`).* That order exists for the cleaner, which deletes part of a chain and must not orphan a child. A deletion removes the whole chain, so one statement is enough.
- *Keeping `CASCADE` until the code moves.* The handler would not change here, but `physical-backups-module` would then change the deletion and the schema at once, which is what this split avoids. `docs/architecture.md`, "Data", also requires `RESTRICT` between modules.
- *A separate handler that only deletes rows.* The handler that reads the file names already runs on the deletion's transaction. A second handler would read the same rows again and add one more ordering rule to `internal/app/wiring.go`.

### 4. A `CHECK` on every enum column the code owns, after the repairs

`CHECK` constraints cover:
- `status` and `compression` of full and incremental backups;
- `error_reason` of full and incremental backups, `NULL` or one of the 17 values;
- `encryption` of full and incremental backups and WAL segments;
- `status` of streamers;
- `source_state` of samples.

The existing check on `backup_type` of the in-flight claims only takes the new name.

The repairs run before the constraints, and each one changes only values a constraint would reject:
- **Unknown backup status.** The backup becomes `ERROR`, and `fail_message` says "The upgrade replaced the unknown status <value>". This is the rule of the logical migration, with the physical failure status.
- **Unknown compression.** The restore set cannot decompress such a file, so the backup becomes `ERROR` with "The upgrade replaced the unknown compression <value>" and takes the column's default `ZSTD`. Its file stays in storage and goes when retention or the user removes the backup, like any failed backup's.
- **Unknown encryption.** The encryption follows the salt, as in the logical migration: a row with `encryption_salt` is `ENCRYPTED`, one without is `NONE`.
- **Unknown error reason.** The reason becomes `NULL`, and "The upgrade cleared the unknown error reason <value>" is appended to `fail_message`, after a `; ` when the message is not empty, so neither the old message nor the old reason is lost.
- **Unknown streamer status.** The streamer becomes `FAILED`. The WAL stream supervisor claims a `FAILED` row again on its next pass, as after a stop or a failure (`PH/repositories/wal_streamer_repository.go:55-68`).
- **Unknown sample source state.** The sample is deleted. A sample with an unknown state cannot be read: its source-side columns are defined only for `REACHABLE`. Samples are monitoring history that the cleaner expires anyway.

The slot's WAL status gets no `CHECK`. Its values come from PostgreSQL, and a PostgreSQL release that adds a value would make every sample insert fail.

Alternatives rejected:
- *Repairing an unknown compression to a guess.* A wrong codec makes a backup that looks restorable fail at restore time, which is worse than an honest failed backup.
- *Rewriting the samples with an unknown state to `UNREACHABLE`.* The row would claim an unreachable source while its source-side columns might still hold values, which the health verdict does not expect.
- *No `CHECK` on the error reason, because it has many values.* `docs/architecture.md` asks for a `CHECK` on every enum column, and the scheduler decides on immediate replacement backups by the stored reason (`BP/scheduler.go:482-595`), so a stray value would silently read as no known reason.

### 5. The old code switches to the new names and keeps its behavior

Every table name in the old physical code changes in one step with the migration. The model `TableName` methods change, and so does every raw SQL string listed in Context. The SQL is not reworded in any other way. Code that would have to move anyway is not refactored here, because `physical-backups-module` replaces it.

The physical catalog cleanup helper also removes WAL stream samples, which can no longer disappear through the database cascade. Completed backup result fixtures supply valid compression and encryption values for the new checks.

The three old test helpers that delete database rows with SQL first delete the physical rows of the databases they remove, in the order of decision 3, as `TestDatabaseDeletion.Finish` already does for `backups_configs`. Their comments that mention the cascade change with them.

The end-to-end helpers change only their table names, which `physical-e2e-on-apptest` allowed in its decision 8. A diff of `internal/modules/backups/e2e` and `internal/app/apptest/physicaltest` then shows only the four renamed queries.

### 6. The upgrade test's source moves to PostgreSQL 17 and also serves the physical database

The upgrade test keeps its one PostgreSQL source container and switches it from `postgres:16` to `postgres:17` (`e2e/upgrade/source_database_test.go:14-15`). Nothing in the logical half needs version 16: the previous release backs up and the current code restores PostgreSQL 17 logically like 16. The container's entrypoint, which `containerSpec` already lets the test replace (`e2e/upgrade/docker_test.go:75-86`), becomes a shell wrapper that writes an init script adding `host replication all all scram-sha-256` to `pg_hba.conf` and then starts the image's own entrypoint with `postgres -c summarize_wal=on`, the two settings the physical test helpers use. The default `wal_level` of `replica` is enough for `pg_basebackup` and `pg_receivewal`.

Before the upgrade, through the previous release's API, the test:
1. creates a physical PostgreSQL database of type `FULL_INCREMENTAL_WAL_STREAM` on the same source server, with the superuser the logical database uses, in the workspace the test already creates;
2. saves its physical backup config with backups turned on and the local storage;
3. starts a full backup through `/backups/physical/database/:id/trigger` and waits until the list shows it `COMPLETED`;
4. writes rows on the source and calls `pg_switch_wal()` until the list shows a WAL range, the row of consecutive archived WAL segments, after the full backup.

After the upgrade, through the current code's API, it:
1. checks that the full backup and that WAL range are listed with their identity, status and segment names;
2. issues a restore token for the database;
3. reads the restore stream to its end as a tar archive and checks that it holds entries under the full backup's `full/` directory and at least one `wal/` entry for a segment of the range listed before the upgrade (`internal/features/backups/backups/download/restore_stream/writer.go:55`, `:238`).

A WAL range may grow after the current instance starts streaming, so the comparison expands each listed range into items identified by segment filename and timeline. It checks that every segment listed before the upgrade still appears, without requiring the range to stop growing.

The comparison goes through the `upgradecheck` items the test already uses, so a missing backup or range is named in the failure. A dropped or renamed table that the current code still reads fails the test earlier, on the first request that reads it or at start-up.

The logical restore after the upgrade writes into a new database on the same server while the current code streams its WAL. That is ordinary write load for the WAL stream and changes nothing the logical checks compare.

Alternatives rejected:
- *A second source container for the physical database.* It would add a container lifecycle, its names, labels, network alias and start-up time to the test for no extra coverage, since nothing needs the logical source to stay on version 16.
- *Running the recovery script after the upgrade.* It needs Docker inside the test's flow and a second PostgreSQL container for the recovered cluster. The physical end-to-end scenarios already run it on every pull request. What only this test can show is that the renamed rows still name readable files, and reading the stream shows that.
- *A full backup without WAL streaming.* The WAL tables would then stay empty across the upgrade, and the renames of the segment and history tables would go unchecked.

### 7. Tests

- **Migration test** in `migrations`, beside `logical_backups_migration_test.go`. It applies the migrations before this one, inserts rows with every kind of unknown value from decision 4 and a chain of a full and two incrementals, applies this migration and checks every repair, the renamed tables, every `CHECK`, and that a reference to a deleted database is now refused.
- **Deletion test** `internal/modules/backups/physical_backup_database_deletion_test.go`, beside `logical_backup_database_deletion_test.go`, for the scenario "Deleting a database with physical backups" of the `databases` delta. A source cannot produce a chain with WAL and history in a test quickly, so the test seeds the rows of all seven tables with SQL, as `storage_change_reactions_test.go:175` already does for a state the API cannot create. It deletes the database through `tm.owner.Client.DeleteDatabaseWithResponse`, as `logical_backup_database_deletion_test.go:29` does, then checks with SQL that no physical row of the database remains and that a pending deletion exists for every file name the rows held, because the API returns neither.
- **Old tests.** The old physical package tests run against the renamed tables. They need no other change.
- **End-to-end.** The 14 physical scenarios run unchanged apart from the table names. They exercise the deletion under real streaming in the slot lifecycle scenarios.

## Risks / Trade-offs

- **A reference that should have been cleared is missed, and a deletion fails.** With `RESTRICT`, a table the handler forgets makes every deletion of a database with such rows fail with a foreign key error. → Decision 3 lists all seven tables. The deletion test seeds a row in every one of them, and the end-to-end scenarios delete databases that stream WAL.
- **Old test helpers outside the physical packages delete databases.** A helper missed in decision 5 makes unrelated old tests fail during cleanup. → Context lists every `DELETE FROM databases` outside the databases module, found with `rg`. Task 2.2 repeats that search.
- **A repair changes data the user cares about.** Only values the code never wrote are repaired. The repairs of a backup's status, compression and error reason keep the old value in `fail_message`. The encryption repair records nothing, because the salt decides the value without loss. → The migration test checks each repair.
- **The upgrade test grows slower.** A full backup of a small cluster and one WAL segment take seconds. → The physical database shares the existing source container, which holds only the rows the test writes.
- **The upgrade test depends on the previous release's physical routes.** If a later release changes them, the "before" half changes with it, as the logical half already does. → Its requests live in `previous_release_client_test.go`, apart from the checks of the current code.

## Migration Plan

The migration runs at start-up like every migration, in one transaction:
1. rename the tables, and rename their constraints and indexes;
2. repair the rows of decision 4;
3. add the `CHECK` constraints;
4. drop the references to `databases` and add them again with `RESTRICT`.

Backups and WAL streams that ran before the upgrade resume as today: the startup recovery fails the backups left in progress, and the WAL stream supervisor resumes streaming from its slot. A rollback means restoring the internal database from a backup taken before the upgrade, because migrations only go forward and a binary refuses a newer schema (`docs/architecture.md`, "Migrations"). The migration's `Down` section restores the old names and the cascades for development use, as the earlier module migrations do.
