## Why

Logical backups, their restores and their downloads are the last part of milestone 9 of the backend rewrite plan. `engines-dump-restore` moved dump and restore into the engines, and `backup-configs-module` created the `backups` module with the backup configs. This change moves the rest of the logical side into that module. The old code it replaces has these problems:

- **Restores live apart from the backups they read.** `features/restores` imports the backup packages in 28 places, holds a GORM relation to the backup row and joins `logical_backups` in SQL. `docs/architecture.md` puts restores into `backups`.
- **A restore can hang forever.** When a restore fails before its tool starts, because the database, the config or the storage is not found, the restorer only logs. The row stays `IN_PROGRESS` until the next start of Databasus, and meanwhile it blocks every new restore of that database (`features/restores/restoring/restorer.go:51-119`).
- **A restore has a scheduler that schedules nothing.** After failing the rows left in progress at startup, it only writes a time nobody reads. The target's credentials go through an in-memory cache with a one-hour lifetime, while the goroutine that runs the restore already holds them.
- **Writes overwrite rows.** Every status change, every progress tick of about one megabyte and every restore transition re-saves the whole row. The restore save may also write the backup row through its GORM association.
- **Deleting a database deletes its backups one by one.** The handler loads every backup and deletes each in its own nested transaction, reading on the pool instead of the deletion's transaction. The reference from `logical_backups` to `databases` is still `CASCADE`, so a missed row would vanish without its file being deleted.
- **The HTTP layer is the old one.**
  - Every error is a 400 with the internal error text, not-found and forbidden included.
  - A manual backup answers 200 "backup started" even when it was skipped and no backup exists.
  - The backup list pages by offset with a total count.
  - A restore answers 200 instead of 202.
  - The download handler prints errors with `fmt.Printf`.
- **Background work is hand-written.** The scheduler, the cleaner, the restore scheduler and the download token cleanup are ticker loops with `hasRun` guards. Running backups and restores start as bare goroutines.

## What Changes

- **The module grows.** `internal/modules/backups` takes over logical backups, their retention, restores, download tokens and the user streams, which let a user run one heavy download at a time. `features/restores`, the logical parts of `features/backups/backups` and the pipelines of `engines-dump-restore` move into it.
  - The double name `backups/backups` disappears.
  - The engine dispatchers become a registry keyed by database type. Each entry calls its engine source and the typed connection getter of `databases`. `docs/architecture.md` and `backend/AGENTS.md` learn that a consumer serving several engines keeps one such registry of its own.
- **Background work runs through `background.Runner`.** Three jobs: the backup scheduler, the retention cleaner and the cleanup of expired download tokens.
  - The scheduler's decision, which configs are due and which failed backups are retried, becomes a pure function of the configs, their last backups and the time.
  - Running backups and restores start through a `lib/background` group and register in the cancellation registry, which moves from `features/tasks/cancellation` into `lib/background`.
  - Work left `IN_PROGRESS` by a previous run is failed once at startup, before the jobs start. When that fails, Databasus refuses to start.
- **Restores are an ordinary long operation.**
  - The restore scheduler and the credential cache are deleted. The target's credentials live only in the memory of the running restore and are never stored.
  - Any failure after the restore was accepted, the early ones included, records `FAILED` with a reason.
  - The target's version is detected once, in the start request, so the run no longer detects it a second time without its own context.
- **One column changes with one `UPDATE`.** Status transitions run as conditional `UPDATE ... WHERE status = 'IN_PROGRESS'`, and progress writes only the size and the duration. A download token is consumed by one conditional `UPDATE`, so two requests cannot use it twice.
- **Deleting a database deletes its logical backups with one statement** in the deletion's transaction, and requests the deletion of all their files with one call. Their restores and download tokens go with them through `CASCADE` inside the module. A running backup or restore refuses the deletion with 409 `in_use`.
- **The `module.go` events and methods for the old code that stays:**
  - `LogicalBackupCompleted`, published after a backup is published, for the verification scheduler;
  - batch lookups of logical backups;
  - the latest verifiable backup;
  - reading a backup's decrypted file;
  - the single-column write of the logical backup's verification status, which the old verification runs keep until milestone 12;
  - the telemetry and health check lookups.

  `LogicalBackupConfigStorageChanging` from `backup-configs-module` is deleted, because its handler is now inside the module.
- **11 operations enter `backend/api/openapi.yaml` under the tag `backups`:**
  - logical backups: list with a cursor and status and date filters, read, start (202 with the backup in progress), cancel and delete;
  - downloads: create a download token, and download a backup's file with that token as a public operation;
  - restores: list a backup's restores with a cursor, read, start (202 with the restore in progress) and cancel.

  The restore target is a `oneOf` with the discriminator `type`, and its secrets are `writeOnly`. The codes `already_running`, `insufficient_disk_space` and `not_running` join the contract and the six dictionaries. Cancelling a backup or a restore that is no longer running answers 409 `not_running`.
