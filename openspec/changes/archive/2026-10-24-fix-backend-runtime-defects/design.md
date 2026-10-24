## Context

See `proposal.md` for the motivation and the four parts. The requirements live in the delta specs. This document covers the current state that shapes each fix and the decisions behind it.

The constraining rules come from `AGENTS.md` and `backend/AGENTS.md`: positional DI and `sync.OnceFunc` wiring (`backend/AGENTS.md:149-184`), no access to another feature's repository (`backend/AGENTS.md:186-206`), background services that panic on a second `Run` (`backend/AGENTS.md:208-239`), the stored-file rules (`backend/AGENTS.md:241-271`), redaction at the logger layer (`AGENTS.md`, Security), and the rule to refactor the code the change touches. The WAL retention rules in `openspec/specs/physical-backups/wal-retention` constrain part 2, and the container stop sequence in `openspec/specs/container-shutdown` constrains part 1.

### Shutdown today

- `cmd/main.go:345` and `cmd/main.go:449` install two independent `SIGTERM`/`SIGINT` handlers. The first drains HTTP for up to 10 seconds, the second cancels the context of the background services. `config.StartListeningForShutdownSignal` (`internal/config/signals.go:13`) would install a third, but nothing calls it, so `config.IsShouldShutdown()` always returns `false`. Its flag is a plain `bool` written from a goroutine, which would be a data race if it were ever set.
- The logical backuper and the restorer detach their work from the caller on purpose (`backuping/logical/backuper.go:71`, `restores/restoring/restorer.go:97`). Only a user's cancellation request cancels their `executionCtx`, and the backuper records a backup as cancelled exactly when `executionCtx` ended (`backuper.go:131-145`). `logical-backup-outcome` depends on that distinction.
- A logical backup learns about shutdown from `NewBackupContext` (`backups/backups/core/logical/backup_interruption.go:20`), which polls `IsShouldShutdown` every second and cancels the engine's own context with `ErrBackupShutdown`. The four restore use cases poll the same flag and return an error containing "shutdown", which the restorer records as failed (`restorer.go:180-186`).
- After the HTTP drain, `main` flushes the log with its own 5-second deadline (`internal/util/logger/logger.go:18`) and returns. Nothing waits for running backups or restores. The startup recovery then marks them failed with a restart message (`backuping/logical/scheduler.go:304`, `restores/restoring/scheduler.go:97`).
- Every running logical backup, restore, physical backup and WAL streamer registers a cancel function in `tasks/cancellation.Registry` (`registry.go:21`), keyed by the task ID.

### Storage handling today

- The restorer loads the backup and then reads from `*backupConfig.StorageID` (`restorer.go:84-105`), although the backup row records its own `StorageID` (`backups/backups/core/logical/model.go:18`).
- The logical config service calls `OnBeforeBackupsStorageChange` before it saves a new storage (`backups/config/logical/service.go:100-110`), and `LogicalBackupService` deletes every backup of the database in response (`backups/backups/services/logical_service.go:75`). The physical config service has the same call (`backups/config/physical/service.go:130-140`), but nothing registers a listener, and no physical implementation exists. The frontend warns about deletion for both kinds (`frontend/src/features/backups/physical/ui/EditPhysicalBackupConfigComponent.tsx:703`).
- Deleting a physical full backup through `PhysicalBackupService.deleteFull` (`backups/backups/services/physical_service.go:421`) stops an in-flight backup of that chain and then removes the chain through the core service, which applies the WAL retention locks. The physical cleaner removes WAL that no full backup anchors (`backuping/physical/cleaner.go:539`).
- The WAL supervisor's `reconcile` (`backuping/physical/wal_supervisor.go:166`) starts a streamer for every config that expects one and has no running streamer. It reads the storage from the config at start (`wal_supervisor.go:242`). `PhysicalBackupCancellationListener.cancelStreamingAndDeleteStreamerRow` (`backuping/physical/cancellation_listener.go:37`) stops a streamer, after which `reconcile` starts a new one.
- Both config repositories call `Save(...).Omit(...)` (`backups/config/logical/repository.go:21`, `backups/config/physical/repository.go:22`). `Omit` after the finisher has no effect, so GORM also upserts the embedded `Storage`.
- `DatabaseService.SetBackupError`, `SetLastBackupTime` and `SetHealthStatus` (`databases/service.go:522`, `:537`, `:661`) load the database, change one field and call `dbRepository.Save`, which rewrites the database row and its engine-specific connection row (`databases/repository.go:24-90`).
- `LogicalBackupService.DeleteBackup` hands deletion to the cleaner (`logical_service.go:220`), which notifies its listeners (`backuping/logical/cleaner.go:75`). Restores subscribe to both the service and the cleaner (`restores/di.go:48-49`). The service never reads its own list.

