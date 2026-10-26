package backuping_logical

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/features/backups/backups/backuping/shared/gfs"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/storages"
	db "databasus-backend/internal/storage"
	"databasus-backend/internal/util/period"
)

const (
	cleanerTickerInterval = 3 * time.Second

	cleanerJobName          = "logical_backup_retention_cleanup"
	recentBackupGracePeriod = 60 * time.Minute
)

type BackupCleaner struct {
	backupRepository *backups_core_logical.BackupRepository
	// getStorages is read per call: each test application installs its own module.
	getStorages           func() storages.Module
	backupConfigService   *backups_config_logical.BackupConfigService
	logger                *slog.Logger
	backupRemoveListeners []backups_core_logical.BackupRemoveListener

	hasRun atomic.Bool
}

func (c *BackupCleaner) Run(ctx context.Context) {
	if c.hasRun.Swap(true) {
		panic(fmt.Sprintf("%T.Run() called multiple times", c))
	}

	if ctx.Err() != nil {
		return
	}

	ticker := time.NewTicker(cleanerTickerInterval)
	defer ticker.Stop()

	lifecycleLogger := c.logger.With("job_name", cleanerJobName)

	lifecycleLogger.InfoContext(ctx, "logical backup retention cleaner started")

	for {
		select {
		case <-ctx.Done():
			lifecycleLogger.InfoContext(ctx, "logical backup retention cleaner stopped")

			return
		case <-ticker.C:
			retentionLog := c.logger.With(
				"job_id", uuid.New(),
				"job_name", cleanerJobName,
				"task_name", "clean_by_retention_policy",
			)

			if err := c.cleanByRetentionPolicy(ctx, retentionLog); err != nil {
				retentionLog.ErrorContext(ctx, "failed to clean backups by retention policy", "error", err)
			}
		}
	}
}

func (c *BackupCleaner) DeleteBackup(ctx context.Context, backup *backups_core_logical.LogicalBackup) error {
	for _, listener := range c.backupRemoveListeners {
		if err := listener.OnBeforeBackupRemove(backup); err != nil {
			return err
		}
	}

	references := []storages.StoredFileReference{
		{StorageID: backup.StorageID, FileName: backup.FileName},
		{StorageID: backup.StorageID, FileName: backup.FileName + metadataSuffix},
	}

	// The row and the obligation to remove its files commit together, so an
	// unreachable storage does not decide whether the row may go, and a rolled
	// back removal leaves the files where the row still names them.
	return db.GetTransactor().Transaction(ctx, func(ctx context.Context) error {
		tx := database.Conn(ctx, db.GetDb())

		if err := c.getStorages().RequestFileDeletions(ctx, references); err != nil {
			return err
		}

		return c.backupRepository.DeleteByIDInTransaction(tx, backup.ID)
	})
}

func (c *BackupCleaner) AddBackupRemoveListener(listener backups_core_logical.BackupRemoveListener) {
	c.backupRemoveListeners = append(c.backupRemoveListeners, listener)
}

func (c *BackupCleaner) cleanByRetentionPolicy(ctx context.Context, logger *slog.Logger) error {
	enabledBackupConfigs, err := c.backupConfigService.GetBackupConfigsWithEnabledBackups()
	if err != nil {
		return err
	}

	for _, backupConfig := range enabledBackupConfigs {
		dbLog := logger.With("database_id", backupConfig.DatabaseID, "policy", backupConfig.RetentionPolicyType)

		var cleanErr error

		switch backupConfig.RetentionPolicyType {
		case backups_config_logical.RetentionPolicyTypeCount:
			cleanErr = c.cleanByCount(ctx, dbLog, backupConfig)
		case backups_config_logical.RetentionPolicyTypeGFS:
			cleanErr = c.cleanByGFS(ctx, dbLog, backupConfig)
		default:
			cleanErr = c.cleanByTimePeriod(ctx, dbLog, backupConfig)
		}

		if cleanErr != nil {
			dbLog.ErrorContext(ctx, "failed to clean backups by retention policy", "error", cleanErr)
		}
	}

	return nil
}

