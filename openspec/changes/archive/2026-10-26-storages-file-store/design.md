## Context

Why the change is needed is in proposal.md, section Why. This document covers only what shapes the approach. It assumes `notifiers-module` has landed: `in_use` exists in `apierror.Code`, `database.IsForeignKeyViolation` and `database.WithoutTransaction` exist in `lib/database`, and `workspaces.WorkspaceDeleting` passes a handler's `Problem` back to the caller.

**References to storages.** On the dev database, `storages` is referenced by eight tables outside its own provider tables:

| Table | Constraint | On delete |
|---|---|---|
| `logical_backup_configs` | `fk_backup_config_storage_id` | `CASCADE` |
| `logical_backups` | `fk_backups_storage_id` | `CASCADE` |
| `physical_backup_configs` | `fk_physical_backup_configs_storage_id` | `NO ACTION` |
| `physical_full_backups` | `fk_physical_full_backups_storage_id` | `NO ACTION` |
| `physical_incremental_backups` | `fk_physical_incremental_backups_storage_id` | `NO ACTION` |
| `physical_wal_segments` | `fk_physical_wal_segments_storage_id` | `NO ACTION` |
| `physical_wal_history_files` | `fk_physical_wal_history_files_storage_id` | `NO ACTION` |
| `storage_pending_deletions` | `fk_storage_pending_deletions_storage_id` | `CASCADE` |

The last one belongs to the storages feature itself.

**Deletion.** `StorageService.DeleteStorage` (`internal/features/storages/service.go:197`) loads the storage and checks access. It then asks `GetStorageAttachedDatabasesIDs` (`:49`), which merges the `StorageDatabaseCounter` registered by both backup config services (`backups/config/logical/di.go:43`, `backups/config/physical/di.go:46`). Next it asks `GetStorageBackupReferenceCount` (`:34`), which sums the `StorageBackupCounter` registered by the logical backup service (`backups/backups/services/di.go:86`, counting with `CountByStorageID` at `core/logical/repository.go:201`) and the physical one (`backuping/physical/di.go:146`, counting four tables in `core/physical/service/storage_backup_counter.go:14`). Only then does it call `drainPendingDeletions` (`pending_deletion_drain.go:22`), which runs `DeletionWorker.DrainStorage` (`files/worker.go:110`) with a 10-second budget, and deletes the rows with `StorageRepository.Delete`. The drain reads the pending deletions on the pool and removes every file it finds, including one whose write is still running in this process, because it does not consult the store's live writes the way `claimDue` (`files/worker.go:143`) does. `OnWorkspaceDeleting` (`service.go:94`) drains and deletes each storage of the workspace without the checks. The two refusals are `ErrStorageHasAttachedDatabases` and `ErrStorageHasBackups` (`errors.go`), answered as 400 `{"error": ...}`. The frontend asks `/backup-configs/storage/:id/is-using` before it deletes, so the backend refusal shows mostly for backups.

**The file store.** `Store.ConfirmFileWrites` (`files/store.go:97`) and `RequestFileDeletions` (`:121`) take a `tx *gorm.DB` and refuse unless it is a transaction (`requireTransaction`, `:193`). `WriteFile` (`:45`) writes its pending deletion on the pool with `db.GetDb()`, so the obligation commits before the upload starts. 16 transactions of the backup code reach the two methods, all opened with `GetDb().Transaction(func(tx *gorm.DB) error)` or `GetDb().WithContext(ctx).Transaction(...)`:

- `backuping/logical/backuper.go:362` and `:386`, `backuping/logical/cleaner.go:89`;
- `backuping/physical/backuper.go:584`, `backuping/physical/scheduler.go:787`;
- `core/physical/service/database_removal_listener.go:19`, `core/physical/service/service.go:145`, `:166`, `:225` and `:666`, whose helpers `deleteWalInSpanBudgeted`, `deleteIncrementalSet`, `deleteOrphanedHistoryFiles` and `deleteFullArtifactAndRow` pass `tx` down to `requestFileDeletions` (`:1111`);
- `usecases/physical/postgresql/create_full_uc.go:206`, `executor.go:136`, `wal_upload.go:469` and `:490`, `timeline.go:284` and `:307`.

The tests of `files` call the two methods with a transaction 38 times (`files/publication_test.go`, `files/worker_test.go`).

**Reads.** Backup code reads a stored file by calling the provider's `GetFile` with the field encryptor:

- the four restore use cases (`restores/usecases/{postgresql,mysql,mariadb,mongodb}/restore_backup_uc.go`), which receive `*storages.Storage` through `restores_core.RestoreBackupUsecase.Execute` (`restores/core/interfaces.go:20`) from the restorer (`restores/restoring/restorer.go`);
- `LogicalBackupService.GetBackupReader` (`backups/backups/services/logical_service.go:271`) for downloads;
- the physical restore stream (`backups/backups/download/restore_stream/artifact.go:85` and `:186`), which keeps a `storageCache` per stream (`restore_stream/writer.go:53`) so that thousands of WAL segments do not load their storage thousands of times;
- the parent manifest of an incremental backup (`usecases/physical/postgresql/create_incremental_uc.go:265`), through `CommonBackupSpec.Storage storages.StorageFileSaver` (`usecases/physical/postgresql/dto.go:29`) asserted to `parentManifestFetcher`.

