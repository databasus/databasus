package backups_config_physical

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/storage"
)

type BackupConfigRepository struct{}

func (r *BackupConfigRepository) Save(
	backupConfig *PhysicalBackupConfig,
) (*PhysicalBackupConfig, error) {
	if backupConfig.Storage != nil && backupConfig.Storage.ID != uuid.Nil {
		backupConfig.StorageID = &backupConfig.Storage.ID
	}

	if err := storage.GetDb().Save(backupConfig).Omit("Storage", "PostgresqlPhysical").Error; err != nil {
		return nil, err
	}

	return r.FindByDatabaseID(backupConfig.DatabaseID)
}

func (r *BackupConfigRepository) FindByDatabaseID(
	databaseID uuid.UUID,
) (*PhysicalBackupConfig, error) {
	var backupConfig PhysicalBackupConfig

	if err := storage.
		GetDb().
		Preload("PostgresqlPhysical").
		Preload("Storage").
		Preload("Storage.LocalStorage").
		Preload("Storage.S3Storage").
		Preload("Storage.GoogleDriveStorage").
		Preload("Storage.NASStorage").
		Preload("Storage.AzureBlobStorage").
		Preload("Storage.FTPStorage").
		Preload("Storage.SFTPStorage").
		Preload("Storage.RcloneStorage").
		Where("database_id = ?", databaseID).
		First(&backupConfig).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &backupConfig, nil
}

func (r *BackupConfigRepository) GetWithEnabledBackups() ([]*PhysicalBackupConfig, error) {
	var backupConfigs []*PhysicalBackupConfig

	if err := storage.
		GetDb().
		Preload("PostgresqlPhysical").
		Preload("Storage").
		Preload("Storage.LocalStorage").
		Preload("Storage.S3Storage").
		Preload("Storage.GoogleDriveStorage").
		Preload("Storage.NASStorage").
		Preload("Storage.AzureBlobStorage").
		Preload("Storage.FTPStorage").
		Preload("Storage.SFTPStorage").
		Preload("Storage.RcloneStorage").
		Where("is_backups_enabled = ?", true).
		Find(&backupConfigs).Error; err != nil {
		return nil, err
	}

	return backupConfigs, nil
}

func (r *BackupConfigRepository) RequestFullBackupNow(databaseID uuid.UUID) error {
	return storage.
		GetDb().
		Model(&PhysicalBackupConfig{}).
		Where("database_id = ?", databaseID).
		Update("force_full_requested_at", time.Now().UTC()).Error
}

func (r *BackupConfigRepository) ClearFullBackupRequest(databaseID uuid.UUID, requestedAt *time.Time) error {
	query := storage.
		GetDb().
		Model(&PhysicalBackupConfig{}).
		Where("database_id = ?", databaseID)
	if requestedAt != nil {
		query = query.Where("force_full_requested_at = ?", *requestedAt)
	}

	return query.Update("force_full_requested_at", nil).Error
}

func (r *BackupConfigRepository) RequestIncrementalBackupNow(databaseID uuid.UUID) error {
	return storage.
		GetDb().
		Model(&PhysicalBackupConfig{}).
		Where("database_id = ?", databaseID).
		Update("force_incremental_requested_at", time.Now().UTC()).Error
}

func (r *BackupConfigRepository) ClearIncrementalBackupRequest(databaseID uuid.UUID, requestedAt *time.Time) error {
	query := storage.
		GetDb().
		Model(&PhysicalBackupConfig{}).
		Where("database_id = ?", databaseID)
	if requestedAt != nil {
		query = query.Where("force_incremental_requested_at = ?", *requestedAt)
	}

	return query.Update("force_incremental_requested_at", nil).Error
}

func (r *BackupConfigRepository) IsStorageInUse(storageID uuid.UUID) (bool, error) {
	var count int64

	if err := storage.
		GetDb().
		Table("physical_backup_configs").
		Where("storage_id = ?", storageID).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *BackupConfigRepository) GetDatabasesIDsByStorageID(
	storageID uuid.UUID,
) ([]uuid.UUID, error) {
	var databasesIDs []uuid.UUID

	if err := storage.
		GetDb().
		Table("physical_backup_configs").
		Where("storage_id = ?", storageID).
		Pluck("database_id", &databasesIDs).Error; err != nil {
		return nil, err
	}

	return databasesIDs, nil
}

func (r *BackupConfigRepository) FindByDatabaseIDs(
	databaseIDs []uuid.UUID,
) ([]*PhysicalBackupConfig, error) {
	backupConfigs := []*PhysicalBackupConfig{}

	if len(databaseIDs) == 0 {
		return backupConfigs, nil
	}

	if err := storage.
		GetDb().
		Preload("Storage").
		Where("database_id IN ?", databaseIDs).
		Find(&backupConfigs).Error; err != nil {
		return nil, err
	}

	return backupConfigs, nil
}
