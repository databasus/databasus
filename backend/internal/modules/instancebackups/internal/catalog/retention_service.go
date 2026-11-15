package catalog

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/storages"
)

const (
	RetentionJobName     = "instancebackups_retention"
	RetentionJobInterval = time.Hour
	day                  = 24 * time.Hour
)

// RunRetentionPass deletes, per current destination, the stored copies older than the retention
// period, always keeping the destination's newest stored copy, so failed later runs never leave it
// without one. Deselected storages keep their copies. It also drops expired download tokens.
func (s *Service) RunRetentionPass(ctx context.Context, now time.Time) error {
	config, err := s.configs.GetConfig(ctx)
	if err != nil {
		return err
	}

	destinations, err := s.configs.GetDestinations(ctx)
	if err != nil {
		return err
	}

	storedBefore := now.Add(-time.Duration(config.RetentionDays) * day)

	for _, destination := range destinations {
		if err := s.deleteExpiredCopies(ctx, destination.StorageID, storedBefore); err != nil {
			return err
		}
	}

	_, err = s.tokens.DeleteExpired(ctx, now)

	return err
}

func (s *Service) deleteExpiredCopies(ctx context.Context, storageID uuid.UUID, storedBefore time.Time) error {
	expiredCopies, err := s.copies.listExpired(ctx, storageID, storedBefore)
	if err != nil || len(expiredCopies) == 0 {
		return err
	}

	copyIDs := make([]uuid.UUID, 0, len(expiredCopies))
	references := make([]storages.StoredFileReference, 0, len(expiredCopies))

	for _, expiredCopy := range expiredCopies {
		copyIDs = append(copyIDs, expiredCopy.ID)
		references = append(
			references,
			storages.StoredFileReference{StorageID: storageID, FileName: *expiredCopy.FileName},
		)
	}

	err = s.transactor.Transaction(ctx, func(txCtx context.Context) error {
		if err := s.storages.RequestFileDeletions(txCtx, references); err != nil {
			return err
		}

		return s.copies.markDeleted(txCtx, copyIDs)
	})
	if err != nil {
		return err
	}

	s.logger.InfoContext(ctx, "expired databasus backup copies deleted",
		slog.String("storage_id", storageID.String()), slog.Int("copy_count", len(copyIDs)))

	return nil
}
