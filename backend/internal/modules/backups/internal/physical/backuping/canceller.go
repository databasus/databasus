package backuping

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

type Canceller struct {
	inFlightRepo             *catalog.InFlightBackupRepository
	taskCancellationRegistry *background.CancellationRegistry
	logger                   *slog.Logger
}

func (c *Canceller) CancelInFlightForDatabase(ctx context.Context, databaseID uuid.UUID) {
	logger := c.logger.With(slog.Any("database_id", databaseID))

	claim, _, err := c.inFlightRepo.FindByDatabaseID(ctx, databaseID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to look up in-flight backup for cancellation", slog.Any("error", err))

		return
	}

	if claim == nil {
		return
	}

	c.cancelClaim(ctx, logger, CancelInFlightBackupSpec{
		DatabaseID: databaseID,
		BackupID:   claim.BackupID,
	})
}

func (c *Canceller) CancelInFlightBackup(
	ctx context.Context,
	spec CancelInFlightBackupSpec,
) (bool, error) {
	claim, _, err := c.inFlightRepo.FindByDatabaseID(ctx, spec.DatabaseID)
	if err != nil {
		return false, err
	}

	if claim == nil || claim.BackupID != spec.BackupID {
		return false, nil
	}

	c.cancelClaim(ctx, c.logger.With(slog.Any("database_id", spec.DatabaseID)), spec)

	return true, nil
}

func (c *Canceller) cancelClaim(
	ctx context.Context,
	logger *slog.Logger,
	spec CancelInFlightBackupSpec,
) {
	c.taskCancellationRegistry.Cancel(spec.BackupID)

	if err := c.inFlightRepo.ReleaseOwned(ctx, spec.DatabaseID, spec.BackupID); err != nil {
		logger.ErrorContext(
			ctx,
			"failed to release in-flight claim",
			slog.Any("backup_id", spec.BackupID),
			slog.Any("error", err),
		)
	}
}
