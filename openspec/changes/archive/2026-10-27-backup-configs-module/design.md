## Context

Why the change is needed is in proposal.md, section Why. This document covers what shapes the approach. Paths are under `backend/internal/` unless they start with another top-level directory. `CL` stands for `features/backups/config/logical` and `CP` for `features/backups/config/physical`.

**Models and tables.**
- The logical config is `CL/model.go:16-43`, table `logical_backup_configs`. It embeds the schedule without a column prefix (`interval_type`, `time_of_day`, `weekday`, `day_of_month`, `cron_expression`). It holds the retention policy (`TIME_PERIOD`, `COUNT` or `GFS`) with its period, count and five GFS counts, the retry flag with its count, the storage, the encryption and `send_notifications_on` as a comma-joined string.
- The physical config is `CP/model.go:17-43`, table `physical_backup_configs`, created in `migrations/20260523120000_add_physical_pg_17_backups.sql:84-132`. It holds two schedules with the prefixes `full_` and `incremental_`, the retention (`CHAINS`, `FULL_BACKUPS` or `CHAINS_AND_FULL_BACKUPS`) with the chain count and the full-backup policy, the WAL lag threshold, the two force-request timestamps, the storage, the encryption and the notification string.
- Both tables use `database_id` as their primary key and reference `databases` with `ON DELETE CASCADE`. Their storage references are `RESTRICT` since `migrations/20261026090000_restrict_storage_references.sql`. No enum column has a `CHECK`.

**Reads that write.**
- `GetBackupConfigByDbId` inserts a default row when none exists (`CL/service.go:107-125`, `CP/service.go:130-147`). Every old caller reaches configs through it: the logical scheduler and backuper, the restorer, the physical backuper, the physical service and the WAL stream statistics.
- The physical GET also repairs a row left invalid by an earlier backup type switch and saves it (`CP/service.go:107-128`, `:520-550`).

**Saving.**
- Both services check the storage's workspace before the save, and again in the saving transaction under `storages.GetStorageBlockingTransfer`, a `FOR SHARE` lock (`CL/service.go:295-357`, `CP/service.go:486-519`).
- The logical service calls `OnBeforeBackupsStorageChange` before that transaction, and the old logical backup service deletes every logical backup of the database there (`CL/service.go:314-325`, `features/backups/backups/services/logical_service.go:68-75`). A running backup makes it fail with an unmapped error.
- The physical service calls three listeners after the write, outside any transaction: cancel on disable, cancel on a storage change, and remove the backups kept in another storage. The last one runs after every save that names a storage and returns its error to the caller (`CP/service.go:340-365`), as the `backup-storage-assignment` requirement on physical configs demands.

**Transfer.** `TransferDatabaseToWorkspace` is implemented twice (`CL/service.go:210-291`, `CP/service.go:223-303`) and opens no outer transaction. It checks both workspaces through `workspaces_bridge`, then follows one of two paths:
- moving the storage too: refuse when other databases use the storage, then `databases.TransferDatabaseToWorkspace`, then `storages.TransferStorage` (`CL/service.go:359-389`);
- switching to a target storage: save the config without a lock, then `databases.TransferDatabaseToWorkspace` (`:391-414`).

After either path it moves each notifier through `notifiers.TransferNotifier`, skipping `in_use` (`:532-555`), and calls `databases.ReplaceDatabaseNotifiers`. Each of those calls commits on its own. `databases.TransferDatabaseToWorkspace` writes its audit event with no author (`modules/databases/internal/userdatabase/transfer_service.go:14-52`).

**Events today** (`app/wiring.go:52-78`):
- `CL.OnStorageTransferring` refuses a storage move while a database outside the target workspace uses the storage, reading both tables through `storage_usage` (`CL/service.go:155-173`, `storage_usage/storage_usage.go:15-29`);
- `CL.OnDatabaseCopied` copies the logical config (`CL/service.go:184-204`);
- `CP.OnBackupTypeChanged` coerces the physical config in the event's transaction and stops the WAL stream after commit (`CP/service.go:175-217`).

**Schedules.** `features/intervals/model.go` holds `Interval` with gorm column tags, `Validate`, `ShouldTriggerBackup`, `ApproxPeriod` and `Copy`, with 796 lines of tests. The old verification config embeds it as columns (`features/verification/config/model.go:21`).

