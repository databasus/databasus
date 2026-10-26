package mongodb

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	mongodbengine "databasus-backend/internal/engines/mongodb"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/namelist"
	"databasus-backend/internal/modules/databases/internal/providers"
)

// settingsRepository keeps the settings of MongoDB databases in databases_mongodb.
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
	DatabaseID uuid.UUID             `gorm:"column:database_id;primaryKey"`
	Version    mongodbengine.Version `gorm:"column:version"`
	Host       string                `gorm:"column:host"`
	// Port is NULL for an SRV connection.
	Port               *int                       `gorm:"column:port"`
	Username           string                     `gorm:"column:username"`
	Password           string                     `gorm:"column:password"`
	Database           string                     `gorm:"column:database"`
	AuthDatabase       string                     `gorm:"column:auth_database"`
	IsHTTPS            bool                       `gorm:"column:is_https"`
	IsSrv              bool                       `gorm:"column:is_srv"`
	IsDirectConnection bool                       `gorm:"column:is_direct_connection"`
	CPUCount           int                        `gorm:"column:cpu_count"`
	SSHTunnel          providers.SSHTunnelColumns `gorm:"embedded;embeddedPrefix:ssh_"`
	ExcludeCollections pq.StringArray             `gorm:"column:exclude_collections;type:text[]"`
}

func (settingsRow) TableName() string {
	return "databases_mongodb"
}

func (r *settingsRepository) Upsert(ctx context.Context, databaseID uuid.UUID, settings providers.Settings) error {
	mongodbSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	encryptedPassword, err := r.fieldEncryptor.Encrypt(mongodbSettings.Password)
	if err != nil {
		return fmt.Errorf("encrypt the MongoDB password: %w", err)
	}

	sshTunnelColumns, err := providers.NewSSHTunnelColumns(r.fieldEncryptor, mongodbSettings.SSHTunnel)
	if err != nil {
		return err
	}

	var port *int
	if mongodbSettings.Port != 0 {
		port = &mongodbSettings.Port
	}

	row := settingsRow{
		DatabaseID:         databaseID,
		Version:            mongodbSettings.Version,
		Host:               mongodbSettings.Host,
		Port:               port,
		Username:           mongodbSettings.Username,
		Password:           encryptedPassword,
		Database:           mongodbSettings.Database,
		AuthDatabase:       mongodbSettings.AuthDatabase,
		IsHTTPS:            mongodbSettings.IsHTTPS,
		IsSrv:              mongodbSettings.IsSrv,
		IsDirectConnection: mongodbSettings.IsDirectConnection,
		CPUCount:           mongodbSettings.CPUCount,
		SSHTunnel:          sshTunnelColumns,
		ExcludeCollections: namelist.NormalizeUniqueNames(mongodbSettings.ExcludeCollections),
	}

	err = database.Conn(ctx, r.db).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "database_id"}}, UpdateAll: true}).
		Create(&row).Error
	if err != nil {
		return fmt.Errorf("upsert the MongoDB settings: %w", err)
	}

	return nil
}

func (r *settingsRepository) ListByDatabaseIDs(
	ctx context.Context,
	databaseIDs []uuid.UUID,
) (map[uuid.UUID]providers.Settings, error) {
	var rows []settingsRow

	err := database.Conn(ctx, r.db).Where("database_id IN ?", databaseIDs).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list the MongoDB settings: %w", err)
	}

	settingsByDatabaseID := make(map[uuid.UUID]providers.Settings, len(rows))

	for _, row := range rows {
		port := 0
		if row.Port != nil {
			port = *row.Port
		}

		settingsByDatabaseID[row.DatabaseID] = Settings{
			Version:            row.Version,
			Host:               row.Host,
			Port:               port,
			Username:           row.Username,
			Password:           row.Password,
			Database:           row.Database,
			AuthDatabase:       row.AuthDatabase,
			IsHTTPS:            row.IsHTTPS,
			IsSrv:              row.IsSrv,
			IsDirectConnection: row.IsDirectConnection,
			CPUCount:           row.CPUCount,
			SSHTunnel:          row.SSHTunnel.ToSettings(),
			ExcludeCollections: row.ExcludeCollections,
		}
	}

	return settingsByDatabaseID, nil
}
