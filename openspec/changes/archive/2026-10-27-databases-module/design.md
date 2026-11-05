## Context

Why the change is needed is in proposal.md, section Why. This document covers what shapes the approach. It assumes `engines-layer` has landed: the engine behavior lives in `internal/engines`, and the old engine models in `features/databases/databases/*` delegate to it. From `notifiers-module` and `storages-module` it takes the module and bridge pattern, the `in_use` code, `database.NameCursor`, `database.IsForeignKeyViolation`, `database.WithoutTransaction` and the `/…-tests` resource for unsaved settings. Paths are under `backend/internal/` unless they start with another top-level directory.

**Operations.** `DatabaseController` (`features/databases/controller.go:21-33`) serves 13 routes:

| Route | Service | Access |
|---|---|---|
| `POST /databases/create` | `CreateDatabase` | edit |
| `POST /databases/update` | `UpdateDatabase`, answers with the request body | edit |
| `DELETE /databases/:id` | `DeleteDatabase` | edit |
| `GET /databases/:id` | `GetDatabase` | view |
| `GET /databases?workspace_id=` | `GetDatabasesByWorkspace` | view |
| `POST /databases/:id/test-connection` | `TestDatabaseConnection` | view |
| `POST /databases/test-connection-direct` | `TestDatabaseConnectionDirect` | edit |
| `POST /databases/:id/copy` | `CopyDatabase` | edit |
| `GET /databases/notifier/:id/is-using` | `IsNotifierUsing` | view of the notifier |
| `GET /databases/notifier/:id/databases-count` | `CountDatabasesByNotifier` | view of the notifier |
| `POST /databases/should-suggest-readonly-user` | `ShouldSuggestReadOnlyUser` | view |
| `POST /databases/create-readonly-user` | `CreateReadOnlyUser` | edit |
| `POST /databases/create-replication-only-user` | `CreateReplicationOnlyUser` | edit |

Six routes bind the request body into the GORM model `Database` (`controller.go:46,73,187,300,329,354`). The controller holds a `workspaceService` it never uses (`controller.go:15-18`). Errors go out as `{"error": err.Error()}`. `respondConnectionTestError` (`controller.go:201-223`) answers 403, 500 with two fixed messages, or 400 with `{code}` of `postgresql_shared.ConnectionTestError`.

**Defects in the service.**
- *Create takes an ID.* `CreateDatabase` (`service.go:64-140`) keeps the `id` of the body, and `DatabaseRepository.Save` (`repository.go:20-60`) treats a non-nil ID as an existing row. It runs a GORM `Save`, which updates by primary key, with `workspace_id` set to the caller's workspace. The child row of MySQL, MariaDB and MongoDB is then inserted, because those tables have no unique `database_id`. The takeover commits after a successful connection test against the caller's own server.
- *A viewer can leak secrets through the read-only user check.* `ShouldSuggestReadOnlyUser` (`service.go:661-738`) loads a saved database by the body's ID with `ActionView`. It merges the body through `Update`, which keeps the stored password and TLS key when the incoming ones are empty (`logical/model.go:199-205`) and the SSH secrets the same way. Then it connects to the merged host.
- *Testing a saved database rewrites it.* `TestDatabaseConnection` (`service.go:389-438`) needs only `ActionView`, yet it saves the whole row to clear `last_backup_error_message` (`:432`).
- *Deletion is not atomic.* `DeleteDatabase` (`service.go:254-294`) runs the `OnBeforeDatabaseRemove` listeners, each committing its own work, writes the audit entry and then deletes the row (`repository.go:179`).
- *Attaching a notifier races its transfer.* `validateNotifiersInWorkspace` (`service.go:84,177`) reads the notifiers before the connection test and the save. `replaceNotifierIDs` (`repository.go:301`) writes the attachments later in another transaction.
- *No transactions.* No service method opens a transaction. Most repository methods use `storage.GetDb()` and ignore the context (`repository.go:122,135,227,241,262`).

**The cycle.** `di.go:30` injects `physical_core_service.GetPhysicalBackupService()`, which is used only by `fillPhysicalLastBackupTimes` (`service.go:995-1020`). That function computes the last backup time of physical databases on every list from `MAX` over full, incremental and committed WAL rows (`core/physical/service/service.go:436-475`). Telemetry calls the same function (`telemetry/service.go:311`).

The last backup time is written in two ways:
- logical backups call `SetLastBackupTime` and `SetBackupError` (`backuping/logical/backuper.go:154,219,415`), single-column updates;
- health checks call `SetHealthStatus` (`healthcheck/attempt/check_database_health_uc.go:91,129`, `healthcheck/config/service.go:72`).

**Listeners.** `interfaces.go` declares four listener interfaces with `Add*Listener` setters (`service.go:40-61`). Each feature's `di.go` registers them, and `internal/app/wiring.go:22-32` calls those setups:
- creation: the health check config (`healthcheck/config/di.go:44`);
- copy: the logical backup config and the verification config (`backups/services/di.go:86`, `verification/config/di.go:41`);
- backup type change: the physical backup config (`config/physical/di.go:46`);
- removal: the logical backup service, the physical cancellation listener, the physical backup service and the physical slot cleanup listener, in that order. The comment at `backuping/physical/di.go:136` says the order matters.

`config/physical/service.go:171` has an `OnDatabaseCopied` that nothing registers. `wiring.go:42-44` subscribes `OnWorkspaceDeleting` of databases, notifiers and storages in that order, and `:50` subscribes `OnNotifierTransferring`.