**Rules in force.** `docs/architecture.md` "Data" asks for `RESTRICT` between modules and for the database to refuse deleting a referenced storage. "Transactions travel in the context" asks that a transaction is never a parameter. "Module boundaries" allows an event handler to delete files that already have a pending deletion inside the transaction, because that stays correct on rollback. The old code keeps its structure until its milestone, per `AGENTS.md` "Until the migration finishes". The change answers to the root `AGENTS.md` and `backend/AGENTS.md`.

## Goals / Non-Goals

**Goals:**

- The database refuses to delete a storage that a backup config or a backup row references, and the counters that duplicated that rule are gone.
- No file leaves a storage before its deletion is known to succeed, and no file whose write is running is removed.
- The file store offers exactly the operations, with exactly the signatures, that `storages-module` puts into `module.go`, so that change only moves the code and changes imports.

**Non-Goals:**

- Moving any code into `internal/modules`, or touching the storage routes, the provider models or the frontend.
- Changing what the backup code does inside its transactions. Only the way each transaction is opened and how its `tx` reaches the store changes.

## Decisions

### 1. Seven references become `RESTRICT`

One goose migration drops and re-adds the seven constraints of the table above, keeping their names, with `ON DELETE RESTRICT`. `storage_pending_deletions` keeps `CASCADE`, because it belongs to the storages feature. No row is repaired: every constraint already exists, so no row can reference a missing storage. The plan's step "check that no backup references a missing storage" therefore needs no SQL. The down section restores the old actions and serves development only.

`RESTRICT` instead of `NO ACTION`, because `RESTRICT` cannot be deferred to the end of a transaction. The deletion of decision 2 needs the refusal at the `DELETE` statement, before it removes any file.

*Rejected: keep `NO ACTION` on the physical tables.* It behaves the same today, but a later `SET CONSTRAINTS ALL DEFERRED` would move the refusal after the drain. `docs/architecture.md` names `RESTRICT`.

### 2. A storage is deleted first, and its files are removed after

`DeleteStorage` and `OnWorkspaceDeleting` run the same unexported `deleteStorageAndDrainFiles(ctx, storage)` inside a transaction opened with the `Transactor` of decision 4. `OnWorkspaceDeleting` already runs in the workspace deletion's transaction, so the call joins it. The steps:

1. Lock the storage row with `SELECT ... FOR UPDATE`, then read its pending deletions through `database.Conn(ctx, ...)` in the transaction. The lock matters: `WriteFile` inserts its pending deletion on the pool, and that insert takes only a key share lock on `storages`. Without the row lock an insert committed between this read and the `DELETE` would be removed by the cascade without reaching the list. With it, a concurrent insert waits for the deletion and then fails on the foreign key, or proceeds when the deletion rolls back.
2. Delete the provider row and the storage row. A foreign key violation, detected with `database.IsForeignKeyViolation`, ends the transaction with `ErrStorageInUse`, and nothing else happens.
3. Remove the files of step 1 from the provider within the 10-second budget, with the storage loaded before step 2, and skip the files whose write is running in this process.
4. Write the audit event `Storage deleted`, and the event `Storage deleted with N file(s) left behind` when files were skipped or failed.

```
Transaction(ctx)
  SELECT ... FROM storages WHERE id = ? FOR UPDATE
  pending := SELECT ... FROM storage_pending_deletions WHERE storage_id = ?
  DELETE provider row, DELETE FROM storages WHERE id = ?
      | FK violation --> rollback, ErrStorageInUse (old route: 400, workspace deletion: 409 in_use)
      v
  DrainFiles(storage, pending, budget)   -- skips live writes, returns the names left behind
  audit
commit
```

`DeletionWorker.DrainStorage(ctx, storageID, budget)` becomes `DrainFiles(ctx, DrainRequest{Remover, PendingDeletions, Budget})`. It takes the remover and the list from the caller, because the storage row is gone from the transaction's view when the files are removed. It checks each file against the store's live writes under the same mutex as `claimDue`, which is what the `storage-file-cleanup` requirement "Cleanup never removes a file that is still being written or already published" asks.

`ErrStorageHasAttachedDatabases` and `ErrStorageHasBackups` become one `ErrStorageInUse`: "storage is used by a backup config or still holds backups; remove them first, they are the only record of the files it holds". The database names no reason in a way worth parsing: a constraint name would be one more contract between the schema and the old code, and `storages-module` answers both reasons with one `in_use` anyway. In the workspace deletion path the handler returns a `Problem` with `in_use` instead, because the workspaces module answers any other error with 500, which the `workspace-deletion` delta forbids.

*Rejected: drain first, delete after, as today.* Without the counters the drain would run before the storage is known to be unused. A backup writing to a storage that its config still uses holds a valid receipt, because the drain does not raise the generation. The drain would remove that file, the storage deletion would fail, and the backup would publish a file that no longer exists.

