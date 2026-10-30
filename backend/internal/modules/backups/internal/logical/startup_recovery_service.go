package logical

import (
	"context"
	"log/slog"

	"databasus-backend/internal/modules/backups/internal/config"
)

const restartFailMessage = "Backup failed due to application restart"

// Start keeps the application's context, from which every run derives its own, and fails the
// backups the previous run left in progress: their files go to the deletion worker in the same
// transaction, and the failure notifications follow the commit.
func (s *Service) Start(appCtx context.Context) error {
	s.appCtx.Store(&appCtx)

	var failedBackups []LogicalBackup

	err := s.transactor.Transaction(appCtx, func(ctx context.Context) error {
		var err error

		failedBackups, err = s.repository.failAllInProgress(ctx, restartFailMessage)
		if err != nil {
			return err
		}

		return s.requestFileDeletions(ctx, failedBackups)
	})
	if err != nil {
		return err
	}

	if len(failedBackups) > 0 {
		s.logger.InfoContext(appCtx, "failed the logical backups left in progress by the previous run",
			slog.Int("count", len(failedBackups)))
	}

	for _, failedBackup := range failedBackups {
		s.notifyRestartFailure(appCtx, failedBackup)
	}

	return nil
}

func (s *Service) notifyRestartFailure(ctx context.Context, failedBackup LogicalBackup) {
	backupDatabase, err := s.getDatabase(ctx, failedBackup.DatabaseID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to read the database of a backup failed by the restart",
			slog.Any("backup_id", failedBackup.ID), slog.Any("error", err))

		return
	}

	backupConfig, err := s.configs.GetLogicalBackupConfig(ctx, failedBackup.DatabaseID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to read the config of a backup failed by the restart",
			slog.Any("backup_id", failedBackup.ID), slog.Any("error", err))

		return
	}

	failMessage := restartFailMessage
	failedBackup.Status = StatusFailed
	failedBackup.FailMessage = &failMessage

	s.notifyBackupOutcome(ctx, failedBackup, backupDatabase, backupConfig, config.NotificationTriggerBackupFailed)
}
