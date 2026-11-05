## Why

Every 15 seconds the logical backup scheduler reads the whole backup history of every enabled database, although it only needs the newest few backups of each, so each pass gets slower as history grows. Separately, four foreign keys have no index that starts with the referencing column, so deleting a storage or a notifier scans each of those tables, and the WAL segments table can hold many rows.

## What Changes

- The scheduler's read of the newest backups per database stops at the rows it needs instead of ranking the whole history.
- One migration adds an index on each of `backups_physical.storage_id`, `backups_physical_wal_segments.storage_id`, `backups_physical_wal_history_files.storage_id` and `databases_notifiers.notifier_id`.
- The scheduler's API tests gain a scenario with old history and several databases, and a migration test pins the four indexes.

No change is breaking. Which databases the scheduler picks, the HTTP API and the results of deleting a storage or a notifier all stay as they are.

### Out of scope

- A general test or rule that requires an index on every future foreign key.
- `CREATE INDEX CONCURRENTLY` or any other change to how migrations run.
- Other scheduler queries, the physical backup scheduler, and retention.
- Any change to scheduling or retry behavior.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. The change affects only how fast the behavior specified in `logical-backups`, `storages` and `notifiers` runs, so it sets `skip_specs: true`.

## Impact

- **Rules:** the change answers to the root `AGENTS.md` and `backend/AGENTS.md`.
- **Code:** the backups module's logical repository, a new migration in `backend/migrations`, and tests in `backend/migrations` and `backend/internal/modules/backups`.
- **Existing test:** `TestMergePhysicalBackupsMigration` compares the full index list of `backups_physical` after all migrations, so it must list the new index.
- **Operations:** the first start after the upgrade takes slightly longer while the indexes build. Each insert into the four tables also maintains one more index, and the indexes take some disk space.
- **Dependencies and API:** none change.
