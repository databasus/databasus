package googledrive

import (
	"context"
	"io"
	"log/slog"

	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/modules/storages/internal/providers"
)

type Deps struct {
	Repositories   Repositories
	FieldEncryptor *crypto.FieldEncryptor
	// Logger records the token refreshes of a connection test, which runs without a file's logger.
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
	return providers.TypeGoogleDrive
}

func (p *Provider) GetRepository() providers.SettingsRepository {
	return p.repository
}

func (p *Provider) Validate(settings providers.Settings) error {
	driveSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	return validateSettings(driveSettings)
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

	if incomingSettings.ClientSecret == "" {
		incomingSettings.ClientSecret = storedSettings.ClientSecret
	}

	if incomingSettings.TokenJSON == "" {
		incomingSettings.TokenJSON = storedSettings.TokenJSON
	}

	return incomingSettings, nil
}

func (p *Provider) SaveFile(ctx context.Context, file providers.FileAccess, content io.Reader) error {
	client, err := p.newDriveClient(file.Settings)
	if err != nil {
		return err
	}

	return client.saveFile(ctx, file.Logger, file.FileName, content)
}

func (p *Provider) OpenFile(ctx context.Context, file providers.FileAccess) (io.ReadCloser, error) {
	client, err := p.newDriveClient(file.Settings)
	if err != nil {
		return nil, err
	}

	return client.openFile(ctx, file.Logger, file.FileName)
}

func (p *Provider) DeleteFile(ctx context.Context, file providers.FileAccess) error {
	client, err := p.newDriveClient(file.Settings)
	if err != nil {
		return err
	}

	return client.deleteFile(ctx, file.Logger, file.FileName)
}

func (p *Provider) TestConnection(ctx context.Context, location providers.Location) error {
	client, err := p.newDriveClient(location.Settings)
	if err != nil {
		return err
	}

	return client.testConnection(ctx, p.logger)
}

func (p *Provider) newDriveClient(settings providers.Settings) (*driveClient, error) {
	driveSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return nil, err
	}

	return &driveClient{provider: p, settings: driveSettings}, nil
}
