package azureblob

import (
	"context"
	"io"
	"net"
	"net/http"

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
	// httpClient keeps the connections of every Azure Blob storage. Clients are still built per call
	// from freshly decrypted credentials. Only the connections are shared: every request is signed or
	// carries its SAS token, so a changed credential takes effect on the next request even over a
	// connection opened before the change.
	httpClient *http.Client
}

func New(deps Deps) *Provider {
	return &Provider{
		repository:     deps.Repositories.settings,
		fieldEncryptor: deps.FieldEncryptor,
		httpClient: &http.Client{Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout: azureConnectTimeout,
			}).DialContext,
			TLSHandshakeTimeout:   azureTLSHandshakeTimeout,
			ResponseHeaderTimeout: azureResponseTimeout,
			IdleConnTimeout:       azureIdleConnTimeout,
			MaxIdleConnsPerHost:   azureMaxIdleConnsPerHost,
		}},
	}
}

func (p *Provider) GetType() providers.Type {
	return providers.TypeAzureBlob
}

func (p *Provider) GetRepository() providers.SettingsRepository {
	return p.repository
}

func (p *Provider) Validate(settings providers.Settings) error {
	azureSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	if azureSettings.ContainerName == "" {
		return providers.NewInvalidSettingError("containerName", "is required")
	}

	switch azureSettings.AuthMethod {
	case AuthMethodConnectionString:
		if azureSettings.ConnectionString == "" {
			return providers.NewInvalidSettingError("connectionString", "is required with CONNECTION_STRING")
		}
	case AuthMethodAccountKey:
		if azureSettings.AccountName == "" {
			return providers.NewInvalidSettingError("accountName", "is required with ACCOUNT_KEY")
		}

		if azureSettings.AccountKey == "" {
			return providers.NewInvalidSettingError("accountKey", "is required with ACCOUNT_KEY")
		}
	default:
		return providers.NewInvalidSettingError("authMethod", "is not an Azure Blob authentication method")
	}

	return nil
}

// MergeStoredSettings also keeps a stored account name the request leaves out, as the form of a
// saved storage sends none when the user only changes the container.
func (p *Provider) MergeStoredSettings(merge providers.SettingsMerge) (providers.Settings, error) {
	incomingSettings, err := providers.CastSettings[Settings](merge.Incoming)
	if err != nil {
		return nil, err
	}

	storedSettings, err := providers.CastSettings[Settings](merge.Stored)
	if err != nil {
		return nil, err
	}

	if incomingSettings.Prefix != storedSettings.Prefix {
		return nil, providers.NewFixedSettingError("prefix")
	}

	if incomingSettings.ConnectionString == "" {
		incomingSettings.ConnectionString = storedSettings.ConnectionString
	}

	if incomingSettings.AccountName == "" {
		incomingSettings.AccountName = storedSettings.AccountName
	}

	isAddressChanged := incomingSettings.Endpoint != storedSettings.Endpoint ||
		incomingSettings.AccountName != storedSettings.AccountName

	if incomingSettings.AccountKey == "" && !isAddressChanged {
		incomingSettings.AccountKey = storedSettings.AccountKey
	}

	return incomingSettings, nil
}

func (p *Provider) SaveFile(ctx context.Context, file providers.FileAccess, content io.Reader) error {
	container, err := p.newBlobContainer(file.Settings)
	if err != nil {
		return err
	}

	return container.saveFile(ctx, file.Logger, file.FileName, content)
}

func (p *Provider) OpenFile(ctx context.Context, file providers.FileAccess) (io.ReadCloser, error) {
	container, err := p.newBlobContainer(file.Settings)
	if err != nil {
		return nil, err
	}

	return container.openFile(ctx, file.Logger, file.FileName)
}

func (p *Provider) DeleteFile(ctx context.Context, file providers.FileAccess) error {
	container, err := p.newBlobContainer(file.Settings)
	if err != nil {
		return err
	}

	return container.deleteFile(ctx, file.Logger, file.FileName)
}

func (p *Provider) TestConnection(ctx context.Context, location providers.Location) error {
	container, err := p.newBlobContainer(location.Settings)
	if err != nil {
		return err
	}

	return container.testConnection(ctx)
}

func (p *Provider) newBlobContainer(settings providers.Settings) (*blobContainer, error) {
	azureSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return nil, err
	}

	return &blobContainer{provider: p, settings: azureSettings}, nil
}
