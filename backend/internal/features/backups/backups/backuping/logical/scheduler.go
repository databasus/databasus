package backuping_logical

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	databases_bridge "databasus-backend/internal/features/databases"
	task_cancellation "databasus-backend/internal/features/tasks/cancellation"
	"databasus-backend/internal/modules/databases"
)

const (
	schedulerTickerInterval = 15 * time.Second

	// Stable log job_name; never the struct type name, since a rename would silently break log
	// queries.
	schedulerJobName              = "logical_backup_scheduler"
	schedulerHealthcheckThreshold = 5 * time.Minute
)

type BackupsScheduler struct {
	backupRepository          *backups_core_logical.BackupRepository
	backupConfigService       *backups_config_logical.BackupConfigService
	taskCancellationRequester *task_cancellation.Requester
	databaseAccess            *databases_bridge.Access

	lastBackupTime time.Time
	logger         *slog.Logger

	backuper *Backuper

	backupCompletionListeners []backups_core_logical.BackupCompletionListener

	hasRun  atomic.Bool
	isReady atomic.Bool
}

// IsRunning reports whether the scheduler has finished startup recovery and is
// ready to dispatch work. Tests use it to gate the start of work.
func (s *BackupsScheduler) IsRunning() bool {
	return s.isReady.Load()
}

func (s *BackupsScheduler) AddBackupCompletionListener(
	listener backups_core_logical.BackupCompletionListener,
) {
	s.backupCompletionListeners = append(s.backupCompletionListeners, listener)
}

func (s *BackupsScheduler) Run(ctx context.Context) {
	if s.hasRun.Swap(true) {
		panic(fmt.Sprintf("%T.Run() called multiple times", s))
	}

	s.lastBackupTime = time.Now().UTC()

	startupRecoveryLogger := s.logger.With("job_id", uuid.New(), "job_name", schedulerJobName)
	lifecycleLogger := s.logger.With("job_name", schedulerJobName)

	if err := s.failBackupsInProgress(ctx, startupRecoveryLogger); err != nil {
		startupRecoveryLogger.ErrorContext(ctx, "failed to fail backups in progress", "error", err)

		panic(err)
	}

	s.isReady.Store(true)
	defer s.isReady.Store(false)

	if ctx.Err() != nil {
		return
	}

	ticker := time.NewTicker(schedulerTickerInterval)
	defer ticker.Stop()

	lifecycleLogger.InfoContext(ctx, "logical backup scheduler started")

	for {
		select {
		case <-ctx.Done():
			lifecycleLogger.InfoContext(ctx, "logical backup scheduler stopped")

			return
		case <-ticker.C:
			tickLogger := s.logger.With("job_id", uuid.New(), "job_name", schedulerJobName)

			if err := s.runPendingBackups(ctx, tickLogger); err != nil {
				tickLogger.ErrorContext(ctx, "failed to run pending backups", "error", err)
			}

			s.lastBackupTime = time.Now().UTC()
		}
	}
}

func (s *BackupsScheduler) IsSchedulerRunning() bool {
	// if last backup time is more than 5 minutes ago, return false
	return s.lastBackupTime.After(time.Now().UTC().Add(-schedulerHealthcheckThreshold))
}

func (s *BackupsScheduler) StartBackup(ctx context.Context, database databases.Database, isCallNotifier bool) {
	backupConfig, err := s.backupConfigService.GetBackupConfigByDbId(database.ID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get backup config by database ID", "error", err)
		return
	}

	if backupConfig.StorageID == nil {
		s.logger.ErrorContext(ctx, "backup config storage ID is nil", "database_id", database.ID)
		return
	}

	// Check for existing in-progress backups
	inProgressBackups, err := s.backupRepository.FindByDatabaseIdAndStatus(
		database.ID,
		backups_core_logical.BackupStatusInProgress,
	)
	if err != nil {
		s.logger.ErrorContext(ctx,
			"failed to check for in-progress backups",
			"database_id",
			database.ID,
			"error",
			err,
		)
		return
	}

	if len(inProgressBackups) > 0 {
		s.logger.WarnContext(ctx,
			"backup already in progress for database, skipping new backup",
			"database_id",
			database.ID,
			"existingBackupId",
			inProgressBackups[0].ID,
		)
		return
	}

	backupID := uuid.New()
	timestamp := time.Now().UTC()

	backup := &backups_core_logical.LogicalBackup{
		ID:           backupID,
		DatabaseID:   backupConfig.DatabaseID,
		StorageID:    *backupConfig.StorageID,
		Status:       backups_core_logical.BackupStatusInProgress,
		BackupSizeMb: 0,
		CreatedAt:    timestamp,
	}

	backup.GenerateFilename(database.Name)

	if err := s.backupRepository.Save(backup); err != nil {
		s.logger.ErrorContext(ctx,
			"failed to save backup",
			"database_id",
			backupConfig.DatabaseID,
			"error",
			err,
		)
		return
	}

	go func() {
		// Detached so a finished HTTP request cannot cancel completion work such as
		// sending notifications; context values (request_id) survive for logging.
		s.backuper.MakeBackup(context.WithoutCancel(ctx), backup.ID, isCallNotifier)
		s.onBackupCompleted(backup.ID)
	}()

	s.logger.InfoContext(ctx,
		"successfully triggered scheduled backup",
		"database_id",
		backupConfig.DatabaseID,
		"backup_id",
		backup.ID,
	)
}

