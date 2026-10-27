package restoring

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	backups_services "databasus-backend/internal/features/backups/backups/services"
	databases_bridge "databasus-backend/internal/features/databases"
	restores_core "databasus-backend/internal/features/restores/core"
	tasks_cancellation "databasus-backend/internal/features/tasks/cancellation"
	"databasus-backend/internal/lib/cache"
	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/storages"
)

type Restorer struct {
	databaseAccess *databases_bridge.Access
	// getDatabases is read per call: each test application installs its own module.
	getDatabases      func() databases.Module
	backupService     *backups_services.LogicalBackupService
	restoreRepository *restores_core.RestoreRepository
	getBackups        func() backups.Module
	// getStorages is read per call: each test application installs its own module.
	getStorages              func() storages.Module
	logger                   *slog.Logger
	restoreBackupUsecase     restores_core.RestoreBackupUsecase
	restoreDatabaseCache     *cache.JSONStore[RestoreDatabaseCache]
	taskCancellationRegistry *tasks_cancellation.Registry
	shutdownSignal           <-chan struct{}
	clientToolsDir           string
}

type restoreMetadataLookup struct {
	restoreID             uuid.UUID
	fallbackDatabaseCache *RestoreDatabaseCache
}

func (r *Restorer) MakeRestore(
	ctx context.Context,
	restoreID uuid.UUID,
	fallbackDatabaseCache *RestoreDatabaseCache,
) {
	logger := r.logger.With("restore_id", restoreID)

	restore, err := r.restoreRepository.FindByID(restoreID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get restore by ID", "error", err)
		return
	}

	databaseCache := r.getRestoreDatabaseCache(ctx, restoreMetadataLookup{
		restoreID:             restore.ID,
		fallbackDatabaseCache: fallbackDatabaseCache,
	}, logger)
	if !databaseCache.HasDatabaseConfiguration() {
		failMessage := "Database credentials expired or missing from cache (most likely due to instance restart)"
		restore.FailMessage = &failMessage
		restore.Status = restores_core.RestoreStatusFailed
		if err := r.restoreRepository.Save(restore); err != nil {
			logger.ErrorContext(ctx, "failed to save restore after missing metadata", "error", err)
		}

		logger.ErrorContext(ctx, "restore failed because database credentials are missing")

		return
	}

	backup, err := r.backupService.GetBackup(restore.BackupID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get backup by ID", "backup_id", restore.BackupID, "error", err)
		return
	}

	databaseID := backup.DatabaseID

	database, err := r.databaseAccess.GetDatabaseByID(ctx, databaseID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get database by ID", "database_id", databaseID, "error", err)
		return
	}

	source, err := restores_core.ReadSourceDatabase(ctx, r.getDatabases(), database)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"failed to read the source database's settings",
			"database_id",
			databaseID,
			"error",
			err,
		)
		return
	}

	backupConfig, err := r.getBackups().GetLogicalBackupConfig(ctx, databaseID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get backup config by database ID", "error", err)
		return
	}

	// Detached from the caller so a finished HTTP request cannot cancel a running restore.
	executionCtx, cancel := context.WithCancel(context.Background())
	r.taskCancellationRegistry.RegisterTask(restore.ID, cancel)
	defer r.taskCancellationRegistry.UnregisterTask(restore.ID)

	restoreCtx, cancelRestore := restores_core.NewRestoreContext(executionCtx, r.shutdownSignal)
	defer cancelRestore(nil)

	storageReader, err := r.getStorages().GetStorageReader(restoreCtx, backup.StorageID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to open the backup's storage", "storage_id", backup.StorageID, "error", err)
		return
	}

	start := time.Now().UTC()

	// The restore target is the only endpoint this function connects to, so one tunnel covers
	// version detection, the TimescaleDB hooks and pg_restore itself.
	targetConnection, err := restores_core.OpenTargetConnection(restoreCtx, restores_core.TargetConnectionSpec{
		Type:           database.Type,
		Request:        databaseCache.getRequest(),
		ClientToolsDir: r.clientToolsDir,
		Logger:         logger,
	})
	if err != nil {
		errMsg := fmt.Sprintf("failed to open the SSH tunnel to the restore target: %v", err)
		restore.FailMessage = &errMsg
		restore.Status = restores_core.RestoreStatusFailed
		restore.RestoreDurationMs = time.Since(start).Milliseconds()

		logger.ErrorContext(ctx, "restore failed to open the ssh tunnel to the target", "error", err)

		if err := r.restoreRepository.Save(restore); err != nil {
			logger.ErrorContext(ctx, "failed to save restore", "error", err)
		}

		return
	}

	defer targetConnection.Close()

	if err := targetConnection.DetectVersion(ctx); err != nil {
		errMsg := fmt.Sprintf("failed to auto-detect database data: %v", err)
		restore.FailMessage = &errMsg
		restore.Status = restores_core.RestoreStatusFailed
		restore.RestoreDurationMs = time.Since(start).Milliseconds()

		logger.ErrorContext(ctx, "restore failed to auto-detect target database data", "error", err)

		if err := r.restoreRepository.Save(restore); err != nil {
			logger.ErrorContext(ctx, "failed to save restore", "error", err)
		}

		return
	}

	// IsExcludeExtensions is a transient choice carried on the target config from the restore
	// request; IsSkipUserMappings is a persisted property of the source database being restored.
	restoreOptions := restores_core.RestoreOptions{}
	if databaseCache.PostgresqlLogicalDatabase != nil {
		restoreOptions.IsExcludeExtensions = databaseCache.PostgresqlLogicalDatabase.IsExcludeExtensions
	}
	if source.PostgresqlLogical != nil {
		restoreOptions.IsSkipUserMappings = source.PostgresqlLogical.IsSkipUserMappings
	}

	logger.InfoContext(ctx, fmt.Sprintf("restore started: %s database %q", database.Type, database.Name),
		"backup_id", backup.ID, "database_id", databaseID, "storage_id", backup.StorageID)

	err = r.restoreBackupUsecase.Execute(
		restoreCtx,
		backupConfig,
		*restore,
		source,
		targetConnection.Target,
		backup,
		storageReader,
		restoreOptions,
	)
	if err != nil {
		errMsg := err.Error()

		switch {
		case errors.Is(context.Cause(restoreCtx), restores_core.ErrRestoreShutdown):
			errMsg = restores_core.ErrRestoreShutdown.Error()
		case executionCtx.Err() != nil:
			logger.WarnContext(ctx, "restore was cancelled by user")

			restore.Status = restores_core.RestoreStatusCanceled
			restore.RestoreDurationMs = time.Since(start).Milliseconds()

			if err := r.restoreRepository.Save(restore); err != nil {
				logger.ErrorContext(ctx, "failed to save cancelled restore", "error", err)
			}

			return
		}

		logger.ErrorContext(ctx, "restore execution failed",
			"backup_id", backup.ID,
			"database_id", databaseID,
			"database_type", database.Type,
			"storage_id", backup.StorageID,
			"error", err,
		)

		restore.FailMessage = &errMsg
		restore.Status = restores_core.RestoreStatusFailed
		restore.RestoreDurationMs = time.Since(start).Milliseconds()

		if err := r.restoreRepository.Save(restore); err != nil {
			logger.ErrorContext(ctx, "failed to save restore", "error", err)
		}

		return
	}

	restore.Status = restores_core.RestoreStatusCompleted
	restore.RestoreDurationMs = time.Since(start).Milliseconds()

	if err := r.restoreRepository.Save(restore); err != nil {
		logger.ErrorContext(ctx, "failed to save restore", "error", err)
		return
	}

	logger.InfoContext(ctx, fmt.Sprintf("restore finished in %d ms", restore.RestoreDurationMs),
		"backup_id", backup.ID)
}

func (r *Restorer) getRestoreDatabaseCache(
	ctx context.Context,
	lookup restoreMetadataLookup,
	logger *slog.Logger,
) *RestoreDatabaseCache {
	databaseCache, err := r.restoreDatabaseCache.ReadAndDelete(ctx, lookup.restoreID.String())
	if err != nil {
		logger.WarnContext(ctx, "failed to read restore metadata from cache", "error", err)
	}
	if databaseCache != nil {
		return databaseCache
	}
	if lookup.fallbackDatabaseCache != nil {
		return lookup.fallbackDatabaseCache
	}

	return &RestoreDatabaseCache{}
}