**Readers of the old model.** 125 Go files outside `features/databases` import it.
- Backups and restores call the engine methods of the old models through `OpenTunnel` and read engine fields such as `PostgresqlPhysical.BackupType`, `ReplicationSlotName` and `SshTunnel`.
- `backups/config/physical/repository.go:40,96` preloads `postgresql_physical_databases` through a GORM association (`config/physical/model.go:19`).
- The restore request embeds the engine models as payload fields (`restores/core/model.go:22-25`, `dto.go:11-14`). `restores/service.go:321` and `restores/restoring/restorer.go:109` build a `Database` from them to open a tunnel and detect the version.
- `telemetry/service.go` reads the type, health, version, tunnel flag and backup type of every database.
- `backup_slot.go:143` lists every database.
- The database transfer lives in the old backup configs (`config/logical/service.go:198-284` and its physical twin). It calls `TransferDatabaseToWorkspace` (`service.go:560`) and `UpdateDatabaseNotifiers` (`:602`), neither of which checks access or opens a transaction.

**Schema**, from the migrations:
- `databases`: `type text NOT NULL` without `CHECK`; `health_status` nullable without `CHECK`; `workspace_id` nullable with `fk_databases_workspace_id … ON DELETE RESTRICT`.
- `20251031155005_connect_workspaces_with_dbs.sql` left NULL on databases that restores created before workspaces existed. No current code creates such rows.
- `postgresql_logical_databases` has its own `id`, a unique `database_id` and **no foreign key** to `databases`.
- `postgresql_physical_databases` has a unique `database_id` with `CASCADE`.
- `mysql_databases`, `mariadb_databases` and `mongodb_databases` reference `databases` with `CASCADE`, without a unique `database_id`.
- The list columns are comma-separated `text` and are parsed by `namelist` on commas, newlines and tabs, trimmed, empty entries dropped, duplicates dropped in first-seen order:
  - `include_schemas`, `exclude_tables` and `exclude_table_data` of logical PostgreSQL;
  - `exclude_tables` and `privileges` of MySQL and MariaDB;
  - `exclude_collections` of MongoDB.
- `database_notifiers` has the primary key `(database_id, notifier_id)`, `CASCADE` to `databases` and `RESTRICT` to `notifiers`.
- Logical and physical backup tables, health check tables and verification tables reference `databases` with `CASCADE`.

**Tests.**
- `features/databases/controller_test.go` (2638 lines) drives the old routes.
- `service_test.go`, `model_test.go`, `physical_model_test.go` and `tunneled_database_test.go` cover the status setters, copying, hiding secrets and the tunnel wrapper.
- Outside the package, 37 test files in 20 packages call `CreateTestDatabase` 107 times and `RemoveTestDatabase` 198 times.
- `apptest.InsertDatabase` inserts into `databases` and `postgresql_logical_databases` with SQL, and `apptest.AttachNotifierToDatabase` inserts into `database_notifiers`.
- The upgrade test writes one logical PostgreSQL database (`e2e/upgrade/previous_release_client_test.go:175`).

**Frontend.**
- `entity/databases` has a hand-written `databaseApi` with 13 methods and model files per engine.
- 40 files outside the entity import the `Database` type: 27 in `features/databases`, 10 in `features/backups`, and one each in `features/healthcheck`, `features/restores` and `features/verification`.
- The transfer dialogs call the two notifier usage routes.

**Rules in force.**
- `docs/architecture.md` fixes:
  - the registry with a provider-owned table and the composite key `(id, type)`;
  - `CHECK` on enums and the single-column `UPDATE`;
  - decryption where a secret is used;
  - events published in the publisher's transaction, with external effects after the commit;
  - `RESTRICT` between modules;
  - `text[]` for lists;
  - a row that continues another row takes its ID.
- The api-design skill fixes paths, cursor pages, the `oneOf` settings with `type`, `writeOnly` secrets kept on `PUT`, booleans with `is`, and the Problem codes.
- The change answers to the root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`.

## Goals / Non-Goals

**Goals:**

- `databases` is a module with its operations in the contract, and adding a database type takes an engine, a provider package, a migration, a registry entry, a contract schema and its conversion in `httpapi`.
- The defects of Context are closed by construction: create carries no ID, stored secrets reach only hosts chosen by an editor, a test writes nothing, deletion and attachment are each one transaction.
- `databases` no longer imports `backups`, and no other package reads its tables.

**Non-Goals:**

- Moving the database transfer into the module. It carries storage and notifier choices of the backup configs, and moves with them in milestone 9.
- `RESTRICT` on the references from backups, health checks and verification. Each switches with its module, as `storages-file-store` did for storages.
- A provider framework shared with `notifiers` and `storages`.

## Decisions

### 1. The module's packages and its contract

```
internal/modules/databases/
  module.go
  internal/
    userdatabase/  Database, Type, HealthStatus, the events; create, list, read, update, copy, delete,
                   status writes; the parent and attachment repositories
    providers/     Provider, ConnectedDatabase, the registry
      postgresqllogical/ postgresqlphysical/ mysql/ mariadb/ mongodb/
    limiteduser/   checking and creating limited users through the registry
    connection/    connection tests of saved and unsaved settings, the liveness probe
    httpapi/       the generated strict server and its handlers
    module/        the Module implementation that ties the services together
