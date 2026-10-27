## Why

A backup config says how a database is backed up: whether backups run, on what schedule, into which storage, how long they are kept, whether they are encrypted and which backup outcomes send a notification. Today two old packages hold it, `features/backups/config/logical` and `features/backups/config/physical`. They are copies of each other in their storage handling, transfer, errors and hooks, and they keep their rows in two unrelated tables keyed by the database ID. The old configs have these problems:

- **Reading writes.** A GET of a config that does not exist yet inserts a default row. A GET of a physical config may rewrite the row to repair it. A GET on the logical endpoint for a physical database creates a logical row, and nothing in the schema keeps a database from having both.
- **Changing the logical storage loses backups early.** When a logical config moves to another storage, its old backups are deleted before the transaction that saves the config. If the save then fails, the backups are gone and the config still names the old storage (`features/backups/config/logical/service.go:314-325`).
- **A database transfer is not atomic.** Moving a database to another workspace runs the database move, the storage move, the notifier moves and the notifier replacement as separate transactions. A failure in a later step leaves the earlier ones committed. The rewrite plan lists this as a relaxation to remove in milestone 9.
- **Errors leak and lie.** Every save error is a 400 with the internal error text, a permission error included. Every read error is a 404, and the storage usage endpoints answer 500 for a missing storage.
- **The SPA asks once per database card.** It reads the config of every database separately just to show the storage name.

Milestone 9 of the backend rewrite plan moves backups into the `backups` module in three changes. `engines-dump-restore` moved dump and restore into the engines. This change is the second part: the `backups` module appears, owns the backup configs of both methods and the database transfer, and serves them through the API contract. Logical backups, restores and downloads follow in `logical-backups-module`, and physical backups in milestone 10.

## What Changes

- **New module.** `internal/modules/backups` appears with `module.go`, `Deps`, `New` and its code under `internal/`. In this change it owns the backup configs and the database transfer. Its services open transactions through the `database.Transactor`, check access through `workspaces.Authorize` and write audit events through `lib/audit` in the same transaction.
- **One config per database, of the database's method.** A logical database (PostgreSQL logical, MySQL, MariaDB, MongoDB) has a logical config and a physical PostgreSQL database a physical one. The fields both methods share sit in one parent row, and the schedule and retention of each method sit in a table of their own. A database whose config was never saved reads as the default config of its method, and the read writes nothing. The first save stores the config.
- **What other code gets from `module.go`:**
  - lookups of the config of one database, and of every config with backups turned on, of each method;
  - the single-column writes of the physical full and incremental backup requests;
  - the handlers of `DatabaseCopied`, `DatabaseBackupTypeChanged`, `DatabaseDeleting` and `StorageTransferring`;
  - two temporary events for the old backup code: `LogicalBackupConfigStorageChanging`, inside the transaction that saves a logical config with another storage, and `PhysicalBackupConfigSaved`, after the transaction that saves a physical config commits. They let the old logical and physical backups react to a config change until each moves into the module. `docs/architecture.md` names the second one as the only event published after a commit whose handler's error becomes the request's answer.
- **Changing the logical storage deletes the old logical backups in the same transaction as the save.** A refusal, such as a logical backup still running, rolls the save back.
- **The database transfer is one transaction.** It checks edit rights in both workspaces, moves the database, then its storage or switches it to a storage of the target workspace, then moves or replaces its notifiers. A failure in any step leaves everything as it was. A database whose config was never saved and that switches to a target storage gets the default config with that storage saved, as today. The transfer calls the `databases`, `storages` and `notifiers` modules with the transaction in the context. `databases.TransferDatabaseToWorkspace` and `databases.ReplaceDatabaseNotifiers` lose their "until the backup configs move" status and stay as the methods the transfer calls.
- **A copy keeps today's rule.** A copied logical database gets a copy of the original's saved config. A copied physical database starts with the default config.
- **A deleted database takes its config with it.** The deletion's transaction removes the config through the `DatabaseDeleting` handler, because the reference from the config to the database becomes `RESTRICT`.
- **Schedules move to `lib/schedule`.** The schedule type and its rules move from `features/intervals` to `lib/schedule`, the package `docs/architecture.md` names for schedules. `features/intervals` keeps only the stored schedule of the old verification config, which delegates to `lib/schedule` until milestone 12.
- **4 operations enter `backend/api/openapi.yaml` under the tag `backups`:**
  - `GET /backup-configs` lists saved configs with a cursor, filtered by exactly one of `workspaceId` or `storageId`;
  - `GET /backup-configs/{databaseId}` reads the config of a database, the default when none is saved;
  - `PUT /backup-configs/{databaseId}` saves it;
  - `POST /databases/{databaseId}/transfer` moves a database to another workspace.

  A config carries the storage by its ID. Its method-specific part is a `oneOf` with the discriminator `type`, `LOGICAL` or `PHYSICAL`.
