package s3

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

// settingsRepository keeps the settings of S3 storages in storages_s3.
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
	StorageID            uuid.UUID `gorm:"column:storage_id;primaryKey"`
	Bucket               string    `gorm:"column:s3_bucket"`
	Region               string    `gorm:"column:s3_region"`
	AccessKey            string    `gorm:"column:s3_access_key"`
	SecretKey            string    `gorm:"column:s3_secret_key"`
	Endpoint             *string   `gorm:"column:s3_endpoint"`
	Prefix               *string   `gorm:"column:s3_prefix"`
	IsVirtualHostedStyle bool      `gorm:"column:s3_use_virtual_hosted_style"`
	IsSkipTLSVerify      bool      `gorm:"column:skip_tls_verify"`
	StorageClass         string    `gorm:"column:s3_storage_class"`
}

func (settingsRow) TableName() string {
	return "storages_s3"
}

func (r *settingsRepository) Upsert(ctx context.Context, storageID uuid.UUID, settings providers.Settings) error {
	s3Settings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	encryptedAccessKey, err := r.fieldEncryptor.Encrypt(s3Settings.AccessKey)
	if err != nil {
		return fmt.Errorf("encrypt the S3 access key: %w", err)
	}

	encryptedSecretKey, err := r.fieldEncryptor.Encrypt(s3Settings.SecretKey)
	if err != nil {
		return fmt.Errorf("encrypt the S3 secret key: %w", err)
	}

	row := settingsRow{
		StorageID:            storageID,
		Bucket:               s3Settings.Bucket,
		Region:               s3Settings.Region,
		AccessKey:            encryptedAccessKey,
		SecretKey:            encryptedSecretKey,
		Endpoint:             &s3Settings.Endpoint,
		Prefix:               &s3Settings.Prefix,
		IsVirtualHostedStyle: s3Settings.IsVirtualHostedStyle,
		IsSkipTLSVerify:      s3Settings.IsSkipTLSVerify,
		StorageClass:         s3Settings.StorageClass,
	}

	err = database.Conn(ctx, r.db).Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error
	if err != nil {
		return fmt.Errorf("upsert the S3 settings: %w", err)
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
		return nil, fmt.Errorf("list the S3 settings: %w", err)
	}

	settingsByStorageID := make(map[uuid.UUID]providers.Settings, len(rows))
	for _, row := range rows {
		settingsByStorageID[row.StorageID] = Settings{
			Bucket:               row.Bucket,
			Region:               row.Region,
			AccessKey:            row.AccessKey,
			SecretKey:            row.SecretKey,
			Endpoint:             valueOrEmpty(row.Endpoint),
			Prefix:               valueOrEmpty(row.Prefix),
			IsVirtualHostedStyle: row.IsVirtualHostedStyle,
			IsSkipTLSVerify:      row.IsSkipTLSVerify,
			StorageClass:         row.StorageClass,
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
