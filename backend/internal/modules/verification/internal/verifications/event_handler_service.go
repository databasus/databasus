package verifications

import (
	"context"

	"databasus-backend/internal/modules/backups"
)

func (s *Service) OnLogicalBackupCompleted(ctx context.Context, event backups.LogicalBackupCompleted) error {
	return s.scheduleVerificationForCompletedBackup(ctx, event.DatabaseID)
}

func (s *Service) OnLogicalBackupsDeleting(ctx context.Context, event backups.LogicalBackupsDeleting) error {
	return s.verificationRepository.DeleteByBackupIDs(ctx, event.BackupIDs)
}
