package mysql

import (
	"context"
	"log/slog"
	"slices"

	"github.com/google/uuid"

	mysqlengine "databasus-backend/internal/engines/mysql"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/modules/databases/internal/providers"
)

const maxPort = 65535

type Deps struct {
	Repositories   Repositories
	FieldEncryptor *crypto.FieldEncryptor
	// Logger receives the records of the SSH tunnels the provider opens.
	Logger *slog.Logger
	// ClientToolsDir holds the client tool bundles; the version detection refuses a server version
	// without one.
	ClientToolsDir string
}

type Provider struct {
	repository     *settingsRepository
	fieldEncryptor *crypto.FieldEncryptor
	logger         *slog.Logger
	clientTools    mysqlengine.ClientTools
}

func New(deps Deps) *Provider {
	return &Provider{
		repository:     deps.Repositories.settings,
		fieldEncryptor: deps.FieldEncryptor,
		logger:         deps.Logger,
		clientTools:    mysqlengine.ClientTools{Root: deps.ClientToolsDir},
	}
}

func (p *Provider) GetType() providers.Type {
	return providers.TypeMysql
}

func (p *Provider) GetRepository() providers.SettingsRepository {
	return p.repository
}

func (p *Provider) Validate(settings providers.Settings) error {
	mysqlSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	switch {
	case mysqlSettings.Host == "":
		return providers.NewInvalidSettingError("host", "is required")
	case mysqlSettings.Port <= 0 || mysqlSettings.Port > maxPort:
		return providers.NewInvalidSettingError("port", "must be between 1 and 65535")
	case mysqlSettings.Username == "":
		return providers.NewInvalidSettingError("username", "is required")
	case mysqlSettings.Password == "":
		return providers.NewInvalidSettingError("password", "is required")
	}

	return providers.ValidateSSHTunnel(mysqlSettings.SSHTunnel)
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
		providers.IsSSHTunnelAddressChanged(incomingSettings.SSHTunnel, storedSettings.SSHTunnel)

	if incomingSettings.Password == "" && !isAddressChanged {
		incomingSettings.Password = storedSettings.Password
	}

	incomingSettings.Version = storedSettings.Version
	incomingSettings.Privileges = storedSettings.Privileges
	incomingSettings.SSHTunnel = providers.MergeStoredSSHTunnel(incomingSettings.SSHTunnel, storedSettings.SSHTunnel)

	return incomingSettings, nil
}

func (p *Provider) PrepareForCreate(settings providers.Settings) providers.Settings {
	return settings
}

func (p *Provider) CopyForNewDatabase(settings providers.Settings) providers.Settings {
	mysqlSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return settings
	}

	mysqlSettings.ExcludeTables = slices.Clone(mysqlSettings.ExcludeTables)
	mysqlSettings.Privileges = slices.Clone(mysqlSettings.Privileges)

	return mysqlSettings
}

func (p *Provider) Connect(ctx context.Context, settings providers.Settings) (providers.ConnectedDatabase, error) {
	mysqlSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return nil, err
	}

	engineSettings, err := p.decryptEngineSettings(mysqlSettings)
	if err != nil {
		return nil, err
	}

	tunnel, err := mysqlengine.NewSource(engineSettings, p.clientTools).OpenTunnel(ctx, p.logger)
	if err != nil {
		return nil, err
	}

	return &connectedDatabase{settings: mysqlSettings, tunnel: tunnel}, nil
}

func (p *Provider) GetTelemetryFacts(settings providers.Settings) (providers.TelemetryFacts, error) {
	mysqlSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return providers.TelemetryFacts{}, err
	}

	return providers.TelemetryFacts{
		Version:            string(mysqlSettings.Version),
		PhysicalBackupType: "",
		IsSSHTunnelEnabled: mysqlSettings.SSHTunnel.IsEnabled,
	}, nil
}

func (p *Provider) GetConnectionSettings(
	databaseID uuid.UUID,
	settings providers.Settings,
) (providers.ConnectionSettings, error) {
	mysqlSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return nil, err
	}

	engineSettings, err := p.decryptEngineSettings(mysqlSettings)
	if err != nil {
		return nil, err
	}

	return providers.MysqlConnectionSettings{
		DatabaseID:    databaseID,
		Engine:        engineSettings,
		Version:       mysqlSettings.Version,
		ExcludeTables: slices.Clone(mysqlSettings.ExcludeTables),
		Privileges:    slices.Clone(mysqlSettings.Privileges),
	}, nil
}

func (p *Provider) decryptEngineSettings(settings Settings) (mysqlengine.Settings, error) {
	password, err := providers.DecryptSecret(p.fieldEncryptor, settings.Password, "the MySQL password")
	if err != nil {
		return mysqlengine.Settings{}, err
	}

	tunnelSettings, err := providers.DecryptSSHTunnel(p.fieldEncryptor, settings.SSHTunnel)
	if err != nil {
		return mysqlengine.Settings{}, err
	}

	return mysqlengine.Settings{
		Host:      settings.Host,
		Port:      settings.Port,
		Username:  settings.Username,
		Password:  password,
		Database:  settings.Database,
		IsHTTPS:   settings.IsHTTPS,
		SSHTunnel: tunnelSettings,
	}, nil
}
