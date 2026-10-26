package postgresql_logical

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	pglogical "databasus-backend/internal/engines/postgresql/logical"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/features/sshtunnel"
	"databasus-backend/internal/lib/namelist"
	libsshtunnel "databasus-backend/internal/lib/sshtunnel"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

type PostgresqlLogicalDatabase struct {
	ID uuid.UUID `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`

	DatabaseID *uuid.UUID `json:"databaseId" gorm:"type:uuid;column:database_id"`

	Version pgshared.Version `json:"version" gorm:"type:text;not null"`

	Host     string  `json:"host"     gorm:"type:text;not null"`
	Port     int     `json:"port"     gorm:"type:int;not null"`
	Username string  `json:"username" gorm:"type:text;not null"`
	Password string  `json:"password" gorm:"type:text;not null"`
	Database *string `json:"database" gorm:"type:text"`

	// SSL / TLS connection settings
	SslMode       pgshared.SslMode `json:"sslMode"       gorm:"column:ssl_mode;type:text;not null;default:'disable'"`
	SslClientCert string           `json:"sslClientCert" gorm:"column:ssl_client_cert;type:text;not null;default:''"`
	SslClientKey  string           `json:"sslClientKey"  gorm:"column:ssl_client_key;type:text;not null;default:''"`
	SslRootCert   string           `json:"sslRootCert"   gorm:"column:ssl_root_cert;type:text;not null;default:''"`

	// When the tunnel is enabled, Host and Port above address the database as the bastion sees it.
	SshTunnel sshtunnel.Config `json:"sshTunnel" gorm:"embedded;embeddedPrefix:ssh_"`

	// Set only on the copy handed out by OpenTunnel, so the credential spec can point libpq at the
	// forwarded port while Host keeps the name that TLS and .pgpass are matched against.
	LocalTunnelEndpoint *libsshtunnel.Endpoint `json:"-" gorm:"-"`

	// backup settings
	IncludeSchemas         []string `json:"includeSchemas"     gorm:"-"`
	IncludeSchemasString   string   `json:"-"                  gorm:"column:include_schemas;type:text;not null;default:''"`
	ExcludeTables          []string `json:"excludeTables"      gorm:"-"`
	ExcludeTablesString    string   `json:"-"                  gorm:"column:exclude_tables;type:text;not null;default:''"`
	ExcludeTableData       []string `json:"excludeTableData"   gorm:"-"`
	ExcludeTableDataString string   `json:"-"                  gorm:"column:exclude_table_data;type:text;not null;default:''"`
	CpuCount               int      `json:"cpuCount"           gorm:"column:cpu_count;type:int;not null;default:1"`
	IsSkipUserMappings     bool     `json:"isSkipUserMappings" gorm:"column:is_skip_user_mappings;type:bool;not null;default:false"`

	// restore settings (not saved to DB)
	IsExcludeExtensions bool `json:"isExcludeExtensions" gorm:"-"`
	IsRestoreOwnership  bool `json:"isRestoreOwnership"  gorm:"-"`
	IsRestorePrivileges bool `json:"isRestorePrivileges" gorm:"-"`
}

func (p *PostgresqlLogicalDatabase) TableName() string {
	return "postgresql_logical_databases"
}

func (p *PostgresqlLogicalDatabase) BeforeSave(_ *gorm.DB) error {
	p.IncludeSchemasString = namelist.FormatUniqueNames(p.IncludeSchemas)
	p.ExcludeTablesString = namelist.FormatUniqueNames(p.ExcludeTables)
	p.ExcludeTableDataString = namelist.FormatUniqueNames(p.ExcludeTableData)

	return nil
}

func (p *PostgresqlLogicalDatabase) AfterFind(_ *gorm.DB) error {
	p.IncludeSchemas = namelist.ParseUniqueNames(p.IncludeSchemasString)
	p.ExcludeTables = namelist.ParseUniqueNames(p.ExcludeTablesString)
	p.ExcludeTableData = namelist.ParseUniqueNames(p.ExcludeTableDataString)

	return nil
}

func (p *PostgresqlLogicalDatabase) Validate() error {
	if p.SslMode == "" {
		p.SslMode = pgshared.SslModeDisable
	}

	if p.Host == "" {
		return errors.New("host is required")
	}

	if p.Port == 0 {
		return errors.New("port is required")
	}

	if p.Username == "" {
		return errors.New("username is required")
	}

	if p.Password == "" {
		return errors.New("password is required")
	}

	if p.CpuCount <= 0 {
		return errors.New("cpu count must be greater than 0")
	}

	if err := p.validateSslConfig(); err != nil {
		return err
	}

	if err := p.SshTunnel.Validate(); err != nil {
		return err
	}

	if err := p.ValidateNotEmbeddedTarget(); err != nil {
		return err
	}

	return nil
}

func (p *PostgresqlLogicalDatabase) TestConnection(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	source, err := p.newSource(encryptor)
	if err != nil {
		return err
	}

	detectedVersion, err := source.TestConnection(ctx, logger, pglogical.DumpCheck{
		Filter:             p.getDumpFilter(),
		IsSkipUserMappings: p.IsSkipUserMappings,
	})
	if err != nil {
		return err
	}

	p.Version = detectedVersion

	return nil
}

// The value feeds verification disk planning and the restored-size sanity check,
// so it must cover exactly the relations pg_dump emits for this database.
func (p *PostgresqlLogicalDatabase) GetRawDbSizeMb(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) (float64, error) {
	if p.Database == nil || *p.Database == "" {
		return 0, nil
	}

	source, err := p.newSource(encryptor)
	if err != nil {
		return 0, err
	}

	return source.GetDumpedSizeMb(ctx, logger, p.getDumpFilter())
}

