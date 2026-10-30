package logical

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/modules/backups/internal/config"
)

const (
	SchedulerJobName  = "backups_logical_scheduler"
	schedulerInterval = 15 * time.Second
)

func (s *Service) GetSchedulerJob() background.Job {
	return background.Job{Name: SchedulerJobName, Interval: schedulerInterval, Run: s.runSchedulerPass}
}

// runSchedulerPass starts a backup of every database the schedule or a retry makes due. A start that
// fails is logged and the pass goes on with the next database.
func (s *Service) runSchedulerPass(ctx context.Context, now time.Time) error {
	enabledConfigs, err := s.configs.ListEnabledLogicalBackupConfigs(ctx)
	if err != nil {
		return err
	}

	if len(enabledConfigs) == 0 {
		return nil
	}

	newestBackups, err := s.listNewestBackups(ctx, enabledConfigs)
	if err != nil {
		return err
	}

	for _, databaseID := range getDueBackups(enabledConfigs, newestBackups, now) {
		s.startScheduledBackup(ctx, databaseID)
	}

	return nil
}

func (s *Service) startScheduledBackup(ctx context.Context, databaseID uuid.UUID) {
	backupDatabase, err := s.getDatabase(ctx, databaseID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to read the database of a due backup",
			slog.Any("database_id", databaseID), slog.Any("error", err))

		return
	}

	startedBackup, err := s.startBackup(ctx, backupDatabase, func(context.Context) error { return nil })
	if errors.Is(err, ErrBackupAlreadyRunning) {
		return
	}

	if err != nil {
		s.logger.ErrorContext(ctx, "failed to start a scheduled backup",
			slog.Any("database_id", databaseID), slog.Any("error", err))

		return
	}

	s.logger.InfoContext(ctx, "scheduled backup started",
		slog.Any("database_id", databaseID), slog.Any("backup_id", startedBackup.ID))
}

// listNewestBackups reads, per database, as many newest backups as its config's retry count needs,
// and at least the last one.
func (s *Service) listNewestBackups(
	ctx context.Context,
	enabledConfigs []config.LogicalBackupConfig,
) (map[uuid.UUID][]LogicalBackup, error) {
	databaseIDs := make([]uuid.UUID, 0, len(enabledConfigs))
	readCount := 1

	for _, enabledConfig := range enabledConfigs {
		databaseIDs = append(databaseIDs, enabledConfig.DatabaseID)
		readCount = max(readCount, enabledConfig.MaxFailedTriesCount)
	}

	listedBackups, err := s.repository.listNewestPerDatabase(ctx, databaseIDs, readCount)
	if err != nil {
		return nil, err
	}

	newestBackups := make(map[uuid.UUID][]LogicalBackup, len(databaseIDs))
	for _, listedBackup := range listedBackups {
		newestBackups[listedBackup.DatabaseID] = append(newestBackups[listedBackup.DatabaseID], listedBackup)
	}

	return newestBackups, nil
}

// getDueBackups returns the databases whose backup is due by the schedule since the last backup, or
// by a retry: the last backup failed, the config retries, and fewer of the newest MaxFailedTriesCount
// backups failed than it allows. newestBackups holds each database's newest backups, newest first. A
// database with a running backup or without a storage is never due.
func getDueBackups(
	enabledConfigs []config.LogicalBackupConfig,
	newestBackups map[uuid.UUID][]LogicalBackup,
	now time.Time,
) []uuid.UUID {
	dueDatabaseIDs := make([]uuid.UUID, 0)

	for _, enabledConfig := range enabledConfigs {
		if enabledConfig.StorageID == nil {
			continue
		}

		databaseBackups := newestBackups[enabledConfig.DatabaseID]

		if len(databaseBackups) == 0 {
			if enabledConfig.Schedule.IsDue(now, nil) {
				dueDatabaseIDs = append(dueDatabaseIDs, enabledConfig.DatabaseID)
			}

			continue
		}

		lastBackup := databaseBackups[0]
		if lastBackup.Status == StatusInProgress {
			continue
		}

		if enabledConfig.Schedule.IsDue(now, &lastBackup.CreatedAt) ||
			isRetryDue(enabledConfig, databaseBackups) {
			dueDatabaseIDs = append(dueDatabaseIDs, enabledConfig.DatabaseID)
		}
	}

	return dueDatabaseIDs
}

func isRetryDue(enabledConfig config.LogicalBackupConfig, databaseBackups []LogicalBackup) bool {
	if databaseBackups[0].Status != StatusFailed || !enabledConfig.IsRetryIfFailed {
		return false
	}

	failedCount := 0

	for _, recentBackup := range databaseBackups[:min(len(databaseBackups), enabledConfig.MaxFailedTriesCount)] {
		if recentBackup.Status == StatusFailed {
			failedCount++
		}
	}

	return failedCount < enabledConfig.MaxFailedTriesCount
}
