package postgresql_physical

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"gorm.io/gorm"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/features/sshtunnel"
	libsshtunnel "databasus-backend/internal/lib/sshtunnel"
	"databasus-backend/internal/util/encryption"
)

type PostgresqlPhysicalDatabase struct {
	ID uuid.UUID `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`

	DatabaseID *uuid.UUID `json:"databaseId" gorm:"type:uuid;column:database_id"`

	Version    pgshared.Version `json:"version"    gorm:"type:text;not null"`
	BackupType BackupType       `json:"backupType" gorm:"column:backup_type;type:text;not null;default:'FULL'"`

	Host     string `json:"host"     gorm:"type:text;not null"`
	Port     int    `json:"port"     gorm:"type:int;not null"`
	Username string `json:"username" gorm:"type:text;not null"`
	Password string `json:"password" gorm:"type:text;not null"`

	// SSL / TLS connection settings
	SslMode       pgshared.SslMode `json:"sslMode"       gorm:"column:ssl_mode;type:text;not null;default:'disable'"`
	SslClientCert string           `json:"sslClientCert" gorm:"column:ssl_client_cert;type:text;not null;default:''"`
	SslClientKey  string           `json:"sslClientKey"  gorm:"column:ssl_client_key;type:text;not null;default:''"`
	SslRootCert   string           `json:"sslRootCert"   gorm:"column:ssl_root_cert;type:text;not null;default:''"`

	// When the tunnel is enabled, Host and Port above address the cluster as the bastion sees it.
	SshTunnel sshtunnel.Config `json:"sshTunnel" gorm:"embedded;embeddedPrefix:ssh_"`

	// Set only on the copy handed out by OpenTunnel, so the credential spec can point libpq at the
	// forwarded port while Host keeps the name that TLS and .pgpass are matched against.
	LocalTunnelEndpoint *libsshtunnel.Endpoint `json:"-" gorm:"-"`

	ReplicationSlotName string  `json:"-"                gorm:"column:replication_slot_name;type:text;not null"`
	SystemIdentifier    *string `json:"systemIdentifier" gorm:"column:system_identifier;type:text"`

	// WalSegmentSizeBytes captures the source cluster's wal_segment_size at first connect.
	WalSegmentSizeBytes *int64 `json:"walSegmentSizeBytes" gorm:"column:wal_segment_size_bytes;type:bigint"`
}

func (p *PostgresqlPhysicalDatabase) TableName() string {
	return "postgresql_physical_databases"
}

func (p *PostgresqlPhysicalDatabase) Validate() error {
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

	tls := pgshared.TLS{
		Mode:       p.SslMode,
		ClientCert: p.SslClientCert,
		ClientKey:  p.SslClientKey,
		RootCert:   p.SslRootCert,
	}
	if err := tls.Validate(); err != nil {
		return err
	}

	if err := p.SshTunnel.Validate(); err != nil {
		return err
	}

	if err := p.ValidateNotEmbeddedTarget(); err != nil {
		return err
	}

	switch p.BackupType {
	case BackupTypeFullOnly, BackupTypeFullAndIncremental, BackupTypeFullIncrementalAndWalStream:
	case "":
		p.BackupType = BackupTypeFullOnly
	default:
		return fmt.Errorf("invalid backup type: %q", p.BackupType)
	}

	return nil
}

func (p *PostgresqlPhysicalDatabase) ValidateNotEmbeddedTarget() error {
	return pgshared.ValidateNotEmbeddedTarget(pgshared.EmbeddedTargetSpec{
		Host:               p.Host,
		Port:               p.Port,
		IsPhysical:         true,
		IsSSHTunnelEnabled: p.SshTunnel.IsEnabled,
		SSHBastionHost:     p.SshTunnel.Host,
	})
}

func (p *PostgresqlPhysicalDatabase) ValidateUpdate(old *PostgresqlPhysicalDatabase) error {
	if old == nil {
		return nil
	}

	if old.SystemIdentifier != nil && p.SystemIdentifier != nil &&
		*old.SystemIdentifier != *p.SystemIdentifier {
		return errors.New(
			"system_identifier is immutable; cluster swap refused",
		)
	}

	if old.WalSegmentSizeBytes != nil && p.WalSegmentSizeBytes != nil &&
		*old.WalSegmentSizeBytes != *p.WalSegmentSizeBytes {
		return errors.New(
			"wal_segment_size_bytes is immutable; cluster swap refused",
		)
	}

	return nil
}

