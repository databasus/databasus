package logical

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/databases"
)

// OnDatabaseDeleting removes every backup of the database in the deletion's transaction, so a refused
// deletion keeps them. It refuses while a backup runs, because the backup still writes its file, and
// while a restore reads one of the backups.
func (s *Service) OnDatabaseDeleting(ctx context.Context, event databases.DatabaseDeleting) error {
	return s.removeDatabaseBackups(ctx, event.DatabaseID, databaseBackupsRemovalRefusals{
		backupRunning:  ErrDatabaseBackupRunning,
		restoreRunning: ErrDatabaseRestoreRunning,
	})
}

// RemoveBackupsForStorageChange removes the database's backups in the transaction of a config save
// that names another storage. The save holds the config's row lock, which a backup start shares, so
// every backup that started before the save is visible here.
func (s *Service) RemoveBackupsForStorageChange(ctx context.Context, databaseID uuid.UUID) error {
	return s.removeDatabaseBackups(ctx, databaseID, databaseBackupsRemovalRefusals{
		backupRunning:  ErrStorageChangeBackupRunning,
		restoreRunning: ErrStorageChangeRestoreRunning,
	})
}

// databaseBackupsRemovalRefusals are the errors that refuse the removal, worded for its cause.
type databaseBackupsRemovalRefusals struct {
	backupRunning  error
	restoreRunning error
}

func (s *Service) removeDatabaseBackups(
	ctx context.Context,
	databaseID uuid.UUID,
	refusals databaseBackupsRemovalRefusals,
) error {
	// A restore start holds a key share lock on its backup's row until it commits, so this lock waits
	// for a restore that is being inserted and the check below sees it; a later insert waits for the
	// deletion and fails its reference to the deleted backup.
	backupIDs, err := s.repository.lockAndListBackupIDsOfDatabase(ctx, databaseID)
	if err != nil {
		return err
	}

	isBackupRunning, err := s.repository.isInProgressForDatabase(ctx, databaseID)
	if err != nil {
		return err
	}

	if isBackupRunning {
		return refusals.backupRunning
	}

	isRestoreRunning, err := s.restores.IsRestoreRunningForDatabase(ctx, databaseID)
	if err != nil {
		return err
	}

	if isRestoreRunning {
		return refusals.restoreRunning
	}

	return s.deleteBackupsWithFiles(ctx, backupIDs)
}