func (c *BackupCleaner) cleanByTimePeriod(
	ctx context.Context,
	logger *slog.Logger,
	backupConfig *backups_config_logical.LogicalBackupConfig,
) error {
	if backupConfig.RetentionTimePeriod == "" || backupConfig.RetentionTimePeriod == period.PeriodForever {
		logger.DebugContext(ctx, "time-period retention keeps everything, nothing to evaluate")

		return nil
	}

	cutoff := time.Now().UTC().Add(-backupConfig.RetentionTimePeriod.ToDuration())

	oldBackups, err := c.backupRepository.FindBackupsBeforeDate(backupConfig.DatabaseID, cutoff)
	if err != nil {
		return fmt.Errorf("failed to find old backups for database %s: %w", backupConfig.DatabaseID, err)
	}

	deletedCount, withinGraceCount := 0, 0

	for _, backup := range oldBackups {
		if isRecentBackup(backup) {
			withinGraceCount++

			logger.DebugContext(ctx, fmt.Sprintf("keeping backup, still within the %s grace period",
				recentBackupGracePeriod), "backup_id", backup.ID)

			continue
		}

		if err := c.DeleteBackup(ctx, backup); err != nil {
			logger.ErrorContext(ctx, "failed to delete backup", "backup_id", backup.ID, "error", err)

			continue
		}

		deletedCount++

		logger.InfoContext(ctx, fmt.Sprintf("deleted backup older than %s (%.1f MB)",
			backupConfig.RetentionTimePeriod, backup.BackupSizeMb), "backup_id", backup.ID)
	}

	summary := fmt.Sprintf("time-period retention: %d backups past the %s cutoff, %d deleted, %d within grace",
		len(oldBackups), backupConfig.RetentionTimePeriod, deletedCount, withinGraceCount)

	// The cleaner ticks every few seconds, so a summary of a run that changed nothing belongs at
	// debug; only an actual deletion is a state change.
	if deletedCount > 0 {
		logger.InfoContext(ctx, summary)
	} else {
		logger.DebugContext(ctx, summary)
	}

	return nil
}

func (c *BackupCleaner) cleanByCount(
	ctx context.Context,
	logger *slog.Logger,
	backupConfig *backups_config_logical.LogicalBackupConfig,
) error {
	if backupConfig.RetentionCount <= 0 {
		logger.DebugContext(ctx, "count retention is not configured, nothing to evaluate")

		return nil
	}

	completedBackups, err := c.findCompletedBackups(backupConfig.DatabaseID)
	if err != nil {
		return err
	}

	if len(completedBackups) <= backupConfig.RetentionCount {
		logger.DebugContext(ctx, fmt.Sprintf("count retention: keeping all %d backups, limit is %d",
			len(completedBackups), backupConfig.RetentionCount))

		return nil
	}

	deletedCount, withinGraceCount := 0, 0

	for _, backup := range completedBackups[backupConfig.RetentionCount:] {
		if isRecentBackup(backup) {
			withinGraceCount++

			logger.DebugContext(ctx, fmt.Sprintf("keeping backup, still within the %s grace period",
				recentBackupGracePeriod), "backup_id", backup.ID)

			continue
		}

		if err := c.DeleteBackup(ctx, backup); err != nil {
			logger.ErrorContext(ctx, "failed to delete backup", "backup_id", backup.ID, "error", err)

			continue
		}

		deletedCount++

		logger.InfoContext(ctx, fmt.Sprintf("deleted backup past the retention count of %d (%.1f MB)",
			backupConfig.RetentionCount, backup.BackupSizeMb), "backup_id", backup.ID)
	}

	summary := fmt.Sprintf("count retention: %d completed backups, %d kept by limit, %d deleted, %d within grace",
		len(completedBackups), backupConfig.RetentionCount, deletedCount, withinGraceCount)

	if deletedCount > 0 {
		logger.InfoContext(ctx, summary)
	} else {
		logger.DebugContext(ctx, summary)
	}

	return nil
}

