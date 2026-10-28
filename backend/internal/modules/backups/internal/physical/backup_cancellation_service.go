package physical

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/backups/internal/physical/backuping"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/users"
)

func (s *Service) CancelBackup(ctx context.Context, actor users.Actor, backupID uuid.UUID) error {
	databaseID, status, found, err := s.resolveBackupDatabaseAndStatus(ctx, backupID)
	if err != nil {
		return err
	}
	if !found {
		return ErrBackupNotFound
	}

	database, err := s.authorizeManage(ctx, actor, databaseID)
	if err != nil {
		return toBackupAccessProblem(err)
	}

	if status != catalog.BackupStatusInProgress {
		return ErrBackupNotInProgress
	}

	cancelled, err := s.canceller.CancelInFlightBackup(ctx, backuping.CancelInFlightBackupSpec{
		DatabaseID: databaseID,
		BackupID:   backupID,
	})
	if err != nil {
		return err
	}

	if !cancelled {
		return ErrBackupNotInProgress
	}

	s.recordAuditAfterAction(ctx, actor, database,
		fmt.Sprintf("Physical backup cancelled for database: %s", database.Name))

	return nil
}