*Rejected: drain after commit.* A crash between the commit and the drain loses the list of files without a trace, while the list in the audit event is the user's only way to find them.

*Rejected: a constraint-name check that keeps two error texts.* It would need a helper in `lib/database` and a table of constraint names in the old service, both thrown away in `storages-module`.

The cost of removing files inside the transaction: the storage row stays locked for up to the budget plus one provider timeout. A backup config save that names this storage waits that long and then fails on the foreign key. A storage is deleted rarely, and the same holds inside a workspace deletion today.

### 3. The counters of backups go

`StorageBackupCounter`, `AddStorageBackupCounter`, `GetStorageBackupReferenceCount` of `StorageService` and of both backup services, `core/physical/service/storage_backup_counter.go`, `BackupRepository.CountByStorageID` and `SetStorageBackupCountersForTest` are deleted with their registrations. `StorageDatabaseCounter` and `GetStorageAttachedDatabasesIDs` stay for `TransferStorageToWorkspace` and the routes `/backup-configs/storage/:id/is-using` and `.../databases-count`; `storages-module` replaces them.

### 4. The store takes the transaction from the context

`ConfirmFileWrites(ctx, receipts)` and `RequestFileDeletions(ctx, references)` drop their `tx` parameter and run through `database.Conn(ctx, db)`. `requireTransaction` checks the handle that `Conn` returns, so a call outside a transaction still fails with `ErrTransactionRequired`. `WriteFile` writes its pending deletion through `database.Conn(database.WithoutTransaction(ctx), db)`, because the obligation has to commit before the first byte leaves, even when a caller writes from inside a transaction.

The old `internal/storage` package gains `GetTransactor()`, a `database.Transactor` over the pool it already hands out, next to `GetDb()`. Each of the 16 transactions of the Context section changes from

```go
db.GetDb().Transaction(func(tx *gorm.DB) error { ... })
```

to

```go
storage.GetTransactor().Transaction(ctx, func(ctx context.Context) error {
	tx := database.Conn(ctx, storage.GetDb())
	...
})
```

The body keeps passing `tx` to the old repositories and helpers, and it passes the shadowed `ctx` to the store. The helpers of `core/physical/service/service.go` that receive both keep both, and the inner `ctx` they get is the transaction's.

*Rejected: keep `tx *gorm.DB` in the store's signature until `storages-module`.* The module's `module.go` would then carry a GORM handle, or `storages-module` would have to change every transaction of the backup code on top of the move.

*Rejected: an exported `database.ContextWithTransaction(ctx, tx)`.* It is shorter at each call site, but it lets any code put any handle into the context as "the transaction", which `Transactor` exists to prevent, and nothing would ever remove it.

### 5. Reads go through a storage reader of the store

```go
// StorageReader opens files of one storage. The returned reader may keep fetching from the provider
// while it is read, so ctx must outlive the stream.
type StorageReader interface {
	OpenFile(ctx context.Context, fileName string) (io.ReadCloser, error)
}

func (s *Store) GetStorageReader(ctx context.Context, storageID uuid.UUID) (StorageReader, error)
```

`StorageLocator` gains `GetFileReader`, implemented in `file_locator.go` next to `GetFileWriter`, so the provider lookup stays in the parent package. It returns a `FileReader`, the provider-level interface with `GetFile` that takes the encryptor, matching `FileWriter` and `FileRemover`. `StorageReader` is what the store hands out: a `FileReader` bound to the field encryptor once, so a caller does not need an encryptor to read.

- The restorer gets the reader for `backup.StorageID` and passes it to `Execute` instead of `*storages.Storage`; its logs keep `storage_id` from the backup and drop `storage_type`.
- `GetBackupReader` opens through the reader of the backup's storage.
- The restore stream keeps its per-stream cache, now of readers keyed by storage ID.
- `CommonBackupSpec.Storage` becomes `StorageReader`, and the incremental backup opens its parent manifest through it; `parentManifestFetcher` goes.

*Rejected: `OpenFile(ctx, StoredFileReference)` on the store.* It reads well for one file, but each call would load the storage and its provider row again, and a restore stream opens thousands of WAL segments.

## Risks / Trade-offs

- [A storage deletion holds a row lock while it talks to the provider] → Bounded by the 10-second budget plus one provider timeout, and only nothing referencing the storage can wait on it.
- [The mechanical rewrite of 16 transactions could pass the outer `ctx` to the store] → The store refuses a call without a transaction, so a missed site fails its existing tests at once instead of committing a file without its row.
- [One error text replaces two on the old route] → The frontend checks `is-using` before it deletes, and `storages-module` changes the answer to a `Problem` anyway.
- [Files written by an in-flight write to a storage being deleted are left behind] → They are named in the audit event. The write's publication fails on the foreign key, as it would today.

## Migration Plan

1. The migration runs at startup before the server accepts requests. It replaces seven constraints on tables whose rows are already valid. Adding a foreign key validates the existing rows, which takes a scan of each table under a lock. `physical_wal_segments` is the largest table, and the scan finishes before the server starts.
2. The upgrade is one-way, as `docs/architecture.md` says for every migration.
