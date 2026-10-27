package backuping_physical

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	"databasus-backend/internal/lib/background"
)

// PhysicalBackupCanceller stands a database's in-flight physical backup down: it
// cancels the running backup task (the executor unwinds on the cancelled
// context) and releases the cross-table single-in-flight claim. It is the one
// place that knows how to stop a running FULL/INCR, shared by the config-change
// listener, the database-remove listener, and the user-facing cancel/delete
// endpoints.
type PhysicalBackupCanceller struct {
	inFlightRepo             *physical_repositories.PhysicalInFlightBackupRepository
	taskCancellationRegistry *background.CancellationRegistry
	logger                   *slog.Logger
}

func NewPhysicalBackupCanceller(
	inFlightRepo *physical_repositories.PhysicalInFlightBackupRepository,
	taskCancellationRegistry *background.CancellationRegistry,
	logger *slog.Logger,
) *PhysicalBackupCanceller {
	return &PhysicalBackupCanceller{inFlightRepo, taskCancellationRegistry, logger}
}

// CancelInFlightForDatabase cancels whatever backup the database currently holds
// in flight, whichever it is. Use it for teardown (config disable, db removal)
// where any running backup must stop. A no claim is a no-op.
func (c *PhysicalBackupCanceller) CancelInFlightForDatabase(ctx context.Context, databaseID uuid.UUID) {
	logger := c.logger.With("database_id", databaseID)

	claim, err := c.inFlightRepo.FindByDatabaseID(databaseID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to look up in-flight backup for cancellation", "error", err)

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

// CancelInFlightBackup cancels the database's in-flight backup only when the
// claim still names backupID. It returns whether a matching claim was found and
// cancelled, so a delete path can tell "I stopped the running backup" from
// "nothing was running for this row". Scoping by backupID avoids stopping a
// newer backup that took the claim after the targeted one finished.
func (c *PhysicalBackupCanceller) CancelInFlightBackup(
	ctx context.Context,
	spec CancelInFlightBackupSpec,
) (bool, error) {
	claim, err := c.inFlightRepo.FindByDatabaseID(spec.DatabaseID)
	if err != nil {
		return false, err
	}

	if claim == nil || claim.BackupID != spec.BackupID {
		return false, nil
	}

	c.cancelClaim(ctx, c.logger.With("database_id", spec.DatabaseID), spec)

	return true, nil
}

func (c *PhysicalBackupCanceller) cancelClaim(
	ctx context.Context,
	logger *slog.Logger,
	spec CancelInFlightBackupSpec,
) {
	c.taskCancellationRegistry.Cancel(spec.BackupID)

	if err := c.inFlightRepo.ReleaseOwned(spec.DatabaseID, spec.BackupID); err != nil {
		logger.ErrorContext(ctx, "failed to release in-flight claim",
			"backup_id", spec.BackupID, "error", err)
	}
}
