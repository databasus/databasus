package nas

import (
	"context"
	"io"

	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/modules/storages/internal/providers"
	"databasus-backend/internal/modules/storages/internal/providers/connections"
)

const maxPort = 65535

type Deps struct {
	Repositories   Repositories
	FieldEncryptor *crypto.FieldEncryptor
	// ChunkWrites runs the write of each chunk, so a write blocked on a silent server cannot hold
	// the upload past its context.
	ChunkWrites *background.Group
}

type Provider struct {
	repository     *settingsRepository
	fieldEncryptor *crypto.FieldEncryptor
	chunkWrites    *background.Group
	sessionPool    *connections.SessionPool[*nasSession]
}

func New(deps Deps) *Provider {
	return &Provider{
		repository:     deps.Repositories.settings,
		fieldEncryptor: deps.FieldEncryptor,
		chunkWrites:    deps.ChunkWrites,
		sessionPool: connections.NewSessionPool(connections.SessionOperations[*nasSession]{
			Close: closeSession,
			Probe: probeSession,
		}, connections.GetDefaultPoolSettings()),
	}
}

func (p *Provider) GetType() providers.Type {
	return providers.TypeNAS
}

func (p *Provider) GetRepository() providers.SettingsRepository {
	return p.repository
}

func (p *Provider) Validate(settings providers.Settings) error {
	nasSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	switch {
	case nasSettings.Host == "":
		return providers.NewInvalidSettingError("host", "is required")
	case nasSettings.Share == "":
		return providers.NewInvalidSettingError("share", "is required")
	case nasSettings.Username == "":
		return providers.NewInvalidSettingError("username", "is required")
	case nasSettings.Password == "":
		return providers.NewInvalidSettingError("password", "is required")
	case nasSettings.Port <= 0 || nasSettings.Port > maxPort:
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

	isAddressChanged := incomingSettings.Host != storedSettings.Host || incomingSettings.Port != storedSettings.Port

	if incomingSettings.Password == "" && !isAddressChanged {
		incomingSettings.Password = storedSettings.Password
	}

	return incomingSettings, nil
}

func (p *Provider) SaveFile(ctx context.Context, file providers.FileAccess, content io.Reader) error {
	share, err := p.newNasShare(file.Location)
	if err != nil {
		return err
	}

	return share.saveFile(ctx, file.Logger, file.FileName, content)
}

func (p *Provider) OpenFile(ctx context.Context, file providers.FileAccess) (io.ReadCloser, error) {
	share, err := p.newNasShare(file.Location)
	if err != nil {
		return nil, err
	}

	return share.openFile(ctx, file.Logger, file.FileName)
}

func (p *Provider) DeleteFile(ctx context.Context, file providers.FileAccess) error {
	share, err := p.newNasShare(file.Location)
	if err != nil {
		return err
	}

	return share.deleteFile(ctx, file.Logger, file.FileName)
}

func (p *Provider) TestConnection(ctx context.Context, location providers.Location) error {
	share, err := p.newNasShare(location)
	if err != nil {
		return err
	}

	return share.testConnection(ctx)
}

func (p *Provider) newNasShare(location providers.Location) (*nasShare, error) {
	nasSettings, err := providers.CastSettings[Settings](location.Settings)
	if err != nil {
		return nil, err
	}

	return &nasShare{provider: p, settings: nasSettings, storageID: location.StorageID}, nil
}
