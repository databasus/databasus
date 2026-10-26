package sftp

import (
	"context"
	"io"

	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/modules/storages/internal/providers"
	"databasus-backend/internal/modules/storages/internal/providers/connections"
)

const maxPort = 65535

type Deps struct {
	Repositories   Repositories
	FieldEncryptor *crypto.FieldEncryptor
}

type Provider struct {
	repository     *settingsRepository
	fieldEncryptor *crypto.FieldEncryptor
	sessionPool    *connections.SessionPool[*sftpSession]
}

func New(deps Deps) *Provider {
	return &Provider{
		repository:     deps.Repositories.settings,
		fieldEncryptor: deps.FieldEncryptor,
		sessionPool: connections.NewSessionPool(connections.SessionOperations[*sftpSession]{
			Close: closeSession,
			Probe: probeSession,
		}, connections.GetDefaultPoolSettings()),
	}
}

func (p *Provider) GetType() providers.Type {
	return providers.TypeSFTP
}

func (p *Provider) GetRepository() providers.SettingsRepository {
	return p.repository
}

func (p *Provider) Validate(settings providers.Settings) error {
	sftpSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	switch {
	case sftpSettings.Host == "":
		return providers.NewInvalidSettingError("host", "is required")
	case sftpSettings.Username == "":
		return providers.NewInvalidSettingError("username", "is required")
	case sftpSettings.Password == "" && sftpSettings.PrivateKey == "":
		return providers.NewInvalidSettingError("password", "is required when no private key is given")
	case sftpSettings.Port <= 0 || sftpSettings.Port > maxPort:
		return providers.NewInvalidSettingError("port", "must be between 1 and 65535")
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

	if incomingSettings.Password == "" {
		incomingSettings.Password = storedSettings.Password
	}

	if incomingSettings.PrivateKey == "" {
		incomingSettings.PrivateKey = storedSettings.PrivateKey
	}

	return incomingSettings, nil
}

func (p *Provider) SaveFile(ctx context.Context, file providers.FileAccess, content io.Reader) error {
	server, err := p.newSftpServer(file.Location)
	if err != nil {
		return err
	}

	return server.saveFile(ctx, file.Logger, file.FileName, content)
}

func (p *Provider) OpenFile(ctx context.Context, file providers.FileAccess) (io.ReadCloser, error) {
	server, err := p.newSftpServer(file.Location)
	if err != nil {
		return nil, err
	}

	return server.openFile(ctx, file.Logger, file.FileName)
}

func (p *Provider) DeleteFile(ctx context.Context, file providers.FileAccess) error {
	server, err := p.newSftpServer(file.Location)
	if err != nil {
		return err
	}

	return server.deleteFile(ctx, file.Logger, file.FileName)
}

func (p *Provider) TestConnection(ctx context.Context, location providers.Location) error {
	server, err := p.newSftpServer(location)
	if err != nil {
		return err
	}

	return server.testConnection(ctx)
}

func (p *Provider) newSftpServer(location providers.Location) (*sftpServer, error) {
	sftpSettings, err := providers.CastSettings[Settings](location.Settings)
	if err != nil {
		return nil, err
	}

	return &sftpServer{provider: p, settings: sftpSettings, storageID: location.StorageID}, nil
}
