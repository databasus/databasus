## Context

Why the change is needed is in proposal.md, section Why. This document covers only what shapes the approach. It assumes `notifiers-module` and `storages-file-store` have landed. From `notifiers-module` it takes the module and bridge pattern, the `in_use` code, `database.NameCursor`, `database.IsForeignKeyViolation` and `database.WithoutTransaction`. From `storages-file-store` it takes `RESTRICT` on every reference from backup tables, the deletion that removes the storage row before its files, `ConfirmFileWrites` and `RequestFileDeletions` with the transaction in the context, `GetStorageReader`, and the `Transactor` of `internal/storage`.

**Operations.** `StorageController` (`internal/features/storages/controller.go:20-28`) serves seven routes:
- `POST /storages` creates a storage, or updates one when the body carries an ID. It answers with the request body. On create that body has just been through `EncryptSensitiveData` (`service.go:172`), so the response carries the encrypted secrets.
- `GET /storages?workspace_id=` returns every storage of a workspace as an array.
- `GET /storages/:id`, `DELETE /storages/:id`, `POST /storages/:id/test`, `POST /storages/:id/transfer` and `POST /storages/direct-test` complete the set.

Errors go out as `{"error": err.Error()}`, with 403 for the permission and rclone errors and 400 for everything else. A failed test is a 400 as well. Both tests need only `ActionView`. `TestStorageConnectionDirect` (`service.go:330`) merges the request into the stored storage when the request carries an ID. So a viewer can send a host of their own with the ID of a saved NAS, FTP or SFTP storage and receive its stored password. The `TestStorageConnection` of a saved storage (`service.go:296`) re-saves the whole storage after a success. This can write a refreshed Google Drive token to `token_json` without encryption, because the save skips `EncryptSensitiveData`.

**Model and repository.** `Storage` (`model.go:22`) has one pointer field per provider with a GORM `foreignKey` association, and `getSpecificStorage` (`:157`) switches on the type and panics on an unknown one. `StorageRepository.Save` (`repository.go:14`) runs two `switch` statements and an `Omit` of eight associations. `FindByID` and `FindByWorkspaceID` run eight `Preload` calls, and `Delete` runs one more `switch`. `Storage.Update` (`model.go:101`) copies the incoming type and assigns the incoming provider struct when the stored one is nil. A type change therefore works, and the row of the old provider stays behind. The edit form offers a type selector for a saved storage (`frontend/src/features/storages/ui/edit/EditStorageComponent.tsx:346`). `last_save_error` is set only in memory by `Storage.SaveFile` and by a failed test, and is cleared and saved by a successful test. No code stores a value in it.

**Providers.** Eight packages under `internal/features/storages/models/<name>` each define a GORM model with `SaveFile`, `GetFile`, `DeleteFile`, `Validate`, `TestConnection`, `HideSensitiveData`, `EncryptSensitiveData` and `Update`. The encryptor is a parameter.
- Secrets per provider:
  - S3: `s3_access_key`, `s3_secret_key`;
  - Google Drive: `client_secret`, `token_json`;
  - NAS and FTP: `password`;
  - Azure Blob: `connection_string`, `account_key`;
  - SFTP: `password`, `private_key`;
  - rclone: `config_content`.
- Each `Update` keeps a stored secret when the incoming one is empty. Azure also keeps an empty `account_name` that way. S3 and Azure never copy the prefix, and the S3 form disables it once saved (`EditS3StorageComponent.tsx:232`).
- Enum-like columns: `azure_blob_storages.auth_method` (`CONNECTION_STRING`, `ACCOUNT_KEY`, `models/azure_blob/model.go:52-57`) and `s3_storages.s3_storage_class` (`''` and six classes, `models/s3/enums.go:3-13`). Neither has a `CHECK`.
- Package state that survives a call:
  - session pools of NAS, FTP and SFTP (`models/{nas,ftp,sftp}/session_pool.go`), built on `connections.SessionPool` and invalidated by a fingerprint of the stored settings;
  - the rclone remote cache (`models/rclone/remote_cache.go:38`) and the `rcloneConfigMu` lock (`models/rclone/model.go:30`);
  - the two S3 transports and the bucket region cache (`models/s3/transport.go:18-54`);
  - the Azure HTTP client (`models/azure_blob/model.go:510`).
- `context.Background()` appears 20 times. Every delete carries its own 30-second deadline, because cleanup often calls with a cancelled context. The connection tests have their own deadlines, and the Google Drive test and its Drive service have none.
- `//nolint:bodyclose` at `models/google_drive/model.go:716` has no reason.
- Local storage reads `config.GetEnv().TempFolder` and `DataFolder`.
- NAS copies every chunk in a bare `go func()` (`models/nas/model.go:366`), and `files/background_write.go:32` starts the upload of a background write the same way.

