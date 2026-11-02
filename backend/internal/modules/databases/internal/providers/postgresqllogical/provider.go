package postgresqllogical

import (
	"context"
	"log/slog"
	"slices"

	"github.com/google/uuid"

	pglogical "databasus-backend/internal/engines/postgresql/logical"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
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
	return providers.TypePostgresqlLogical
}

func (p *Provider) GetRepository() providers.SettingsRepository {
	return p.repository
}

func (p *Provider) Validate(settings providers.Settings) error {
	logicalSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	switch {
	case logicalSettings.Host == "":
		return providers.NewInvalidSettingError("host", "is required")
	case logicalSettings.Port <= 0 || logicalSettings.Port > maxPort:
		return providers.NewInvalidSettingError("port", "must be between 1 and 65535")
	case logicalSettings.Username == "":
		return providers.NewInvalidSettingError("username", "is required")
	case logicalSettings.Password == "":
		return providers.NewInvalidSettingError("password", "is required")
	case logicalSettings.CPUCount <= 0:
		return providers.NewInvalidSettingError("cpuCount", "must be greater than 0")
	}

	err = providers.ValidatePostgresqlTLS(logicalSettings.TLS)
	if err != nil {
		return err
	}

	err = providers.ValidateSSHTunnel(logicalSettings.SSHTunnel)
	if err != nil {
		return err
	}

	err = pgshared.ValidateNotEmbeddedTarget(pgshared.EmbeddedTargetSpec{
		Host:               logicalSettings.Host,
		Port:               logicalSettings.Port,
		DatabaseName:       logicalSettings.Database,
		IsPhysical:         false,
		IsSSHTunnelEnabled: logicalSettings.SSHTunnel.IsEnabled,
		SSHBastionHost:     logicalSettings.SSHTunnel.Host,
	})
	if err != nil {
		return providers.NewInvalidSettingError("host", err.Error())
	}

	return nil
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
	incomingSettings.TLS = providers.MergeStoredPostgresqlTLS(
		incomingSettings.TLS,
		storedSettings.TLS,
		isAddressChanged,
	)
	incomingSettings.SSHTunnel = providers.MergeStoredSSHTunnel(incomingSettings.SSHTunnel, storedSettings.SSHTunnel)

	return incomingSettings, nil
}

func (p *Provider) PrepareForCreate(settings providers.Settings) providers.Settings {
	return settings
}

func (p *Provider) CopyForNewDatabase(settings providers.Settings) providers.Settings {
	logicalSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return settings
	}

	logicalSettings.IncludeSchemas = slices.Clone(logicalSettings.IncludeSchemas)
	logicalSettings.ExcludeTables = slices.Clone(logicalSettings.ExcludeTables)
	logicalSettings.ExcludeTableData = slices.Clone(logicalSettings.ExcludeTableData)

	return logicalSettings
}

func (p *Provider) Connect(ctx context.Context, settings providers.Settings) (providers.ConnectedDatabase, error) {
	logicalSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return nil, err
	}

	engineSettings, err := p.decryptEngineSettings(logicalSettings)
	if err != nil {
		return nil, err
	}

	return openConnectedDatabase(ctx, p.logger, connection{settings: logicalSettings, engineSettings: engineSettings})
}

func (p *Provider) GetTelemetryFacts(settings providers.Settings) (providers.TelemetryFacts, error) {
	logicalSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return providers.TelemetryFacts{}, err
	}

	return providers.TelemetryFacts{
		Version:            string(logicalSettings.Version),
		PhysicalBackupType: "",
		IsSSHTunnelEnabled: logicalSettings.SSHTunnel.IsEnabled,
	}, nil
}

func (p *Provider) GetConnectionSettings(
	databaseID uuid.UUID,
	settings providers.Settings,
) (providers.ConnectionSettings, error) {
	logicalSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return nil, err
	}

	engineSettings, err := p.decryptEngineSettings(logicalSettings)
	if err != nil {
		return nil, err
	}

	return providers.PostgresqlLogicalConnectionSettings{
		DatabaseID:         databaseID,
		Engine:             engineSettings,
		Version:            logicalSettings.Version,
		IncludeSchemas:     slices.Clone(logicalSettings.IncludeSchemas),
		ExcludeTables:      slices.Clone(logicalSettings.ExcludeTables),
		ExcludeTableData:   slices.Clone(logicalSettings.ExcludeTableData),
		CPUCount:           logicalSettings.CPUCount,
		IsSkipUserMappings: logicalSettings.IsSkipUserMappings,
	}, nil
}

func (p *Provider) GetVersions(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	return p.repository.listVersions(ctx, ids)
}

func (p *Provider) decryptEngineSettings(settings Settings) (pglogical.Settings, error) {
	password, err := providers.DecryptSecret(p.fieldEncryptor, settings.Password, "the PostgreSQL password")
	if err != nil {
		return pglogical.Settings{}, err
	}

	tls, err := providers.DecryptPostgresqlTLS(p.fieldEncryptor, settings.TLS)
	if err != nil {
		return pglogical.Settings{}, err
	}

	tunnelSettings, err := providers.DecryptSSHTunnel(p.fieldEncryptor, settings.SSHTunnel)
	if err != nil {
		return pglogical.Settings{}, err
	}

	return pglogical.Settings{
		Host:      settings.Host,
		HostAddr:  "",
		Port:      settings.Port,
		Username:  settings.Username,
		Password:  password,
		Database:  settings.Database,
		TLS:       tls,
		SSHTunnel: tunnelSettings,
	}, nil
}