### Secrets and HTTP today

- `GoogleDriveStorage.refreshToken` and `withRetryOnAuth` print token fragments and the token JSON with `fmt.Printf` (`storages/models/google_drive/model.go:406-536`). `SaveFile`, `GetFile` and `DeleteFile` receive a `*slog.Logger`, `TestConnection` does not.
- `SecretKeyService.GetSecretKey` (`encryption/secrets/service.go:52`) reads and writes `cachedKey` without a lock. When the key file is missing, two concurrent first calls each generate a key and write the file.
- `WebhookNotifier.AfterFind` (`notifiers/models/webhook/model.go:73-101`) decrypts the URL and body template and silently keeps the ciphertext on failure. `decryptHeadersForSending` (`model.go:302`) does the same for headers. The other notifiers decrypt at send time and return the error (`notifiers/models/discord/model.go:41`).
- `http.Server` sets only `Addr` and `Handler` (`cmd/main.go:331`). `mountFrontend` (`cmd/main.go:619`) answers every unknown path with a file from `ui/build` or `index.html`. `http.ServeFile` rejects paths containing `..`, so there is no path traversal.

## Goals / Non-Goals

**Goals:**

- One owner of process signals, and a shutdown signal that is safe to read from any goroutine.
- Shutdown that records the outcome of logical backups and restores without changing how a user cancellation is recorded.
- Each fix reuses the deletion, cancellation and retention paths that already exist, instead of adding parallel ones.

**Non-Goals:**

- Making physical full and incremental backups react to shutdown. Their detached contexts stay as they are.
- Making the logical backuper build the backup context itself, as the restorer will. The four engines keep calling `NewBackupContext`, which also owns the 23-hour limit.
- Changing how logical storage changes work. They keep deleting backups before the save.

## Decisions

### 1. One signal handler, and a shutdown signal that is a closed channel

`main` installs the only handler, with `signal.NotifyContext`. When it fires, `main` calls `config.BeginShutdown()`, which closes a channel once through `sync.OnceFunc`, and cancels the background context. `config.GetShutdownSignal()` returns that channel for reading. `StartListeningForShutdownSignal` and `IsShouldShutdown` are deleted, and the two handlers in `main.go` merge into this one. The `stop` function of `signal.NotifyContext` is called only after the whole shutdown sequence has ended. Until then the registration stays, so a second `SIGTERM` or `SIGINT` lands in the already fired context instead of triggering the default action that kills the process. The second signal therefore changes nothing, and the sequence runs once.

The signal reaches the tasks at the two places that build a task's own context, and each of them takes the channel as a parameter:

- `NewBackupContext(callerCtx, shutdownSignal)` selects on the channel instead of polling every second, and still ends the engine context with `ErrBackupShutdown`. The four logical engines pass `config.GetShutdownSignal()`.
- The restorer builds the restore context the same way, with a new `NewRestoreContext(executionCtx, shutdownSignal)` in `restores/core`, which ends it with `ErrRestoreShutdown` ("restore interrupted by application shutdown"). It passes that context to the use case. The restorer receives the channel as a positional DI field. The four restore use cases lose their own polling goroutines, which today cancel an inner context when the flag is set.