**The file lifecycle.** After `storages-file-store`, `features/storages/files` holds `Store`, `WriteReceipt`, `StoredFileReference`, `StorageReader`, `BackgroundWrite`, `PendingDeletionRepository`, `DeletionWorker` and `Timings`. The worker runs as an old background loop (`internal/app/background_loops.go:32`), so `apptest` cannot run a pass of it. It finds the provider through `StorageLocator`, which `StorageService` implements in `file_locator.go`.

**Readers in the old code.**
- The backup configs carry `Storage *storages.Storage` with `gorm:"foreignKey:StorageID"` (`backups/config/logical/model.go:34`, `physical/model.go:36`) and preload it (`config/{logical,physical}/repository.go`). The frontend reads `storage.id`, `storage.name` and `storage.type` from `/backup-configs` in seven places, among them `EditLogicalBackupConfigComponent.tsx:278`, `DatabaseCardComponent.tsx:36` and `DatabaseConfigComponent.tsx:85`.
- The config save takes the storage ID from the embedded object (`takeStorageIDFromEmbeddedStorage`, `config/logical/service.go:409`) and checks that the storage is in the database's workspace with a plain read before it saves.
- The database transfer (`config/logical/service.go:205`, and its physical twin) moves the storage before the database through `TransferStorageToWorkspace` with `transferingWithDbID`.
- The routes `/backup-configs/storage/:id/is-using` and `.../databases-count` (`config/logical/controller.go:20-21`) ask `StorageService.GetStorageAttachedDatabasesIDs`, which merges the `StorageDatabaseCounter` of both config services.
- `telemetry` reads every storage for its type (`telemetry/service.go:375`).
- `internal/app` subscribes `OnWorkspaceDeleting` (`wiring.go:41`), mounts the old controller (`routes.go:73`), runs the worker loop, parses the GORM schema of `Storage` (`model_schemas.go:22`) and probes the local storage for `--test-storage` (`storage_probe.go`).

**Tests.** `internal/features/storages/controller_test.go` (1370 lines) drives the old routes:
- the role tests;
- `Test_StorageSensitiveDataLifecycle_AllTypes` (`:608`), the secrets of seven types;
- the rclone admin rule;
- the transfer permissions;
- the deletion tests that `storages-file-store` rewrote.

`model_test.go` runs every provider except Google Drive against containers, with the reuse, credential change and silent-connection tests at `:255`, `:386` and `:451`. Each provider package has unit tests of its own rules, and `s3_cleanup_test.go` pins the S3 delete.

Outside the package, 47 test files call `CreateTestStorage`, `RemoveTestStorage`, `CreateTestFlakyS3Storage` or `DrainStorageFileDeletions` 339 times. Every caller uses only `.ID`, apart from:
- `.Name` and `.S3Storage` on the flaky S3 storage (`config/logical/controller_test.go:1675`, `config/physical/controller_test.go:740`);
- provider methods called on the storage in five files (`restore_stream/writer_test.go:132`, `controllers/physical/controller_test.go:1557`, `core/physical/service/service_test.go:62`, `controllers/logical/controller_test.go:1332`, `restores/controller_test.go:715`);
- the pointer handed to a backup config's `Storage` field in about 40 places and to `EnableBackupsForTestDatabase`.

`databases.CreateTestDatabase` takes a storage it never uses (`databases/testing.go:260`, 108 call sites). The `workspaces` deletion tests create storages through `apptest.App.CreateLocalStorage` (`internal/app/apptest/old_api.go:59`) and count them through the old list (`modules/workspaces/helpers_test.go:219`). The upgrade test writes one local storage through the previous release (`e2e/upgrade/previous_release_client_test.go:159`) and reads it through `GET /storages?workspace_id=` (`current_release_checks_test.go:272`).

**Frontend.** `entity/storages` has a hand-written `storageApi` with seven methods and 15 model files. `features/storages` has 27 files, and no test touches storages. `pages/OauthStorageComponent.tsx` finishes the Google Drive authorization: it exchanges the code with Google from the browser, puts the token into `tokenJson`, tests and saves the storage. The backup config editors, the database card, the database config and the database transfer dialog read storages.

**Rules in force.** The module lands in the target layout and passes every linter and `archlint`, the goroutine rule included. `docs/architecture.md` fixes:
- the registry, the provider-owned table and repository, the batch load per type and the composite key `(id, type)`;
- `CHECK` on enums, `JSONB` only for opaque data, the single-column `UPDATE`;
- decryption where a secret is used;
- `RESTRICT` with 409 `in_use`, and goroutines only through `lib`.

