package postgresqlphysical

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/databases/internal/providers"
)

// settingsRepository keeps the settings of physical PostgreSQL databases in
// databases_postgresql_physical.
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
	DatabaseID          uuid.UUID                      `gorm:"column:database_id;primaryKey"`
	Version             pgshared.Version               `gorm:"column:version"`
	Host                string                         `gorm:"column:host"`
	Port                int                            `gorm:"column:port"`
	Username            string                         `gorm:"column:username"`
	Password            string                         `gorm:"column:password"`
	TLS                 providers.PostgresqlTLSColumns `gorm:"embedded"`
	SSHTunnel           providers.SSHTunnelColumns     `gorm:"embedded;embeddedPrefix:ssh_"`
	BackupType          providers.PhysicalBackupType   `gorm:"column:backup_type"`
	ReplicationSlotName string                         `gorm:"column:replication_slot_name"`
	BackupSlotName      string                         `gorm:"column:backup_slot_name"`
	SystemIdentifier    *string                        `gorm:"column:system_identifier"`
	WalSegmentSizeBytes *int64                         `gorm:"column:wal_segment_size_bytes"`
}

func (settingsRow) TableName() string {
	return "databases_postgresql_physical"
}

func (r *settingsRepository) Upsert(ctx context.Context, databaseID uuid.UUID, settings providers.Settings) error {
	physicalSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	encryptedPassword, err := r.fieldEncryptor.Encrypt(physicalSettings.Password)
	if err != nil {
		return fmt.Errorf("encrypt the PostgreSQL password: %w", err)
	}

	tlsColumns, err := providers.NewPostgresqlTLSColumns(r.fieldEncryptor, physicalSettings.TLS)
	if err != nil {
		return err
	}

	sshTunnelColumns, err := providers.NewSSHTunnelColumns(r.fieldEncryptor, physicalSettings.SSHTunnel)
	if err != nil {
		return err
	}

	row := settingsRow{
		DatabaseID:          databaseID,
		Version:             physicalSettings.Version,
		Host:                physicalSettings.Host,
		Port:                physicalSettings.Port,
		Username:            physicalSettings.Username,
		Password:            encryptedPassword,
		TLS:                 tlsColumns,
		SSHTunnel:           sshTunnelColumns,
		BackupType:          physicalSettings.BackupType,
		ReplicationSlotName: physicalSettings.ReplicationSlotName,
		BackupSlotName:      physicalSettings.BackupSlotName,
		SystemIdentifier:    physicalSettings.SystemIdentifier,
		WalSegmentSizeBytes: physicalSettings.WalSegmentSizeBytes,
	}

	err = database.Conn(ctx, r.db).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "database_id"}}, UpdateAll: true}).
		Create(&row).Error
	if err != nil {
		return fmt.Errorf("upsert the physical PostgreSQL settings: %w", err)
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
		return nil, fmt.Errorf("list the physical PostgreSQL settings: %w", err)
	}

	settingsByDatabaseID := make(map[uuid.UUID]providers.Settings, len(rows))

	for _, row := range rows {
		tlsSettings, err := row.TLS.ToSettings(r.fieldEncryptor)
		if err != nil {
			return nil, err
		}

		settingsByDatabaseID[row.DatabaseID] = Settings{
			Version:             row.Version,
			Host:                row.Host,
			Port:                row.Port,
			Username:            row.Username,
			Password:            row.Password,
			TLS:                 tlsSettings,
			SSHTunnel:           row.SSHTunnel.ToSettings(),
			BackupType:          row.BackupType,
			ReplicationSlotName: row.ReplicationSlotName,
			BackupSlotName:      row.BackupSlotName,
			SystemIdentifier:    row.SystemIdentifier,
			WalSegmentSizeBytes: row.WalSegmentSizeBytes,
		}
	}

	return settingsByDatabaseID, nil
}

// listBackupTypes reads only the type column, so no secret is decrypted.
func (r *settingsRepository) listBackupTypes(
	ctx context.Context,
	databaseIDs []uuid.UUID,
) (map[uuid.UUID]providers.PhysicalBackupType, error) {
	var rows []struct {
		DatabaseID uuid.UUID                    `gorm:"column:database_id"`
		BackupType providers.PhysicalBackupType `gorm:"column:backup_type"`
	}

	err := database.Conn(ctx, r.db).
		Raw("SELECT database_id, backup_type FROM databases_postgresql_physical WHERE database_id IN ?", databaseIDs).
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list the physical backup types: %w", err)
	}

	backupTypesByDatabaseID := make(map[uuid.UUID]providers.PhysicalBackupType, len(rows))
	for _, row := range rows {
		backupTypesByDatabaseID[row.DatabaseID] = row.BackupType
	}

	return backupTypesByDatabaseID, nil
}