The 17 direct flag checks inside copy loops and use cases are deleted, not replaced. Each runs under a context that now ends on shutdown and checks `ctx.Done()` right before. After the use case returns, the restorer decides the outcome from causes, as the logical backuper does (`backuper.go:131`). A shutdown cause means failed with the shutdown message, a cancelled `executionCtx` means cancelled, and anything else means failed. This replaces the substring matching on "restore cancelled", "context canceled" and "shutdown" at `restorer.go:180-186`. A storage error that wraps `context.Canceled` without a user request is therefore recorded as failed instead of cancelled, matching the logical rule.

Tests pass their own channel, so no test closes the process-wide signal and affects the tests that follow it.

Alternatives rejected:

- Calling `StartListeningForShutdownSignal` from `main`. It is one line, but it keeps a racy `bool`, three handlers for one signal and 17 polling sites, and no test can trigger it without poisoning its package.
- Cancelling `executionCtx` on shutdown through the registry. The backuper would then record every interrupted backup as cancelled, because `executionCtx` ending means "user cancelled" (`backuper.go:131`). That breaks `logical-backup-outcome`.
- Keeping a global predicate such as `config.IsShuttingDown()` for the existing call sites. It keeps the duplicated checks and leaves tests with a signal they cannot reset.

### 2. Bounded wait through the cancellation registry

`Registry` gains a method that blocks until no task is registered or its context ends. After the signal, `main` runs the HTTP drain and this wait at the same time, under one 6-second deadline. It then flushes the log under its own 2-second deadline. The total stays within the 8 seconds the spec allows and leaves Docker's default 10-second stop timeout room for the embedded PostgreSQL fast shutdown. The flush keeps a separate deadline for the reason the comment at `cmd/main.go:359` gives: a slow drain must not hand the flush an expired context. The 5-second constant stays for the console commands that call `ExitAfterFlush`.

`CancelRegisteredTask` removes a task from the registry before the task has finished (`tasks/cancellation/registry.go:33-45`). A task that a user cancels just before the stop signal is therefore not waited for, and its cancelled status may not be written before the exit. This case is accepted: the startup recovery records such a task as failed, which is today's outcome for every task interrupted by a stop.

Alternatives rejected:

- A separate `sync.WaitGroup` in the backuper and the restorer. It duplicates the bookkeeping the registry already does for every task.
- Not waiting. The process would usually exit before a backup has written its status, so wiring the signal would change nothing that users see.
- Waiting only for the task kinds that react to shutdown. That needs a task kind in the registry. A physical backup that ignores the signal costs at most the 6-second deadline, and today's drain alone may take 10 seconds.

### 3. Restores read from the backup's storage

The restorer loads the storage by `backup.StorageID` and drops the check that the config names a storage. The config is still loaded, because the restore use cases take it as an argument.

Alternative rejected: keeping the config as a fallback when the backup's storage is missing. A backup always records its storage (`model.go:18`, `not null`), so the fallback would only hide a broken row.

### 4. Physical storage change runs after the save and reuses the full-backup deletion

The physical storage change listener changes contract: `OnBackupsStorageChanged(ctx, databaseID, storageID)`, called after the config is saved and not before. `PhysicalBackupService` in `backups/backups/services` implements it, because it already holds `deleteFull`, the canceller and the core service. It deletes, through `deleteFull`, every full backup of the database that is kept in a storage other than `storageID`. `deleteFull` stops an in-flight backup of that chain and removes its incrementals, WAL and history under the retention locks. WAL that is left without an anchoring full backup goes through the existing orphan cleanup.

