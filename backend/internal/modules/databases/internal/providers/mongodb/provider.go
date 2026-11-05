package mongodb

import (
	"context"
	"log/slog"
	"slices"

	"github.com/google/uuid"

	mongodbengine "databasus-backend/internal/engines/mongodb"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/modules/databases/internal/providers"
)

const maxPort = 65535

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
	return providers.TypeMongodb
}

func (p *Provider) GetRepository() providers.SettingsRepository {
	return p.repository
}

func (p *Provider) Validate(settings providers.Settings) error {
	mongodbSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	switch {
	case mongodbSettings.Host == "":
		return providers.NewInvalidSettingError("host", "is required")
	case !mongodbSettings.IsSrv && (mongodbSettings.Port <= 0 || mongodbSettings.Port > maxPort):
		return providers.NewInvalidSettingError("port", "must be between 1 and 65535 without SRV")
	case mongodbSettings.Username == "":
		return providers.NewInvalidSettingError("username", "is required")
	case mongodbSettings.Password == "":
		return providers.NewInvalidSettingError("password", "is required")
	case mongodbSettings.Database == "":
		return providers.NewInvalidSettingError("database", "is required")
	case mongodbSettings.CPUCount <= 0:
		return providers.NewInvalidSettingError("cpuCount", "must be greater than 0")
	// SRV resolves its own host list from DNS and never carries a port, so there is nothing for a
	// local forward to stand in front of.
	case mongodbSettings.SSHTunnel.IsEnabled && mongodbSettings.IsSrv:
		return providers.NewInvalidSettingError("sshTunnel.isEnabled", "cannot be used with an SRV connection")
	}

	return providers.ValidateSSHTunnel(mongodbSettings.SSHTunnel)
}

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
		incomingSettings.IsSrv != storedSettings.IsSrv ||
		providers.IsSSHTunnelAddressChanged(incomingSettings.SSHTunnel, storedSettings.SSHTunnel)

	if incomingSettings.Password == "" && !isAddressChanged {
		incomingSettings.Password = storedSettings.Password
	}

	incomingSettings.Version = storedSettings.Version
	incomingSettings.SSHTunnel = providers.MergeStoredSSHTunnel(incomingSettings.SSHTunnel, storedSettings.SSHTunnel)

	return incomingSettings, nil
}

func (p *Provider) PrepareForCreate(settings providers.Settings) providers.Settings {
	return settings
}

func (p *Provider) CopyForNewDatabase(settings providers.Settings) providers.Settings {
	mongodbSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return settings
	}

	mongodbSettings.ExcludeCollections = slices.Clone(mongodbSettings.ExcludeCollections)

	return mongodbSettings
}

func (p *Provider) Connect(ctx context.Context, settings providers.Settings) (providers.ConnectedDatabase, error) {
	mongodbSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return nil, err
	}

	engineSettings, err := p.decryptEngineSettings(mongodbSettings)
	if err != nil {
		return nil, err
	}

	tunnel, err := mongodbengine.NewSource(engineSettings).OpenTunnel(ctx, p.logger)
	if err != nil {
		return nil, err
	}

	return &connectedDatabase{settings: mongodbSettings, tunnel: tunnel}, nil
}

func (p *Provider) GetTelemetryFacts(settings providers.Settings) (providers.TelemetryFacts, error) {
	mongodbSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return providers.TelemetryFacts{}, err
	}

	return providers.TelemetryFacts{
		Version:            string(mongodbSettings.Version),
		PhysicalBackupType: "",
		IsSSHTunnelEnabled: mongodbSettings.SSHTunnel.IsEnabled,
	}, nil
}

func (p *Provider) GetConnectionSettings(
	databaseID uuid.UUID,
	settings providers.Settings,
) (providers.ConnectionSettings, error) {
	mongodbSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return nil, err
	}

	engineSettings, err := p.decryptEngineSettings(mongodbSettings)
	if err != nil {
		return nil, err
	}

	return providers.MongodbConnectionSettings{
		DatabaseID:         databaseID,
		Engine:             engineSettings,
		Version:            mongodbSettings.Version,
		ExcludeCollections: slices.Clone(mongodbSettings.ExcludeCollections),
		CPUCount:           mongodbSettings.CPUCount,
	}, nil
}

func (p *Provider) decryptEngineSettings(settings Settings) (mongodbengine.Settings, error) {
	password, err := providers.DecryptSecret(p.fieldEncryptor, settings.Password, "the MongoDB password")
	if err != nil {
		return mongodbengine.Settings{}, err
	}

	tunnelSettings, err := providers.DecryptSSHTunnel(p.fieldEncryptor, settings.SSHTunnel)
	if err != nil {
		return mongodbengine.Settings{}, err
	}

	return mongodbengine.Settings{
		Host:               settings.Host,
		Port:               settings.Port,
		Username:           settings.Username,
		Password:           password,
		Database:           settings.Database,
		AuthDatabase:       settings.AuthDatabase,
		IsTLS:              settings.IsHTTPS,
		IsSrv:              settings.IsSrv,
		IsDirectConnection: settings.IsDirectConnection,
		SSHTunnel:          tunnelSettings,
	}, nil
}
