package postgresqlphysical

import (
	"context"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/modules/databases/internal/providers"
)

const (
	maxPort                   = 65535
	replicationSlotNamePrefix = "databasus_slot_"
	backupSlotNamePrefix      = "databasus_basebackup_"
)

type Deps struct {
	Repositories   Repositories
	FieldEncryptor *crypto.FieldEncryptor
	// Logger receives the records of the SSH tunnels the provider opens.
	Logger *slog.Logger
}

type Provider struct {
	repository     *settingsRepository
	fieldEncryptor *crypto.FieldEncryptor
	logger         *slog.Logger
}

func New(deps Deps) *Provider {
	return &Provider{
		repository:     deps.Repositories.settings,
		fieldEncryptor: deps.FieldEncryptor,
		logger:         deps.Logger,
	}
}

func (p *Provider) GetType() providers.Type {
	return providers.TypePostgresqlPhysical
}

func (p *Provider) GetRepository() providers.SettingsRepository {
	return p.repository
}

// GetBackupTypes leaves out the IDs that match no physical PostgreSQL database.
func (p *Provider) GetBackupTypes(
	ctx context.Context,
	databaseIDs []uuid.UUID,
) (map[uuid.UUID]providers.PhysicalBackupType, error) {
	if len(databaseIDs) == 0 {
		return map[uuid.UUID]providers.PhysicalBackupType{}, nil
	}

	return p.repository.listBackupTypes(ctx, databaseIDs)
}

func (p *Provider) Validate(settings providers.Settings) error {
	physicalSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	switch {
	case physicalSettings.Host == "":
		return providers.NewInvalidSettingError("host", "is required")
	case physicalSettings.Port <= 0 || physicalSettings.Port > maxPort:
		return providers.NewInvalidSettingError("port", "must be between 1 and 65535")
	case physicalSettings.Username == "":
		return providers.NewInvalidSettingError("username", "is required")
	case physicalSettings.Password == "":
		return providers.NewInvalidSettingError("password", "is required")
	case !physicalSettings.BackupType.IsValid():
		return providers.NewInvalidSettingError(
			"backupType", "must be FULL, FULL_INCREMENTAL or FULL_INCREMENTAL_WAL_STREAM",
		)
	}

	err = providers.ValidatePostgresqlTLS(physicalSettings.TLS)
	if err != nil {
		return err
	}

	err = providers.ValidateSSHTunnel(physicalSettings.SSHTunnel)
	if err != nil {
		return err
	}

	err = pgshared.ValidateNotEmbeddedTarget(pgshared.EmbeddedTargetSpec{
		Host:               physicalSettings.Host,
		Port:               physicalSettings.Port,
		DatabaseName:       "",
		IsPhysical:         true,
		IsSSHTunnelEnabled: physicalSettings.SSHTunnel.IsEnabled,
		SSHBastionHost:     physicalSettings.SSHTunnel.Host,
	})
	if err != nil {
		return providers.NewInvalidSettingError("host", err.Error())
	}

	return nil
}

// MergeStoredSettings keeps what the provider and the server own: the replication slots, the version
// and the identity of the cluster.
func (p *Provider) MergeStoredSettings(merge providers.SettingsMerge) (providers.Settings, error) {
	incomingSettings, err := providers.CastSettings[Settings](merge.Incoming)
	if err != nil {
		return nil, err
	}

	storedSettings, err := providers.CastSettings[Settings](merge.Stored)
	if err != nil {
		return nil, err
	}

	isAddressChanged := incomingSettings.Host != storedSettings.Host || incomingSettings.Port != storedSettings.Port ||
		providers.IsSSHTunnelAddressChanged(incomingSettings.SSHTunnel, storedSettings.SSHTunnel)

	if incomingSettings.Password == "" && !isAddressChanged {
		incomingSettings.Password = storedSettings.Password
	}

	incomingSettings.Version = storedSettings.Version
	incomingSettings.ReplicationSlotName = storedSettings.ReplicationSlotName
	incomingSettings.BackupSlotName = storedSettings.BackupSlotName
	incomingSettings.SystemIdentifier = storedSettings.SystemIdentifier
	incomingSettings.WalSegmentSizeBytes = storedSettings.WalSegmentSizeBytes
	incomingSettings.TLS = providers.MergeStoredPostgresqlTLS(
		incomingSettings.TLS,
		storedSettings.TLS,
		isAddressChanged,
	)
	incomingSettings.SSHTunnel = providers.MergeStoredSSHTunnel(incomingSettings.SSHTunnel, storedSettings.SSHTunnel)

	return incomingSettings, nil
}

