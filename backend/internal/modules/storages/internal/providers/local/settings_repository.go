package local

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/storages/internal/providers"
)

// settingsRepository keeps the rows of local storages in storages_local, which hold no settings.
type settingsRepository struct {
	db *gorm.DB
}

// Repositories are the repositories of the package. module.go builds them from the pool, so the
// provider never holds it.
type Repositories struct {
	settings *settingsRepository
}

func NewRepositories(db *gorm.DB) Repositories {
	return Repositories{settings: &settingsRepository{db: db}}
}

type settingsRow struct {
	StorageID uuid.UUID `gorm:"column:storage_id;primaryKey"`
}

func (settingsRow) TableName() string {
	return "storages_local"
}

func (r *settingsRepository) Upsert(ctx context.Context, storageID uuid.UUID, settings providers.Settings) error {
	_, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	row := settingsRow{StorageID: storageID}

	err = database.Conn(ctx, r.db).Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error
	if err != nil {
		return fmt.Errorf("upsert the local storage settings: %w", err)
	}

	return nil
}

func (r *settingsRepository) ListByStorageIDs(
	ctx context.Context,
	storageIDs []uuid.UUID,
) (map[uuid.UUID]providers.Settings, error) {
	var rows []settingsRow

	err := database.Conn(ctx, r.db).Where("storage_id IN ?", storageIDs).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list the local storage settings: %w", err)
	}

	settingsByStorageID := make(map[uuid.UUID]providers.Settings, len(rows))
	for _, row := range rows {
		settingsByStorageID[row.StorageID] = Settings{}
	}

	return settingsByStorageID, nil
}
