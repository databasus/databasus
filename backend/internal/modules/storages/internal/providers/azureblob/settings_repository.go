package azureblob

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

// settingsRepository keeps the settings of Azure Blob storages in storages_azure_blob.
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
	StorageID        uuid.UUID  `gorm:"column:storage_id;primaryKey"`
	AuthMethod       AuthMethod `gorm:"column:auth_method"`
	ConnectionString *string    `gorm:"column:connection_string"`
	AccountName      *string    `gorm:"column:account_name"`
	AccountKey       *string    `gorm:"column:account_key"`
	ContainerName    string     `gorm:"column:container_name"`
	Endpoint         *string    `gorm:"column:endpoint"`
	Prefix           *string    `gorm:"column:prefix"`
}

func (settingsRow) TableName() string {
	return "storages_azure_blob"
}

func (r *settingsRepository) Upsert(ctx context.Context, storageID uuid.UUID, settings providers.Settings) error {
	azureSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	encryptedConnectionString, err := r.fieldEncryptor.Encrypt(azureSettings.ConnectionString)
	if err != nil {
		return fmt.Errorf("encrypt the Azure connection string: %w", err)
	}

	encryptedAccountKey, err := r.fieldEncryptor.Encrypt(azureSettings.AccountKey)
	if err != nil {
		return fmt.Errorf("encrypt the Azure account key: %w", err)
	}

	row := settingsRow{
		StorageID:        storageID,
		AuthMethod:       azureSettings.AuthMethod,
		ConnectionString: &encryptedConnectionString,
		AccountName:      &azureSettings.AccountName,
		AccountKey:       &encryptedAccountKey,
		ContainerName:    azureSettings.ContainerName,
		Endpoint:         &azureSettings.Endpoint,
		Prefix:           &azureSettings.Prefix,
	}

	err = database.Conn(ctx, r.db).Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error
	if err != nil {
		return fmt.Errorf("upsert the Azure Blob settings: %w", err)
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
		return nil, fmt.Errorf("list the Azure Blob settings: %w", err)
	}

	settingsByStorageID := make(map[uuid.UUID]providers.Settings, len(rows))
	for _, row := range rows {
		settingsByStorageID[row.StorageID] = Settings{
			AuthMethod:       row.AuthMethod,
			ConnectionString: valueOrEmpty(row.ConnectionString),
			AccountName:      valueOrEmpty(row.AccountName),
			AccountKey:       valueOrEmpty(row.AccountKey),
			ContainerName:    row.ContainerName,
			Endpoint:         valueOrEmpty(row.Endpoint),
			Prefix:           valueOrEmpty(row.Prefix),
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