func (p *PostgresqlPhysicalDatabase) BeforeCreate(_ *gorm.DB) error {
	if p.ReplicationSlotName == "" {
		p.ReplicationSlotName = "databasus_slot_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	}

	return nil
}

func (p *PostgresqlPhysicalDatabase) Update(incoming *PostgresqlPhysicalDatabase) {
	p.Host = incoming.Host
	p.Port = incoming.Port
	p.Username = incoming.Username
	p.SslMode = incoming.SslMode
	p.SslClientCert = incoming.SslClientCert
	p.SslRootCert = incoming.SslRootCert
	p.BackupType = incoming.BackupType
	p.SshTunnel.Update(&incoming.SshTunnel)

	if incoming.Password != "" {
		p.Password = incoming.Password
	}

	if incoming.SslClientKey != "" {
		p.SslClientKey = incoming.SslClientKey
	}

	// SystemIdentifier and ReplicationSlotName are server-managed; never overwritten from UI input.
}

func (p *PostgresqlPhysicalDatabase) HideSensitiveData() {
	if p == nil {
		return
	}

	p.Password = ""
	p.SslClientKey = ""
	p.SshTunnel.HideSensitiveData()
}

// Copying the struct rather than listing fields is the point: a field added later travels with it
// instead of being silently dropped. The three server-managed fields must not travel: BeforeCreate
// only mints a replication slot name when the field is empty, so an inherited one would leave two
// databases sharing a single slot on the source cluster, and the identifier and segment size are
// read back from whichever cluster the copy is pointed at. LocalTunnelEndpoint belongs to the
// operation that opened the tunnel, not to the configuration.
func (p *PostgresqlPhysicalDatabase) CopyForNewDatabase() *PostgresqlPhysicalDatabase {
	if p == nil {
		return nil
	}

	copiedDatabase := *p
	copiedDatabase.ID = uuid.Nil
	copiedDatabase.DatabaseID = nil
	copiedDatabase.ReplicationSlotName = ""
	copiedDatabase.SystemIdentifier = nil
	copiedDatabase.WalSegmentSizeBytes = nil
	copiedDatabase.LocalTunnelEndpoint = nil

	return &copiedDatabase
}

func (p *PostgresqlPhysicalDatabase) EncryptSensitiveFields(
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

func (p *PostgresqlPhysicalDatabase) PopulateDbData(
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

	clusterIdentity, err := source.InspectCluster(ctx, logger)
	if err != nil {
		return err
	}

	p.Version = clusterIdentity.Version

	if p.SystemIdentifier == nil {
		p.SystemIdentifier = &clusterIdentity.SystemIdentifier
	}

	if p.WalSegmentSizeBytes == nil {
		p.WalSegmentSizeBytes = &clusterIdentity.WalSegmentSizeBytes
	}

	return nil
}

// ParentDatabaseID returns the parent databases.id this physical record hangs
// off — the target of every physical catalog FK (fk_physical_wal_*_database_id,
// fk_physical_full_backups_database_id, …), never the
// postgresql_physical_databases PK. The fallback to ID covers an unassociated
// in-memory row whose DatabaseID was not loaded, mirroring
// receivewalApplicationName's fallback.
func (p *PostgresqlPhysicalDatabase) ParentDatabaseID() uuid.UUID {
	if p.DatabaseID != nil {
		return *p.DatabaseID
	}

	return p.ID
}

func (p *PostgresqlPhysicalDatabase) SystemIdentifierUint64() uint64 {
	if p.SystemIdentifier == nil {
		return 0
	}

	if value, err := strconv.ParseUint(*p.SystemIdentifier, 10, 64); err == nil {
		return value
	}

	// pg_control_system() returns system_identifier as a signed bigint; the cast restores its unsigned value bit for bit.
	// nosemgrep: trailofbits.go.string-to-int-signedness-cast.string-to-int-signedness-cast
	if value, err := strconv.ParseInt(*p.SystemIdentifier, 10, 64); err == nil {
		return uint64(value)
	}

	return 0
}

// The summarize_wal=on gate and the custom-tablespace refusal are required by the spec
// physical-backups/backup-method.
func (p *PostgresqlPhysicalDatabase) TestReplicationConnection(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	source, err := p.newSource(encryptor)
	if err != nil {
		return err
	}

	expectedSystemIdentifier := ""
	if p.SystemIdentifier != nil {
		expectedSystemIdentifier = *p.SystemIdentifier
	}

	return source.TestReplicationConnection(ctx, logger, pgphysical.ReplicationCheck{
		IsWalSummaryRequired:     p.BackupType.IsRequireWalSummary(),
		IsWalRotationRequired:    p.BackupType.IsWalStreaming(),
		ExpectedSystemIdentifier: expectedSystemIdentifier,
	})
}

func (p *PostgresqlPhysicalDatabase) ProbeLiveness(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	source, err := p.newSource(encryptor)
	if err != nil {
		return err
	}

	return source.ProbeLiveness(ctx, logger)
}

func (p *PostgresqlPhysicalDatabase) VerifyWalSlot(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	source, err := p.newSource(encryptor)
	if err != nil {
		return err
	}

	return source.VerifySlot(ctx, logger)
}

func (p *PostgresqlPhysicalDatabase) DropWalSlot(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	source, err := p.newSource(encryptor)
	if err != nil {
		return err
	}

	return source.DropSlot(ctx, logger)
}

func (p *PostgresqlPhysicalDatabase) DropWalSlotForRemoval(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	source, err := p.newSource(encryptor)
	if err != nil {
		return err
	}

	return source.DropSlotForRemoval(ctx, logger)
}

func (p *PostgresqlPhysicalDatabase) GetClusterSizeMb(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) (float64, error) {
	source, err := p.newSource(encryptor)
	if err != nil {
		return 0, err
	}

	return source.GetClusterSizeMb(ctx, logger)
}

func (p *PostgresqlPhysicalDatabase) ShouldSuggestReplicationOnlyUser(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) (bool, []string, error) {
	source, err := p.newSource(encryptor)
	if err != nil {
		return false, nil, err
	}

	return source.ShouldSuggestReplicationOnlyUser(ctx, logger)
}

func (p *PostgresqlPhysicalDatabase) CreateReplicationOnlyUser(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) (*pgphysical.ReplicationOnlyUser, error) {
	source, err := p.newSource(encryptor)
	if err != nil {
		return nil, err
	}

	replicationUser, err := source.CreateReplicationOnlyUser(ctx, logger)
	if err != nil {
		return nil, err
	}

	return &replicationUser, nil
}

// OpenInspectionConn opens an ordinary connection to the cluster for the SQL identity checks and
// the timeline history files, which a replication connection cannot read.
func (p *PostgresqlPhysicalDatabase) OpenInspectionConn(
	ctx context.Context,
	encryptor encryption.FieldEncryptor,
) (*pgx.Conn, error) {
	source, err := p.newSource(encryptor)
	if err != nil {
		return nil, err
	}

	return source.OpenInspectionConn(ctx)
}
