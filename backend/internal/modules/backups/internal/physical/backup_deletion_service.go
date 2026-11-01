package physical

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/users"
)

func (s *Service) DeleteBackup(ctx context.Context, actor users.Actor, backupID uuid.UUID) error {
	backup, isFound, err := s.backupRepository.FindByID(ctx, backupID)
	if err != nil {
		return err
	}
	if !isFound {
		return ErrBackupNotFound
	}

	backupDatabase, err := s.authorizeManage(ctx, actor, backup.DatabaseID)
	if err != nil {
		return toBackupAccessProblem(err)
	}

	return s.transactor.Transaction(ctx, func(ctx context.Context) error {
		if err := s.deleteBackup(ctx, backup); err != nil {
			return err
		}
		kind := "full"
		if backup.Kind == catalog.BackupKindIncremental {
			kind = "incremental"
		}
		return s.auditing.Record(ctx, actor, backupDatabase.WorkspaceID,
			fmt.Sprintf("Physical %s backup deleted for database: %s", kind, backupDatabase.Name))
	})
}

func (s *Service) deleteBackup(ctx context.Context, backup *catalog.Backup) error {
	return s.transactor.Transaction(ctx, func(ctx context.Context) error {
		runningBackup, _, err := s.backupRepository.FindRunningByDatabaseID(ctx, backup.DatabaseID)
		if err != nil {
			return err
		}

		if backup.Kind == catalog.BackupKindFull {
			_, err = s.chains.DeleteFullCompletely(ctx, backup.ID)
		} else {
			_, err = s.chains.DeleteIncrementalCascade(ctx, backup.ID)
		}
		if err != nil {
			return err
		}

		if runningBackup == nil {
			return nil
		}

		_, isFound, err := s.backupRepository.FindByID(ctx, runningBackup.ID)
		if err != nil || isFound {
			return err
		}
		return database.AfterCommit(ctx, func(context.Context) { s.cancellations.Cancel(runningBackup.ID) })
	})
}