The api-design skill fixes the paths, the workspace filter, cursor pages, the `oneOf` settings with `type`, `writeOnly` secrets kept on `PUT`, booleans with the `is` prefix, and the Problem codes. The change answers to the root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`.

## Goals / Non-Goals

**Goals:**

- `storages` is a module with its operations in the contract, and adding a provider takes a package, a migration, a registry entry, a contract schema and its conversion in the module's `httpapi`.
- Backup code reaches stored files only through `storages.Module`, and nothing outside the module can reach a provider.
- The old features keep their routes and answers and reach the module only through the bridge.

**Non-Goals:**

- A provider framework shared with `notifiers`. The two registries share a shape, not code. Engines in milestone 8 are the third family, and extracting common code waits until then.
- Moving the old `/backup-configs/storage/...` routes into the contract. They move with `backups` in milestone 9.
- Changing what a provider does with a file, its chunking, its retries or its deadlines.

## Decisions

### 1. The module's packages and its contract

```
internal/modules/storages/
  module.go          Module, Deps, New, ProbeLocalStorage and the aliases below
  internal/
    storage/         Storage, Type, Transfer, Transferring; create, list, read, update, delete,
                     transfer, connection tests; the parent repository
    files/           Store, WriteReceipt, StoredFileReference, StorageReader, BackgroundWrite,
                     the pending deletion repository and the deletion job
    providers/       the Provider interface, the registry, connections (session pool, fingerprint)
      local/ s3/ googledrive/ nas/ azureblob/ ftp/ sftp/ rclone/
    httpapi/         the generated strict server and its handlers
    module/          the Module implementation that ties the services together
```

The types the contract exposes are declared in the internal packages and aliased in `module.go`, as in `notifiers`:

```go
type (
	Type                       = storage.Type // TypeLocal, TypeS3, ... in package storage
	Storage                    = storage.Storage // ID, WorkspaceID, Name, Type
	StorageTransfer            = storage.Transfer
	StorageTransferring        = storage.Transferring
	StorageTransferringHandler = storage.TransferringHandler
	StoredFileReference        = files.StoredFileReference
	WriteReceipt               = files.WriteReceipt
	StorageReader              = files.StorageReader
	BackgroundWrite            = files.BackgroundWrite
	BackgroundWriteSpec        = files.BackgroundWriteSpec
	LocalFolders               = providers.LocalFolders // TempFolder, DataFolder
	FileTimings                = files.Timings
)

// ProductionFileTimings is the policy of a running instance; tests pass shorter timings.
func ProductionFileTimings() FileTimings

// LocalStorageProbe is what the local storage probe needs; it runs without Deps.
type LocalStorageProbe struct {
	Folders LocalFolders
	Logger  *slog.Logger
}

type Module interface {
	RegisterRoutes(routes gin.IRouter, hooks *apierror.Hooks)

	// WriteFile records the obligation to delete the file before the first byte leaves.
	WriteFile(ctx context.Context, reference StoredFileReference, file io.Reader) (WriteReceipt, error)
	StartBackgroundWrite(ctx context.Context, spec BackgroundWriteSpec) BackgroundWrite
	// ConfirmFileWrites and RequestFileDeletions refuse to run outside a transaction in ctx.
	ConfirmFileWrites(ctx context.Context, receipts []WriteReceipt) error
	RequestFileDeletions(ctx context.Context, references []StoredFileReference) error
	GetStorageReader(ctx context.Context, storageID uuid.UUID) (StorageReader, error)

	// GetStorages leaves out the IDs that match no storage.
	GetStorages(ctx context.Context, storageIDs []uuid.UUID) ([]Storage, error)
	// GetStorageBlockingTransfer reads the storage in the caller's transaction and makes a transfer of it
	// wait until that transaction ends, so the workspace the caller checked stays the storage's.
	GetStorageBlockingTransfer(ctx context.Context, storageID uuid.UUID) (Storage, bool, error)
	// GetStorageTypes returns the type of every storage of the instance, for telemetry.
	GetStorageTypes(ctx context.Context) ([]Type, error)
	TransferStorage(ctx context.Context, actor users.Actor, transfer StorageTransfer) error
	OnWorkspaceDeleting(ctx context.Context, event workspaces.WorkspaceDeleting) error
	SubscribeToStorageTransferring(handler StorageTransferringHandler)
	GetJobs() []background.Job
}

type Deps struct {
	DB             *gorm.DB
	Transactor     *database.Transactor
	Logger         *slog.Logger
	AuditLog       *audit.Log
	FieldEncryptor *crypto.FieldEncryptor
	Users          users.Module
	Workspaces     workspaces.Module
	Clock          clock.Source
	LocalFolders   LocalFolders
	FileTimings    FileTimings
}

