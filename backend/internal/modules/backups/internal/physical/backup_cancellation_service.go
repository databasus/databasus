package physical

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/backups/internal/physical/backuping"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/users"
)

func (s *Service) CancelBackup(
	ctx context.Context,
	actor users.Actor,
	backupID uuid.UUID,
) (PhysicalBackupListItem, error) {
	databaseID, status, found, err := s.resolveBackupDatabaseAndStatus(ctx, backupID)
	if err != nil {
		return PhysicalBackupListItem{}, err
	}
	if !found {
		return PhysicalBackupListItem{}, ErrBackupNotFound
	}

	database, err := s.authorizeManage(ctx, actor, databaseID)
	if err != nil {
		return PhysicalBackupListItem{}, toBackupAccessProblem(err)
	}

	if status != catalog.BackupStatusInProgress {
		return PhysicalBackupListItem{}, ErrBackupNotInProgress
	}

	var cancelledBackup PhysicalBackupListItem

	err = s.transactor.Transaction(ctx, func(ctx context.Context) error {
		isCanceled, err := s.canceller.CancelRunningBackup(ctx, backuping.CancelRunningBackupSpec{
			DatabaseID: databaseID, BackupID: backupID,
		})
		if err != nil {
			return err
		}
		if !isCanceled {
			return ErrBackupNotInProgress
		}

		err = s.auditing.Record(ctx, actor, database.WorkspaceID,
			fmt.Sprintf("Physical backup cancelled for database: %s", database.Name))
		if err != nil {
			return err
		}

		cancelledBackup, err = s.getBackupListItem(ctx, databaseID, backupID)
		return err
	})
	if err != nil {
		return PhysicalBackupListItem{}, err
	}

	return cancelledBackup, nil
}

// getBackupListItem reads through the list query, so the answer cannot differ from the row the
// list shows for the same backup.
func (s *Service) getBackupListItem(
	ctx context.Context,
	databaseID uuid.UUID,
	backupID uuid.UUID,
) (PhysicalBackupListItem, error) {
	rows, err := s.chains.ListBackups(ctx, catalog.BackupListQuery{
		DatabaseID: databaseID,
		Filter:     catalog.BackupListFilter{ID: &backupID},
		After:      nil,
		Limit:      1,
	})
	if err != nil {
		return PhysicalBackupListItem{}, err
	}
	if len(rows) == 0 {
		return PhysicalBackupListItem{}, ErrBackupNotFound
	}

	return rowToListItem(rows[0]), nil
}
