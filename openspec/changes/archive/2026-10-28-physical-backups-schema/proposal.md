## Why

Milestone 10c of the backend rewrite plan moves the physical backups into the `backups` module, with its tables. That move is too large for one review, so it is split in two. This change, the first half, changes only the schema. The second half, `physical-backups-module`, moves the code onto a schema that is already in place and already checked by the upgrade test.

The seven physical tables break the data rules of `docs/architecture.md`:

- **Names.** `physical_full_backups` and the six others carry no module prefix, while the module's other tables are `backups_configs_physical`, `backups_logical` and `backups_restores`.
- **References to `databases` are `CASCADE`.** A reference from a table of one module to a table of another must be `RESTRICT`. A backup row describes a file in a storage. A cascade deletes the row without asking for the file's deletion, so a missed cleanup leaves orphan files silently instead of failing. Today the physical rows of a deleted database vanish through the cascade, and only a handler that runs first records the file deletions (`core/physical/service/database_deletion_handler.go:17-30`).
- **Enum columns have no `CHECK`.** Only `physical_in_flight_backups.backup_type` has one (`migrations/20260523120000_add_physical_pg_17_backups.sql:250`). The status, the error reason, the compression and the encryption of a backup, the streamer status and the sample's source state accept any text.
- **The upgrade test does not cover physical backups.** It writes and restores logical data only, so a migration of these tables would ship unchecked against data of a real release.

## What Changes

- **Table names.** One migration renames:
  - `physical_full_backups` to `backups_physical_full`;
  - `physical_incremental_backups` to `backups_physical_incremental`;
  - `physical_in_flight_backups` to `backups_physical_in_flight`;
  - `physical_wal_segments` to `backups_physical_wal_segments`;
  - `physical_wal_history_files` to `backups_physical_wal_history_files`;
  - `physical_wal_streamers` to `backups_physical_wal_streamers`;
  - `physical_wal_stream_samples` to `backups_physical_wal_stream_samples`.

  Their primary keys, foreign keys, unique constraints, check constraints and indexes take the new prefix.
- **References to `databases` become `RESTRICT`** on all seven tables. The references inside the module stay as they are: an incremental references its full and its parent incremental with `RESTRICT`.
- **The database deletion removes the physical rows itself.** The old physical handler of `DatabaseDeleting` already runs in the deletion's transaction and records the file deletions. It now also deletes the rows of the seven tables, after it reads the file names. The order of the deletion's handlers and its after-commit effects stays as it is.
- **`CHECK` constraints** cover the status, the error reason, the compression and the encryption of full and incremental backups, the encryption of WAL segments, the streamer status and the sample's source state. The slot's WAL status in a sample gets none, because its values come from PostgreSQL rather than from Databasus.
- **Rows that a new constraint would reject are repaired first.** An unknown status or compression fails the backup with a message that keeps the stored value. An unknown encryption is set by whether the row has a salt. An unknown error reason is cleared, and its value is appended to the fail message. An unknown streamer status becomes `FAILED`, and a monitoring sample with an unknown source state is deleted.
- **The old physical code reads and writes the renamed tables.** The model table names, the raw SQL of the scheduler, the WAL stream supervisor, the repositories and the catalog service, and the old test helpers that delete a database row directly all switch to the new names. Behavior stays as it is.
- **The upgrade test covers physical backups.** Its PostgreSQL source moves from version 16 to 17 and allows replication. The previous release creates a physical PostgreSQL database with WAL streaming on that source, saves its config and makes a full backup, and the test waits for at least one archived WAL segment. After the upgrade, the current code lists that backup and the WAL, issues a restore token and streams the whole restore set.
- **BREAKING** for the schema: seven tables and their constraints and indexes are renamed, the references to `databases` change, `CHECK` constraints are added and invalid rows are repaired. A user's own SQL against the old table names stops working. The API, the frontend and every stored backup stay as they are.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `upgrade-test`: the previous release also writes a physical PostgreSQL database with a completed full backup and archived WAL, and after the upgrade the current code lists them and streams their restore set.
- `databases`: the deletion requirement gains a scenario for a physical database with a chain of backups and WAL, whose records are removed and whose files are handed for deletion. The requirement itself does not change; the scenario pins it now that the deletion removes those records itself.

## Out of scope

- Moving the physical code into the `backups` module, the loops onto `background.Runner`, the removal of `PhysicalBackupConfigSaved` and of the bridges, and the move of the physical tests. That is `physical-backups-module`.
- Merging full and incremental backups into one table, which milestone 11 does. The name `backups_physical` stays free for that table.
- Changing which handlers run on a database deletion, their order, or what a deletion refuses. Milestone 11 merges the three physical handlers into one.
- A `CHECK` on the slot's WAL status, which PostgreSQL defines.
- Running the recovery script in the upgrade test. The physical end-to-end tests already run it.
- The physical operations in the API contract, which `physical-backups-api` adds.

## Impact

- **Migrations:** one new migration in `backend/migrations` with its test, which applies the migrations before it to rows with unknown values and checks every repair and every constraint.
- **Backend, changed:**
  - the table names of `features/backups/backups/core/physical/models` and the raw SQL in `core/physical/repositories`, `core/physical/service/service.go`, `backuping/physical/scheduler.go`, `backuping/physical/wal_supervisor.go` and `usecases/physical/postgresql/wal_stream.go`;
  - the database deletion handler `core/physical/service/database_deletion_handler.go`;
  - the test helpers that delete database rows directly: `features/workspaces/testing/testing.go`, `features/databases/test_database_deletion.go` and `usecases/physical/postgresql/backup_slot_test.go`, together with `core/physical/testing/testing.go`;
  - the SQL of the physical end-to-end helpers `internal/app/apptest/physicaltest/{restore_wal_gap,promotion}.go`, which `physical-e2e-on-apptest` allows to change at exactly this point, and of `internal/modules/backups/{physical_config,storage_change_reactions}_test.go`;
  - `backend/e2e/upgrade`: the source server on PostgreSQL 17 with replication allowed, and the physical data before and after the upgrade.
- **Not affected:** the contract, the frontend, the old physical routes and every stored backup, file and WAL segment.
- **Running instances:** the migration renames the tables in place, changes the references and adds the constraints after the repairs. Physical backups, WAL streams and restores keep working across the upgrade.
- **Documents:**
  - `docs/glossary.md` gains the terms the specs and the design use: WAL range, timeline history file, in-flight claim and WAL stream sample;
  - the rewrite plan `temp/refactor/plan.md`, which lives only in the main checkout: the 10c line is split into 10c1 and 10c2, and 10c1 is checked off.

  `docs/architecture.md` does not change. The change follows its data rules: the module prefix, `RESTRICT` between modules and `CHECK` on enum columns. The rule of "Physical backups" that full and incremental backups share one table stays for milestone 11.
- **Rules the change answers to:** the root `AGENTS.md` and `backend/AGENTS.md`.
