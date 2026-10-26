package rclone

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/storages/internal/providers"
)

// settingsRepository keeps the settings of rclone storages in storages_rclone.
type settingsRepository struct {
	db             *gorm.DB
	fieldEncryptor *crypto.FieldEncryptor
}

// Repositories are the repositories of the package. module.go builds them from the pool, so the
// provider never holds it.
type Repositories struct {
	settings *settingsRepository
}

func NewRepositories(db *gorm.DB, fieldEncryptor *crypto.FieldEncryptor) Repositories {
	return Repositories{settings: &settingsRepository{db: db, fieldEncryptor: fieldEncryptor}}
}

type settingsRow struct {
	StorageID     uuid.UUID `gorm:"column:storage_id;primaryKey"`
	ConfigContent string    `gorm:"column:config_content"`
	RemotePath    *string   `gorm:"column:remote_path"`
}

func (settingsRow) TableName() string {
	return "storages_rclone"
}

func (r *settingsRepository) Upsert(ctx context.Context, storageID uuid.UUID, settings providers.Settings) error {
	rcloneSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	encryptedConfigContent, err := r.fieldEncryptor.Encrypt(rcloneSettings.ConfigContent)
	if err != nil {
		return fmt.Errorf("encrypt the rclone configuration: %w", err)
	}

	row := settingsRow{
		StorageID:     storageID,
		ConfigContent: encryptedConfigContent,
		RemotePath:    &rcloneSettings.RemotePath,
	}

	err = database.Conn(ctx, r.db).Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error
	if err != nil {
		return fmt.Errorf("upsert the rclone settings: %w", err)
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
		return nil, fmt.Errorf("list the rclone settings: %w", err)
	}

	settingsByStorageID := make(map[uuid.UUID]providers.Settings, len(rows))
	for _, row := range rows {
		remotePath := ""
		if row.RemotePath != nil {
			remotePath = *row.RemotePath
		}

		settingsByStorageID[row.StorageID] = Settings{ConfigContent: row.ConfigContent, RemotePath: remotePath}
	}

	return settingsByStorageID, nil
}
