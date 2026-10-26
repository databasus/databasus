package googledrive

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

// settingsRepository keeps the settings of Google Drive storages in storages_google_drive.
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
	StorageID    uuid.UUID `gorm:"column:storage_id;primaryKey"`
	ClientID     string    `gorm:"column:client_id"`
	ClientSecret string    `gorm:"column:client_secret"`
	TokenJSON    string    `gorm:"column:token_json"`
}

func (settingsRow) TableName() string {
	return "storages_google_drive"
}

func (r *settingsRepository) Upsert(ctx context.Context, storageID uuid.UUID, settings providers.Settings) error {
	driveSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	encryptedClientSecret, err := r.fieldEncryptor.Encrypt(driveSettings.ClientSecret)
	if err != nil {
		return fmt.Errorf("encrypt the Google Drive client secret: %w", err)
	}

	encryptedTokenJSON, err := r.fieldEncryptor.Encrypt(driveSettings.TokenJSON)
	if err != nil {
		return fmt.Errorf("encrypt the Google Drive token: %w", err)
	}

	row := settingsRow{
		StorageID:    storageID,
		ClientID:     driveSettings.ClientID,
		ClientSecret: encryptedClientSecret,
		TokenJSON:    encryptedTokenJSON,
	}

	err = database.Conn(ctx, r.db).Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error
	if err != nil {
		return fmt.Errorf("upsert the Google Drive settings: %w", err)
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
		return nil, fmt.Errorf("list the Google Drive settings: %w", err)
	}

	settingsByStorageID := make(map[uuid.UUID]providers.Settings, len(rows))
	for _, row := range rows {
		settingsByStorageID[row.StorageID] = Settings{
			ClientID:     row.ClientID,
			ClientSecret: row.ClientSecret,
			TokenJSON:    row.TokenJSON,
		}
	}

	return settingsByStorageID, nil
}
