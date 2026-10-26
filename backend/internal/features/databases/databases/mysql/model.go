package mysql

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	mysql_engine "databasus-backend/internal/engines/mysql"
	"databasus-backend/internal/features/sshtunnel"
	"databasus-backend/internal/lib/namelist"
	"databasus-backend/internal/util/encryption"
)

const connectionTimeout = 15 * time.Second

type MysqlDatabase struct {
	ID         uuid.UUID  `json:"id"         gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	DatabaseID *uuid.UUID `json:"databaseId" gorm:"type:uuid;column:database_id"`

	Version mysql_engine.Version `json:"version" gorm:"type:text;not null"`

	Host     string  `json:"host"     gorm:"type:text;not null"`
	Port     int     `json:"port"     gorm:"type:int;not null"`
	Username string  `json:"username" gorm:"type:text;not null"`
	Password string  `json:"password" gorm:"type:text;not null"`
	Database *string `json:"database" gorm:"type:text"`
	IsHttps  bool    `json:"isHttps"  gorm:"type:boolean;default:false"`

	// When the tunnel is enabled, Host and Port above address the database as the bastion sees it.
	SshTunnel sshtunnel.Config `json:"sshTunnel" gorm:"embedded;embeddedPrefix:ssh_"`

	ExcludeTables       []string `json:"excludeTables" gorm:"-"`
	ExcludeTablesString string   `json:"-"             gorm:"column:exclude_tables;type:text;not null;default:''"`
	Privileges          string   `json:"privileges"    gorm:"column:privileges;type:text;not null;default:''"`
}

func (m *MysqlDatabase) TableName() string {
	return "mysql_databases"
}

func (m *MysqlDatabase) BeforeSave(_ *gorm.DB) error {
	m.ExcludeTablesString = namelist.FormatUniqueNames(m.ExcludeTables)

	return nil
}

func (m *MysqlDatabase) AfterFind(_ *gorm.DB) error {
	m.ExcludeTables = namelist.ParseUniqueNames(m.ExcludeTablesString)

	return nil
}

func (m *MysqlDatabase) Validate() error {
	if m.Host == "" {
		return errors.New("host is required")
	}
	if m.Port == 0 {
		return errors.New("port is required")
	}
	if m.Username == "" {
		return errors.New("username is required")
	}
	if m.Password == "" {
		return errors.New("password is required")
	}

	return m.SshTunnel.Validate()
}

func (m *MysqlDatabase) TestConnection(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	ctx, cancel := context.WithTimeout(ctx, connectionTimeout)
	defer cancel()

	if m.Database == nil || *m.Database == "" {
		return mysql_engine.ErrDatabaseNameRequired
	}

	source, err := m.newSource(encryptor)
	if err != nil {
		return err
	}

	inspection, err := source.TestConnection(ctx, logger, m.ExcludeTables)
	if err != nil {
		return err
	}

	m.writeInspection(inspection)

	return nil
}

func (m *MysqlDatabase) GetRawDbSizeMb(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) (float64, error) {
	if m.Database == nil || *m.Database == "" {
		return 0, nil
	}

	source, err := m.newSource(encryptor)
	if err != nil {
		return 0, err
	}

	return source.GetRawSizeMb(ctx, logger, m.Version)
}

func (m *MysqlDatabase) HideSensitiveData() {
	if m == nil {
		return
	}
	m.Password = ""
	m.SshTunnel.HideSensitiveData()
}

func (m *MysqlDatabase) Update(incoming *MysqlDatabase) {
	m.Version = incoming.Version
	m.Host = incoming.Host
	m.Port = incoming.Port
	m.Username = incoming.Username
	m.Database = incoming.Database
	m.IsHttps = incoming.IsHttps
	m.ExcludeTables = incoming.ExcludeTables
	m.Privileges = incoming.Privileges
	m.SshTunnel.Update(&incoming.SshTunnel)

	if incoming.Password != "" {
		m.Password = incoming.Password
	}
}

func (m *MysqlDatabase) CopyForNewDatabase() *MysqlDatabase {
	if m == nil {
		return nil
	}

	copiedDatabase := *m
	copiedDatabase.ID = uuid.Nil
	copiedDatabase.DatabaseID = nil
	copiedDatabase.ExcludeTables = slices.Clone(m.ExcludeTables)

	if m.Database != nil {
		copiedDatabase.Database = new(*m.Database)
	}

	return &copiedDatabase
}

func (m *MysqlDatabase) EncryptSensitiveFields(
	encryptor encryption.FieldEncryptor,
) error {
	if m.Password != "" {
		encrypted, err := encryptor.Encrypt(m.Password)
		if err != nil {
			return err
		}
		m.Password = encrypted
	}

	return m.SshTunnel.EncryptSensitiveFields(encryptor)
}

func (m *MysqlDatabase) PopulateDbData(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	if m.Database == nil || *m.Database == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, connectionTimeout)
	defer cancel()

	source, err := m.newSource(encryptor)
	if err != nil {
		return err
	}

	inspection, err := source.Inspect(ctx, logger, m.ExcludeTables)
	if err != nil {
		return err
	}

	m.writeInspection(inspection)

	return nil
}

func (m *MysqlDatabase) PopulateVersion(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	if m.Database == nil || *m.Database == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, connectionTimeout)
	defer cancel()

	source, err := m.newSource(encryptor)
	if err != nil {
		return err
	}

	detectedVersion, err := source.DetectVersion(ctx, logger)
	if err != nil {
		return err
	}

	m.Version = detectedVersion

	return nil
}

func (m *MysqlDatabase) ShouldSuggestReadOnlyUser(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) (bool, []string, error) {
	source, err := m.newSource(encryptor)
	if err != nil {
		return false, nil, err
	}

	return source.ShouldSuggestReadOnlyUser(ctx, logger)
}

func (m *MysqlDatabase) CreateReadOnlyUser(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) (string, string, error) {
	source, err := m.newSource(encryptor)
	if err != nil {
		return "", "", err
	}

	return source.CreateReadOnlyUser(ctx, logger)
}
