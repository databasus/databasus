// Package backups_config_storage_usage tells which databases a storage serves through their backup
// configs, logical and physical alike.
package backups_config_storage_usage

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/storage"
)

// GetDatabaseIDsUsingStorage runs in the transaction ctx carries.
func GetDatabaseIDsUsingStorage(ctx context.Context, storageID uuid.UUID) ([]uuid.UUID, error) {
	var databaseIDs []uuid.UUID

	err := database.Conn(ctx, storage.GetDb()).Raw(
		`SELECT database_id FROM logical_backup_configs WHERE storage_id = @storageID
		UNION
		SELECT database_id FROM physical_backup_configs WHERE storage_id = @storageID`,
		map[string]any{"storageID": storageID},
	).Scan(&databaseIDs).Error
	if err != nil {
		return nil, err
	}

	return databaseIDs, nil
}
