package backups_config_physical

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/storage"
)

type BackupConfigRepository struct{}

// Save runs in the transaction ctx carries and reads the saved row back in it.
func (r *BackupConfigRepository) Save(
	ctx context.Context,
	backupConfig *PhysicalBackupConfig,
) (*PhysicalBackupConfig, error) {
	if err := database.Conn(ctx, storage.GetDb()).Omit("PostgresqlPhysical").Save(backupConfig).Error; err != nil {
		return nil, err
	}

	return r.findByDatabaseID(ctx, backupConfig.DatabaseID)
}

func (r *BackupConfigRepository) FindByDatabaseID(
	databaseID uuid.UUID,
) (*PhysicalBackupConfig, error) {
	return r.findByDatabaseID(context.Background(), databaseID)
}

func (r *BackupConfigRepository) GetWithEnabledBackups() ([]*PhysicalBackupConfig, error) {
	var backupConfigs []*PhysicalBackupConfig

	if err := storage.
		GetDb().
		Preload("PostgresqlPhysical").
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

func (r *BackupConfigRepository) findByDatabaseID(
	ctx context.Context,
	databaseID uuid.UUID,
) (*PhysicalBackupConfig, error) {
	var backupConfig PhysicalBackupConfig

	if err := database.Conn(ctx, storage.GetDb()).
		Preload("PostgresqlPhysical").
		Where("database_id = ?", databaseID).
		First(&backupConfig).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &backupConfig, nil
}