**HTTP.** Eight Gin routes (`CL/controller.go:18-22`, `CP/controller.go:18-20`) answer `gin.H{"error": err.Error()}`. The SPA reads the config of every database card separately (`frontend/src/features/databases/ui/DatabaseCardComponent.tsx:30-38`). It checks storage use through `is-using` (`features/storages/ui/StorageTransferDialogComponent.tsx`) and `databases-count` (`DatabaseTransferDialogComponent.tsx:88`).

**Rules in force.**
- `docs/architecture.md`, "Modules": `module.go` as the whole contract, events with handlers in the publisher's transaction and after-commit hooks.
- "Engines and providers": a parent row with a table per kind, referenced through `(id, type)`.
- "Data": prefixes, `RESTRICT` between modules, `CHECK` on enums, one `UPDATE` per column, upgrade-safe migrations.
- "HTTP API": contract first, `Problem`, cursor pages.
- `backend/AGENTS.md`: handler, service and repository roles; `Deps`; `workspaces.Authorize`.
- `frontend/AGENTS.md`: generated client, `translateApiError`, no literal strings.
- The `api-design` skill.
- The change answers to the root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`.

## Goals / Non-Goals

**Goals:**

- The `backups` module exists, passes every rule of the target layout, and owns the configs and the database transfer.
- The old logical and physical backup code keep their behavior and reach the configs only through `module.go`.
- No read writes, and every multi-step write is one transaction.

**Non-Goals:**

- Moving any backup, restore, download or retention code into the module.
- Changing defaults, validation rules or what a config change does to running or stored backups.

## Decisions

### 1. The packages

```
internal/modules/backups/
  module.go            Module, Deps, New, the config types, the two temporary events
  internal/
    module/            the Module implementation that wires the services
    config/            the configs: models per method, defaults, validation, type coercion,
                       repositories, service, event handlers
    transfer/          the database transfer service
    httpapi/           the generated strict server, handlers and conversions
internal/lib/schedule/ the schedule type, its validation and when it is due
```

`config` holds one concept, the backup config of a database, split into files per method and role: `logical_config.go`, `physical_config.go`, `defaults.go`, `physical_type_coercion.go`, `config_repository.go`, `service.go`, `errors.go`. `transfer` is separate because it changes rows of three other modules and owns no table.

*Rejected: a subpackage per method.* The two methods share the parent row, the storage check, the events and the service. A split would put that shared code into a third package that both import, which is the old layout again.

### 2. A parent row with a table per method

```
backups_configs                       backups_configs_logical / backups_configs_physical
  database_id  PK, -> databases RESTRICT    database_id  PK
  method       LOGICAL | PHYSICAL           method       CHECK = 'LOGICAL' / 'PHYSICAL'
  is_backups_enabled                        (database_id, method) -> backups_configs CASCADE
  storage_id   -> storages RESTRICT         schedule and retention columns of the method
  encryption   NONE | ENCRYPTED             physical only: wal_lag_threshold_bytes,
  notification_triggers text[]                force_full_requested_at, force_incremental_requested_at
  UNIQUE (database_id, method)
