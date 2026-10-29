package verification_runs

import (
	"context"

	"databasus-backend/internal/modules/backups"
)

func (s *VerificationService) OnLogicalBackupsDeleting(
	ctx context.Context,
	event backups.LogicalBackupsDeleting,
) error {
	return s.verificationRepository.DeleteByBackupIDs(ctx, event.BackupIDs)
}
