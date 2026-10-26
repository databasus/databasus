package rclone

import (
	"context"
	"fmt"
	"io"
	"sync"

	// Every backend of rclone registers itself on import, so a configuration may name any of them.
	_ "github.com/rclone/rclone/backend/all"

	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/modules/storages/internal/providers"
)

type Deps struct {
	Repositories   Repositories
	FieldEncryptor *crypto.FieldEncryptor
}

type Provider struct {
	repository     *settingsRepository
	fieldEncryptor *crypto.FieldEncryptor
	// configMutex guards rclone's own configuration store, which the library keeps for the whole
	// process: building a remote writes its section there and reads it back.
	configMutex sync.Mutex
	remotes     *remoteCache
}

func New(deps Deps) *Provider {
	return &Provider{
		repository:     deps.Repositories.settings,
		fieldEncryptor: deps.FieldEncryptor,
		configMutex:    sync.Mutex{},
		remotes:        newRemoteCache(rcloneRemoteIdleTimeout),
	}
}

func (p *Provider) GetType() providers.Type {
	return providers.TypeRclone
}

func (p *Provider) GetRepository() providers.SettingsRepository {
	return p.repository
}

func (p *Provider) Validate(settings providers.Settings) error {
	rcloneSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	if rcloneSettings.ConfigContent == "" {
		return providers.NewInvalidSettingError("configContent", "is required")
	}

	configContent, err := p.fieldEncryptor.Decrypt(rcloneSettings.ConfigContent)
	if err != nil {
		return fmt.Errorf("decrypt the rclone configuration: %w", err)
	}

	parsedConfig, err := parseConfigContent(configContent)
	if err != nil {
		return providers.NewInvalidSettingError("configContent", "is not an rclone configuration")
	}

	if len(parsedConfig) != 1 {
		return providers.NewInvalidSettingError(
			"configContent",
			fmt.Sprintf(
				"must contain exactly one remote section, but found %d; create a separate storage for each remote",
				len(parsedConfig),
			),
		)
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

	if incomingSettings.ConfigContent == "" {
		incomingSettings.ConfigContent = storedSettings.ConfigContent
	}

	return incomingSettings, nil
}

func (p *Provider) SaveFile(ctx context.Context, file providers.FileAccess, content io.Reader) error {
	remote, err := p.newConfiguredRemote(file.Location)
	if err != nil {
		return err
	}

	return remote.saveFile(ctx, file.Logger, file.FileName, content)
}

func (p *Provider) OpenFile(ctx context.Context, file providers.FileAccess) (io.ReadCloser, error) {
	remote, err := p.newConfiguredRemote(file.Location)
	if err != nil {
		return nil, err
	}

	return remote.openFile(ctx, file.Logger, file.FileName)
}

func (p *Provider) DeleteFile(ctx context.Context, file providers.FileAccess) error {
	remote, err := p.newConfiguredRemote(file.Location)
	if err != nil {
		return err
	}

	return remote.deleteFile(ctx, file.Logger, file.FileName)
}

func (p *Provider) TestConnection(ctx context.Context, location providers.Location) error {
	remote, err := p.newConfiguredRemote(location)
	if err != nil {
		return err
	}

	return remote.testConnection(ctx)
}

func (p *Provider) newConfiguredRemote(location providers.Location) (*configuredRemote, error) {
	rcloneSettings, err := providers.CastSettings[Settings](location.Settings)
	if err != nil {
		return nil, err
	}

	return &configuredRemote{provider: p, settings: rcloneSettings, storageID: location.StorageID}, nil
}
