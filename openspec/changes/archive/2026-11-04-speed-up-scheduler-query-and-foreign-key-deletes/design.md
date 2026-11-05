## Context

See `proposal.md` for the motivation. `listNewestPerDatabase` ranks every backup of the requested databases with `row_number()` and filters afterwards (`backend/internal/modules/backups/internal/logical/backup_repository.go:134-151`). The scheduler calls it on every pass, with the highest `MaxFailedTriesCount` among the enabled configs as the count (`backup_scheduler_service.go:80-87`). The index that fits the read already exists: `idx_backups_logical_database_created_at` on `(database_id, created_at DESC, id DESC)` (`backend/migrations/20261027140000_move_logical_backups_into_module.sql:196-197`).

A catalog query on a database with every migration applied, up to `20261102114936`, found exactly four foreign keys with no valid, non-partial index starting with their first column. These are the four in the proposal. All of them are `ON DELETE RESTRICT`, so PostgreSQL looks for referencing rows when a parent row is deleted (`storage_repository.go:175`, `notifier_repository.go:184` and `:193`). The delete holds the parent row's lock during the scan. Writes that reference other parents do not wait for it.

Measurements taken while planning used synthetic data in temporary tables, so they show the shape of the gain rather than production timings:

| Measured | Current | After the change |
|---|---|---|
| Newest 3 backups of 100 databases with 5000 backups each | 500 000 rows read, 713 ms | 300 rows read, 0.6 ms |
| Foreign key lookup on 1 million WAL segment rows | sequential scan, 43 ms | index scan, 0.02 ms |

Building the index on those 1 million rows took 296 ms and produced 6.9 MB next to a 142 MB table. Temporary tables write no WAL, so a build on a real table takes somewhat longer.

The approach follows "Data" and "Migrations" in `docs/architecture.md` and the migration format in `backend/AGENTS.md`: one statement per index, each in its own `StatementBegin` block.

## Goals / Non-Goals

**Goals:**

- A scheduler pass reads a number of rows bounded by the number of databases and the count, whatever the length of the history.
- A storage or notifier deletion finds referencing rows through an index.

**Non-Goals:**

- Pinning a query plan in an automated test. Plans depend on statistics and table size, so such a test would be brittle.

## Decisions

### Read each database's newest backups with `LATERAL`

The query iterates over the requested IDs and, for each one, reads from the existing index with `ORDER BY created_at DESC, id DESC LIMIT ?`. The outer `ORDER BY database_id, created_at DESC, id DESC` stays, so `listNewestBackups` and `getDueBackups` receive the same rows in the same order. The unnested column gets a distinct alias such as `requested_database_id`, so it does not clash with `id` in `backupColumns`.

Rejected alternatives:

- **Keep `row_number()`.** The measured plan on PostgreSQL 17 applied the limit as a window run condition, yet its index scan still read all 500 000 rows.
- **One query per database.** This reads the same rows with one round trip per database on every pass.
- **Join the requested databases to `backups_configs_logical`.** That table belongs to the same module's config package, and the join would make this read depend on the config schema and repeat the "enabled" filter that the caller has already applied.

### Pass the IDs through `unnest(?::uuid[])` with `pq.Array`

`IN ?` expands a slice into a row list that cannot drive `LATERAL`. `notifier_attachment_repository.go:44-46` already uses `unnest` with `pq.Array`.

Rejected alternative: a generated `VALUES` list needs one parameter per database and explicit casts, for no gain.

Duplicates would produce repeated rows, unlike `IN`. They cannot occur here, because the IDs come from `backups_configs_logical`, whose primary key is `database_id`. An empty list returns no rows, and `runSchedulerPass` already returns before this read when no config is enabled.

### Build the indexes with a plain `CREATE INDEX` inside the migration's transaction

`docs/architecture.md` forbids `-- +goose NO TRANSACTION`, `archlint` enforces it, and migrations finish before the server accepts requests, so nothing waits for the table lock. A failed migration leaves nothing behind, and the next start applies it again.

Rejected alternative: `CREATE INDEX CONCURRENTLY` would need a change to that architecture rule, gain nothing because no requests run during migrations, and could leave an invalid index after a failure.

### Add all four indexes

Only the WAL segments table is likely to grow large. The other three are small in typical installs, yet their indexes cost little, and the `notifier_id` index also serves the database list filter (`backend/internal/modules/databases/internal/userdatabase/database_repository.go:155`).

Rejected alternative: indexing only WAL segments leaves three of the same defect behind and saves almost nothing.

Names follow the existing `idx_backups_logical_storage_id`: `idx_backups_physical_storage_id`, `idx_backups_physical_wal_segments_storage_id`, `idx_backups_physical_wal_history_files_storage_id` and `idx_databases_notifiers_notifier_id`. The Down section drops them with plain `DROP INDEX`, like the newest migrations that drop an index (`20261029175054_healthcheck_module.sql:54`), so a wrong name fails instead of passing silently.

### Test the behavior that must not change, and the indexes that must appear

- **Scheduler.** A new subtest of the existing API test `TestLogicalBackupScheduler` (`backend/internal/modules/backups/logical_backup_schedule_test.go:25`) prepares two databases with long histories. Old history cannot be created through the API, so it is inserted with `ExecuteSQL`, as the testing skill permits. One database's newest backups exhaust its tries and the other's do not. Only the second gets a retry. This test passes before and after the rewrite, because behavior does not change. The bug-fix rule of starting from a failing test cannot apply to it, so the task runs it on the old query first.
- **Migration.** A new test reads the catalog and requires an index whose first column is each of the four columns. It fails first on the current schema for that reason. It checks columns rather than index names, because the columns are what the foreign key lookup needs. It then rolls back this migration with `provider.ApplyVersion(ctx, version, false)`, as `backend/migrations/merge_physical_backups_migration_test.go:219` calls it, and requires those indexes to be gone, because the Migration Plan relies on the Down section. `provider.Down` is rejected here, because it rolls back whichever migration is newest and would stop testing this one once a later migration lands.

Rejected alternative: a generic catalog test requiring an index on every foreign key would catch future omissions, but it adds a new rule to every future migration. That rule is out of scope and can be a separate change.

## Risks / Trade-offs

- [The planner picks another plan for the rewritten query on small tables] → That is harmless, because a table small enough for another plan to win is cheap to read either way.
- [The upgrade of a large install starts more slowly] → The measured build time on 1 million rows is under a second, and the build happens once. The server is not accepting requests at that time anyway.
- [Inserts into the four tables maintain one more index] → A row is written per WAL segment or per backup, so the added cost is small next to the work that produces the row.
- [`TestMergePhysicalBackupsMigration` breaks because it lists every index of `backups_physical` after all migrations] → The new index joins that list in this change.

## Migration Plan

The server applies the migration at startup. A failed build rolls back and is retried at the next start. No data changes and no rows need repair, because adding a non-unique index cannot be violated by existing rows. Rollback drops the indexes through the Down section, and the old query needs no schema support.