```

The primary key on `database_id` in the parent keeps one config per database. The composite reference keeps a method's row on a parent of that method. The schedule, retention and retry columns keep their names, so the migration moves no data between child tables. `notification_triggers` holds the union of both methods' triggers, and a `CHECK` limits a logical row to `BACKUP_FAILED` and `BACKUP_SUCCESS`.

The parent shares the shape of `storages` and `notifiers` from "Engines and providers" in `docs/architecture.md`, with the method in the role of the type. The storage check, the storage-change event, the transfer and the storage filter of the list read the parent alone.

*Rejected: rename the two tables and keep them apart.* The storage filter would stay a `UNION`, every storage and transfer path would exist twice, and nothing would stop both rows.

*Rejected: one wide table with nullable columns per method.* That is the nullable-pointer shape `docs/architecture.md` removed from providers, and a `CHECK` per method would have to list which columns must be null.

### 3. The migration

One migration, in its transaction:
1. Rename `logical_backup_configs` to `backups_configs_logical` and `physical_backup_configs` to `backups_configs_physical`, with their constraints and indexes.
2. Delete the row of the other method where a database has both. The method follows `databases.type`: `POSTGRES_PHYSICAL` keeps the physical row, every other type the logical one.
3. Create `backups_configs`, and insert from each child its database, method, enabled flag, storage, encryption and notification triggers, splitting the comma-joined notification string into `notification_triggers` and dropping empty, duplicate and unknown entries.
4. Repair the values the new `CHECK` constraints would reject. An unknown retention policy, retention period, schedule type, physical retention, full-backup policy or encryption becomes the default of its method from `config/defaults.go`, written as literals in the migration. The repair touches only those columns.
5. Drop the moved columns from the children, add `method` with its `DEFAULT` and `CHECK`, the composite reference to the parent with `CASCADE`, and the parent's references to `databases` and `storages` with `RESTRICT`.
6. Add the `CHECK` constraints on every enum column.

Rows that fit the old validation keep every value. The upgrade test of decision 12 writes every logical setting through the previous release and reads it back.

### 4. Defaults are computed, not stored

`GetBackupConfig` returns the saved config, or the default of the database's method when no row exists, and writes nothing. The physical default is built and then coerced to the database's physical backup type. Today `CP/repository.go:122-131` reads that type through the decrypted connection settings, but the type needs no secret, and the list would read the settings once per physical config. `databases` therefore gains a batch method:

```go
// GetPhysicalBackupTypes leaves out the IDs that match no physical PostgreSQL database.
GetPhysicalBackupTypes(ctx context.Context, databaseIDs []uuid.UUID) (map[uuid.UUID]PhysicalBackupType, error)
```

It reads the type column of the physical provider's table and decrypts nothing. A read asks for one ID, and a list page asks once for the IDs of its physical configs. `PUT` inserts or updates. The schedulers of the old code list only saved configs with backups turned on, so an unsaved config runs no backup, as an unsaved default does today.

A saved physical config that does not fit its database's type is returned coerced, by a read and by the list, and is not saved. Such a row comes only from an earlier release, whose type switch left the config as it was. The next `PUT` saves the coerced values, and the type-change handler keeps saved rows fitting from now on.

*Rejected: create the default in a `DatabaseCreated` handler.* Every existing database without a row would need a default written by the migration, and the physical default depends on the backup type, so the migration would repeat the coercion rules in SQL. A row nobody saved has no value of its own to keep.

*Rejected: keep the lazy insert.* A GET by a viewer would write, and a GET on the wrong method would create the second row the schema now forbids.

### 5. The config types in `module.go`

```go
type Method string // MethodLogical and MethodPhysical, stored and sent as LOGICAL and PHYSICAL

type BackupConfig struct {
	DatabaseID       uuid.UUID
	IsBackupsEnabled bool
	StorageID        *uuid.UUID
	Encryption       Encryption
	NotificationTriggers []NotificationTrigger
}

type LogicalBackupConfig struct {
	BackupConfig
	Schedule            schedule.Schedule
	Retention           LogicalRetention
	IsRetryIfFailed     bool
	MaxFailedTriesCount int
}

type PhysicalBackupConfig struct {
	BackupConfig
	BackupType                  databases.PhysicalBackupType
	FullBackupSchedule          schedule.Schedule
	IncrementalBackupSchedule   *schedule.Schedule
	Retention                   PhysicalRetention
	WalLagThresholdBytes        int64
	ForceFullRequestedAt        *time.Time
	ForceIncrementalRequestedAt *time.Time
}
```

The old code reads these through the module. Methods that only read the values, such as `IsWalStreamingExpected` and the GFS counts, move with them. The enums `Encryption`, `NotificationTrigger`, `LogicalRetentionPolicy`, `PhysicalRetentionPolicy` and `FullBackupsRetentionPolicy` sit in `module.go` as the types of these fields, with the same string values as today.

The module offers the old code:
- `GetLogicalBackupConfig` and `GetPhysicalBackupConfig` (saved or default);
- `ListEnabledLogicalBackupConfigs` and `ListEnabledPhysicalBackupConfigs`;
- `RequestFullBackup`, `ClearFullBackupRequest`, `RequestIncrementalBackup` and `ClearIncrementalBackupRequest`, each one conditional `UPDATE` of one column, the clear guarded by the timestamp as `CP/repository.go:59-97` does;
- `SaveBackupConfig(ctx, actor, config)`, the service behind `PUT`, for the old test helpers and `seed_dev`.

### 6. Two temporary events for the old backup code

```go
// Published inside the transaction that saves a logical config with another storage.
type LogicalBackupConfigStorageChanging struct {
	DatabaseID        uuid.UUID
	PreviousStorageID *uuid.UUID
	StorageID         uuid.UUID
}