// ProbeLocalStorage saves and deletes a file through local storage. Container startup runs it before
// the internal database exists, so it needs no Deps.
func ProbeLocalStorage(ctx context.Context, probe LocalStorageProbe) error
```

`BackgroundWriteSpec` names the reference, the pipe reader and the failure callback that `StartBackgroundWrite` takes today as three parameters. `FileTimings` and `ProductionFileTimings` are declared in `module.go`, because `internal/app` and `apptest` cannot import `internal/files`. `internal/app` passes `ProductionFileTimings()`, and `apptest` passes shorter timings of its own, as `TimingsForTest()` does today.

*Rejected: hand out a provider handle with `SaveFile` and `DeleteFile`.* Backup code could then bypass the pending deletion again, and the `archlint` rule of decision 10 would have to stay. The reader is the only provider capability that leaves the module, because reading cannot strand a file.

*Rejected: `OpenFile(ctx, StoredFileReference)` on `Module` instead of `GetStorageReader`.* `storages-file-store` decision 5 explains the cost for a restore stream that opens thousands of WAL segments.

### 2. A provider owns its settings, its table and its file operations

```go
type Provider interface {
	GetType() storage.Type
	Validate(settings Settings) error
	// MergeStoredSettings fills the secrets the incoming settings left out from the stored ones and
	// refuses a change of a setting fixed at creation, such as the S3 prefix.
	MergeStoredSettings(merge SettingsMerge) (Settings, error)
	SaveFile(ctx context.Context, file FileAccess, content io.Reader) error
	OpenFile(ctx context.Context, file FileAccess) (io.ReadCloser, error)
	DeleteFile(ctx context.Context, file FileAccess) error
	TestConnection(ctx context.Context, location Location) error
	GetRepository() SettingsRepository
}

// Location is where a provider works: the storage's ID keys the session pools, and the settings carry
// the encrypted secrets.
type Location struct {
	StorageID uuid.UUID
	Settings  Settings
}

type FileAccess struct {
	Location
	FileName string
	Logger   *slog.Logger
}

