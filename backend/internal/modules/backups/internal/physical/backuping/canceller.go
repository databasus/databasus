package backuping

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

type Canceller struct {
	backupRepo               *catalog.BackupRepository
	lifecycle                *catalog.BackupLifecycle
	transactor               *database.Transactor
	taskCancellationRegistry *background.CancellationRegistry
}

func (c *Canceller) CancelRunningForDatabase(ctx context.Context, databaseID uuid.UUID) error {
	backup, _, err := c.backupRepo.FindRunningByDatabaseID(ctx, databaseID)
	if err != nil {
		return err
	}
	if backup == nil {
		return nil
	}
	_, err = c.cancel(ctx, backup)
	return err
}

func (c *Canceller) CancelRunningBackup(ctx context.Context, spec CancelRunningBackupSpec) (bool, error) {
	backup, _, err := c.backupRepo.FindByID(ctx, spec.BackupID)
	if err != nil {
		return false, err
	}
	if backup == nil || backup.DatabaseID != spec.DatabaseID {
		return false, nil
	}
	return c.cancel(ctx, backup)
}

func (c *Canceller) cancel(ctx context.Context, backup *catalog.Backup) (bool, error) {
	var isCanceled bool
	err := c.transactor.Transaction(ctx, func(ctx context.Context) error {
		currentBackup, isFound, err := c.backupRepo.FindByIDForUpdate(ctx, backup.ID)
		if err != nil || !isFound || currentBackup.Status != catalog.BackupStatusInProgress {
			return err
		}
		isCanceled, err = c.lifecycle.Cancel(ctx, currentBackup)
		if err != nil || !isCanceled {
			return err
		}
		return database.AfterCommit(ctx, func(context.Context) { c.taskCancellationRegistry.Cancel(backup.ID) })
	})
	return isCanceled, err
}