Streaming is switched separately. `BackupCancellationListener` gains `OnBackupsStorageChanged(ctx, databaseID)`. `PhysicalBackupCancellationListener` implements it with `cancelStreamingAndDeleteStreamerRow`, and `reconcile` then starts a streamer on the new storage. The config service calls this only when the storage really changed. It calls the deletion listener on every save of a config that names a storage, and the deletion does nothing when no backup lives outside that storage. A failed removal is therefore retried by the next save, as the spec requires. `backups_services.SetupDependencies` registers the listener next to the logical one (`services/di.go:79`).

Alternatives rejected:

- Deleting before the save, as the logical path does. A full backup or a streamer that starts between the deletion and the save writes to the old storage and survives the change.
- Deleting every physical row of the database directly, like `collectDatabaseFileReferences` does for database removal (`core/physical/service/database_removal_listener.go`). That bypasses the in-flight stop and the WAL retention locks, and it could remove WAL that a new full backup on the new storage needs.
- Keeping physical backups and changing the warning text. That contradicts what users were told and the logical behavior.

### 5. `Omit` before the finisher

Both repositories become `storage.GetDb().Omit(...).Save(backupConfig)`. Both services also reject an embedded storage object without an identifier. Today such an object skips the workspace check at `backups/config/logical/service.go:72-80` and `backups/config/physical/service.go:89-97`, because that check runs only for a non-nil ID, and the association upsert then creates a new storage row. With `Omit` in place nothing would be created, but the configuration would still be saved with a storage it was never given by identifier, so the request is refused instead. An unknown identifier stays rejected by the existing `GetStorageByID` lookup.

Alternative rejected: `Session(&gorm.Session{FullSaveAssociations: false})`. That is the default already and does not stop the association upsert; only `Omit` or `Select` does.

### 6. Status setters update their columns only

The databases repository gets a method that updates named columns of one database row by ID through `Model(&Database{}).Where("id = ?", id).Updates(map[string]any{...})`. The three setters call it with their own columns: `last_backup_error_message` for `SetBackupError`, `last_backup_time` and a cleared `last_backup_error_message` for `SetLastBackupTime`, and `health_status` for `SetHealthStatus`. A map is used so that clearing a column to `NULL` is not skipped as a zero value. When the update affects no row, the method returns an error that the database was not found. Today the setters fail in that case through `FindByID`, and their callers log the error, so this keeps the behavior.

Alternative rejected: locking the row with `SELECT ... FOR UPDATE` around the read-modify-write. Every user save path would need the same lock, and the setters do not need the rest of the row at all.

### 7. Delete the unused backup removal listener list

`LogicalBackupService.backupRemoveListeners`, `AddBackupRemoveListener` and the registration at `restores/di.go:48` are deleted, and the positional DI literal loses its slice. Restores stay subscribed to the cleaner.

Alternative rejected: making the service notify its own list. Restores would then be notified twice for every deletion, because the service already deletes through the cleaner.

### 8. Google Drive logs through the logger

The `fmt.Printf` calls that show token fragments, the token JSON or the forced expiry are deleted together with `maskSensitiveData` and `truncateString`, which only serve them. The refresh start, success and failure are logged at `Info` or `Warn` through a logger that `withRetryOnAuth` takes as a parameter. `TestConnection` passes `logger.GetLogger()`. The close error at `model.go:314` goes to the same logger.

Alternative rejected: keeping the prints with stronger masking. They would still bypass the logger's redaction and its sinks, which the Security section of `AGENTS.md` forbids.

### 9. Mutex around key loading

`SecretKeyService` gets a `sync.Mutex` held for the whole load-or-generate path, with the cache checked under the lock.

Alternative rejected: `sync.Once`. A transient read error would then be cached for the life of the process.

### 10. Webhook decryption errors surface at send time

`AfterFind` keeps decrypting the URL and body template, because the edit form receives them decrypted today. On failure it records the error in a field that GORM does not persist (`gorm:"-"`) instead of dropping it. `Send` returns that error before it builds a request. `decryptHeadersForSending` returns its error as well.

Alternatives rejected:

