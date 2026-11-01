package physical

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/backups/internal/config"
)

func (s *Service) ReactToSavedConfig(
	ctx context.Context,
	previous *config.PhysicalBackupConfig,
	saved config.PhysicalBackupConfig,
) error {
	isStorageCleared := previous != nil && previous.StorageID != nil && saved.StorageID == nil
	isDisabled := previous != nil && previous.IsBackupsEnabled && !saved.IsBackupsEnabled
	if isStorageCleared || isDisabled {
		if err := s.canceller.CancelRunningForDatabase(ctx, saved.DatabaseID); err != nil {
			return err
		}
		s.cancelStreaming(ctx, saved.DatabaseID)
	}
	return nil
}

func (s *Service) RemoveBackupsForStorageChange(
	ctx context.Context,
	removal config.HistoryRemoval,
) (config.RemovedBackupHistory, error) {
	runningBackup, _, err := s.backupRepository.FindRunningByDatabaseID(ctx, removal.DatabaseID)
	if err != nil {
		return config.RemovedBackupHistory{}, err
	}

	pendingFiles, err := s.storages.GetPendingFileReferences(ctx, removal.DatabaseID.String()+"-")
	if err != nil {
		return config.RemovedBackupHistory{}, err
	}

	history, err := s.chains.RemoveBackupsForStorageChange(ctx, removal)
	if err != nil {
		return config.RemovedBackupHistory{}, err
	}

	for _, pending := range pendingFiles {
		isOldWrite := removal.UnfinishedStorageID != nil && pending.StorageID == *removal.UnfinishedStorageID
		if isOldWrite {
			history.References = append(history.References, pending)
		}
	}

	cancelOldStreamer := s.streams.GetStreamerCancellation(removal.DatabaseID)
	if err := database.AfterCommit(ctx, func(context.Context) {
		if runningBackup != nil {
			s.cancellations.Cancel(runningBackup.ID)
		}
		if cancelOldStreamer != nil {
			cancelOldStreamer()
		}
	}); err != nil {
		return config.RemovedBackupHistory{}, err
	}

	return history, nil
}

func (s *Service) cancelStreaming(ctx context.Context, databaseID uuid.UUID) {
	s.cancellations.Cancel(databaseID)
	if err := s.walStreamers.DeleteByDatabaseID(ctx, databaseID); err != nil {
		s.logger.ErrorContext(ctx, "failed to delete wal streamer row", slog.Any("error", err))
	}
}
