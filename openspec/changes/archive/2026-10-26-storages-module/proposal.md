## Why

`storages` is the second provider family in `docs/architecture.md` to move into the target layout, after `notifiers`. Today it is an old feature:
- `switch` statements on the storage type in four places and a 120-line `StorageRepository.Save`;
- controllers that answer every error as 400 with the internal error text, and a create that answers with the request body, the secrets just encrypted included;
- package-level session pools, caches and transports inside the providers;
- a transfer guarded by counters that the backup configs register in code.

Backups still reach a storage through `storages.Storage`, a GORM association on the backup configs and the services of the old feature, so nothing keeps backup code away from the providers except an `archlint` rule. Two defects come from this shape as well. Testing unsaved settings with the ID of a saved storage needs only the right to view. It fills the stored password or keys into the host the caller sent, so a viewer can make the backend send them to a server of the viewer's choice. And testing a saved storage saves the whole storage afterwards, which can write a refreshed Google Drive token to the database without encryption. `storages-file-store` already gave the file store the signatures the module offers. This change moves the feature into `internal/modules/storages` with its operations in the contract, as milestone 7b of the backend rewrite plan.

## What Changes

- The module `internal/modules/storages` appears with `module.go`, `Deps`, `New` and its code under `internal/`. It owns storages, their provider settings, connection tests, transfers and the whole file lifecycle: the file store, write receipts, pending deletions, the storage reader and the deletion worker. Its services open transactions through the `database.Transactor` from `Deps`, check access through `workspaces.Authorize` and write audit events through `lib/audit` in the same transaction.
- Each of the eight providers (local, S3, Google Drive, NAS, Azure Blob, FTP, SFTP, rclone) becomes a package under `internal/providers` that holds its settings, its validation, its file operations, its connection test and the repository of its own table. A registry keyed by type is the only place that knows every provider. The `switch` statements, the eight `Preload` calls and the GORM associations are deleted. The session pools, the rclone remote cache, the S3 transports and its bucket region cache become fields of their provider instead of package variables.
- `module.go` offers the file store as `storages-file-store` shaped it: `WriteFile`, `ConfirmFileWrites`, `RequestFileDeletions` and `GetStorageReader`. It also offers a batch lookup of storages by ID, the storage types for telemetry, `TransferStorage`, the handler of `workspaces.WorkspaceDeleting`, the event `StorageTransferring`, the deletion worker as a `background.Runner` job, and a probe of the local storage for the `--test-storage` console command. The providers are unreachable from outside the module, so the `archlint` rule against direct provider writes is deleted.
- 8 operations enter `backend/api/openapi.yaml` under the tag `storages`: list with a cursor, create, read, update, delete, test a saved storage, test unsaved settings, and transfer to another workspace. The provider settings are a `oneOf` with the discriminator `type`. Secrets are `writeOnly`, and an omitted secret keeps its stored value on update. A response carries the saved storage without its secrets.
- A storage's type is fixed when it is created. An update with settings of another type answers 400 `invalid_request`, and the edit form shows the type of a saved storage without a selector. **BREAKING** for users: today the form offers a type change for a saved storage and the backend applies it.
- Testing unsaved settings needs the right to edit the workspace's content. A viewer can still test a saved storage with its stored settings.
- Testing a saved storage writes nothing.
- `last_save_error` is deleted together with its banner and its mark on the storage card. No code path ever stores a value in it.
- Deleting a storage that a backup config uses or that holds backups answers 409 `in_use`. A transfer answers 409 `in_use` while a database outside the target workspace uses the storage, through the event `StorageTransferring` that the backup configs handle. `StorageDatabaseCounter` and `transferingWithDbID` are deleted. The database transfer moves the database before its storage.
- The old features reach the module through a bridge in `features/storages`: the installed module, and the test helpers behind the `testhelpers` tag. The old backup configs replace the GORM association `Storage *storages.Storage` with a type of their own, `BackupConfigStorage`, filled from the module's batch lookup, so `/backup-configs` keeps its `storage` field. The routes `/backup-configs/storage/:id/is-using` and `.../databases-count` stay in the old backup configs and count their own two tables.
- The provider tables become `storages_<type>` and the pending deletions `storages_pending_deletions`. The migration adds a composite foreign key `(storage_id, type)` from every provider table, a `CHECK` on `storages.type`, on the Azure authentication method and on the S3 storage class, and `NOT NULL` on `workspace_id`, and drops `last_save_error`. Rows that would break the new constraints are repaired first.
- The frontend calls the 8 operations through the generated client and types. The hand-written models of `entity/storages` are deleted.
- The tests of `storages` become module API tests through `apptest` and the contract-checking client. The upgrade test writes one storage of every type before the upgrade and checks them after it.
- **BREAKING** for users' scripts: the storage operations change shape. The list takes `workspaceId` instead of `workspace_id` and returns a page with `items` and `nextCursor`. A storage carries `type` and `settings` instead of one field per provider. Saving an existing storage is `PUT /storages/{storageId}` instead of `POST /storages` with an ID. The test of unsaved settings moves from `/storages/direct-test` to `/storage-tests`. A connection test answers its outcome in the body. Errors are `Problem` bodies.
- **BREAKING** for the schema: nine tables are renamed, one column is dropped, and constraints are added.

