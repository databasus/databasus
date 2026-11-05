package postgresqllogical

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/namelist"
	"databasus-backend/internal/modules/databases/internal/providers"
)

// settingsRepository keeps the settings of logical PostgreSQL databases in databases_postgresql_logical.
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
	DatabaseID         uuid.UUID                      `gorm:"column:database_id;primaryKey"`
	Version            pgshared.Version               `gorm:"column:version"`
	Host               string                         `gorm:"column:host"`
	Port               int                            `gorm:"column:port"`
	Username           string                         `gorm:"column:username"`
	Password           string                         `gorm:"column:password"`
	Database           *string                        `gorm:"column:database"`
	TLS                providers.PostgresqlTLSColumns `gorm:"embedded"`
	SSHTunnel          providers.SSHTunnelColumns     `gorm:"embedded;embeddedPrefix:ssh_"`
	IncludeSchemas     pq.StringArray                 `gorm:"column:include_schemas;type:text[]"`
	ExcludeTables      pq.StringArray                 `gorm:"column:exclude_tables;type:text[]"`
	ExcludeTableData   pq.StringArray                 `gorm:"column:exclude_table_data;type:text[]"`
	CPUCount           int                            `gorm:"column:cpu_count"`
	IsSkipUserMappings bool                           `gorm:"column:is_skip_user_mappings"`
}

func (settingsRow) TableName() string {
	return "databases_postgresql_logical"
}

func (r *settingsRepository) Upsert(ctx context.Context, databaseID uuid.UUID, settings providers.Settings) error {
	logicalSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	encryptedPassword, err := r.fieldEncryptor.Encrypt(logicalSettings.Password)
	if err != nil {
		return fmt.Errorf("encrypt the PostgreSQL password: %w", err)
	}

	tlsColumns, err := providers.NewPostgresqlTLSColumns(r.fieldEncryptor, logicalSettings.TLS)
	if err != nil {
		return err
	}

	sshTunnelColumns, err := providers.NewSSHTunnelColumns(r.fieldEncryptor, logicalSettings.SSHTunnel)
	if err != nil {
		return err
	}

	row := settingsRow{
		DatabaseID:         databaseID,
		Version:            logicalSettings.Version,
		Host:               logicalSettings.Host,
		Port:               logicalSettings.Port,
		Username:           logicalSettings.Username,
		Password:           encryptedPassword,
		Database:           &logicalSettings.Database,
		TLS:                tlsColumns,
		SSHTunnel:          sshTunnelColumns,
		IncludeSchemas:     namelist.NormalizeUniqueNames(logicalSettings.IncludeSchemas),
		ExcludeTables:      namelist.NormalizeUniqueNames(logicalSettings.ExcludeTables),
		ExcludeTableData:   namelist.NormalizeUniqueNames(logicalSettings.ExcludeTableData),
		CPUCount:           logicalSettings.CPUCount,
		IsSkipUserMappings: logicalSettings.IsSkipUserMappings,
	}

	err = database.Conn(ctx, r.db).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "database_id"}}, UpdateAll: true}).
		Create(&row).Error
	if err != nil {
		return fmt.Errorf("upsert the logical PostgreSQL settings: %w", err)
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
		return nil, fmt.Errorf("list the logical PostgreSQL settings: %w", err)
	}

	settingsByDatabaseID := make(map[uuid.UUID]providers.Settings, len(rows))

	for _, row := range rows {
		tlsSettings, err := row.TLS.ToSettings(r.fieldEncryptor)
		if err != nil {
			return nil, err
		}

		databaseName := ""
		if row.Database != nil {
			databaseName = *row.Database
		}

		settingsByDatabaseID[row.DatabaseID] = Settings{
			Version:            row.Version,
			Host:               row.Host,
			Port:               row.Port,
			Username:           row.Username,
			Password:           row.Password,
			Database:           databaseName,
			TLS:                tlsSettings,
			SSHTunnel:          row.SSHTunnel.ToSettings(),
			IncludeSchemas:     row.IncludeSchemas,
			ExcludeTables:      row.ExcludeTables,
			ExcludeTableData:   row.ExcludeTableData,
			CPUCount:           row.CPUCount,
			IsSkipUserMappings: row.IsSkipUserMappings,
		}
	}

	return settingsByDatabaseID, nil
}

func (r *settingsRepository) listVersions(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	var rows []struct {
		DatabaseID uuid.UUID
		Version    string
	}
	err := database.Conn(ctx, r.db).
		Table("databases_postgresql_logical").
		Select("database_id, version").
		Where("database_id IN ? AND version <> ''", ids).
		Scan(&rows).
		Error
	if err != nil {
		return nil, fmt.Errorf("list logical PostgreSQL versions: %w", err)
	}
	versions := make(map[uuid.UUID]string, len(rows))
	for _, row := range rows {
		versions[row.DatabaseID] = row.Version
	}
	return versions, nil
}
