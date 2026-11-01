package physical

import (
	"context"

	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/databases"
)

func (s *Service) OnDatabaseDeleting(ctx context.Context, event databases.DatabaseDeleting) error {
	runningBackup, _, err := s.backupRepository.FindRunningByDatabaseID(ctx, event.DatabaseID)
	if err != nil {
		return err
	}

	if err := database.AfterCommit(ctx, func(_ context.Context) {
		if runningBackup != nil {
			s.cancellations.Cancel(runningBackup.ID)
		}
		s.cancellations.Cancel(event.DatabaseID)
	}); err != nil {
		return err
	}

	if err := s.chains.OnDatabaseDeleting(ctx, event); err != nil {
		return err
	}

	return s.onDatabaseDeletingDropSlots(ctx, event)
}

func (s *Service) OnDatabaseBackupTypeChanged(ctx context.Context, event databases.DatabaseBackupTypeChanged) error {
	if !event.OldBackupType.IsWalStreaming() || event.NewBackupType.IsWalStreaming() {
		return nil
	}
	return database.AfterCommit(ctx, func(ctx context.Context) { s.cancelStreaming(ctx, event.DatabaseID) })
}
