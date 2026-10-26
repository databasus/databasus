package mariadb

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	mariadbengine "databasus-backend/internal/engines/mariadb"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/namelist"
	"databasus-backend/internal/modules/databases/internal/providers"
)

// settingsRepository keeps the settings of MariaDB databases in databases_mariadb.
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
	DatabaseID          uuid.UUID                  `gorm:"column:database_id;primaryKey"`
	Version             mariadbengine.Version      `gorm:"column:version"`
	Host                string                     `gorm:"column:host"`
	Port                int                        `gorm:"column:port"`
	Username            string                     `gorm:"column:username"`
	Password            string                     `gorm:"column:password"`
	Database            *string                    `gorm:"column:database"`
	IsHTTPS             bool                       `gorm:"column:is_https"`
	SSHTunnel           providers.SSHTunnelColumns `gorm:"embedded;embeddedPrefix:ssh_"`
	ExcludeTables       pq.StringArray             `gorm:"column:exclude_tables;type:text[]"`
	Privileges          pq.StringArray             `gorm:"column:privileges;type:text[]"`
	IsExcludeEvents     bool                       `gorm:"column:is_exclude_events"`
	IsSkipGaleraDisable bool                       `gorm:"column:is_skip_galera_disable"`
}

func (settingsRow) TableName() string {
	return "databases_mariadb"
}

func (r *settingsRepository) Upsert(ctx context.Context, databaseID uuid.UUID, settings providers.Settings) error {
	mariadbSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	encryptedPassword, err := r.fieldEncryptor.Encrypt(mariadbSettings.Password)
	if err != nil {
		return fmt.Errorf("encrypt the MariaDB password: %w", err)
	}

	sshTunnelColumns, err := providers.NewSSHTunnelColumns(r.fieldEncryptor, mariadbSettings.SSHTunnel)
	if err != nil {
		return err
	}

	row := settingsRow{
		DatabaseID:          databaseID,
		Version:             mariadbSettings.Version,
		Host:                mariadbSettings.Host,
		Port:                mariadbSettings.Port,
		Username:            mariadbSettings.Username,
		Password:            encryptedPassword,
		Database:            &mariadbSettings.Database,
		IsHTTPS:             mariadbSettings.IsHTTPS,
		SSHTunnel:           sshTunnelColumns,
		ExcludeTables:       namelist.NormalizeUniqueNames(mariadbSettings.ExcludeTables),
		Privileges:          namelist.NormalizeUniqueNames(mariadbSettings.Privileges),
		IsExcludeEvents:     mariadbSettings.IsExcludeEvents,
		IsSkipGaleraDisable: mariadbSettings.IsSkipGaleraDisable,
	}

	err = database.Conn(ctx, r.db).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "database_id"}}, UpdateAll: true}).
		Create(&row).Error
	if err != nil {
		return fmt.Errorf("upsert the MariaDB settings: %w", err)
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
		return nil, fmt.Errorf("list the MariaDB settings: %w", err)
	}

	settingsByDatabaseID := make(map[uuid.UUID]providers.Settings, len(rows))

	for _, row := range rows {
		databaseName := ""
		if row.Database != nil {
			databaseName = *row.Database
		}

		settingsByDatabaseID[row.DatabaseID] = Settings{
			Version:             row.Version,
			Host:                row.Host,
			Port:                row.Port,
			Username:            row.Username,
			Password:            row.Password,
			Database:            databaseName,
			IsHTTPS:             row.IsHTTPS,
			SSHTunnel:           row.SSHTunnel.ToSettings(),
			ExcludeTables:       row.ExcludeTables,
			Privileges:          row.Privileges,
			IsExcludeEvents:     row.IsExcludeEvents,
			IsSkipGaleraDisable: row.IsSkipGaleraDisable,
		}
	}

	return settingsByDatabaseID, nil
}
