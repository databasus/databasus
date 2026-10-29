package verification_config

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/storage"
)

type VerificationConfigRepository struct{}

func (r *VerificationConfigRepository) Save(ctx context.Context, config *BackupVerificationConfig) error {
	return database.Conn(ctx, storage.GetDb()).Save(config).Error
}

func (r *VerificationConfigRepository) GetByDatabaseID(
	databaseID uuid.UUID,
) (*BackupVerificationConfig, error) {
	var config BackupVerificationConfig

	if err := storage.
		GetDb().
		Where("database_id = ?", databaseID).
		First(&config).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &config, nil
}

func (r *VerificationConfigRepository) FindAllEnabled() ([]*BackupVerificationConfig, error) {
	configs := make([]*BackupVerificationConfig, 0)

	err := storage.GetDb().
		Where("is_scheduled_verification_enabled = ?", true).
		Find(&configs).Error

	return configs, err
}

func (r *VerificationConfigRepository) DeleteByDatabaseID(ctx context.Context, databaseID uuid.UUID) error {
	return database.Conn(ctx, storage.GetDb()).
		Where("database_id = ?", databaseID).
		Delete(&BackupVerificationConfig{}).
		Error
}
