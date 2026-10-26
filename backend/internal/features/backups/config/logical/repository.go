package backups_config_logical

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/storage"
)

type BackupConfigRepository struct{}

func (r *BackupConfigRepository) Save(
	ctx context.Context,
	backupConfig *LogicalBackupConfig,
) (*LogicalBackupConfig, error) {
	if err := database.Conn(ctx, storage.GetDb()).Save(backupConfig).Error; err != nil {
		return nil, err
	}

	return backupConfig, nil
}

func (r *BackupConfigRepository) FindByDatabaseID(databaseID uuid.UUID) (*LogicalBackupConfig, error) {
	var backupConfig LogicalBackupConfig

	if err := storage.
		GetDb().
		Where("database_id = ?", databaseID).
		First(&backupConfig).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &backupConfig, nil
}

func (r *BackupConfigRepository) GetWithEnabledBackups() ([]*LogicalBackupConfig, error) {
	var backupConfigs []*LogicalBackupConfig

	if err := storage.
		GetDb().
		Where("is_backups_enabled = ?", true).
		Find(&backupConfigs).Error; err != nil {
		return nil, err
	}

	return backupConfigs, nil
}
