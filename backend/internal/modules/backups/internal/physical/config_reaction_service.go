package physical

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/backups/internal/config"
)

func (s *Service) ReactToSavedConfig(
	ctx context.Context,
	previous *config.PhysicalBackupConfig,
	saved config.PhysicalBackupConfig,
) error {
	if previous != nil {
		if previous.IsBackupsEnabled && !saved.IsBackupsEnabled {
			if err := s.canceller.CancelRunningForDatabase(ctx, saved.DatabaseID); err != nil {
				return err
			}
			s.cancelStreaming(ctx, saved.DatabaseID)
		}
		if saved.StorageID != nil && !isSameStorage(previous.StorageID, saved.StorageID) {
			s.cancelStreaming(ctx, saved.DatabaseID)
		}
	}
	if saved.StorageID == nil {
		return nil
	}
	return s.removeBackupsOutsideStorage(ctx, saved.DatabaseID, *saved.StorageID)
}

func (s *Service) cancelStreaming(ctx context.Context, databaseID uuid.UUID) {
	s.cancellations.Cancel(databaseID)
	if err := s.walStreamers.DeleteByDatabaseID(ctx, databaseID); err != nil {
		s.logger.ErrorContext(ctx, "failed to delete wal streamer row", slog.Any("error", err))
	}
}

func isSameStorage(first, second *uuid.UUID) bool {
	if first == nil || second == nil {
		return first == second
	}
	return *first == *second
}

func (s *Service) removeBackupsOutsideStorage(
	ctx context.Context,
	databaseID uuid.UUID,
	storageID uuid.UUID,
) error {
	fullsInOtherStorages, err := s.backupRepository.FindAllFullByDatabaseOutsideStorage(ctx, databaseID, storageID)
	if err != nil {
		return err
	}

	if len(fullsInOtherStorages) == 0 {
		return nil
	}

	logger := s.logger.With(slog.Any("database_id", databaseID), slog.Any("storage_id", storageID))

	var removalErrors []error

	for _, full := range fullsInOtherStorages {
		if err := s.deleteFull(ctx, full); err != nil {
			logger.ErrorContext(
				ctx,
				"failed to remove full backup kept in a previous storage",
				slog.Any("backup_id", full.ID),
				slog.Any("error", err),
			)

			removalErrors = append(removalErrors, fmt.Errorf("remove full backup %s: %w", full.ID, err))
		}
	}

	logger.InfoContext(
		ctx,
		"removed full backups kept in previous storages",
		slog.Any("removed_count", len(fullsInOtherStorages)-len(removalErrors)),
		slog.Any("found_count", len(fullsInOtherStorages)),
	)

	return errors.Join(removalErrors...)
}