func (p *PostgresqlLogicalDatabase) HideSensitiveData() {
	if p == nil {
		return
	}

	p.Password = ""
	p.SslClientKey = ""
	p.SshTunnel.HideSensitiveData()
}

func (p *PostgresqlLogicalDatabase) ValidateUpdate(_ *PostgresqlLogicalDatabase) error {
	return nil
}

func (p *PostgresqlLogicalDatabase) Update(incoming *PostgresqlLogicalDatabase) {
	p.Version = incoming.Version
	p.Host = incoming.Host
	p.Port = incoming.Port
	p.Username = incoming.Username
	p.Database = incoming.Database
	p.SslMode = incoming.SslMode
	p.SslClientCert = incoming.SslClientCert
	p.SslRootCert = incoming.SslRootCert
	p.IncludeSchemas = incoming.IncludeSchemas
	p.ExcludeTables = incoming.ExcludeTables
	p.ExcludeTableData = incoming.ExcludeTableData
	p.CpuCount = incoming.CpuCount
	p.IsSkipUserMappings = incoming.IsSkipUserMappings
	p.SshTunnel.Update(&incoming.SshTunnel)

	if incoming.Password != "" {
		p.Password = incoming.Password
	}

	if incoming.SslClientKey != "" {
		p.SslClientKey = incoming.SslClientKey
	}
}

// LocalTunnelEndpoint belongs to the operation that opened the tunnel, not to the configuration.
func (p *PostgresqlLogicalDatabase) CopyForNewDatabase() *PostgresqlLogicalDatabase {
	if p == nil {
		return nil
	}

	copiedDatabase := *p
	copiedDatabase.ID = uuid.Nil
	copiedDatabase.DatabaseID = nil
	copiedDatabase.IncludeSchemas = slices.Clone(p.IncludeSchemas)
	copiedDatabase.ExcludeTables = slices.Clone(p.ExcludeTables)
	copiedDatabase.ExcludeTableData = slices.Clone(p.ExcludeTableData)
	copiedDatabase.LocalTunnelEndpoint = nil

	if p.Database != nil {
		copiedDatabase.Database = new(*p.Database)
	}

	return &copiedDatabase
}

func (p *PostgresqlLogicalDatabase) EncryptSensitiveFields(
	encryptor encryption.FieldEncryptor,
) error {
	for _, field := range []*string{
		&p.Password,
		&p.SslClientCert,
		&p.SslClientKey,
		&p.SslRootCert,
	} {
		if *field == "" {
			continue
		}

		encrypted, err := encryptor.Encrypt(*field)
		if err != nil {
			return err
		}

		*field = encrypted
	}

	return p.SshTunnel.EncryptSensitiveFields(encryptor)
}

// PopulateDbData detects and sets the PostgreSQL version.
// This should be called before encrypting sensitive fields.
func (p *PostgresqlLogicalDatabase) PopulateDbData(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	return p.PopulateVersion(ctx, logger, encryptor)
}

// PopulateVersion detects and sets the PostgreSQL version by querying the database.
func (p *PostgresqlLogicalDatabase) PopulateVersion(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	if p.Database == nil || *p.Database == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	source, err := p.newSource(encryptor)
	if err != nil {
		return err
	}

	detectedVersion, err := source.DetectVersion(ctx, logger)
	if err != nil {
		return err
	}

	p.Version = detectedVersion

	return nil
}

func (p *PostgresqlLogicalDatabase) ShouldSuggestReadOnlyUser(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) (bool, []string, error) {
	source, err := p.newSource(encryptor)
	if err != nil {
		return false, nil, err
	}

	return source.ShouldSuggestReadOnlyUser(ctx, logger, p.IncludeSchemas)
}

func (p *PostgresqlLogicalDatabase) CreateReadOnlyUser(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) (string, string, error) {
	source, err := p.newSource(encryptor)
	if err != nil {
		return "", "", err
	}

	return source.CreateReadOnlyUser(ctx, logger, p.IncludeSchemas)
}

// GetTimescaleDBVersion returns the installed timescaledb extension version, or "" when the
// extension is absent.
func (p *PostgresqlLogicalDatabase) GetTimescaleDBVersion(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
) (string, error) {
	source, err := p.newSource(encryptor)
	if err != nil {
		return "", err
	}

	return source.GetTimescaleDBVersion(ctx, logger.GetLogger())
}

func (p *PostgresqlLogicalDatabase) RunTimescaleDBPreRestore(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
) error {
	source, err := p.newSource(encryptor)
	if err != nil {
		return err
	}

	return source.RunTimescaleDBPreRestore(ctx, logger.GetLogger())
}

func (p *PostgresqlLogicalDatabase) RunTimescaleDBPostRestore(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
) error {
	source, err := p.newSource(encryptor)
	if err != nil {
		return err
	}

	return source.RunTimescaleDBPostRestore(ctx, logger.GetLogger())
}

func (p *PostgresqlLogicalDatabase) ValidateNotEmbeddedTarget() error {
	databaseName := ""
	if p.Database != nil {
		databaseName = *p.Database
	}

	return pgshared.ValidateNotEmbeddedTarget(pgshared.EmbeddedTargetSpec{
		Host:               p.Host,
		Port:               p.Port,
		DatabaseName:       databaseName,
		IsSSHTunnelEnabled: p.SshTunnel.IsEnabled,
		SSHBastionHost:     p.SshTunnel.Host,
	})
}

func (p *PostgresqlLogicalDatabase) validateSslConfig() error {
	return pgshared.TLS{
		Mode:       p.SslMode,
		ClientCert: p.SslClientCert,
		ClientKey:  p.SslClientKey,
		RootCert:   p.SslRootCert,
	}.Validate()
}