func (c *BackupCleaner) cleanByGFS(
	ctx context.Context,
	logger *slog.Logger,
	backupConfig *backups_config_logical.LogicalBackupConfig,
) error {
	if backupConfig.RetentionGfsHours <= 0 && backupConfig.RetentionGfsDays <= 0 &&
		backupConfig.RetentionGfsWeeks <= 0 && backupConfig.RetentionGfsMonths <= 0 &&
		backupConfig.RetentionGfsYears <= 0 {
		// Every tier is zero, so GFS retains nothing and would delete every backup if it ran. Debug
		// rather than Warn: an empty config is a deliberate state, and this runs on every tick.
		logger.DebugContext(ctx, "gfs retention has no tier configured, keeping everything")

		return nil
	}

	completedBackups, err := c.findCompletedBackups(backupConfig.DatabaseID)
	if err != nil {
		return err
	}

	retainedTiersByBackupID := buildRetainedTiersByBackupID(
		completedBackups,
		backupConfig.RetentionGfsHours,
		backupConfig.RetentionGfsDays,
		backupConfig.RetentionGfsWeeks,
		backupConfig.RetentionGfsMonths,
		backupConfig.RetentionGfsYears,
	)

	deletedCount, withinGraceCount := 0, 0

	for _, backup := range completedBackups {
		if tiers := retainedTiersByBackupID[backup.ID]; len(tiers) > 0 {
			logger.DebugContext(ctx, fmt.Sprintf("keeping backup, retained as %s", gfs.FormatTiers(tiers)),
				"backup_id", backup.ID)

			continue
		}

		if isRecentBackup(backup) {
			withinGraceCount++

			logger.DebugContext(ctx, fmt.Sprintf("keeping backup, still within the %s grace period",
				recentBackupGracePeriod), "backup_id", backup.ID)

			continue
		}

		if err := c.DeleteBackup(ctx, backup); err != nil {
			logger.ErrorContext(ctx, "failed to delete backup", "backup_id", backup.ID, "error", err)

			continue
		}

		deletedCount++

		logger.InfoContext(ctx, fmt.Sprintf("deleted backup not retained by any gfs tier (%.1f MB)",
			backup.BackupSizeMb), "backup_id", backup.ID)
	}

	summary := fmt.Sprintf(
		"gfs retention (%dh/%dd/%dw/%dm/%dy): %d completed backups, %d kept by tier, %d deleted, %d within grace",
		backupConfig.RetentionGfsHours, backupConfig.RetentionGfsDays, backupConfig.RetentionGfsWeeks,
		backupConfig.RetentionGfsMonths, backupConfig.RetentionGfsYears,
		len(completedBackups), len(retainedTiersByBackupID), deletedCount, withinGraceCount)

	if deletedCount > 0 {
		logger.InfoContext(ctx, summary)
	} else {
		logger.DebugContext(ctx, summary)
	}

	return nil
}

func (c *BackupCleaner) findCompletedBackups(databaseID uuid.UUID) ([]*backups_core_logical.LogicalBackup, error) {
	completed, err := c.backupRepository.FindByDatabaseIdAndStatus(
		databaseID,
		backups_core_logical.BackupStatusCompleted,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to find completed backups for database %s: %w", databaseID, err)
	}

	return completed, nil
}

func isRecentBackup(backup *backups_core_logical.LogicalBackup) bool {
	return time.Since(backup.CreatedAt) < recentBackupGracePeriod
}

func buildRetainedTiersByBackupID(
	backups []*backups_core_logical.LogicalBackup,
	hours, days, weeks, months, years int,
) map[uuid.UUID][]gfs.Tier {
	items := make([]gfs.Item, len(backups))
	for i, backup := range backups {
		items[i] = gfs.Item{ID: backup.ID, CreatedAt: backup.CreatedAt}
	}

	return gfs.GetItemsToRetain(items, hours, days, weeks, months, years)
}
