package catalog

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/storages"
)

func (r *Repository) DeleteDatabaseRows(ctx context.Context, databaseID uuid.UUID) error {
	for _, table := range []string{"backups_physical_wal_stream_samples", "backups_physical_wal_streamers", "backups_physical_in_flight", "backups_physical_wal_segments", "backups_physical_wal_history_files", "backups_physical_incremental", "backups_physical_full"} {
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

	var fulls []FullBackup
	if err := database.Conn(ctx, r.db).Where("database_id = ?", databaseID).Find(&fulls).Error; err != nil {
		return nil, fmt.Errorf("list full backups of a database: %w", err)
	}

	for _, full := range fulls {
		references = append(references, artifactReferences(
			full.StorageID, getFileName(full.FileName), getFileName(full.ManifestFileName))...)
	}

	var incrementals []IncrementalBackup
	if err := database.Conn(ctx, r.db).Where("database_id = ?", databaseID).Find(&incrementals).Error; err != nil {
		return nil, fmt.Errorf("list incremental backups of a database: %w", err)
	}

	for _, incremental := range incrementals {
		references = append(references, artifactReferences(
			incremental.StorageID,
			getFileName(incremental.FileName),
			getFileName(incremental.ManifestFileName),
		)...)
	}

	var segments []WalSegment
	if err := database.Conn(ctx, r.db).Where("database_id = ?", databaseID).Find(&segments).Error; err != nil {
		return nil, fmt.Errorf("list wal segments of a database: %w", err)
	}

	for _, segment := range segments {
		references = append(references, artifactReferences(
			segment.StorageID, getFileName(segment.FileName), "")...)
	}

	var historyFiles []WalHistoryFile
	if err := database.Conn(ctx, r.db).Where("database_id = ?", databaseID).Find(&historyFiles).Error; err != nil {
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