```

The module's main internal package is `userdatabase`, so its name does not clash with `lib/database`, which keeps its own name in this document: `database.Transactor`, `database.Conn`, `database.AfterCommit` and `database.NameCursor` are from `lib/database`.

`module.go` aliases the types of the internal packages, as `storages` does:

```go
type (
	Type         = userdatabase.Type         // TypePostgresqlLogical, TypePostgresqlPhysical, TypeMysql, ...
	HealthStatus = userdatabase.HealthStatus // HealthStatusAvailable, HealthStatusUnavailable
	// Database is a database as other modules see it, without its settings.
	Database = userdatabase.Database // ID, WorkspaceID, Name, Type, NotifierIDs, LastBackupTime,
	                             // LastBackupErrorMessage, HealthStatus
	PostgresqlLogicalConnectionSettings  = providers.PostgresqlLogicalConnectionSettings
	PostgresqlPhysicalConnectionSettings = providers.PostgresqlPhysicalConnectionSettings
	MysqlConnectionSettings              = providers.MysqlConnectionSettings
	MariadbConnectionSettings            = providers.MariadbConnectionSettings
	MongodbConnectionSettings            = providers.MongodbConnectionSettings
	TelemetryEntry           = userdatabase.TelemetryEntry
	DatabaseCreated          = userdatabase.Created
	DatabaseCopied           = userdatabase.Copied
	DatabaseBackupTypeChanged = userdatabase.BackupTypeChanged
	DatabaseDeleting         = userdatabase.Deleting
	// Handler types of the four events, as StorageTransferringHandler in storages.
)

type Module interface {
	RegisterRoutes(routes gin.IRouter, hooks *apierror.Hooks)

	// GetDatabases leaves out the IDs that match no database.
	GetDatabases(ctx context.Context, databaseIDs []uuid.UUID) ([]Database, error)
	ListAllDatabases(ctx context.Context) ([]Database, error)
	GetTelemetry(ctx context.Context) ([]TelemetryEntry, error)

	// The getters hand out the decrypted settings of one database of a known type. A database of
	// another type is an error, so a caller never switches on the type.
	GetPostgresqlLogicalConnectionSettings(ctx context.Context, databaseID uuid.UUID) (PostgresqlLogicalConnectionSettings, error)
	GetPostgresqlPhysicalConnectionSettings(ctx context.Context, databaseID uuid.UUID) (PostgresqlPhysicalConnectionSettings, error)
	GetMysqlConnectionSettings(ctx context.Context, databaseID uuid.UUID) (MysqlConnectionSettings, error)
	GetMariadbConnectionSettings(ctx context.Context, databaseID uuid.UUID) (MariadbConnectionSettings, error)
	GetMongodbConnectionSettings(ctx context.Context, databaseID uuid.UUID) (MongodbConnectionSettings, error)

	// ProbeLiveness is the trusted path of scheduled health checks; it has no actor.
	ProbeLiveness(ctx context.Context, databaseID uuid.UUID) error

	// The status writes change one column each, and refuse to run on a missing database.
	RecordBackupSuccess(ctx context.Context, databaseID uuid.UUID, backupTime time.Time) error
	RecordBackupFailure(ctx context.Context, databaseID uuid.UUID, message string) error
	SetHealthStatus(ctx context.Context, databaseID uuid.UUID, status *HealthStatus) error

	// DeleteDatabase is the deletion of deleteDatabase. It serves the old test helpers, which delete
	// their databases through the subscribed handlers.
	DeleteDatabase(ctx context.Context, actor users.Actor, databaseID uuid.UUID) error

	// TransferDatabaseToWorkspace and ReplaceDatabaseNotifiers serve the database transfer of the old
	// backup configs, which checks access itself, until milestone 9.
	TransferDatabaseToWorkspace(ctx context.Context, databaseID, workspaceID uuid.UUID) error
	ReplaceDatabaseNotifiers(ctx context.Context, databaseID uuid.UUID, notifierIDs []uuid.UUID) error

	OnWorkspaceDeleting(ctx context.Context, event workspaces.WorkspaceDeleting) error
	OnNotifierTransferring(ctx context.Context, event notifiers.NotifierTransferring) error

	SubscribeToDatabaseCreated(handler DatabaseCreatedHandler)
	SubscribeToDatabaseCopied(handler DatabaseCopiedHandler)
	SubscribeToDatabaseBackupTypeChanged(handler DatabaseBackupTypeChangedHandler)
	SubscribeToDatabaseDeleting(handler DatabaseDeletingHandler)
}

type Deps struct {
	DB             *gorm.DB
	Transactor     *database.Transactor
	Logger         *slog.Logger
	AuditLog       *audit.Log
	FieldEncryptor *crypto.FieldEncryptor
	Clock          clock.Source
	Users          users.Module
	Workspaces     workspaces.Module
	Notifiers      notifiers.Module
}
```

Each `*ConnectionSettings` type carries the database, the engine settings with secrets in clear text, and the type's own options:
- `PostgresqlLogicalConnectionSettings`: the version, the dump scope (`IncludeSchemas`, `ExcludeTables`, `ExcludeTableData`), `CpuCount` and `IsSkipUserMappings`;
- `PostgresqlPhysicalConnectionSettings`: the version, the system identifier, the WAL segment size, the replication slot name, the backup slot name and the backup type. The first migration adds `backup_slot_name` to the physical table and fills it with the name the old code derived from the database ID, so the deletion and the copy no longer derive it;
- `MysqlConnectionSettings` and `MariadbConnectionSettings`: the version, the excluded tables, the privileges and their engine options;
- `MongodbConnectionSettings`: the version, the excluded collections and `CpuCount`.

The caller opens the engine's tunnel and calls the engine itself.

*Rejected: one `GetConnectionSettings` with a pointer per type.* Every caller would test which pointer is set, which is the `switch` the architecture keeps inside the family. Each caller already knows its type: the backup use cases are per engine, and the physical code only handles physical databases.

*Rejected: return settings without secrets and decrypt in the caller.* The callers would need the `FieldEncryptor` and the knowledge of which fields are secret, and the next field would be decrypted in one place and forgotten in another.

### 2. A provider per type, which calls its engine directly

```go
type Provider interface {
	GetType() userdatabase.Type
	Validate(settings Settings) error
	// MergeStoredSettings fills the secrets the incoming settings left out from the stored ones.
	MergeStoredSettings(merge SettingsMerge) (Settings, error)
	// PrepareForCreate sets what the provider owns from the first save on, such as the replication
	// slot name of a physical database.
	PrepareForCreate(settings Settings) Settings
	// Connect decrypts the settings, opens the engine's tunnel and builds the engine source.
	Connect(ctx context.Context, settings Settings) (ConnectedDatabase, error)
	EncryptSecrets(settings Settings) (Settings, error)
	CopyForNewDatabase(settings Settings) Settings
	GetRepository() SettingsRepository
}

