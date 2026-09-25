package healthcheck_config

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/storage"
)

type HealthcheckConfigRepository struct{}

func (r *HealthcheckConfigRepository) Save(
	config *HealthcheckConfig,
) error {
	db := storage.GetDb()

	return db.Save(config).Error
}

func (r *HealthcheckConfigRepository) GetDatabasesWithEnabledHealthcheck() (
	[]HealthcheckConfig, error,
) {
	var configs []HealthcheckConfig

	if err := storage.
		GetDb().
		Where("healthcheck_configs.is_healthcheck_enabled = ?", true).
		Find(&configs).Error; err != nil {
		return nil, err
	}

	return configs, nil
}

func (r *HealthcheckConfigRepository) GetByDatabaseID(
	databaseID uuid.UUID,
) (*HealthcheckConfig, error) {
	var config HealthcheckConfig

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

func (r *HealthcheckConfigRepository) FindByDatabaseIDs(
	databaseIDs []uuid.UUID,
) ([]HealthcheckConfig, error) {
	configs := []HealthcheckConfig{}

	if len(databaseIDs) == 0 {
		return configs, nil
	}

	if err := storage.
		GetDb().
		Where("database_id IN ?", databaseIDs).
		Find(&configs).Error; err != nil {
		return nil, err
	}

	return configs, nil
}
