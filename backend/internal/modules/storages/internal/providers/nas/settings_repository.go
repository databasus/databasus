package nas

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

// settingsRepository keeps the settings of NAS storages in storages_nas.
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
	StorageID uuid.UUID `gorm:"column:storage_id;primaryKey"`
	Host      string    `gorm:"column:host"`
	Port      int       `gorm:"column:port"`
	Share     string    `gorm:"column:share"`
	Username  string    `gorm:"column:username"`
	Password  string    `gorm:"column:password"`
	IsSSL     bool      `gorm:"column:use_ssl"`
	Domain    *string   `gorm:"column:domain"`
	Path      *string   `gorm:"column:path"`
}

func (settingsRow) TableName() string {
	return "storages_nas"
}

func (r *settingsRepository) Upsert(ctx context.Context, storageID uuid.UUID, settings providers.Settings) error {
	nasSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	encryptedPassword, err := r.fieldEncryptor.Encrypt(nasSettings.Password)
	if err != nil {
		return fmt.Errorf("encrypt the NAS password: %w", err)
	}

	row := settingsRow{
		StorageID: storageID,
		Host:      nasSettings.Host,
		Port:      nasSettings.Port,
		Share:     nasSettings.Share,
		Username:  nasSettings.Username,
		Password:  encryptedPassword,
		IsSSL:     nasSettings.IsSSL,
		Domain:    &nasSettings.Domain,
		Path:      &nasSettings.Path,
	}

	err = database.Conn(ctx, r.db).Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error
	if err != nil {
		return fmt.Errorf("upsert the NAS settings: %w", err)
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
		return nil, fmt.Errorf("list the NAS settings: %w", err)
	}

	settingsByStorageID := make(map[uuid.UUID]providers.Settings, len(rows))
	for _, row := range rows {
		settingsByStorageID[row.StorageID] = Settings{
			Host:     row.Host,
			Port:     row.Port,
			Share:    row.Share,
			Username: row.Username,
			Password: row.Password,
			IsSSL:    row.IsSSL,
			Domain:   valueOrEmpty(row.Domain),
			Path:     valueOrEmpty(row.Path),
		}
	}

	return settingsByStorageID, nil
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}