// Published after the transaction that saves a physical config commits; a handler's error becomes
// the save's error.
type PhysicalBackupConfigSaved struct {
	Previous *PhysicalBackupConfig // nil when no config was saved before
	Saved    PhysicalBackupConfig
}
```

- The old logical backup service subscribes to `LogicalBackupConfigStorageChanging` and deletes the database's logical backups inside the save's transaction. Its refusal while a backup runs becomes 409 `in_use` and rolls the save back. That fixes the early deletion in `CL/service.go:314-325`.
- The old physical code subscribes to `PhysicalBackupConfigSaved`. It keeps today's order and today's error answer from `CP/service.go:340-365`: cancel on disable, cancel on a storage change, then remove the backups kept in another storage, whose error the save reports while the config keeps the new storage.

`internal/app/wiring.go` registers both, with the other handlers. `logical-backups-module` removes the first event when the logical backups join the module, and milestone 10 removes the second. The rewrite plan's relaxation table gets a row for each.

`PhysicalBackupConfigSaved` is a kind of event that "Modules" in `docs/architecture.md` does not describe: it is published after the commit, and its handler's error becomes the answer of a request that has already committed. Task 8.1 adds it to that section as the one such event, naming the reason and its removal when physical backups move into `backups`. The reason holds for no new code: the old physical removal must see the new storage committed, so a full backup or a WAL streamer that starts meanwhile already writes there, and the main `backup-storage-assignment` spec wants its error in the save's answer while the config keeps the new storage.

*Rejected: publish `PhysicalBackupConfigSaved` inside the save's transaction and register the reactions with `database.AfterCommit`.* The cancellations would fit, but the removal would either run before the commit, when a backup that starts meanwhile still reads the old storage, or lose its error, which the spec wants in the answer.

*Rejected: after-commit hooks for the physical reactions.* A hook cannot fail the request, and the `backup-storage-assignment` requirement on physical configs wants the removal error in the save's answer.

*Rejected: put an interface of the old code into `Deps`.* The module would declare a dependency on code above it in the layering, and `internal/app` would have to build the old services before the module.

### 7. The database transfer is one transaction

`transfer.Service.TransferDatabase(ctx, actor, request)` runs in one `Transaction`:
1. Look the database up through `databases.GetDatabases`, answering 404 when it is missing. Call `workspaces.Authorize` with `ActionEditContent` for its workspace and for the target. `Authorize` answers 404 for a non-member and 403 for a role too low.
2. Read the database's saved config for update.
3. Move the database with `databases.TransferDatabaseToWorkspace`, which locks its row and writes the audit event. The method gains the actor, so the event names its author instead of none.
4. Take one of the two storage options:
   - **Move the storage with the database.** Refuse with 400 when the config names no storage. Otherwise call `storages.TransferStorage`. Its `StorageTransferring` event reaches this module's handler (decision 8), which refuses with 409 `in_use` when a database outside the target workspace uses the storage. The moved database is already in the target, so a storage only it uses passes. This replaces the separate check through `storage_usage` (`CL/service.go:367-378`), which answered the same question earlier with a 400.
   - **Switch to a target storage.** Check under `GetStorageBlockingTransfer` that the storage belongs to the target workspace, answering 400 otherwise. When the config is saved, write `storage_id` with one `UPDATE`. When it was never saved, insert the default of the database's method with the target storage, as the old transfer does through the lazy insert of `GetBackupConfigByDbId` (`CL/service.go:248`). The backups in the previous storage stay, as `CL/service.go:391-414` keeps them, and no storage-change event is published.
5. When asked, move each notifier of the database through `notifiers.TransferNotifier`, each in a nested transaction, which is a savepoint. A notifier another database uses is refused with `in_use`, and its savepoint is rolled back, so it stays and the transfer goes on.
6. When given replacing notifiers, call `databases.ReplaceDatabaseNotifiers`, which checks that each belongs to the database's new workspace and answers 400 otherwise.

An error in any step rolls back all of them, the database move and the storage move included. There is no up-front notifier check (`CL/service.go:557-580`): step 6 runs the same check, and a refusal there is the late failure the spec's scenario exercises.

*Rejected: keep the steps and add compensation.* Undoing a committed move needs the same access checks in reverse and can fail itself. One transaction is what the other modules' methods already support, because they join the transaction in the context.

### 8. Event handlers

The module registers handlers in `internal/app/wiring.go`:
- `StorageTransferring` from `storages`: decision 7, reading the parent table only.
- `DatabaseCopied` from `databases`: copy the saved logical config of the original. A physical copy gets nothing and reads as the default (`databases` spec, "A copy starts with no backup history").
- `DatabaseBackupTypeChanged` from `databases`: coerce the saved physical config in the event's transaction. Stopping the WAL stream stays with the old physical code, which subscribes to this event itself with an after-commit hook, as `CP/service.go:179-188` registers it today.
- `DatabaseDeleting` from `databases`: delete the config row, registered after the old backup handlers so that they still read it during the deletion.

### 9. The contract

Four operations under the tag `backups`:

| Operation | Request | Response |
|---|---|---|
| `GET /backup-configs` `listBackupConfigs` | exactly one of `workspaceId` and `storageId`; `limit`, `after` | `items` of `BackupConfig` ordered by `databaseId`, `nextCursor` |
| `GET /backup-configs/{databaseId}` `getBackupConfig` | | `BackupConfig` |
| `PUT /backup-configs/{databaseId}` `updateBackupConfig` | `BackupConfigInput` | `BackupConfig` |
| `POST /databases/{databaseId}/transfer` `transferDatabase` | `DatabaseTransfer` | 204 |

- `BackupConfig` holds `databaseId`, `isBackupsEnabled`, `storageId` (nullable), `encryption` and `settings`. `settings` is a `oneOf` with the discriminator `type` between `LogicalBackupSettings` and `PhysicalBackupSettings`.
- `BackupConfigInput` holds the same fields without `databaseId`. Its `storageId` is required and nullable, and `null` saves a config without a storage. The server does not validate request bodies against the schema, so a body that omits `storageId` decodes as `null` too. A storage object in the body is an unknown field and is dropped, which is why the `backup-storage-assignment` scenarios check that no storage changes rather than a status.
- Each settings schema holds the schedule, the retention, the retry fields or the WAL lag threshold, and `notificationTriggers` with the enum of its method.
- `Schedule` keeps today's field names (`type`, `timeOfDay`, `weekday`, `dayOfMonth`, `cronExpression`), so the schedule editor of the SPA keeps working with the verification config's hand-written type.
- The force-request timestamps stay out of the contract, as they are `json:"-"` today.
- The config is addressed by `{databaseId}` because it has no identity of its own: a database has exactly one, and its primary key is the database ID. The contract describes the parameter as the key of the config, so `vacuum` and readers see why it is not `{backupConfigId}`.
- The transfer is an action on an existing database, so it is a `POST` on the verb sub-path `/databases/{databaseId}/transfer`, like `POST /storages/{storageId}/transfer` and `POST /notifiers/{notifierId}/transfer`. Its tag is `backups`, the module that implements it; the `api-design` skill ties the tag to the implementing module, not to the path's prefix.
- `DatabaseTransfer` holds `targetWorkspaceId`, `storage`, `isMovingNotifiers` and `targetNotifierIds`. `storage` is a `oneOf` with the discriminator `type`: `MOVE_WITH_DATABASE`, or `SWITCH_TO_STORAGE` with `storageId`.
- The errors are `invalid_request`, `not_found`, `permission_denied` and `in_use`, all of which the contract and the dictionaries already have.

*Rejected: `/databases/{databaseId}/backup-config`.* The config is a resource read, listed and replaced on its own, and the `api-design` skill lists such workspace content with a filter instead of nesting it.

*Rejected: `POST /database-transfers`.* A `POST` on a collection creates a resource, and a transfer creates nothing that could be read back.

*Rejected: name the triggers after `NotificationCategory`.* A config notifies on `CHAIN_BROKEN` and `WAL_GAP` too, which are no notification categories: the physical code sends both as a failed backup. The config's values are the outcomes that send a notification, so `docs/glossary.md` gets the term "Notification trigger" in task 8.1.

*Rejected: return the storage's name and type in the config.* The SPA already lists the workspace's storages, and the config would read another module's data on every request.

### 10. `lib/schedule`

`schedule.Schedule` has the fields of today's `Interval` without tags, with `Validate`, `IsDue(now, lastRun)` (today's `ShouldTriggerBackup`) and `GetApproxPeriod`. The tests move with it. `features/intervals` keeps the tagged struct the old verification config embeds, with a `ToSchedule` conversion. Its old methods delegate, so the verification scheduler keeps working until milestone 12 moves it. The old backup code switches to `schedule.Schedule` through the config types.

*Rejected: put gorm tags on `schedule.Schedule`.* The column names are a table's business. The module's repositories map their own row structs, and a tag in `lib` would bind every user to one table's columns.

### 11. The bridge and the old code

`features/backups/config` becomes the bridge package `backups_bridge`, named after its module like `storages_bridge` and `databases_bridge`, which `logical-backups-module` extends to the logical backups, with `Install`, `GetModule` and the test helpers behind `testhelpers`, the way `features/databases` serves the databases module. The old callers switch from `*CL.LogicalBackupConfig` and `*CP.PhysicalBackupConfig` to the types of `module.go`. Those callers are the scheduler, backuper and cleaner of each method, the services, the use cases, the restorer, the physical WAL supervisor and statistics, and the end-to-end test helpers. `storage_usage`, both config packages and their controllers are deleted. `internal/app/model_schemas.go` drops the two config models.

### 12. Tests

- **Module API tests** through `apptest` and the contract-checking client cover every requirement of the `backup-configs` delta spec: defaults without a write, save and validation per method, the storage checks with a concurrent storage transfer, the list filters and pages, the copy, the deletion, the type change and every transfer path. The transfer tests include the rollback after a refused replacing notifier, the unsaved config switched to a target storage, and the 404 and 403 refusals.
- **The race of a save and a storage transfer** starts both requests together many times and asserts each round ends in one of the two outcomes of the `backup-storage-assignment` scenario, never both successes.
- **An unfit physical config from an earlier release** is written with `ExecuteSQL`, because no current path can store it, then read, listed and saved through the API.
- **A failed removal of the logical backups** uses a handler of `LogicalBackupConfigStorageChanging` that fails, subscribed by the test through `apptest` after the old logical backup service, and asserts the 500, the old storage and the still-listed backups.
- **Storage changes:** the logical deletion inside the save and its refusal while a backup runs are tested through the old logical backups and the module together. The physical removal tests keep running through the old physical code, which now subscribes to `PhysicalBackupConfigSaved`.
- **`lib/schedule`** takes over the tests of `features/intervals`.
- **The upgrade test** writes a logical config through the previous release's API with every value different from the logical default: backups on, encryption on, the local storage, a weekly schedule on Wednesday at 02:30, GFS retention with every period, five retries and only `BACKUP_FAILED`. It reads every value back through the new operation, so a migration that resets a value to its default fails the test.
- **The old tests** of `CL` and `CP` are deleted once their cases exist as API tests. `cross_strategy_storage_test.go` becomes a list test with the `storageId` filter over a physical config.

### 13. Frontend

- `entity/backups` loses its config clients and config types. The config screens, the database card, the database creation, the database transfer dialog and the storage transfer dialog call the generated client.
- The database list reads the workspace's configs with one `listBackupConfigs` call and finds each card's storage in the storages it has already loaded.
- The storage transfer dialog lists with `storageId` and `limit=1` to learn whether the storage is used. The database transfer dialog lists with `limit=2` to learn whether another database uses it.
- `BackupConfigStorage` and `toBackupConfigStorage` are deleted. The edit forms keep their fields and layout, and send `storageId`.
- `entity/intervals` stays for the verification config.

## Risks / Trade-offs

- [A migration repairs a stored value a user chose] → Only values that no validation accepts are repaired, and only per column. The upgrade test checks that valid values survive.
- [Two temporary events couple the module to old code until milestone 10] → They are declared in `module.go` with the milestone that removes them, and the relaxation table names them.
- [The transfer holds locks on the database, the storage and the notifiers for its whole duration] → The steps are a handful of row updates. A concurrent save of the storage's config waits, which the spec of `backup-storage-assignment` requires anyway.
- [A refused notifier inside a savepoint] → `TransferNotifier` refuses through its handler's query before any write fails at the SQL level, so the outer transaction stays usable. A test pins a transfer where one notifier is shared.
- [The list's order by `databaseId` is not alphabetical] → The SPA matches configs to databases by ID and keeps its own order.

## Migration Plan

The migration runs at startup like every other. It is one-way, as all migrations are. A database with two config rows loses the one of the other method, which no code path reads for that database today.