- Returning the error from `AfterFind`. GORM would then fail the whole query, so one bad notifier would break the notifier list of its workspace and every backup notification that loads notifiers.
- Moving decryption to send time, as the other notifiers do. The API would then return ciphertext to the edit form. That needs the URL hidden like Discord's (`discord/model.go:97`) and a frontend change, which is beyond this change.

### 11. Header and idle timeouts, no write deadline

`http.Server` gets `ReadHeaderTimeout: 10 * time.Second` and `IdleTimeout: 120 * time.Second`. It gets no `ReadTimeout` and no `WriteTimeout`, because both apply to the whole request or response, and long backup downloads would be cut.

Alternative rejected: a `WriteTimeout` with per-handler extensions through `http.ResponseController`. Every streaming handler would then have to remember to extend it, and forgetting breaks downloads silently.

### 12. JSON 404 under `/api/`

The `NoRoute` handler checks the `/api/` prefix first and answers `404` with `{"error": "not found"}`, the error shape the controllers already use (`storages/controller.go:51`). Every other unknown path keeps the frontend fallback.

Alternative rejected: a separate `NoRoute` per route group. Gin has a single `NoRoute` for the engine.

### 13. Hygiene

- `cmd/...` joins the `other` group in `test-group-packages` (`backend/Justfile:23`) and the `go test` package list in `scripts/test.sh:59`. CI takes its package list from that recipe (`.github/workflows/ci-release.yml:438`), so the workflow file needs no edit. `just test-package ./cmd/...` passes today, so adding the packages does not turn CI red.
- `test_once_protection.go` is renamed to `test_once_protection_test.go` with `git mv`. If one of its five tests fails, the fix goes into the code the test protects, and the test stays.
- The 25 `testing.Short()` branches are deleted. Nothing passes `-short`, and the testing skill prescribes no fast mode.
- A new goose migration drops both monitoring tables. Its `Down` recreates them as `migrations/20250912092352_add_monitoring_metrics.sql` defined them.
- `backend/cleanup_test_db` and `agent/verification/mock-server.exe` are removed with `git rm`. `cleanup_test_db` is added to `backend/.gitignore`, and `mock-server.exe` to `agent/verification/.gitignore`.
- `AGENTS.md:195` drops "`revive` fails CI" and keeps the naming rule.

Alternative rejected for the `cmd` packages: a fifth CI group. `ci-pipeline` names exactly four groups and sends every other package to `other`.

## Risks / Trade-offs

- [A physical full backup running at shutdown keeps the wait open until the 6-second deadline] → It still exits within the spec's limit, and the startup recovery records the backup as before. Today's drain may take 10 seconds.
- [The WAL streamers are registered in the same registry] → The supervisor stops all streamers when its context ends (`wal_supervisor.go:128-131`), and the background context is cancelled before the wait starts. A `backuping/physical` test cancels the supervisor's context with a running streamer and asserts that the registry wait returns before its deadline.
- [A streamer that has not stopped yet writes one more segment to the old storage after the change] → The segment has no anchoring full backup, so the physical cleaner's orphan sweep removes it.
- [Calling the physical deletion listener on every config save adds queries to each save] → Config saves are rare user actions, and the lookup filters by database ID.
- [The five tests in `test_once_protection.go` have never run and may fail] → The task that renames them runs them right away and fixes the protected code if needed.
- [The root `AGENTS.md` has an uncommitted change by someone else, outside line 195] → The edit touches only line 195. If that foreign change is still uncommitted when this change is packed, the agent asks the user before committing, as the "Other agents' changes" rule requires.
- [Dropping the monitoring tables removes any rows that old versions wrote] → No code has read or written those tables since they were created, and `Down` restores the schema.

## Migration Plan

Deployment needs no manual step. The goose migration runs at startup as usual. A rollback to the previous image runs the migration's `Down`, which recreates the empty tables. An API client that relied on `200` for an unknown `/api/` path has to handle `404`.
