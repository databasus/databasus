package backuping

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/backups/internal/runcontext"
)

type backupRunOptions struct {
	Backup         *catalog.Backup
	IsCallNotifier bool
}

func (b *Backuper) runBackup(ctx context.Context, options backupRunOptions) {
	backup := options.Backup
	logger := b.logger.With(slog.Any("backup_kind", backup.Kind))
	backupCtx, isOpened := b.openBackupContext(ctx, logger, backup.DatabaseID)
	if !isOpened {
		b.finalizeAsError(ctx, backup, catalog.BackupErrorPgBasebackupFailed, "failed to open backup context")
		return
	}
	defer backupCtx.Close()

	common := CommonBackupSpec{
		Publication: config.PublicationFence{
			Configs: b.configs,
			Publication: config.StoragePublication{
				DatabaseID: backup.DatabaseID,
				StorageID:  backup.StorageID,
				Generation: backup.StorageGeneration,
			},
		},
		ExecutionClock: b.executionClock,
		Clock:          b.clock,
		Transactor:     b.transactor,
		ClientTools:    b.clientTools,
		SourceDB:       backupCtx.Source,
		DatabaseName:   backupCtx.Database.Name,
		StorageID:      backupCtx.Storage.ID,
		FileStore:      b.storages,
		Encryption:     catalog.BackupEncryption(backupCtx.Config.Encryption),
		MasterKey:      backupCtx.MasterKey,
		BackupRepo:     b.backupRepo,
		HistoryRepo:    b.historyRepo,
		Logger:         logger,
		ProgressListener: func(completedMb float64, elapsedMs int64) {
			if err := b.backupRepo.UpdateProgress(ctx, backup.ID, completedMb, elapsedMs); err != nil {
				logger.ErrorContext(ctx, "failed to update physical backup progress", slog.Any("error", err))
			}
		},
	}

	var rawSizeMb *float64
	execution := backupExecutionSpec{CommonBackupSpec: common, Backup: backup}
	switch backup.Kind {
	case catalog.BackupKindFull:
		rawSizeMb = b.getSourceClusterSizeMb(ctx, logger, backupCtx.Source)
		sourceSize := "unknown"
		if rawSizeMb != nil {
			sourceSize = fmt.Sprintf("%.1f MB", *rawSizeMb)
		}

		logger.InfoContext(ctx, "physical full backup started",
			slog.Any("database_name", backupCtx.Database.Name), slog.Any("source_size", sourceSize),
			slog.Any("storage_id", backupCtx.Storage.ID), slog.Any("encryption", backupCtx.Config.Encryption))

	case catalog.BackupKindIncremental:
		parentRef, parentErr := b.resolveParentManifest(ctx, backup)
		if parentErr != nil {
			logger.ErrorContext(ctx, "failed to resolve parent manifest", slog.Any("error", parentErr))
			b.finalizeAsChainBroken(ctx, backup, catalog.BackupErrorParentManifestMissing, parentErr.Error())
			return
		}

		storageReader, storageErr := b.storages.GetStorageReader(ctx, backupCtx.Storage.ID)
		if storageErr != nil {
			logger.ErrorContext(ctx, "failed to open the storage of the parent manifest", slog.Any("error", storageErr))
			b.finalizeAsError(ctx, backup, catalog.BackupErrorPgBasebackupFailed,
				"failed to open the storage of the parent manifest")
			return
		}

		var incrementalCadence time.Duration
		if backupCtx.Config.IncrementalBackupSchedule != nil {
			incrementalCadence = backupCtx.Config.IncrementalBackupSchedule.GetApproxPeriod()
		}

		logger.InfoContext(ctx, "physical incremental backup started",
			slog.Any("database_name", backupCtx.Database.Name), slog.Any("parent_stop_lsn", parentRef.StopLSN),
			slog.Any("parent_backup_id", parentRef.BackupID), slog.Any("storage_id", backupCtx.Storage.ID))

		execution.ParentManifest = parentRef
		execution.StorageReader = storageReader
		execution.IncrementalCadence = incrementalCadence
	}

	backupResult := executeBackup(ctx, execution)

	if backup.Kind == catalog.BackupKindIncremental &&
		!errors.Is(context.Cause(ctx), runcontext.ErrBackupShutdown) &&
		backupResult.Status == catalog.BackupStatusError && b.isIncrRetryBudgetExhausted(ctx, logger, backup) {
		backupResult.Status = catalog.BackupStatusChainBroken
	}

	backupResult = classifyShutdown(ctx, backupResult)
	ctx = context.WithoutCancel(ctx)

	if backupResult.Status != catalog.BackupStatusCompleted {
		logger.WarnContext(ctx, "physical executor returned non-completed result",
			slog.Any("status", backupResult.Status), slog.Any("reason", reasonOrEmpty(backupResult.ErrorReason)),
			slog.Any("message", backupResult.ErrorMessage))
	}

	if err := b.persistResult(ctx, backup, backupResult, rawSizeMb); err != nil {
		logger.ErrorContext(ctx, "failed to persist physical result", slog.Any("error", err))
		return
	}

	logger.InfoContext(ctx, "physical backup finished",
		slog.Any("status", backupResult.Status), slog.Any("size_mb", backupResult.BackupSizeMb),
		slog.Any("duration_ms", backupResult.BackupDurationMs), slog.Any("timeline_id", backupResult.TimelineID),
		slog.Any("start_lsn", backupResult.StartLSN), slog.Any("stop_lsn", backupResult.StopLSN))

	if options.IsCallNotifier {
		b.sendBackupNotification(ctx, backupCtx.Config, backupCtx.Database, backup, backupResult)
	}
}