// ConnectedDatabase is an open connection to one user database through its engine.
type ConnectedDatabase interface {
	TestConnection(ctx context.Context, logger *slog.Logger) error
	// Detect returns what the server reports, for the provider to keep in its settings: the
	// version, the privileges of MySQL and MariaDB, and for physical PostgreSQL the system
	// identifier and the WAL segment size.
	Detect(ctx context.Context, logger *slog.Logger) (Settings, error)
	ProbeLiveness(ctx context.Context, logger *slog.Logger) error
	CheckLimitedUser(ctx context.Context, logger *slog.Logger) (LimitedUserCheck, error)
	CreateLimitedUser(ctx context.Context, logger *slog.Logger) (LimitedUser, error)
	Close() error
}

type SettingsRepository interface {
	Upsert(ctx context.Context, databaseID uuid.UUID, settings Settings) error
	ListByDatabaseIDs(ctx context.Context, databaseIDs []uuid.UUID) (map[uuid.UUID]Settings, error)
}
```

Each provider package holds:
- its settings struct with plain fields and its validation, moved from the old model's `Validate`;
- its GORM row type for `databases_<type>` and its repository through `database.Conn` of `lib/database`;
- the conversion to the engine's settings with decryption;
- the `ConnectedDatabase` over the engine source of `engines-layer`.

Every type can check and create a limited user. Physical PostgreSQL does it with the replication user, so a limited user of a physical database is a replication user and the answer carries `isForcedWalRotationAvailable`.

A provider calls the concrete source of its own engine, so `databases` declares no interface per engine capability. `docs/architecture.md` "Engines and providers" says that an engine *exposes* small capability interfaces. Task 7.1 rewords this: an engine's sources offer small sets of methods, and a consumer that serves several engines through one shape declares the interface it needs, as `ConnectedDatabase` does here. The verification agent of milestone 12 will declare its own restore interface the same way.

*Rejected: declare the capability interfaces in `internal/engines`.* `engines` would then hold interfaces that only `databases` implements against, and every change of a consumer's need would touch the engine layer.

### 3. The parent repository and transactions

`userdatabase.Service` opens one transaction per write, and the repositories run every query through `database.Conn`.
- **List.** A list loads one page of `databases` rows ordered by `name, id` with `database.NameCursor`, reads the attachments of the page in one query, and calls `ListByDatabaseIDs` once per type that occurs on the page.
- **Status writes.** `RecordBackupSuccess`, `RecordBackupFailure` and `SetHealthStatus` are single `UPDATE`s that return `false` for a missing row. The service turns `false` into an error, as the `database-status-updates` requirement asks.
- **Order on the screen.** The list was ordered by health first (`repository.go:157`). The frontend loads every page and keeps showing unavailable databases first by sorting them itself.

*Rejected: a cursor over `(health_status, name, id)`.* Health changes on its own, so a database could move between pages while a client pages through them, and the screen sorts the whole list anyway.

### 4. Create, change and copy

**Create.**
1. Outside any transaction: check `ActionEditContent` through `Authorize`, validate the settings, connect through the provider (tunnel included) and detect.
2. Inside one transaction:
   - insert the parent row with an ID from `gen_random_uuid()`, the settings row and the attachments;
   - check the attached notifiers through `Notifiers.GetNotifiers`;
   - publish `DatabaseCreated` and write the audit event.

The request schema has no `id`, and the repository's create never updates.

**Change.**
1. Outside the transaction: load and authorize, refuse settings of another type with 400, merge the stored secrets, validate, connect and detect.
2. In the transaction:
   - read the row `FOR UPDATE`;
   - write the parent's name, the settings row and the attachments;
   - check the notifiers;
   - publish `DatabaseBackupTypeChanged` when the physical backup type changed;
   - write the audit event with today's messages, "updated" or "updated and renamed".

**Copy.** In one transaction:
1. read the original;
2. insert the copy with the name `<name> (Copy)`, the provider's `CopyForNewDatabase` settings (a new replication slot name, no system identifier) and the same attachments;
3. publish `DatabaseCreated` and then `DatabaseCopied`, as the old order did (`service.go:543-547`);
4. write the audit event.

A copy does not test its connection, as today. The physical config's unregistered `OnDatabaseCopied` is deleted.

**Notifier check.** The check runs after the attachment rows are inserted. Inserting a row that references `notifiers` takes `FOR KEY SHARE` on the notifier's row. That lock conflicts with the `FOR UPDATE` that `transferNotifier` takes before it counts the databases (`modules/notifiers/internal/notifier/notifier_repository.go:79`).
- If the transfer took its lock first, the insert waits for its commit, and the check then sees the notifier in the other workspace and refuses.
- If the insert came first, the transfer waits, then counts the new attachment and answers `in_use`.

The event handlers run in the transaction and return errors, so a failed health check config refuses the creation instead of being lost (today `OnDatabaseCreated` returns nothing). The physical config's handler of `DatabaseBackupTypeChanged` updates the config in the transaction and stops a WAL streamer through an after-commit hook (decision 5), instead of before the write.

*Rejected: keep the connection test inside the transaction.* A slow server or SSH handshake would hold the row lock and a pooled connection for up to the engine's timeout.

### 5. Deleting a database is one transaction with after-commit hooks

`lib/database` gets `AfterCommit(ctx, fn func(ctx context.Context))`. The outermost `Transactor.Transaction` keeps a list of hooks in the context. A nested transaction appends to the same list, and the outermost one runs the hooks in order after its commit, with a context detached from the transaction. A rollback drops them. Called outside a transaction, `AfterCommit` refuses with an error, so a hook is never run on work that may still be undone.

`deleteDatabase` runs in one transaction:
1. read the row `FOR UPDATE`, answering 404 when it is missing, and authorize;
2. publish `DatabaseDeleting{DatabaseID}`;
3. delete the attachments, the settings row and the parent;
4. write the audit event.

`internal/app/wiring.go` subscribes the handlers of the old backup code in today's order:
1. **Logical backups.** Refuse with the Problem `in_use` while a logical backup of the database is in progress, as `deleteDbBackups` refuses today (`features/backups/backups/services/logical_service.go:462-464`). Otherwise request the deletion of every backup file through `RequestFileDeletions`, then delete the backup rows and the config.
2. **Physical cancellation.** Delete the streamer row, and register a hook that cancels a running backup and stops `pg_receivewal`.
3. **Physical backups.** Request the deletion of every full, incremental and WAL file, and delete the rows.
4. **Slot cleanup.** Read the physical connection settings while the rows still exist, and register a hook that drops the per-backup slots and the WAL slot.

The hooks run in that order, but the cancellation reaches the streamer through pubsub, possibly on another instance, so the slot cleanup cannot wait for it. The order is not what keeps the slot gone: the slot cleanup terminates a consumer still attached to the slot, and the WAL supervisor checks that its database exists after every slot creation and drops the slot when the database is gone. The deletion drops the slot after it commits, so a slot a late streamer created before that drop goes with it, and one created after it is dropped by the streamer itself. The health check and verification rows still go by `CASCADE`.

The refusal rolls back the whole deletion, and `deleteDatabase` answers 409 `in_use`. A physical backup that is still running when the deletion commits is cancelled by the hook. If it reaches its publication first, the publication fails on the missing database, and its files stay in the pending deletions for the deletion worker.

*Rejected: a second event `DatabaseDeleted` after the commit.* Its handlers would need the deleted connection settings, which are gone by then, so the event would have to carry decrypted secrets through memory to every subscriber. A hook keeps what it needs in its own closure.

### 6. The last backup time

`RecordBackupSuccess` runs `UPDATE databases SET last_backup_time = GREATEST(COALESCE(last_backup_time, $1), $1), last_backup_error_message = NULL WHERE id = $2`. A missing row returns `false`, and an older time leaves the newer one.

Every backup kind calls it in the transaction that publishes the result, with the time the read uses today:
- logical backups in the transaction of `publishBackup` (`backuping/logical/backuper.go:207`) with the completion time, instead of after it (`:217-219`);
- `backuping/physical/backuper.go:608` for full and incremental backups, with the backup's `created_at`, its start;
- `usecases/physical/postgresql/wal_upload.go:509` for a WAL segment, with its `received_at`, through a narrow interface of that package, as its `FileStore` is.

`RecordBackupFailure` stays where logical backups record a failure today (`backuping/logical/backuper.go:154,415`), after the failure is saved, because a failure publishes no file.

`fillPhysicalLastBackupTimes` and `GetLastBackupTimesByDatabaseIDs` are deleted, and telemetry reads `GetTelemetry`. The migration sets `last_backup_time` of each physical database to the newest of its completed full and incremental backups and committed WAL segments, the same `MAX` the read used. A WAL segment adds one `UPDATE` of one row next to the insert of the segment's row. The column has no index, so PostgreSQL updates the row in place without touching an index.

*Rejected: let `databases` call an interface that `backups` implements.* The lower module would call the higher one at runtime, and the cycle would only move into `internal/app`.

### 7. Ten operations

| Operation | Method and path | Answers |
|---|---|---|
| `listDatabases` | `GET /databases?workspaceId&notifierId&limit&after` | 200 `DatabasePage`, 404 |
| `createDatabase` | `POST /databases` | 201 `Database`, 400, 403, 404 |
| `getDatabase` | `GET /databases/{databaseId}` | 200 `Database`, 404 |
| `updateDatabase` | `PUT /databases/{databaseId}` | 200 `Database`, 400, 403, 404 |
| `deleteDatabase` | `DELETE /databases/{databaseId}` | 204, 403, 404 |
| `copyDatabase` | `POST /databases/{databaseId}/copy` | 201 `Database`, 403, 404 |
| `testDatabase` | `POST /databases/{databaseId}/test` | 200 `DatabaseTestResult`, 404 |
| `testDatabaseSettings` | `POST /database-tests` | 200 `DatabaseTestResult`, 400, 403, 404 |
| `checkLimitedUser` | `POST /limited-user-checks` | 200 `LimitedUserCheck`, 400, 403, 404 |
| `createLimitedUser` | `POST /limited-users` | 201 `LimitedUser`, 400, 403, 404 |

`Database` has `id`, `workspaceId`, `name`, `settings`, `notifierIds`, and the read-only `lastBackupTime`, `lastBackupErrorMessage` and `healthStatus`. `DatabaseSettings` is a `oneOf` of `PostgresqlLogicalSettings`, `PostgresqlPhysicalSettings`, `MysqlSettings`, `MariadbSettings` and `MongodbSettings` with the discriminator `type`.
- **Fields.** Each schema lists its fields in `camelCase` with the names the frontend uses today. The SSH tunnel is a shared `SshTunnelSettings`.
- **Secrets.** `password`, `sslClientKey`, and the tunnel's `password`, `privateKey` and `privateKeyPassphrase` are `writeOnly` and optional.
- **Detected values.** `version`, `systemIdentifier`, `walSegmentSizeBytes` and the `privileges` of MySQL and MariaDB are `readOnly`.
- **Enums.** The SSL mode, the SSH authentication type, the physical backup type and every version are enums. The SSL mode and the versions keep the names the engine uses, such as `verify-full` and `8.0`, and carry `x-engine-values: true`, which exempts them from the contract lint's UPPER_SNAKE_CASE rule. Users read these names in the engine's documentation, so a client would only map them back.

`DatabaseCreate` has `workspaceId`, `name`, `settings` and `notifierIds`, and `DatabaseUpdate` has `name`, `settings` and `notifierIds`.

The three operations on unsaved settings take `DatabaseSettingsTarget` with `workspaceId`, an optional `databaseId` and `settings`. They are resources of their own for the reason `notifiers-module` decision 7 gives for `/notifier-tests`.

`DatabaseTestResult` is `{isConnected, reason, error}`. `reason` is the enum `DatabaseConnectionFailure` with the values of `pgshared.ConnectionErrorCode`, which the form explains. The engine codes are UPPER_SNAKE_CASE, such as `PG_HBA_NO_ENTRY`, so the handler passes them through unchanged. A failed test is an answer to the question asked, not an error.

Create and update answer a failed connection with 400 and the new code `database_connection_failed`, with the server's message in `detail`. That message comes from the user's own server, as the old 400 body did. A server error never contains our secrets, because the engine sends the password through `.pgpass` or the driver's authentication, never in a query.

`notifierId` on the list replaces the two notifier usage routes. The notifier transfer dialog asks whether the list of the notifier's workspace filtered by the notifier is empty, and the database transfer dialog counts the same list.

### 8. Access goes through `Authorize`

Every operation addressed by a database ID loads the database first and answers 404 `not_found` when it is missing. A `not_found` from `Authorize` becomes the database's own `not_found`, as in `storages`.

| Operation | Action |
|---|---|
| reads and `testDatabase` | `ActionView` |
| create, update, copy, delete | `ActionEditContent` |
| `testDatabaseSettings`, `checkLimitedUser`, `createLimitedUser` | `ActionEditContent` in `workspaceId` |

For the three operations on unsaved settings, the `databaseId` is checked first: a `databaseId` that names no database or a database of another workspace than `workspaceId` answers 404 before the workspace is authorized and before anything connects. Then `ActionEditContent` in `workspaceId` is authorized, so a viewer gets 403. The `databases` requirement "Only a user who may change databases sends stored secrets elsewhere" explains why a viewer gets 403. A viewer cannot save settings, so checking or testing unsaved ones gains a viewer nothing. A viewer who opens the edit form sees no limited-user hint, which is the only loss.

*Rejected: keep `ActionView` and refuse only a `databaseId` with a changed host.* The host, the port, the TLS settings and the SSH bastion all decide where a secret goes. A field missed by one type would reopen the leak. `storages-module` decision 7 rejected the same option.

### 9. The old code reaches the module through a bridge

`features/databases` keeps two things:
- `Install(module)` and `GetModule()`, with a module built from the old access points in a `testhelpers` build, as `storages_bridge` does;
- `GetAccess()`, whose `GetDatabaseByID` and `GetAuthorizedDatabase` serve the old controllers that look a database up by ID and check the user's access to its workspace, until those controllers move into modules;
- `SetupDependencies` for the old packages that still wire themselves;
- the test helpers of decision 11, behind the `testhelpers` tag.

**Old code that switches:**
- Backups, restores, health checks and verification replace `*databases.Database` with `databases.Database` of `module.go`. Where they connect, they call the typed getter of their type and open the engine's tunnel. The physical config drops its GORM association and its `Preload`, and reads the connection settings through `GetPostgresqlPhysicalConnectionSettings`.
- The restore request gets target types of its own in `restores/core`, one `*TargetDatabase` per type, with today's JSON field names, including the restore options. The restore service builds engine settings from them to open a tunnel and detect the version. `restores_core.SourceDatabase` holds the connection settings of the backed-up database, read through the typed getter, and `TargetConnection` the engine settings of the target, so the use cases read neither the module nor the old models.
- Health checks reach the module through a `HealthcheckedDatabases` interface of `healthcheck/attempt` with an adapter over `module.go`, so their tests replace it with a mock.
- Verification hands the agent an `AssignedDatabase` of its own with the type and, for logical PostgreSQL, the version, instead of the old model.
- Telemetry calls `GetTelemetry`. `TelemetryEntry` carries the database ID, the type, the health status, the last backup time and the provider's facts: the version, whether an SSH tunnel is on and the physical backup type. `backup_slot.go` calls `ListAllDatabases` and the physical getter.
- The physical config reads the physical backup type through `GetPostgresqlPhysicalConnectionSettings`.
- The database transfer of the old backup configs calls `TransferDatabaseToWorkspace` and `ReplaceDatabaseNotifiers`. Both run in their own transaction. `ReplaceDatabaseNotifiers` runs the notifier check of decision 4. `TransferDatabaseToWorkspace` writes the audit events of today's transfer.
- The four listener interfaces and their setters go, together with `SetupDependencies` calls that only registered them. `internal/app/wiring.go` subscribes the old handlers to the module's events in today's order.

The rewrite plan, which lives only in the main checkout at `/home/rostislav/development/project/databasus/temp/refactor/plan.md`, gets these relaxation rows:
- the old features reach `databases` through `databases_bridge`, and the old backup configs transfer a database through `TransferDatabaseToWorkspace` and `ReplaceDatabaseNotifiers` without a transaction around the whole transfer, until milestone 9;
- the old restore request keeps its own engine-shaped types until milestone 9.

The row about `database_notifiers` and `DatabaseNotifier` is crossed out.

*Rejected: move the transfer into the module now.* The request carries the target storage and whether to move it, which belongs to the backup configs. The module would need an event whose handler decides about storages in the middle of a database transfer, and milestone 9 would rebuild it once more.

### 10. The schema

Two goose files. The first lands while the old models still read the tables, so it keeps each provider table's own `id`. The second lands after the old models are deleted.

**The first migration** (task 2.1):

1. **Renames.**
   - `postgresql_logical_databases`, `postgresql_physical_databases`, `mysql_databases`, `mariadb_databases` and `mongodb_databases` become `databases_postgresql_logical`, `databases_postgresql_physical`, `databases_mysql`, `databases_mariadb` and `databases_mongodb`;
   - `database_notifiers` becomes `databases_notifiers`;
   - constraints and indexes take the new names.
2. **Repairs, before any constraint.**
   - A provider row whose database has another type is deleted.
   - A logical PostgreSQL row whose `database_id` is NULL or names no database is deleted.
   - Of two provider rows of one database, the one with the lowest `id` stays and the others are deleted. Such rows can only come from the takeover defect; GORM's preload read one of them at random.
   - A database of an unknown type that nothing references is deleted. One that backups or a backup config reference stops the migration with an error that names it. No release has written a type outside the five, and deleting it would leave its backup files without an owner, because SQL cannot ask a storage to delete them.
   - A database without a row in the provider table of its type is deleted when nothing references it. When something does, it stays, and the module answers a read of it with 500 `internal`, as `storages-module` decided for the same case. No constraint requires the provider row, and no code produces such a row today.
   - A database without a workspace takes the oldest workspace when logical or physical backups or a backup config reference it. Otherwise it is deleted, and its children go by `CASCADE`.
3. **`databases`.**
   - `ck_databases_type` over the five values and `ck_databases_health_status` over `AVAILABLE` and `UNAVAILABLE` with NULL allowed;
   - `workspace_id NOT NULL`;
   - `uq_databases_id_type UNIQUE (id, type)`;
   - `idx_databases_workspace_id_name_id` replaces `idx_databases_workspace_id`.
4. **Every provider table.**
   - `database_id` becomes `NOT NULL` and `UNIQUE`, and `id` gets `DEFAULT gen_random_uuid()`, so the module's repositories never write it.
   - Gets `type text NOT NULL DEFAULT '<TYPE>'` with `CHECK (type = '<TYPE>')`, and `FOREIGN KEY (database_id, type) REFERENCES databases (id, type) ON DELETE CASCADE`. The logical PostgreSQL table gets this as its first foreign key.
5. **Enum checks.** `ssl_mode`, `ssh_auth_type` of every table, and the physical `backup_type`.
6. **Lists.** Each list column becomes `text[] NOT NULL DEFAULT '{}'`. Its values are split on commas, newlines, carriage returns and tabs, trimmed, and stripped of empty entries and later duplicates, in order. This is the same rule `namelist` applied when it read them. The old models read the arrays in the same task, so the old code keeps working until task 3.1 deletes it.
7. **Physical last backup time.** `last_backup_time` of every physical database is set from its backups as decision 6 says.

**The second migration** (task 3.2) drops each provider table's own `id` and makes `database_id` the primary key, as "Data" in `docs/architecture.md` says for a row that continues another row.

The secret columns keep their `enc:` values.

*Rejected: stop the migration on every row it cannot repair.* An instance would not start over a database that cannot work anyway. The migration stops only on a referenced database of an unknown type, which no release produces and which it cannot delete without stranding files. The other rules follow the ones `storages-module` used.

*Rejected: one migration that also drops `id`.* The old models read the provider tables until task 3.1, and their primary key is `id`. Switching them to `database_id` would rewrite their GORM associations one task before they are deleted.

### 11. Test helpers and tests

**Old test helpers.**
- `databases_bridge.CreateTestDatabase(workspaceID, notifier)` inserts a logical PostgreSQL database that points at the shared test server, its settings row and its attachment with SQL, and returns `databases.Database`.
- `CreateTestPhysicalPostgresDatabase*`, `CreateTestMariadbDatabase` and `CreateTestMongodbDatabase` do the same for their types.
- `RemoveTestDatabase(ctx, database)` deletes through `Module.DeleteDatabase` as the workspace owner with the old handlers subscribed. It does not retry, so a deadlock between a deletion and a backup writer fails the test instead of hiding. In a test build without `internal/app`, `backups_services.SubscribeToDatabaseEventsWithoutApp` and `backuping_physical.SubscribeToDatabaseDeletingWithoutApp` subscribe the handlers that `wiring.go` subscribes in the application.
- `GetTestRoutes()` mounts the module's routes on an old test router, and `CreateTestDatabaseViaAPI`, `GetTestDatabaseViaAPI` and `UpdateTestDatabaseViaAPI` call them.
- The callers take `.ID` and the fields of the new type. The places that read engine fields read the connection settings through the typed getter.

**`apptest`.**
- `apptest.CreateDatabase(tb, session, workspaceID, settings)` goes through `createDatabase` against a container.
- `apptest.InsertDatabase` and `AttachNotifierToDatabase` keep writing SQL into the renamed tables for tests that need a database without a server.

**Module tests.** The tests in `internal/modules/databases` use `apptest` and the contract-checking client, one file per concept: lifecycle, secrets, the fixed type, the list, connection tests, unsaved settings, limited users, copying, deletion, notifiers, status writes. Tests that create a database start a PostgreSQL, MySQL, MariaDB or MongoDB container per package through `util/testing/containers`. The SSH tunnel tests take the tunnel settings of a bastion container from `util/testing/bastion`, which sits outside `containers` because `lib/sshtunnel` tests import `containers`.

**Ported tests.** The tests of the old `controller_test.go` that pin rules the module keeps are ported onto the new operations from the file as it stands before this change: the SSH tunnel cases, the secret lifecycle of all types, the exclusion lists and the physical refusal without WAL summarization.

**New tests.**
- The deletion: a handler that fails rolls everything back, a running WAL streamer is cancelled and its slots are dropped, and a streamer that starts after the deletion leaves no slot.
- The notifier check: `apptest.App.HoldNotifierTransfer(tb, notifierID, targetWorkspaceID)` opens a transaction that locks the notifier's row `FOR UPDATE` and moves it to the target workspace, as `transferNotifier` does, and returns a commit function. While it holds the lock, `createDatabase` and `updateDatabase` with that notifier run through the API and wait; after the commit they answer 400 `invalid_request`. `apptest.BeginNotifierAttachment` writes SQL and passes the module by, so it keeps testing only the transfer side.
- The takeover: a create with another workspace's database ID leaves that database untouched.
- `lib/database`: `AfterCommit` runs after the commit, does not run after a rollback, runs once for a nested transaction, and refuses outside a transaction.

### 12. Terms

`docs/glossary.md` gets the terms this change brings into `module.go`, the contract and `docs/architecture.md`:
- "Limited user" (`limiteduser`, `LimitedUser`, `checkLimitedUser`, `createLimitedUser`): a read-only user, or for physical PostgreSQL a replication-only user, that Databasus creates on the user's server for its backups;
- "Connection settings" (`*ConnectionSettings` of `databases`): the decrypted settings of one database with its type's options, which the module hands to the code that connects;
- "After-commit hook" (`database.AfterCommit`), in the Code section: work outside the database that runs only after the transaction that registered it commits;
- "Provider" extends to database types: one implementation of a database, storage or notifier type, with its own settings table.

### 13. The frontend

`entity/databases/api` is rewritten on the generated client, with `databaseApi.listAllDatabases(workspaceId, notifierId?)` following `nextCursor` with `limit` 100. The hand-written models are deleted, except:
- the label keys, the logo lookup, `isPostgresType` and the connection string parsers, which move onto the generated types;
- the engine types that the restore form sends, which move to `entity/restores`.

`DatabaseType` becomes a `const` object over the generated enum, as `StorageType` did.

`features/databases` changes as follows:
- the create and edit flows switch on `settings.type`;
- secrets are empty in the form of a saved database and are left out of the request through `toOptionalValue`;
- the test button calls `testDatabaseSettings` with the database's ID when editing and shows `reason` and `error` of a failed result;
- the limited-user hint and its creation call `checkLimitedUser` and `createLimitedUser`, and are shown only to users who may edit;
- notifiers are shown from `notifierIds` and the workspace's notifier list;
- the list sorts unavailable databases first.

The database parts of `features/backups`, `features/healthcheck`, `features/restores`, `features/verification`, `features/notifiers` and `features/workspaces` read the generated `Database`. The transfer dialogs use `listAllDatabases` with `notifierId`. The new code `database_connection_failed` gets its text in the six dictionaries.

Screen tests on `mockApi`:
- the edit form of a saved database sends no password when it is left empty;
- a failed test shows its reason;
- a viewer sees no limited-user hint;
- unavailable databases are listed first.

### 14. The upgrade test

The previous release writes a MySQL, a MariaDB and a MongoDB database next to the PostgreSQL one, each with a password and an exclusion list. Each server runs in a container the test starts, as `source_database_test.go` does for PostgreSQL. The PostgreSQL database is still the one that is backed up and restored, so its lists stay out of the rows the restore compares: the schema list names the schema of the seeded rows, and the table and table data exclusion lists name an empty table the test creates beside them. The other three databases are not backed up, and their lists name an object without rows. TLS stays off, because the source servers serve no TLS and the previous release tests every connection it saves. The current checks read the databases through `listDatabases` and compare every non-secret setting and every list entry. A physical database is left to the migration test of task 2.1, because the previous release refuses one without a replication source, and a replication source adds a second kind of container to a test that runs on every pull request.

## Risks / Trade-offs

- [About 125 Go files and 40 frontend files change] → Most of them change the type they import and read `.ID` or a field of the new type. The compiler and `pnpm typecheck` find every one.
- [Viewers lose the limited-user hint] → The proposal marks it BREAKING. The hint is about changing the database's user, which a viewer cannot save.
- [A WAL segment now writes the database row] → It is one in-place `UPDATE` per archived segment of a database, next to the segment's own insert. It takes a row lock that a user's save of the same database waits for only for the length of that small transaction.
- [Slots are dropped after the commit] → When dropping fails, the slot stays on the source server, as it does today, where the cleanup already only logs a failure. The database is gone either way, and the log names the slot.
- [Duplicate provider rows are deleted by the lowest ID] → Only the takeover defect produces them, and the old read picked one at random.
- [The migration deletes databases without a workspace that nothing references] → They date from a restore flow before workspaces, and no screen has shown them since.

## Migration Plan

1. The migration runs at startup before the server accepts requests. The database tables are small. The list conversion and the backfill of the physical last backup time each read their tables once.
2. The upgrade is one-way, as `docs/architecture.md` says for every migration.
3. The upgrade test writes a database of every logical type through the previous release and checks them through `listDatabases` after the upgrade. The migration test of task 2.1 covers every repair, every list conversion, the physical backfill and that every secret column keeps its bytes, and task 3.2 extends it to the second migration.