func (p *Provider) PrepareForCreate(settings providers.Settings) providers.Settings {
	return withOwnIdentity(settings)
}

// CopyForNewDatabase gives the copy replication slots of its own, because two databases sharing one
// slot would consume each other's WAL, and lets the next save detect the cluster the copy points at.
func (p *Provider) CopyForNewDatabase(settings providers.Settings) providers.Settings {
	return withOwnIdentity(settings)
}

func (p *Provider) Connect(ctx context.Context, settings providers.Settings) (providers.ConnectedDatabase, error) {
	physicalSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return nil, err
	}

	engineSettings, err := p.decryptEngineSettings(physicalSettings)
	if err != nil {
		return nil, err
	}

	return openConnectedDatabase(ctx, p.logger, connection{settings: physicalSettings, engineSettings: engineSettings})
}

func (p *Provider) GetTelemetryFacts(settings providers.Settings) (providers.TelemetryFacts, error) {
	physicalSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return providers.TelemetryFacts{}, err
	}

	return providers.TelemetryFacts{
		Version:            string(physicalSettings.Version),
		PhysicalBackupType: physicalSettings.BackupType,
		IsSSHTunnelEnabled: physicalSettings.SSHTunnel.IsEnabled,
	}, nil
}

func (p *Provider) GetConnectionSettings(
	databaseID uuid.UUID,
	settings providers.Settings,
) (providers.ConnectionSettings, error) {
	physicalSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return nil, err
	}

	engineSettings, err := p.decryptEngineSettings(physicalSettings)
	if err != nil {
		return nil, err
	}

	return providers.PostgresqlPhysicalConnectionSettings{
		DatabaseID:          databaseID,
		Engine:              engineSettings,
		Version:             physicalSettings.Version,
		SystemIdentifier:    physicalSettings.SystemIdentifier,
		WalSegmentSizeBytes: physicalSettings.WalSegmentSizeBytes,
		BackupType:          physicalSettings.BackupType,
		WalStreamSlotName:   physicalSettings.ReplicationSlotName,
		BackupSlotName:      physicalSettings.BackupSlotName,
	}, nil
}

func (p *Provider) decryptEngineSettings(settings Settings) (pgphysical.Settings, error) {
	password, err := providers.DecryptSecret(p.fieldEncryptor, settings.Password, "the PostgreSQL password")
	if err != nil {
		return pgphysical.Settings{}, err
	}

	tls, err := providers.DecryptPostgresqlTLS(p.fieldEncryptor, settings.TLS)
	if err != nil {
		return pgphysical.Settings{}, err
	}

	tunnelSettings, err := providers.DecryptSSHTunnel(p.fieldEncryptor, settings.SSHTunnel)
	if err != nil {
		return pgphysical.Settings{}, err
	}

	return pgphysical.Settings{
		Host:      settings.Host,
		HostAddr:  "",
		Port:      settings.Port,
		Username:  settings.Username,
		Password:  password,
		TLS:       tls,
		SSHTunnel: tunnelSettings,
	}, nil
}

func withOwnIdentity(settings providers.Settings) providers.Settings {
	physicalSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return settings
	}

	physicalSettings.ReplicationSlotName = replicationSlotNamePrefix + newSlotSuffix()
	physicalSettings.BackupSlotName = backupSlotNamePrefix + newSlotSuffix()
	physicalSettings.SystemIdentifier = nil
	physicalSettings.WalSegmentSizeBytes = nil

	return physicalSettings
}

func newSlotSuffix() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")
}