type SettingsRepository interface {
	Upsert(ctx context.Context, storageID uuid.UUID, settings Settings) error
	ListByStorageIDs(ctx context.Context, storageIDs []uuid.UUID) (map[uuid.UUID]Settings, error)
}
```

Each provider package holds its settings struct with plain fields, its GORM row type for `storages_<type>`, its repository through `database.Conn`, its validation and its file operations, which keep today's code. A provider encrypts its secrets with the `FieldEncryptor` from its constructor before `Upsert`, and decrypts them inside each operation, as today. A secret of unsaved settings arrives in clear text, which `Decrypt` returns unchanged. The settings of a provider without secrets, such as local storage, have nothing to merge.

The state that lives in package variables today moves into the provider value that `New` builds once: the NAS, FTP and SFTP session pools, the rclone remote cache with its configuration lock, the S3 transports and the bucket region cache, and the Azure HTTP client. A test builds its own provider and gets its own pools, instead of swapping the package variable `remotes` (`models/rclone/model_test.go:218`). rclone's own process-wide configuration store stays global, because the library offers no other.

Every `context.Background()` becomes `context.WithoutCancel(ctx)` under the same deadline. A delete keeps its own 30 seconds after the caller's context ends, and it now also keeps the caller's logging attributes. The Google Drive test and Drive service get a deadline of 30 seconds. The download at `models/google_drive/model.go:716` gets a comment that the body goes to the caller, who closes it, or loses the `nolint` if `bodyclose` accepts the returned body. The NAS chunk copy and the upload of a background write run on a `background.Group` that the module owns, so no bare `go` statement remains. Local storage takes its two folders from `LocalFolders` instead of `config.GetEnv()`.

The conversion between a provider's settings and its contract schema lives in `httpapi`, one file per provider, as decision 2 of `notifiers-module` and "Engines and providers" in `docs/architecture.md` set out.

*Rejected: keep the pools as package variables.* `gochecknoglobals` is off until milestone 13, but the module would then share state across `apptest` instances in one process, and the rclone test would keep swapping a package variable.

### 3. The parent repository saves both rows in one transaction

`storage.Service` opens one transaction per write. The parent repository inserts or updates the `storages` row, then hands the settings to the provider's repository. A list loads one page of parents, groups their IDs by type and calls `ListByStorageIDs` once per type that occurs on the page, so a page costs at most nine queries. The list is ordered by `name, id` with `database.NameCursor`.

An update loads the stored storage and its settings inside the transaction, merges and validates them there, and writes both rows. Settings of another type are refused with 400 `invalid_request` before anything is written. A prefix that differs from the stored one is refused by `MergeStoredSettings` the same way. Neither a test nor a file operation writes to `storages` or its provider tables, so a refreshed Google Drive token stays in memory, as the proposal keeps it, and `last_save_error` goes without a replacement.

*Rejected: record the last write error per storage with a single-column `UPDATE`.* That is the feature `last_save_error` was meant to be, and the plan adds no features. The banner never showed a value, so users lose nothing.

### 4. The file lifecycle moves into the module unchanged

`features/storages/files` moves to `internal/files` with the behavior `storages-file-store` gave it, with one change. Its deadlines come from the `Clock` of `Deps`, as "Long-running work" in `docs/architecture.md` asks: the pending deletion repository takes the current time as a parameter instead of calling `now()` (`databaseTimeIn`, `features/storages/files/repository.go:254`), and the drain budget is measured on the same clock instead of `time.Now()` (`files/worker.go:120`). The comment there keeps every deadline on one clock so that skew cannot shorten a window on one path and lengthen it on another; taking the time from one `Clock` on every path keeps that property. `apptest` can then move a commit window forward with its manual clock instead of waiting for it. `StorageLocator` becomes a small interface over the registry and the parent repository inside the module. The deletion worker becomes the job `storage_file_deletion` of `GetJobs()`, with the period `FileTimings.WorkerTick`. The runner already keeps passes from overlapping and runs the first pass at start, which `Run` does by hand today. `Run`, its `hasRun` guard and the old background loop go. `apptest.RunJobPass(t, "storage_file_deletion")` replaces `DrainStorageFileDeletions` in API tests, and the bridge keeps `DrainStorageFileDeletions` for the old tests, built on the module's job.

Deleting a storage keeps the transaction of `storages-file-store` decision 2. The foreign key refusal becomes the Problem `in_use`.

### 5. The schema

In one goose file:

1. Renames:
   - `local_storages`, `s3_storages`, `google_drive_storages`, `nas_storages`, `azure_blob_storages`, `ftp_storages`, `sftp_storages` and `rclone_storages` become `storages_local`, `storages_s3`, `storages_google_drive`, `storages_nas`, `storages_azure_blob`, `storages_ftp`, `storages_sftp` and `storages_rclone`;
   - `storage_pending_deletions` becomes `storages_pending_deletions`;
   - the primary keys, references, unique constraints, checks and indexes of these tables take the new prefix.
2. Repairs, before any constraint:
   - a provider row whose storage has another type is deleted;
   - a storage with an unknown type, or without a row in the provider table of its type, is deleted when nothing references it;
   - a storage without a workspace takes the workspace of the earliest database, by ID, whose backup config or backups use it and that has a workspace. Without such a database it takes the oldest workspace, as `20251104000000_workspace_scope_storages_notifiers.sql` did when the column appeared. When the instance has no workspace at all, the storage is deleted if nothing references it.
3. `storages.type` gets `ck_storages_type` over the eight values. `workspace_id` becomes `NOT NULL`. `uq_storages_id_type` is a unique constraint on `(id, type)`. `idx_storages_workspace_id_name_id` replaces `idx_storages_workspace_id` and serves the list. `last_save_error` is dropped.
4. Every provider table gets `type text NOT NULL DEFAULT '<TYPE>'` with `CHECK (type = '<TYPE>')`, and its reference becomes `FOREIGN KEY (storage_id, type) REFERENCES storages (id, type) ON DELETE CASCADE`. `CASCADE` stays, because both tables belong to the module.
5. Enum checks: `storages_azure_blob.auth_method` over its two values, and `storages_s3.s3_storage_class` over `''` and the six classes.

`token_json` stays encrypted `text`. An encrypted value is the string `enc:...`, not JSON, so `JSONB` could hold it only as a JSON string, which gains nothing. Task 6.1 replaces "an OAuth token" in the `JSONB` example of `docs/architecture.md` "Data" with an example that is stored in clear, such as webhook headers.

A storage left without its provider row and still referenced keeps its parent row. No constraint requires the provider row, and the module answers a read of such a storage with 500 `internal`, as the old code panics today. The migration test of task 2.1 covers the unreferenced case, and nothing produces the referenced one today: an update to a type without settings panics in `Validate` before it saves.

*Rejected: stop the migration on a storage it cannot repair.* An instance would not start for a storage that cannot work anyway.

### 6. Eight operations

| Operation | Method and path | Answers |
|---|---|---|
| `listStorages` | `GET /storages?workspaceId&limit&after` | 200 `StoragePage`, 404 |
| `createStorage` | `POST /storages` | 201 `Storage`, 400, 403, 404 |
| `getStorage` | `GET /storages/{storageId}` | 200 `Storage`, 404 |
| `updateStorage` | `PUT /storages/{storageId}` | 200 `Storage`, 400, 403, 404 |
| `deleteStorage` | `DELETE /storages/{storageId}` | 204, 403, 404, 409 |
| `testStorage` | `POST /storages/{storageId}/test` | 200 `StorageTestResult`, 404 |
| `testStorageSettings` | `POST /storage-tests` | 200 `StorageTestResult`, 400, 403, 404 |
| `transferStorage` | `POST /storages/{storageId}/transfer` | 204, 403, 404, 409 |

`Storage` has `id`, `workspaceId`, `name` and `settings`. `StorageSettings` is a `oneOf` of eight schemas with the discriminator `type`. Each provider schema lists its fields in `camelCase` with the names the frontend uses today, with the `is` prefix on booleans: `isVirtualHostedStyle`, `isSkipTlsVerify`, `isSsl`, `isSkipHostKeyVerify`. Its secrets are `writeOnly` and optional; `createStorage` requires them through the provider's validation, which answers 400 `invalid_request` with the field in `errors`.

The S3 storage class is the enum `S3StorageClass` with the six classes. An omitted class means the provider's default and maps to `''`. The Azure method is the enum `AzureBlobAuthMethod`. `NotifierCreate`'s pattern gives `StorageCreate` with `workspaceId`, `name` and `settings`, and `StorageUpdate` with `name` and `settings`.

`testStorageSettings` takes `workspaceId`, an optional `storageId` whose stored secrets fill the omitted ones, and `settings`. It is a resource of its own, `/storage-tests`, for the reason `notifiers-module` decision 7 gives for `/notifier-tests`. `StorageTestResult` is `{isConnected, error}`: a failed test answers the question the user asked, and the storage's error is what the user needs to fix the settings.

### 7. Access goes through `Authorize`

Every operation addressed by a storage ID loads the storage first and answers 404 `not_found` when it is missing. A `not_found` from `Authorize` becomes the same storage `not_found` Problem, as in `notifiers`.

| Operation | Action |
|---|---|
| reads and `testStorage` | `ActionView` |
| writes | `ActionEditContent` |
| `testStorageSettings` | `ActionEditContent` in `workspaceId` |
| transfer | `ActionEditContent` in the source, then in the target |

A `not_found` from `Authorize` on the source becomes the storage's `not_found`. On the target, the transfer returns `Authorize`'s workspace `not_found` unchanged, so a target workspace the actor is not a member of answers exactly as a workspace ID that does not exist, as the `workspace-access` scenario "Transferring a storage into another team's workspace" asks.

For `testStorageSettings`, a `storageId` of another workspace answers 404 before anything is sent. The `storages` requirement "Only a user who may change storages tests unsaved settings" explains why a viewer gets 403 there: with a `storageId`, the test sends stored secrets wherever the request points. A viewer cannot save settings anyway, so testing them gains a viewer nothing.

The rclone rule follows `Authorize`. Creating an rclone storage, updating one and testing unsaved rclone settings answer 403 `permission_denied` unless `actor.Role` is `users.RoleAdmin`. `testStorage` has no such check, as today.

*Rejected: keep `ActionView` for `testStorageSettings` and refuse only a `storageId` with changed hosts.* Every provider would need its own list of the fields that decide where a secret goes. A missed field in a new provider would reopen the leak. The plan names this choice for notifiers as well.

### 8. A transfer asks the backup configs through an event, under a row lock

`TransferStorage` takes a `StorageTransfer` and runs in one transaction:
1. It reads the storage `FOR UPDATE` and checks access in the source and then in the target.
2. It publishes `StorageTransferring` to the subscribed handlers in order.
3. It updates `workspace_id` with one `UPDATE` and writes the audit events with both workspace names.

`internal/app` subscribes the old backup configs through the bridge. Their handler asks `GetDatabasesIDsByStorageID` of both config repositories, reads those databases' workspaces through the old databases service, and returns the Problem `in_use` when any of them is outside the target workspace.

The lock closes a race that the old code has as well. A backup config save checks the storage's workspace with a plain read and then writes the reference, while a transfer counts the configs and then updates the workspace. Without a lock both can commit, and a database ends up using a storage of another workspace. With this change, the logical and physical config save runs its storage check and its write in one transaction opened through `storage.GetTransactor()`, and reads the storage through `GetStorageBlockingTransfer`, which reads it `FOR SHARE`. `FOR SHARE` conflicts with the transfer's `FOR UPDATE`, so the second of the two waits. It then sees the first one's result: either the config already uses the storage and the transfer is refused, or the storage is in the other workspace and the save is refused.

The database transfer of the old backup configs, logical and physical, moves the database first and then its storage. With the database already in the target workspace, the handler lets the storage through, so `transferingWithDbID`, `GetStorageAttachedDatabasesIDs`, `StorageDatabaseCounter`, `AddStorageDatabaseCounter` and `SetStorageDatabaseCountersForTest` go. The routes `/backup-configs/storage/:id/is-using` and `.../databases-count` count with `GetDatabasesIDsByStorageID` of both config repositories inside the old backup configs, after `GetStorages` and `workspaces_bridge.Access.IsActorAllowed` check the actor.

Decision 14 says which package counts both config tables and what the config save's transaction holds.

*Rejected: a composite key from the backup configs to `storages (id, workspace_id)`.* The configs have no `workspace_id`; it belongs to `databases`, which moves in milestone 8, and the configs belong to `backups`, which moves in milestone 9.

### 9. Deletion and the workspace

`deleteStorage` loads, authorizes and runs the deletion of decision 4, answering 204, or 409 `in_use` from the foreign key. `OnWorkspaceDeleting` deletes the workspace's storages through the same path inside the deletion's transaction. `internal/app/wiring.go` subscribes it where the old service stood, after `databases` and `notifiers`, so the order of the `workspace-deletion` spec stays. Audit events go through `lib/audit` in the transaction of the write, with today's messages: created, updated, renamed, deleted, deleted with files left behind, transferred out and transferred in.

### 10. The old code reaches the module through a bridge

`features/storages` keeps:

- `Install(module)` and `GetModule()`, with a module built from the old access points in a `testhelpers` build, as `notifiers_bridge` does;
- the test helpers of decision 11, behind the `testhelpers` tag.

The old code switches as follows:

- The backup configs replace `Storage *storages.Storage` with `Storage *BackupConfigStorage`, a type of their own with `gorm:"-"` and the JSON fields `id`, `name` and `type`. They fill it from one `GetStorages` call per request, a list included, and the repositories drop the `Preload("Storage...")` calls. `takeStorageIDFromEmbeddedStorage` keeps reading `storage.id`. The `backup-storage-assignment` requirement "Saving a backup configuration never creates or changes a storage" holds by construction, because the field is no longer a GORM association.
- Backups, restores and downloads switch from `storages.GetStorageFileStore()`, `*storage_files.Store` and `storages.StorageReader` to `storages_bridge.GetModule()` and the aliases of `module.go`. The restore mocks of `restores/restoring/mocks.go` take a `StorageReader`.
- `telemetry` calls `GetStorageTypes`.
- `internal/app` builds the module in `app.New` with the environment's clock, `LocalFolders` from the configuration and `storages.ProductionFileTimings()`, installs it into the bridge, registers its routes on the contract group, adds its jobs to the runner, and subscribes `OnWorkspaceDeleting` and the backup configs' `StorageTransferring` handler in `wiring.go`. It deletes the old route, the background loop, the GORM schema entry of `Storage` and the old getters from `getters_test.go`. `ProbeStorage` calls `storages.ProbeLocalStorage` with a `LocalStorageProbe` of the configured folders and its logger.
- `archlint` loses `provider_writes.go`, its test and its call in `archlint.go`, per the `backend-code-checks` delta.

Decision 14 covers the narrow file store interfaces of the old backup code.

`temp/refactor/plan.md` gets a row in the table of temporary relaxations: the old features reach storages through `storages_bridge`, and the old backup configs return their own `BackupConfigStorage`, until milestone 9. The row of the provider-writes rule is crossed out.

*Rejected: return only `storageId` from `/backup-configs`.* The database card and the backup config screens would then load every storage of the workspace to show one name, and the `/backup-configs` shape belongs to milestone 9.

### 11. The old test helpers write SQL

`storages_bridge.CreateTestStorage(workspaceID)` inserts a local storage into `storages` and `storages_local` and returns a `storages.Storage` of `module.go`. `CreateTestFlakyS3Storage` inserts an S3 storage with encrypted keys the same way. `RemoveTestStorage` deletes the `storages` row with SQL, and the provider row and the pending deletions go by `CASCADE`, as the old helper removed them without a drain. `DrainStorageFileDeletions` runs passes of the module's deletion job until the given references are gone.

The callers change as follows:
- they take `.ID` from the new type;
- the about 40 places that hand the storage to a backup config's `Storage` field hand `&BackupConfigStorage{ID: storage.ID}`;
- the five files that call provider methods on the storage read through `GetStorageReader` and prepare files through `storages_bridge.WriteTestFile`, which writes a file through the module's `WriteFile` and confirms its receipt in a transaction, so no test reaches a provider;
- the two readers of `.S3Storage` read the bucket from a query of `storages_s3`.

`databases.CreateTestDatabase` drops its unused storage parameter, which removes the storage from 108 call sites with a `gofmt -r` rule.

Decision 14 lists the helpers the bridge and `apptest` gained beyond these.

*Rejected: let the helpers call the module's create operation.* It needs an actor and a workspace member, which the old tests do not set up for storages, and the helpers of `users`, `workspaces` and `notifiers` already write SQL.

### 12. The frontend calls the generated client

`entity/storages/api` is rewritten on the generated client, with `storageApi.listAllStorages(workspaceId)` following `nextCursor` with `limit` 100. The hand-written models are deleted, apart from the label keys and the logo lookup, which move onto the generated types. `StorageType` becomes a `const` object over the generated enum, as `NotifierType` did.

`features/storages` changes as follows:
- the edit component keeps one sub-form per provider and switches on `settings.type`;
- the type selector is offered only for a new storage, and a saved storage shows its type as text;
- the S3 and Azure prefixes are disabled once saved;
- secrets are empty in the form of a saved storage, and an empty secret is left out of the request through the `toOptionalValue` helper of `features/notifiers`, which moves to `shared/lib`;
- the test button calls `testStorageSettings` with the storage's ID when editing and shows `error` of a failed result;
- the detail view calls `testStorage`;
- the `lastSaveError` box and the card's error mark are deleted with their keys in the six dictionaries.

`OauthStorageComponent` keeps the browser exchange and saves through `createStorage` or `updateStorage`. Its `state` keeps carrying the form, the client secret included, as today. The backup config screens, the database card, the database config and the transfer dialog read `BackupConfigStorage` from `entity/backups` and call `listAllStorages`.

Deleting a used storage shows the storage text of the new key `storages.config.removeBlockedByBackups`, because the generic `errors.inUse` names only databases. Testing unsaved settings as a viewer shows the `permission_denied` text.

Screen tests on `mockApi`:
- the edit form of a saved storage offers no type selector;
- saving a renamed S3 storage sends no keys;
- a failed test shows the storage's error;
- deleting a used storage shows the new text;
- the detail view has no save error box.

### 13. Tests

The tests in `internal/modules/storages` use `apptest` and the contract-checking client, one file per concept: lifecycle, secrets, the fixed type and prefix, the list, the rclone rule, connection tests, unsaved settings, deletion, transfer, the file store. `apptest` gets:
- `CreateStorage(tb, session, workspaceID, settings)`;
- `App.InsertBackupConfigUsingStorage` for the deletion and transfer tests, through `ExecuteSQL`, because no contract operation saves a backup config yet;
- an option `WithFileTimings(storages.FileTimings)`, so a test can shorten the timings, and the manual clock moves a commit window forward before `RunJobPass`.

`CreateLocalStorage` goes through `createStorage`. The `workspaces` deletion tests count storages through `listStorages`.

The provider tests move with their code, and the container tests of `model_test.go` move to the provider packages and run against a provider built by the test. `s3_cleanup_test.go` moves to `providers/s3`. The tests of `files` move with the package, `DrainForTest` included.

The requirement "An upgrade keeps every storage's secrets" is tested in two parts. The migration test of task 2.1 inserts rows of the old shape with encrypted secrets, applies the migration and shows that every secret column holds the same bytes. The rename test of task 3.1 inserts an S3 storage with its keys in the stored shape through `ExecuteSQL` and `EncryptSecret`, renames it and makes a backup file reach MinIO with the stored keys, which covers both this requirement and "Changing a storage keeps the secrets the request leaves out".

The tests of the old `controller_test.go` that pin rules the module keeps are ported onto the new operations from the file as it stands before this change: the rclone tests, `Test_StorageSensitiveDataLifecycle_AllTypes` and the transfer permission tests. The old controller and its test file are deleted in task 2.4.

### 14. What the implementation settled

- The physical backup use cases depend on a `FileStore` interface of their own package (`WriteFile`, `StartBackgroundWrite`, `ConfirmFileWrites`, `RequestFileDeletions`), which `storages.Module` satisfies. Their WAL tests fail, hold and count writes through an in-memory `FakeFileStore` over a fake storage, because `files` and the providers are internal to the module and a test can no longer build a real store over a fake provider. These tests no longer see real `storages_pending_deletions` rows; the tests of `internal/files` and the module API tests cover them.
- The logical `BackupFileStore` gains `StartBackgroundWrite`. The old fakes start their writes through `storages_bridge.StartFakeBackgroundWrite`, and the bridge's test helpers also offer `OpenTestFile` and `DeleteTestFile`.
- One package, `features/backups/config/storage_usage`, lists the databases whose logical or physical config uses a storage. The two `/backup-configs/storage/...` routes, the `StorageTransferring` handler and the precheck of a database transfer with its storage read it, because the logical and physical config packages do not import each other.
- The config save's transaction holds only the locked storage check and the row write. The listeners stay outside it: the logical one deletes the old storage's backups before the transaction, after a plain check of the storage, and the physical ones run after the commit, so a backup they start reads the new config.
- `BackupConfigStorage` is declared in both the logical and the physical config package.
- An update checks the rclone rule against the stored and the incoming type, so a member who sends rclone settings to another storage gets 403 rather than 400.
- `StorageProbeError` wraps the module's probe error, which names the failed operation. The app test of a probe whose delete fails is gone, because `internal/app` can no longer reach the provider to fake it.
- `apptest` gets short file timings by default, `App.WriteStorageFile`, the `StorageUse` struct for `InsertBackupConfigUsingStorage`, `LockStorageRow` for the lock test, and `DecryptSecret`.
- The old tests in `features/backups/config` that deleted or transferred a storage through the old routes are gone; the module API tests cover their scenarios.
- Each FTP test container takes a free passive port range, because the module API tests and the provider tests now start FTP containers at the same time.

## Risks / Trade-offs

- [The migration deletes provider rows of another type] → They are never read: the parent's type selects the row. The release notes name the rule.
- [Users lose the type change of a saved storage] → The proposal marks it BREAKING. A user creates a storage of the new type and points the backup configs at it, which also keeps the old backups readable on the old storage.
- [The config save now holds a row lock on the storage] → Only `FOR SHARE`, which waits only on a transfer of the same storage.
- [339 helper calls in 47 test files change] → All but a handful use `.ID`; the rest are listed in Context.
- [Eight providers in one change] → They share one interface and one repository shape, their file operations keep today's code, and each keeps its tests.

## Migration Plan

1. The migration runs at startup before the server accepts requests. The storage tables are small, so the renames, repairs and constraints finish in milliseconds. `storages_pending_deletions` is renamed, not rewritten.
2. The upgrade is one-way, as `docs/architecture.md` says for every migration.
3. The upgrade test writes a storage of every type through the previous release and checks after the upgrade that each is listed with its non-secret settings, and that the backup config still uses the local storage.
