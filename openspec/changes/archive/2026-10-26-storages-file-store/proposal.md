## Why

A storage that a backup config uses, or that still holds backups, must not be deleted, because backup rows are the only record of the files in it. Today the schema does not enforce this. `logical_backup_configs` and `logical_backups` reference `storages` with `ON DELETE CASCADE`, and the rule lives in counters that the backup features register with the storage service in code. A counter that goes missing deletes a database's backup config or its backup rows together with the storage and leaves their files without any record. The deletion also empties the storage's pending deletions before it knows whether the storage is in use. Backup code reads files by calling the provider directly, and it hands the file store its transaction as a `*gorm.DB` parameter, which the storages module of milestone 7 cannot offer. This change fixes the integrity rule in the schema and gives the file store, inside the old code, the shape the module will expose, so that `storages-module` becomes a move.

## What Changes

- The seven references from backup tables to `storages` become `ON DELETE RESTRICT`: `logical_backup_configs` and `logical_backups` instead of `CASCADE`, and `physical_backup_configs`, `physical_full_backups`, `physical_incremental_backups`, `physical_wal_segments` and `physical_wal_history_files` instead of `NO ACTION`.
- Deleting a storage runs in one transaction that deletes the storage row first. When the database refuses because of a reference, the deletion answers with one error that names both reasons, and no file is removed. Only after the row is gone does the deletion remove the files of the storage's pending deletions, within the existing time budget, and write the audit events. A file whose write is still running in this process is not removed and is counted among the files left behind.
- Deleting a workspace uses the same path for each of its storages. When one of them still holds backups of a database in another workspace, the deletion answers `409` with `code` `in_use` instead of `500`, and nothing is deleted.
- `StorageBackupCounter`, its two registrations, `GetStorageBackupReferenceCount` of the logical and physical backup services, `CountByStorageID` and `SetStorageBackupCountersForTest` are deleted. `StorageDatabaseCounter` stays for the storage transfer and the two `/backup-configs/storage/:id/...` routes until `storages-module` replaces it.
- `ConfirmFileWrites` and `RequestFileDeletions` of the file store take the caller's transaction from the context and no longer accept a `*gorm.DB`. They still refuse to run without a transaction. The transactions of the backup code that reach them open through a `Transactor` that the old `internal/storage` package hands out.
- The file store gains a reader of one storage's files. The four restore use cases, the logical backup download, the physical restore stream and the parent manifest read of an incremental backup read through it instead of calling the provider. The restore use cases no longer take a `*storages.Storage`.
- The old error texts `storage has attached databases and cannot be deleted` and `storage still holds backups; ...` are replaced by one text for both reasons. The old route keeps answering `400` with `{"error": ...}`.
- **BREAKING** for the schema: the seven foreign keys change their delete action.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `storage-file-cleanup`: a storage that a backup config uses cannot be deleted either, a refused deletion removes no file, and the deletion never removes a file whose write is still running.
- `workspace-deletion`: a workspace whose storage still holds backups of another workspace cannot be deleted, and deleting a workspace removes the pending files of its storages.

## Out of scope

- The `storages` module, its contract, the provider registry, the table renames and the frontend. They are `storages-module`.
- The storage transfer, the `transferingWithDbID` parameter and `StorageDatabaseCounter`. `storages-module` replaces them with an event.
- Moving the old backup code, or changing its transactions beyond how they are opened. Milestones 9 and 10 move it.
- The `archlint` rule against direct provider writes. It stays until the providers become unreachable in `storages-module`.
- What happens to logical backups when a backup config switches to another storage. They stay on the old storage, as today.
- New error codes on the old storage routes. They move to `Problem` bodies in `storages-module`.

## Impact

- Backend: `backend/internal/features/storages` (service, repository, `files`); `backend/internal/storage` (the `Transactor`); the transactions in `features/backups/backups/backuping/{logical,physical}`, `core/physical/service` and `usecases/physical/postgresql`; the read paths in `features/restores`, `features/backups/backups/services`, `features/backups/backups/download/restore_stream` and `usecases/physical/postgresql`; the counters in `features/backups/backups/services` and `core/physical/service`.
- Migrations: one migration replaces seven foreign keys.
- Running instances: nothing is repaired or deleted, because the foreign keys already exist and no row can break them.
- Documents: `temp/refactor/plan.md` (the checklist), `docs/glossary.md` (the file store also reads).
- Rules the change answers to: the root `AGENTS.md` and `backend/AGENTS.md`.
