package mariadb

import (
	"context"
	"log/slog"
	"slices"

	"github.com/google/uuid"

	mariadbengine "databasus-backend/internal/engines/mariadb"
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
	clientTools    mariadbengine.ClientTools
}

func New(deps Deps) *Provider {
	return &Provider{
		repository:     deps.Repositories.settings,
		fieldEncryptor: deps.FieldEncryptor,
		logger:         deps.Logger,
		clientTools:    mariadbengine.ClientTools{Root: deps.ClientToolsDir},
	}
}

func (p *Provider) GetType() providers.Type {
	return providers.TypeMariadb
}

func (p *Provider) GetRepository() providers.SettingsRepository {
	return p.repository
}

func (p *Provider) Validate(settings providers.Settings) error {
	mariadbSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	switch {
	case mariadbSettings.Host == "":
		return providers.NewInvalidSettingError("host", "is required")
	case mariadbSettings.Port <= 0 || mariadbSettings.Port > maxPort:
		return providers.NewInvalidSettingError("port", "must be between 1 and 65535")
	case mariadbSettings.Username == "":
		return providers.NewInvalidSettingError("username", "is required")
	case mariadbSettings.Password == "":
		return providers.NewInvalidSettingError("password", "is required")
	}

	return providers.ValidateSSHTunnel(mariadbSettings.SSHTunnel)
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
	mariadbSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return settings
	}

	mariadbSettings.ExcludeTables = slices.Clone(mariadbSettings.ExcludeTables)
	mariadbSettings.Privileges = slices.Clone(mariadbSettings.Privileges)

	return mariadbSettings
}

func (p *Provider) Connect(ctx context.Context, settings providers.Settings) (providers.ConnectedDatabase, error) {
	mariadbSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return nil, err
	}

	engineSettings, err := p.decryptEngineSettings(mariadbSettings)
	if err != nil {
		return nil, err
	}

	tunnel, err := mariadbengine.NewSource(engineSettings, p.clientTools).OpenTunnel(ctx, p.logger)
	if err != nil {
		return nil, err
	}

	return &connectedDatabase{settings: mariadbSettings, tunnel: tunnel}, nil
}

func (p *Provider) GetTelemetryFacts(settings providers.Settings) (providers.TelemetryFacts, error) {
	mariadbSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return providers.TelemetryFacts{}, err
	}

	return providers.TelemetryFacts{
		Version:            string(mariadbSettings.Version),
		PhysicalBackupType: "",
		IsSSHTunnelEnabled: mariadbSettings.SSHTunnel.IsEnabled,
	}, nil
}

func (p *Provider) GetConnectionSettings(
	databaseID uuid.UUID,
	settings providers.Settings,
) (providers.ConnectionSettings, error) {
	mariadbSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return nil, err
	}

	engineSettings, err := p.decryptEngineSettings(mariadbSettings)
	if err != nil {
		return nil, err
	}

	return providers.MariadbConnectionSettings{
		DatabaseID:          databaseID,
		Engine:              engineSettings,
		Version:             mariadbSettings.Version,
		ExcludeTables:       slices.Clone(mariadbSettings.ExcludeTables),
		Privileges:          slices.Clone(mariadbSettings.Privileges),
		IsExcludeEvents:     mariadbSettings.IsExcludeEvents,
		IsSkipGaleraDisable: mariadbSettings.IsSkipGaleraDisable,
	}, nil
}

func (p *Provider) decryptEngineSettings(settings Settings) (mariadbengine.Settings, error) {
	password, err := providers.DecryptSecret(p.fieldEncryptor, settings.Password, "the MariaDB password")
	if err != nil {
		return mariadbengine.Settings{}, err
	}

	tunnelSettings, err := providers.DecryptSSHTunnel(p.fieldEncryptor, settings.SSHTunnel)
	if err != nil {
		return mariadbengine.Settings{}, err
	}

	return mariadbengine.Settings{
		Host:      settings.Host,
		Port:      settings.Port,
		Username:  settings.Username,
		Password:  password,
		Database:  settings.Database,
		IsHTTPS:   settings.IsHTTPS,
		SSHTunnel: tunnelSettings,
	}, nil
}
