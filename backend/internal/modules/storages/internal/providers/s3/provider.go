package s3

import (
	"context"
	"io"

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
	transports     sharedTransports
	bucketRegions  *bucketRegionCache
	partLayout     partLayout
}

// partLayout is how a file is split into objects. Tests shrink it, because rolling at the production
// 15.6 GiB object boundary would need about 15 GiB per case.
type partLayout struct {
	innerPartSize     int
	maxPartsPerObject int
}

func New(deps Deps) *Provider {
	return &Provider{
		repository:     deps.Repositories.settings,
		fieldEncryptor: deps.FieldEncryptor,
		transports:     newSharedTransports(),
		bucketRegions:  newBucketRegionCache(),
		partLayout:     partLayout{innerPartSize: defaultInnerPartSize, maxPartsPerObject: defaultMaxPartsPerObject},
	}
}

func (p *Provider) GetType() providers.Type {
	return providers.TypeS3
}

func (p *Provider) GetRepository() providers.SettingsRepository {
	return p.repository
}

func (p *Provider) Validate(settings providers.Settings) error {
	s3Settings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	if s3Settings.Bucket == "" {
		return providers.NewInvalidSettingError("s3Bucket", "is required")
	}

	if s3Settings.AccessKey == "" {
		return providers.NewInvalidSettingError("s3AccessKey", "is required")
	}

	if s3Settings.SecretKey == "" {
		return providers.NewInvalidSettingError("s3SecretKey", "is required")
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

	if incomingSettings.Prefix != storedSettings.Prefix {
		return nil, providers.NewFixedSettingError("s3Prefix")
	}

	isAddressChanged := incomingSettings.Endpoint != storedSettings.Endpoint

	if incomingSettings.AccessKey == "" && !isAddressChanged {
		incomingSettings.AccessKey = storedSettings.AccessKey
	}

	if incomingSettings.SecretKey == "" && !isAddressChanged {
		incomingSettings.SecretKey = storedSettings.SecretKey
	}

	return incomingSettings, nil
}

func (p *Provider) SaveFile(ctx context.Context, file providers.FileAccess, content io.Reader) error {
	targetBucket, err := p.newBucket(file.Settings)
	if err != nil {
		return err
	}

	return targetBucket.saveFile(ctx, file.Logger, file.FileName, content)
}

func (p *Provider) OpenFile(ctx context.Context, file providers.FileAccess) (io.ReadCloser, error) {
	targetBucket, err := p.newBucket(file.Settings)
	if err != nil {
		return nil, err
	}

	return targetBucket.openFile(ctx, file.Logger, file.FileName)
}

func (p *Provider) DeleteFile(ctx context.Context, file providers.FileAccess) error {
	targetBucket, err := p.newBucket(file.Settings)
	if err != nil {
		return err
	}

	return targetBucket.deleteFile(ctx, file.Logger, file.FileName)
}

func (p *Provider) TestConnection(ctx context.Context, location providers.Location) error {
	targetBucket, err := p.newBucket(location.Settings)
	if err != nil {
		return err
	}

	return targetBucket.testConnection(ctx)
}

func (p *Provider) newBucket(settings providers.Settings) (*bucket, error) {
	s3Settings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return nil, err
	}

	return &bucket{provider: p, settings: s3Settings}, nil
}
