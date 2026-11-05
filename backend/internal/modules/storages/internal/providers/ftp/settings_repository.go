package ftp

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

// settingsRepository keeps the settings of FTP storages in storages_ftp.
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
	StorageID       uuid.UUID `gorm:"column:storage_id;primaryKey"`
	Host            string    `gorm:"column:host"`
	Port            int       `gorm:"column:port"`
	Username        string    `gorm:"column:username"`
	Password        string    `gorm:"column:password"`
	Path            *string   `gorm:"column:path"`
	IsSSL           bool      `gorm:"column:use_ssl"`
	IsSkipTLSVerify bool      `gorm:"column:skip_tls_verify"`
}

func (settingsRow) TableName() string {
	return "storages_ftp"
}

func (r *settingsRepository) Upsert(ctx context.Context, storageID uuid.UUID, settings providers.Settings) error {
	ftpSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	encryptedPassword, err := r.fieldEncryptor.Encrypt(ftpSettings.Password)
	if err != nil {
		return fmt.Errorf("encrypt the FTP password: %w", err)
	}

	row := settingsRow{
		StorageID:       storageID,
		Host:            ftpSettings.Host,
		Port:            ftpSettings.Port,
		Username:        ftpSettings.Username,
		Password:        encryptedPassword,
		Path:            &ftpSettings.Path,
		IsSSL:           ftpSettings.IsSSL,
		IsSkipTLSVerify: ftpSettings.IsSkipTLSVerify,
	}

	err = database.Conn(ctx, r.db).Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error
	if err != nil {
		return fmt.Errorf("upsert the FTP settings: %w", err)
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
		return nil, fmt.Errorf("list the FTP settings: %w", err)
	}

	settingsByStorageID := make(map[uuid.UUID]providers.Settings, len(rows))
	for _, row := range rows {
		settingsByStorageID[row.StorageID] = Settings{
			Host:            row.Host,
			Port:            row.Port,
			Username:        row.Username,
			Password:        row.Password,
			Path:            valueOrEmpty(row.Path),
			IsSSL:           row.IsSSL,
			IsSkipTLSVerify: row.IsSkipTLSVerify,
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
