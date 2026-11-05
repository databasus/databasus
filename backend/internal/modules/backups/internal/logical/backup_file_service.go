package logical

import (
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"

	files_utils "databasus-backend/internal/lib/filename"
	"databasus-backend/internal/modules/databases"
)

// GetBackupWithDatabase reads the backup and its database without an access check, for a download
// whose token already names the backup.
func (s *Service) GetBackupWithDatabase(
	ctx context.Context,
	backupID uuid.UUID,
) (LogicalBackup, databases.Database, error) {
	backup, isFound, err := s.repository.get(ctx, backupID)
	if err != nil {
		return LogicalBackup{}, databases.Database{}, err
	}

	if !isFound {
		return LogicalBackup{}, databases.Database{}, ErrBackupNotFound
	}

	backupDatabase, err := s.getDatabase(ctx, backup.DatabaseID)
	if err != nil {
		return LogicalBackup{}, databases.Database{}, err
	}

	return backup, backupDatabase, nil
}

// OpenBackupFile opens the stored file of the backup, decrypted.
func (s *Service) OpenBackupFile(ctx context.Context, backup LogicalBackup) (io.ReadCloser, error) {
	storedBackup, err := backup.GetStoredBackup()
	if err != nil {
		return nil, err
	}

	return s.pipeline.OpenBackupFile(ctx, storedBackup)
}

// GetDownloadFileName names a downloaded copy of the backup after its database, its time and the
// format of its engine.
func (s *Service) GetDownloadFileName(backup LogicalBackup, backupDatabase databases.Database) (string, error) {
	extension, err := s.pipeline.GetFileExtension(backupDatabase.Type)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s_backup_%s%s",
		files_utils.SanitizeFilename(backupDatabase.Name),
		backup.CreatedAt.UTC().Format("2006-01-02_15-04-05"),
		extension,
	), nil
}
