package backuping_physical

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	tasks_cancellation "databasus-backend/internal/features/tasks/cancellation"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/modules/databases"
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

// OnPhysicalBackupConfigSaved stops the work the saved config no longer allows: every physical backup
// when backups were turned off, and the WAL streamer when the storage changed. A first save stops
// nothing, because nothing ran before it.
func (l *PhysicalBackupCancellationListener) OnPhysicalBackupConfigSaved(
	ctx context.Context,
	event backups.PhysicalBackupConfigSaved,
) error {
	if event.Previous == nil {
		return nil
	}

	databaseID := event.Saved.DatabaseID

	if event.Previous.IsBackupsEnabled && !event.Saved.IsBackupsEnabled {
		l.OnBackupsDisabled(ctx, databaseID)
	}

	if event.Saved.StorageID != nil && !isSameStorage(event.Previous.StorageID, event.Saved.StorageID) {
		l.OnBackupsStorageChanged(ctx, databaseID)
	}

	return nil
}

// OnDatabaseBackupTypeChanged stops the WAL streamer after the commit when the new backup type streams
// no WAL, because a streamer left running keeps pinning WAL on the source cluster.
func (l *PhysicalBackupCancellationListener) OnDatabaseBackupTypeChanged(
	ctx context.Context,
	event databases.DatabaseBackupTypeChanged,
) error {
	if !event.OldBackupType.IsWalStreaming() || event.NewBackupType.IsWalStreaming() {
		return nil
	}

	return database.AfterCommit(ctx, func(ctx context.Context) {
		l.OnWalStreamingDisabled(ctx, event.DatabaseID)
	})
}

// OnDatabaseDeleting stops the running backup and the WAL streamer once the deletion commits. The
// in-flight claim and the streamer row go with the database by cascade, so the claim is read now.
func (l *PhysicalBackupCancellationListener) OnDatabaseDeleting(
	ctx context.Context,
	event databases.DatabaseDeleting,
) error {
	claim, err := l.canceller.inFlightRepo.FindByDatabaseID(event.DatabaseID)
	if err != nil {
		return err
	}

	return database.AfterCommit(ctx, func(ctx context.Context) {
		logger := l.logger.With("database_id", event.DatabaseID)

		if claim != nil {
			err := l.taskCancellationRequester.RequestCancellation(ctx, claim.BackupID)
			if err != nil {
				logger.ErrorContext(ctx, "failed to request in-flight backup cancellation",
					"backup_id", claim.BackupID, "error", err)
			}
		}

		err := l.taskCancellationRequester.RequestCancellation(ctx, event.DatabaseID)
		if err != nil {
			logger.ErrorContext(ctx, "failed to cancel wal streamer task", "error", err)
		}
	})
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

func isSameStorage(first, second *uuid.UUID) bool {
	if first == nil || second == nil {
		return first == second
	}

	return *first == *second
}
