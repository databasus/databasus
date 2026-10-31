package sftp

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

// settingsRepository keeps the settings of SFTP storages in storages_sftp.
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
	StorageID           uuid.UUID `gorm:"column:storage_id;primaryKey"`
	Host                string    `gorm:"column:host"`
	Port                int       `gorm:"column:port"`
	Username            string    `gorm:"column:username"`
	Password            *string   `gorm:"column:password"`
	PrivateKey          *string   `gorm:"column:private_key"`
	Path                *string   `gorm:"column:path"`
	IsSkipHostKeyVerify bool      `gorm:"column:skip_host_key_verify"`
}

func (settingsRow) TableName() string {
	return "storages_sftp"
}

func (r *settingsRepository) Upsert(ctx context.Context, storageID uuid.UUID, settings providers.Settings) error {
	sftpSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	encryptedPassword, err := r.fieldEncryptor.Encrypt(sftpSettings.Password)
	if err != nil {
		return fmt.Errorf("encrypt the SFTP password: %w", err)
	}

	encryptedPrivateKey, err := r.fieldEncryptor.Encrypt(sftpSettings.PrivateKey)
	if err != nil {
		return fmt.Errorf("encrypt the SFTP private key: %w", err)
	}

	row := settingsRow{
		StorageID:           storageID,
		Host:                sftpSettings.Host,
		Port:                sftpSettings.Port,
		Username:            sftpSettings.Username,
		Password:            &encryptedPassword,
		PrivateKey:          &encryptedPrivateKey,
		Path:                &sftpSettings.Path,
		IsSkipHostKeyVerify: sftpSettings.IsSkipHostKeyVerify,
	}

	err = database.Conn(ctx, r.db).Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error
	if err != nil {
		return fmt.Errorf("upsert the SFTP settings: %w", err)
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
		return nil, fmt.Errorf("list the SFTP settings: %w", err)
	}

	settingsByStorageID := make(map[uuid.UUID]providers.Settings, len(rows))
	for _, row := range rows {
		authMethods := make([]AuthMethod, 0, 2)
		if valueOrEmpty(row.Password) != "" {
			authMethods = append(authMethods, AuthMethodPassword)
		}

		if valueOrEmpty(row.PrivateKey) != "" {
			authMethods = append(authMethods, AuthMethodPrivateKey)
		}

		settingsByStorageID[row.StorageID] = Settings{
			AuthMethods:         authMethods,
			Host:                row.Host,
			Port:                row.Port,
			Username:            row.Username,
			Password:            valueOrEmpty(row.Password),
			PrivateKey:          valueOrEmpty(row.PrivateKey),
			Path:                valueOrEmpty(row.Path),
			IsSkipHostKeyVerify: row.IsSkipHostKeyVerify,
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
