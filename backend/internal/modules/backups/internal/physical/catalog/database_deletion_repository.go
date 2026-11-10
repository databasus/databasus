package catalog

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/storages"
)

func (r *Repository) DeleteDatabaseRows(ctx context.Context, databaseID uuid.UUID) error {
	for _, table := range []string{"backups_physical_wal_stream_samples", "backups_physical_wal_streamers", "backups_physical_wal_segments", "backups_physical_wal_history_files", "backups_physical"} {
		if err := database.Conn(ctx, r.db).
			Exec("DELETE FROM "+table+" WHERE database_id = ?", databaseID).
			Error; err != nil {
			return fmt.Errorf("delete physical records from %s: %w", table, err)
		}
	}
	return nil
}

func (r *Repository) GetDatabaseFileReferences(
	ctx context.Context,
	databaseID uuid.UUID,
) ([]storages.StoredFileReference, error) {
	references := make([]storages.StoredFileReference, 0)

	var backups []Backup
	if err := database.Conn(ctx, r.db).Where("database_id = ?", databaseID).Find(&backups).Error; err != nil {
		return nil, fmt.Errorf("list physical backups of a database: %w", err)
	}
	for _, backup := range backups {
		references = append(references, backupFileReferences(&backup)...)
	}

	var segments []WalSegment
	if err := database.Conn(ctx, r.db).Where("database_id = ?", databaseID).Find(&segments).Error; err != nil {
		return nil, fmt.Errorf("list wal segments of a database: %w", err)
	}

	for _, segment := range segments {
		references = append(references, artifactReferences(
			segment.StorageID, getFileName(segment.FileName), "",
		)...)
	}

	var historyFiles []WalHistoryFile
	if err := database.Conn(ctx, r.db).Where("database_id = ?", databaseID).Find(&historyFiles).Error; err != nil {
		return nil, fmt.Errorf("list history files of a database: %w", err)
	}

	for _, historyFile := range historyFiles {
		references = append(references, artifactReferences(
			historyFile.StorageID, historyFile.FileName, "",
		)...)
	}

	return references, nil
}

func artifactReferences(
	storageID uuid.UUID,
	fileName, manifestFileName string,
) []storages.StoredFileReference {
	references := make([]storages.StoredFileReference, 0, 3)

	for _, name := range []string{fileName, fileName + MetadataSuffix, manifestFileName} {
		if name == "" || name == MetadataSuffix {
			continue
		}

		references = append(references,
			storages.StoredFileReference{StorageID: storageID, FileName: name})
	}

	return references
}

func getFileName(name *string) string {
	if name == nil {
		return ""
	}
	return *name
}

func (r *Repository) HasDatabaseHistory(ctx context.Context, databaseID uuid.UUID) (bool, error) {
	var hasHistory bool
	err := database.Conn(ctx, r.db).
		Raw(`SELECT EXISTS(SELECT 1 FROM backups_physical WHERE database_id=?) OR EXISTS(SELECT 1 FROM backups_physical_wal_segments WHERE database_id=?) OR EXISTS(SELECT 1 FROM backups_physical_wal_history_files WHERE database_id=?)`, databaseID, databaseID, databaseID).
		Scan(&hasHistory).
		Error
	if err != nil {
		return false, fmt.Errorf("check physical history: %w", err)
	}
	return hasHistory, nil
}