## Capabilities

### New Capabilities

- `storages`: creating, changing, testing, transferring and deleting a storage, its secrets, its fixed type, the rclone rule, and who may test unsaved settings.

### Modified Capabilities

- `workspace-access`: the storage operations answer access refusals with 404 `not_found` and 403 `permission_denied`, like the workspace and notifier operations.
- `storage-file-cleanup`: deleting a storage that is still referenced answers 409 `in_use`.
- `upgrade-test`: the test writes one storage of every type instead of one local storage.
- `backend-code-checks`: the rule against direct provider writes from old backup code goes, because the providers are inside the module.

## Out of scope

- Moving `backups`, `restores`, `databases` or `telemetry` into modules. They call the module through the bridge and keep their routes, `/backup-configs/storage/:id/is-using` and `.../databases-count` included.
- Running the database transfer in one transaction. It belongs to the backup configs and is rebuilt in milestone 9; this change only reorders it.
- Moving the Google Drive OAuth code exchange to the backend, or persisting a refreshed Google Drive token. The browser keeps exchanging the code, and a refresh stays in memory for one operation, as today.
- Moving the old storage connection test of a saved storage behind a different permission: a viewer keeps it.
- New providers, new provider settings, and changing what a provider does with a file.
- Renaming provider columns such as the `s3_` prefix of the S3 table.

## Impact

- Backend: new `backend/internal/modules/storages`; `backend/api/openapi.yaml`; `backend/internal/app` (construction, routes, the `WorkspaceDeleting` and `StorageTransferring` subscriptions, the deletion worker, the local storage probe) and `backend/internal/app/apptest`; `backend/internal/features/storages` reduced to the bridge and the test helpers; the backup configs, backuping, core, download and usecases packages of `features/backups`, and `features/restores`, which switch their imports to the bridge; `features/telemetry`; `backend/tools/archlint`; the test files that use `storages.CreateTestStorage`, listed in design.md Context; `backend/e2e/upgrade`.
- Frontend: `frontend/src/entity/storages`, the screens of `features/storages`, the storage parts of `features/backups` and `features/databases`, `pages/OauthStorageComponent.tsx`, and the six dictionaries in `frontend/src/shared/i18n/locales`.
- Migrations: one migration renames nine tables, repairs rows, drops a column and adds the constraints.
- Running instances: storages keep their settings, secrets, backups and backup configs. A provider row whose type differs from its storage's is deleted; these rows are left behind by type changes and are never read. A storage without a workspace takes the workspace of a database whose backup config or backups use it, or else the oldest workspace; only on an instance without any workspace is it deleted, and only when nothing references it.
- Documents: `temp/refactor/plan.md` (the checklist and the table of temporary relaxations), `docs/architecture.md` (the example of an OAuth token as `JSONB`, the stored files section), `docs/glossary.md` (stored file reference, storage reader, background write), `openspec/specs/backend-code-checks`.
- Rules the change answers to: the root `AGENTS.md`, `backend/AGENTS.md` and `frontend/AGENTS.md`.
