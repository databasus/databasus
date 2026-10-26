package physical_service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/storages"
	db "databasus-backend/internal/storage"
)

// OnDatabaseDeleting records the obligation to remove every file of the database's physical backups.
// Every physical table cascades on databases.id, so the deletion drops the rows that are the only
// record of the file names; the obligation commits with it in the same transaction.
func (s *PhysicalBackupService) OnDatabaseDeleting(ctx context.Context, event databases.DatabaseDeleting) error {
	return db.GetTransactor().Transaction(ctx, func(ctx context.Context) error {
		tx := database.Conn(ctx, db.GetDb())

		references, err := s.collectDatabaseFileReferences(tx, event.DatabaseID)
		if err != nil {
			return err
		}

		return s.getStorages().RequestFileDeletions(ctx, references)
	})
}

func (s *PhysicalBackupService) collectDatabaseFileReferences(
	tx *gorm.DB,
	databaseID uuid.UUID,
) ([]storages.StoredFileReference, error) {
	references := make([]storages.StoredFileReference, 0)

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
) []storages.StoredFileReference {
	references := make([]storages.StoredFileReference, 0, 3)

	for _, name := range []string{fileName, fileName + metadataSuffix, manifestFileName} {
		if name == "" || name == metadataSuffix {
			continue
		}

		references = append(references,
			storages.StoredFileReference{StorageID: storageID, FileName: name})
	}

	return references
}
