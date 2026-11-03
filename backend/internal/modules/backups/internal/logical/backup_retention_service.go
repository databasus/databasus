package logical

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/modules/backups/internal/config"
)

const (
	RetentionJobName  = "backups_logical_retention"
	retentionInterval = time.Minute
)

func (s *Service) GetRetentionJob() background.Job {
	return background.Job{Name: RetentionJobName, Interval: retentionInterval, Run: s.runRetentionPass}
}

// runRetentionPass removes, per enabled config, the backups its retention no longer keeps. A backup
// that still runs or that a running restore reads stays until a later pass. A database whose removal
// fails is logged and the pass goes on with the next one.
func (s *Service) runRetentionPass(ctx context.Context, now time.Time) error {
	enabledConfigs, err := s.configs.ListEnabledLogicalBackupConfigs(ctx)
	if err != nil {
		return err
	}

	for _, enabledConfig := range enabledConfigs {
		err := s.applyRetention(ctx, enabledConfig, now)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to apply the retention of logical backups",
				slog.Any("database_id", enabledConfig.DatabaseID), slog.Any("error", err))
		}
	}

	return nil
}

func (s *Service) applyRetention(ctx context.Context, enabledConfig config.LogicalBackupConfig, now time.Time) error {
	databaseBackups, err := s.repository.listByDatabase(ctx, enabledConfig.DatabaseID)
	if err != nil {
		return err
	}

	for _, expiredBackup := range getExpiredBackups(enabledConfig.Retention, databaseBackups, now) {
		if expiredBackup.Status == StatusInProgress {
			continue
		}

		isRemoved, err := s.removeExpiredBackup(ctx, expiredBackup.ID)
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if err != nil {
			s.logger.ErrorContext(ctx, "failed to remove an expired logical backup",
				slog.Any("backup_id", expiredBackup.ID),
				slog.Any("database_id", expiredBackup.DatabaseID),
				slog.Any("error", err))

			continue
		}

		if isRemoved {
			s.logger.InfoContext(ctx, "logical backup removed by retention",
				slog.Any("backup_id", expiredBackup.ID),
				slog.Any("database_id", expiredBackup.DatabaseID),
				slog.Any("policy", enabledConfig.Retention.Policy))
		}
	}

	return nil
}

// removeExpiredBackup takes the lock a single-backup delete takes, so a restore that starts meanwhile
// either commits first and keeps the backup or finds it gone.
func (s *Service) removeExpiredBackup(ctx context.Context, backupID uuid.UUID) (bool, error) {
	isRemoved := false

	err := s.transactor.Transaction(ctx, func(ctx context.Context) error {
		lockedBackup, isFound, err := s.repository.getForUpdate(ctx, backupID)
		if err != nil || !isFound || lockedBackup.Status == StatusInProgress {
			return err
		}

		isReadByRestore, err := s.restores.IsRestoreRunningForBackups(ctx, []uuid.UUID{backupID})
		if err != nil || isReadByRestore {
			return err
		}

		err = s.deleteBackupsWithFiles(ctx, []uuid.UUID{backupID})
		if err != nil {
			return err
		}

		isRemoved = true

		return nil
	})

	return isRemoved, err
}