- **Who may do what is unchanged and now explicit.** A viewer may list, read, start and download backups, and list, read and start restores. A member may also cancel and delete backups and cancel restores. Refusals answer 404 `not_found` and 403 `permission_denied`.
- **The old code reaches the module through the bridge** of `backup-configs-module`. The old verification runs, telemetry, health check, physical backup code and end-to-end tests switch to `module.go`. The old verification repository loses its raw `JOIN` on the backup table.
- **Schema.**
  - `logical_backups` becomes `backups_logical`, `restores` becomes `backups_restores` and `download_tokens` becomes `backups_download_tokens`.
  - The token's `used` becomes `is_used`. The size and duration columns keep their names and units, like those of the physical backups.
  - A restore keeps its backup's database in a new `database_id` column, tied to the backup by a composite reference, so a partial unique index allows one running restore per database. A second partial unique index allows one running backup per database.
  - Rows that a new constraint would reject are repaired first: an unknown status becomes `FAILED`, an unknown encryption without a salt becomes `NONE`, and all but the newest in-progress row per database are failed.
  - `CHECK` constraints cover the status columns, the encryption and the verification status.
  - The references to `databases` and `users` become `RESTRICT`. The references inside the module stay `CASCADE`.
  - The column `is_skip_retry`, which no code ever sets, is dropped.
- **Frontend.** The logical backup list, the manual backup, the download, the restore dialog and the restore list call the 11 operations through the generated client and types. The hand-written clients and types of `entity/backups/logical` and `entity/restores` are deleted.
- **Tests.** The end-to-end tests of `features/tests/logical` become API tests of the module through `apptest` against real PostgreSQL, MySQL, MariaDB and MongoDB servers. The tests of private methods become tests through the API and the clock. The upgrade test keeps restoring the backup of the previous release through the API of the current code, which is now the new restore operation.
- **BREAKING** for users' scripts:
  - `/backups/...` and `/restores/...` move to the 11 operations above;
  - a manual backup answers 202 with the backup;
  - the backup list pages with a cursor and has no total;
  - a restore answers 202 with the restore;
  - the restore target changes shape;
  - errors are `Problem` bodies.
- **BREAKING** for the schema: three tables are renamed, one column is renamed, one is added and one dropped, and references, constraints and indexes change.

## Capabilities

### New Capabilities

- `logical-backups`: starting, listing, cancelling and deleting logical backups, the scheduler's retries, retention, deletion with the database, and downloading a backup.
- `restores`: starting, listing and cancelling a restore of a logical backup, its target and credentials, and its outcome.

### Modified Capabilities

- `backup-list-paging`: the logical backup list pages with a continuation cursor, like the physical one.
- `databases`: a running restore of a database's backup refuses the database's deletion, like a running backup.
- `workspace-access`: the operations on logical backups, downloads and restores apply the role rules with 404 and 403.

## Out of scope

- Physical backups, the physical restore token and stream, the recovery script and the WAL stream. Milestone 10.
- Changing what the verification agent, the verification runs or the restore verification records do. Milestone 12. The reference from the verification table to the backup table keeps its `CASCADE` until then.
- Moving the health check's view of the scheduler onto the runner's results. Milestone 13.
- Removing the failed and cancelled backups that count and GFS retention keep, and sending the failure notification only after the last retry. Both are today's behavior, and changing them is a product decision.
- Changing which role may do what.

## Impact

- **Backend, new and changed:**
  - `backend/internal/modules/backups` (backups, retention, restores, downloads, the engine registry and the pipelines);
  - `backend/internal/lib/background` (the cancellation registry), new `backend/internal/lib/gfs` (the GFS selection) and new `backend/internal/lib/diskusage` (moved from `features/disk`);
  - `backend/api/openapi.yaml` and `backend/internal/lib/apierror` (three codes);
  - `backend/internal/lib/database` (the time and ID keyset cursor `TimeCursor`);
  - `backend/internal/app` and `backend/internal/app/apptest`.
- **Backend, deleted:**
  - `backend/internal/features/restores`;
  - the logical parts of `backend/internal/features/backups/backups`: core, backuping, controllers, services, dto, use cases and the logical download token;
  - `backend/internal/features/tasks/cancellation`;
  - `backend/internal/features/tests/logical`.
- **Backend, switched to `module.go`:**
  - `features/verification`, `features/telemetry` and `features/system/healthcheck`;
  - the physical backup code where it shares the user streams or the cancellation registry;
  - `backend/e2e/upgrade` and `backend/cmd/seed_dev`.
- **Frontend:**
  - `frontend/src/entity/backups/logical` and `entity/restores`;
  - the logical backup and restore screens of `features/backups` and `features/restores`;
  - the six dictionaries.
- **Migrations:** one migration renames three tables and the token's `used`, adds the restore's `database_id` with its composite reference, drops `is_skip_retry`, repairs the rows the new constraints would reject, and adds the `CHECK` constraints, the two partial unique indexes and the `RESTRICT` references.
- **Running instances:** every logical backup, restore and download token stays with its values. Backups and restores left in progress are failed at the first start, as today.
- **Documents:**
  - the rewrite plan `temp/refactor/plan.md`, which lives only in the main checkout: the checklist and the table of temporary relaxations;
  - `docs/glossary.md`: "Download token" takes its new name in the API, and "Restore target" and "User stream" are added;
  - `docs/architecture.md`: the consumer's engine registry in "Engines and providers", and the `gfs` row in "Shared code";
  - `backend/AGENTS.md`: the same registry rule in "Engines and providers".
- **Rules the change answers to:** the root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`.