- **Who may do what is unchanged.** A viewer reads configs, a member saves them, and a transfer needs edit rights in both workspaces. Refusals now answer 404 `not_found` and 403 `permission_denied`.
- **The old code reaches the module through a bridge in `features/backups/config`:** the installed module, plus the test helpers behind the `testhelpers` tag. The old logical and physical backups, restores, verification and the end-to-end tests switch to the config types of `module.go`. The storage usage query `storage_usage` and both old config packages are deleted.
- **Schema.**
  - `logical_backup_configs` becomes `backups_configs_logical` and `physical_backup_configs` becomes `backups_configs_physical`. The new parent table `backups_configs` takes their shared columns: the database, the method, whether backups run, the storage, the encryption and the notification triggers.
  - The comma-joined notification string becomes the `text[]` column `notification_triggers`.
  - `CHECK` constraints cover every enum column.
  - The references to `databases` become `RESTRICT`.
  - A database with both a logical and a physical row keeps the one of its method. A value that breaks a new constraint is repaired before the constraint is added.
- **Frontend.** The config screens, the database card, the database creation, the database transfer dialog and the storage transfer dialog call the 4 operations through the generated client and types. The hand-written config clients and types of `entity/backups` are deleted. `entity/intervals` stays for the verification config, which is not on the contract yet. The database list reads the configs of a workspace in one request.
- **Tests and the upgrade test.** The config tests become module API tests through `apptest` and the contract-checking client. The upgrade test fills every setting of the logical config it writes and checks each one after the upgrade.
- **BREAKING** for users' scripts:
  - the eight `/backup-configs/...` routes are replaced by the four operations above;
  - a config carries `storageId` instead of an embedded storage, and its method-specific fields under `settings`;
  - the two storage usage routes are replaced by the list's `storageId` filter;
  - errors are `Problem` bodies.
- **BREAKING** for the schema: two tables are renamed, one table is added, columns move into it, and constraints are added.

## Capabilities

### New Capabilities

- `backup-configs`: reading, saving and listing the backup config of a database, its defaults, its validation per method, its storage, its copy and deletion with the database, and moving a database to another workspace.

### Modified Capabilities

- `backup-storage-assignment`: a config names its storage only by ID, and moving a logical config to another storage deletes the old logical backups in the same save.
- `workspace-access`: the backup configs and the database transfer apply the role rules with 404 and 403 like the rest of the workspace's content, and the transfer loses its exemption.
- `upgrade-test`: the test writes a logical backup config with every setting filled in and checks every value after the upgrade.

## Out of scope

- Moving logical backups, restores, downloads, retention or the backup scheduler into the module. `logical-backups-module`.
- Moving physical backups or the WAL stream into the module. Milestone 10.
- Switching the references from backup tables to `databases` from `CASCADE` to `RESTRICT`. Each moves with its backups.
- Deleting the backups that stay in the old storage when a database is transferred to a storage of the target workspace. The transfer keeps them, as it does today.
- A physical config in the upgrade test. It needs a physical database on a source that allows replication, which milestone 10 adds together with the physical backups.
- Changing a default value, a validation rule or a notification trigger of either method.

## Impact

- **Backend, new and changed:**
  - new `backend/internal/modules/backups`;
  - new `backend/internal/lib/schedule`, moved from `backend/internal/features/intervals`;
  - `backend/api/openapi.yaml`;
  - `backend/internal/app` (construction, routes, event wiring) and `backend/internal/app/apptest`;
  - `backend/internal/modules/databases/module.go`: the new batch method `GetPhysicalBackupTypes`, which returns the physical backup types of a set of databases without their settings; the actor in `TransferDatabaseToWorkspace`; the comment on the two transfer methods;
  - `backend/internal/modules/databases/internal/userdatabase/transfer_service.go`, whose audit event gets its author;
  - the tests of `backend/internal/modules/databases`, `storages` and `notifiers` and the `apptest` helpers that use the old config tables and routes.
- **Backend, reduced:** `backend/internal/features/backups/config` down to the bridge and its test helpers.
- **Old callers that switch to the bridge and the types of `module.go`:**
  - `features/backups/backups` (backuping, services, use cases, core);
  - `features/restores` and `features/verification`;
  - `features/tests`;
  - `backend/e2e/upgrade` and `backend/cmd/seed_dev`.
- **Frontend:**
  - `frontend/src/entity/backups` (the config clients and types), which moves to the generated client;
  - the config screens of `features/backups`;
  - the database card, database creation and database transfer of `features/databases`;
  - the storage transfer of `features/storages`;
  - no dictionary: the operations answer only error codes the contract already has.
- **Migrations:** one migration renames the two tables, creates the parent, moves the shared columns, repairs rows and adds the constraints.
- **Running instances:** every saved config keeps its values and its storage. A database that has both a logical and a physical row keeps the row of its method, and the other row is deleted. A database without a saved config keeps reading as the default.
- **Documents:**
  - the rewrite plan `temp/refactor/plan.md`, which lives only in the main checkout: the checklist and the table of temporary relaxations;
  - `docs/glossary.md`: "Backup config" no longer says a database may have two, and "Backup method" and "Notification trigger" are added;
  - `docs/architecture.md`, "Modules": `PhysicalBackupConfigSaved` as the one event published after a commit that can fail its request, until physical backups move into `backups`; and, only if the parent row with a table per method does not read as covered by "Engines and providers", a sentence there.
- **Rules the change answers to:** the root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`.
