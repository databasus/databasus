package backuping_physical

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	tasks_cancellation "databasus-backend/internal/features/tasks/cancellation"
)

type PhysicalBackupCancellationListener struct {
	canceller                 *PhysicalBackupCanceller
	walStreamerRepo           *physical_repositories.PhysicalWalStreamerRepository
	taskCancellationRequester *tasks_cancellation.Requester
	logger                    *slog.Logger
}

func (l *PhysicalBackupCancellationListener) OnBackupsDisabled(ctx context.Context, databaseID uuid.UUID) {
	l.canceller.CancelInFlightForDatabase(ctx, databaseID)
	l.cancelStreamingAndDeleteStreamerRow(ctx, databaseID)
}

func (l *PhysicalBackupCancellationListener) OnWalStreamingDisabled(ctx context.Context, databaseID uuid.UUID) {
	l.cancelStreamingAndDeleteStreamerRow(ctx, databaseID)
}

// The streamer the supervisor starts next reads the storage from the saved configuration, so
// stopping the current one moves WAL streaming to the new storage.
func (l *PhysicalBackupCancellationListener) OnBackupsStorageChanged(ctx context.Context, databaseID uuid.UUID) {
	l.cancelStreamingAndDeleteStreamerRow(ctx, databaseID)
}

func (l *PhysicalBackupCancellationListener) OnBeforeDatabaseRemove(ctx context.Context, databaseID uuid.UUID) error {
	l.canceller.CancelInFlightForDatabase(ctx, databaseID)
	l.cancelStreamingAndDeleteStreamerRow(ctx, databaseID)

	return nil
}

// Deleting the row first would leave an orphaned pg_receivewal on the slot. The
// slot itself stays: re-enabling backups or WAL streaming reuses it.
func (l *PhysicalBackupCancellationListener) cancelStreamingAndDeleteStreamerRow(
	ctx context.Context,
	databaseID uuid.UUID,
) {
	logger := l.logger.With("database_id", databaseID)

	if err := l.taskCancellationRequester.RequestCancellation(ctx, databaseID); err != nil {
		logger.ErrorContext(ctx, "failed to cancel wal streamer task", "error", err)
	}

	if err := l.walStreamerRepo.DeleteByDatabaseID(databaseID); err != nil {
		logger.ErrorContext(ctx, "failed to delete wal streamer row", "error", err)
	}
}
