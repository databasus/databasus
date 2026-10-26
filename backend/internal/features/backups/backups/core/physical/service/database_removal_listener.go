package physical_service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	storage_files "databasus-backend/internal/features/storages/files"
	"databasus-backend/internal/lib/database"
	db "databasus-backend/internal/storage"
)

// Every physical table cascades on databases.id, so removing a database drops the
// rows that are the only record of its object names. This records the obligation
// to remove those objects before the cascade can take the names away.
func (s *PhysicalBackupService) OnBeforeDatabaseRemove(ctx context.Context, databaseID uuid.UUID) error {
	return db.GetTransactor().Transaction(ctx, func(ctx context.Context) error {
		tx := database.Conn(ctx, db.GetDb())

		references, err := s.collectDatabaseFileReferences(tx, databaseID)
		if err != nil {
			return err
		}

		return s.fileStore.RequestFileDeletions(ctx, references)
	})
}

func (s *PhysicalBackupService) collectDatabaseFileReferences(
	tx *gorm.DB,
	databaseID uuid.UUID,
) ([]storage_files.StoredFileReference, error) {
	references := make([]storage_files.StoredFileReference, 0)

	var fulls []physical_models.PhysicalFullBackup
	if err := tx.Where("database_id = ?", databaseID).Find(&fulls).Error; err != nil {
		return nil, fmt.Errorf("list full backups of a database: %w", err)
	}

	for _, full := range fulls {
		references = append(references, artifactReferences(
			full.StorageID, valueOrEmptyString(full.FileName), valueOrEmptyString(full.ManifestFileName))...)
	}

	var incrementals []physical_models.PhysicalIncrementalBackup
	if err := tx.Where("database_id = ?", databaseID).Find(&incrementals).Error; err != nil {
		return nil, fmt.Errorf("list incremental backups of a database: %w", err)
	}

	for _, incremental := range incrementals {
		references = append(references, artifactReferences(
			incremental.StorageID,
			valueOrEmptyString(incremental.FileName),
			valueOrEmptyString(incremental.ManifestFileName),
		)...)
	}

	var segments []physical_models.PhysicalWalSegment
	if err := tx.Where("database_id = ?", databaseID).Find(&segments).Error; err != nil {
		return nil, fmt.Errorf("list wal segments of a database: %w", err)
	}

	for _, segment := range segments {
		references = append(references, artifactReferences(
			segment.StorageID, valueOrEmptyString(segment.FileName), "")...)
	}

	var historyFiles []physical_models.PhysicalWalHistoryFile
	if err := tx.Where("database_id = ?", databaseID).Find(&historyFiles).Error; err != nil {
		return nil, fmt.Errorf("list history files of a database: %w", err)
	}

	for _, historyFile := range historyFiles {
		references = append(references, artifactReferences(
			historyFile.StorageID, historyFile.FileName, "")...)
	}

	return references, nil
}

func artifactReferences(
	storageID uuid.UUID,
	fileName, manifestFileName string,
) []storage_files.StoredFileReference {
	references := make([]storage_files.StoredFileReference, 0, 3)

	for _, name := range []string{fileName, fileName + metadataSuffix, manifestFileName} {
		if name == "" || name == metadataSuffix {
			continue
		}

		references = append(references,
			storage_files.StoredFileReference{StorageID: storageID, FileName: name})
	}

	return references
}