// GetRemainedBackupTryCount returns the number of remaining backup tries for a given backup.
// If the backup is not failed or the backup config does not allow retries, it returns 0.
// If the backup is failed and the backup config allows retries, it returns the number of remaining tries.
// If the backup is failed and the backup config does not allow retries, it returns 0.
func (s *BackupsScheduler) GetRemainedBackupTryCount(
	ctx context.Context,
	logger *slog.Logger,
	lastBackup *backups_core_logical.LogicalBackup,
) int {
	if lastBackup == nil || lastBackup.Status != backups_core_logical.BackupStatusFailed {
		return 0
	}

	logger = logger.With("backup_id", lastBackup.ID, "database_id", lastBackup.DatabaseID)

	if lastBackup.IsSkipRetry {
		logger.DebugContext(ctx, "not retrying the last failed backup, it is marked skip-retry")

		return 0
	}

	backupConfig, err := s.backupConfigService.GetBackupConfigByDbId(lastBackup.DatabaseID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get backup config by database ID", "error", err)

		return 0
	}

	if !backupConfig.IsRetryIfFailed {
		logger.DebugContext(ctx, "not retrying the last failed backup, retries are disabled")

		return 0
	}

	maxFailedTriesCount := backupConfig.MaxFailedTriesCount

	lastBackups, err := s.backupRepository.FindByDatabaseIDWithLimit(
		lastBackup.DatabaseID,
		maxFailedTriesCount,
	)
	if err != nil {
		logger.ErrorContext(ctx, "failed to find last backups by database ID", "error", err)

		return 0
	}

	lastFailedBackups := make([]*backups_core_logical.LogicalBackup, 0)

	for _, backup := range lastBackups {
		if backup.Status == backups_core_logical.BackupStatusFailed {
			lastFailedBackups = append(lastFailedBackups, backup)
		}
	}

	remainedTryCount := maxFailedTriesCount - len(lastFailedBackups)

	if remainedTryCount <= 0 {
		logger.InfoContext(ctx, fmt.Sprintf(
			"not retrying the last failed backup, all %d tries are used up", maxFailedTriesCount))
	} else {
		logger.DebugContext(ctx, fmt.Sprintf("%d of %d backup retries remain",
			remainedTryCount, maxFailedTriesCount))
	}

	return remainedTryCount
}

func (s *BackupsScheduler) runPendingBackups(ctx context.Context, logger *slog.Logger) error {
	enabledBackupConfigs, err := s.backupConfigService.GetBackupConfigsWithEnabledBackups()
	if err != nil {
		return err
	}

	for _, backupConfig := range enabledBackupConfigs {
		lastBackup, err := s.backupRepository.FindLastByDatabaseID(backupConfig.DatabaseID)
		if err != nil {
			logger.ErrorContext(ctx,
				"failed to get last backup for database",
				"database_id",
				backupConfig.DatabaseID,
				"error",
				err,
			)
			continue
		}

		var lastBackupTime *time.Time
		if lastBackup != nil {
			lastBackupTime = &lastBackup.CreatedAt
		}

		remainedBackupTryCount := s.GetRemainedBackupTryCount(ctx, logger, lastBackup)

		if backupConfig.BackupInterval.ShouldTriggerBackup(time.Now().UTC(), lastBackupTime) ||
			remainedBackupTryCount > 0 {
			logger.InfoContext(ctx,
				"triggering scheduled backup",
				"database_id",
				backupConfig.DatabaseID,
				"intervalType",
				backupConfig.BackupInterval.Type,
			)

			database, err := s.databaseAccess.GetDatabaseByID(ctx, backupConfig.DatabaseID)
			if err != nil {
				logger.ErrorContext(ctx, "failed to get database by ID", "error", err)
				continue
			}

			s.StartBackup(ctx, database, remainedBackupTryCount == 1)
			continue
		}
	}

	return nil
}

func (s *BackupsScheduler) failBackupsInProgress(ctx context.Context, logger *slog.Logger) error {
	backupsInProgress, err := s.backupRepository.FindByStatus(backups_core_logical.BackupStatusInProgress)
	if err != nil {
		return err
	}

	if len(backupsInProgress) > 0 {
		logger.InfoContext(ctx, fmt.Sprintf(
			"failing %d logical backups left in progress by the previous run", len(backupsInProgress)))
	}

	for _, backup := range backupsInProgress {
		if err := s.taskCancellationRequester.RequestCancellation(ctx, backup.ID); err != nil {
			logger.ErrorContext(ctx,
				"failed to request backup cancellation",
				"backup_id",
				backup.ID,
				"error",
				err,
			)
		}

		backupConfig, err := s.backupConfigService.GetBackupConfigByDbId(backup.DatabaseID)
		if err != nil {
			logger.ErrorContext(ctx, "failed to get backup config by database ID", "error", err)
			continue
		}

		failMessage := "Backup failed due to application restart"
		backup.FailMessage = &failMessage
		backup.Status = backups_core_logical.BackupStatusFailed
		backup.BackupSizeMb = 0

		s.backuper.SendBackupNotification(ctx,
			backupConfig,
			backup,
			backups_config_logical.NotificationBackupFailed,
			&failMessage,
		)

		// The row already names a file, and the previous run may have uploaded it,
		// so the terminal state and the obligation commit together.
		if err := s.backuper.saveTerminalStateAndDiscardFiles(ctx, backup); err != nil {
			return err
		}

		logger.InfoContext(ctx, "failed a logical backup left in progress by the previous run",
			"backup_id", backup.ID, "database_id", backup.DatabaseID)
	}

	return nil
}

func (s *BackupsScheduler) onBackupCompleted(backupID uuid.UUID) {
	for _, listener := range s.backupCompletionListeners {
		go listener.OnBackupCompleted(backupID)
	}
}
