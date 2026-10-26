package mongodb

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	mongodb_engine "databasus-backend/internal/engines/mongodb"
	"databasus-backend/internal/features/sshtunnel"
	"databasus-backend/internal/lib/namelist"
	"databasus-backend/internal/util/encryption"
)

type MongodbDatabase struct {
	ID         uuid.UUID  `json:"id"         gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	DatabaseID *uuid.UUID `json:"databaseId" gorm:"type:uuid;column:database_id"`

	Version mongodb_engine.Version `json:"version" gorm:"type:text;not null"`

	Host               string `json:"host"               gorm:"type:text;not null"`
	Port               *int   `json:"port"               gorm:"type:int"`
	Username           string `json:"username"           gorm:"type:text;not null"`
	Password           string `json:"password"           gorm:"type:text;not null"`
	Database           string `json:"database"           gorm:"type:text;not null"`
	AuthDatabase       string `json:"authDatabase"       gorm:"type:text;not null;default:'admin'"`
	IsHttps            bool   `json:"isHttps"            gorm:"type:boolean;default:false"`
	IsSrv              bool   `json:"isSrv"              gorm:"column:is_srv;type:boolean;not null;default:false"`
	IsDirectConnection bool   `json:"isDirectConnection" gorm:"column:is_direct_connection;type:boolean;not null;default:false"`
	CpuCount           int    `json:"cpuCount"           gorm:"column:cpu_count;type:int;not null;default:1"`

	// When the tunnel is enabled, Host and Port above address the database as the bastion sees it.
	SshTunnel sshtunnel.Config `json:"sshTunnel" gorm:"embedded;embeddedPrefix:ssh_"`

	ExcludeCollections       []string `json:"excludeCollections" gorm:"-"`
	ExcludeCollectionsString string   `json:"-"                  gorm:"column:exclude_collections;type:text;not null;default:''"`
}

func (m *MongodbDatabase) TableName() string {
	return "mongodb_databases"
}

func (m *MongodbDatabase) BeforeSave(_ *gorm.DB) error {
	m.ExcludeCollectionsString = namelist.FormatUniqueNames(m.ExcludeCollections)

	return nil
}

func (m *MongodbDatabase) AfterFind(_ *gorm.DB) error {
	m.ExcludeCollections = namelist.ParseUniqueNames(m.ExcludeCollectionsString)

	return nil
}

func (m *MongodbDatabase) Validate() error {
	if m.Host == "" {
		return errors.New("host is required")
	}

	if !m.IsSrv {
		if m.Port == nil || *m.Port == 0 {
			return errors.New("port is required for standard connections")
		}
	}

	if m.Username == "" {
		return errors.New("username is required")
	}
	if m.Password == "" {
		return errors.New("password is required")
	}
	if m.Database == "" {
		return errors.New("database is required")
	}
	if m.CpuCount <= 0 {
		return errors.New("cpu count must be greater than 0")
	}

	// SRV resolves its own host list from DNS and never carries a port, so there is nothing for a
	// local forward to stand in front of.
	if m.SshTunnel.IsEnabled && m.IsSrv {
		return errors.New(
			"SSH tunnel cannot be used with an SRV connection: " +
				"SRV resolves its own host list and bypasses the forwarded port",
		)
	}

	return m.SshTunnel.Validate()
}

func (m *MongodbDatabase) TestConnection(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	source, err := m.newSource(encryptor)
	if err != nil {
		return err
	}

	detectedVersion, err := source.TestConnection(ctx, logger, m.ExcludeCollections)
	if err != nil {
		return err
	}

	m.Version = detectedVersion

	return nil
}

func (m *MongodbDatabase) GetRawDbSizeMb(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) (float64, error) {
	if m.Database == "" {
		return 0, nil
	}

	source, err := m.newSource(encryptor)
	if err != nil {
		return 0, err
	}

	return source.GetRawSizeMb(ctx, logger)
}

func (m *MongodbDatabase) HideSensitiveData() {
	if m == nil {
		return
	}
	m.Password = ""
	m.SshTunnel.HideSensitiveData()
}

func (m *MongodbDatabase) Update(incoming *MongodbDatabase) {
	m.Version = incoming.Version
	m.Host = incoming.Host
	m.Port = incoming.Port
	m.Username = incoming.Username
	m.Database = incoming.Database
	m.AuthDatabase = incoming.AuthDatabase
	m.IsHttps = incoming.IsHttps
	m.IsSrv = incoming.IsSrv
	m.IsDirectConnection = incoming.IsDirectConnection
	m.CpuCount = incoming.CpuCount
	m.ExcludeCollections = incoming.ExcludeCollections
	m.SshTunnel.Update(&incoming.SshTunnel)

	if incoming.Password != "" {
		m.Password = incoming.Password
	}
}

func (m *MongodbDatabase) CopyForNewDatabase() *MongodbDatabase {
	if m == nil {
		return nil
	}

	copiedDatabase := *m
	copiedDatabase.ID = uuid.Nil
	copiedDatabase.DatabaseID = nil
	copiedDatabase.ExcludeCollections = slices.Clone(m.ExcludeCollections)

	if m.Port != nil {
		copiedDatabase.Port = new(*m.Port)
	}

	return &copiedDatabase
}

func (m *MongodbDatabase) EncryptSensitiveFields(
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

func (m *MongodbDatabase) PopulateDbData(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	return m.PopulateVersion(ctx, logger, encryptor)
}

func (m *MongodbDatabase) PopulateVersion(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
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

func (m *MongodbDatabase) ShouldSuggestReadOnlyUser(
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

func (m *MongodbDatabase) CreateReadOnlyUser(
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

// BuildConnectionURI builds the URI the Go driver and mongodump connect with, with the database in
// its path.
func (m *MongodbDatabase) BuildConnectionURI(password string) string {
	return m.buildSettings(password).BuildConnectionURI()
}

// BuildRestoreURI builds the URI mongorestore connects with, without the database in its path.
func (m *MongodbDatabase) BuildRestoreURI(password string) string {
	return m.buildSettings(password).BuildRestoreURI()
}
