package httpapi

import (
	"fmt"

	"databasus-backend/internal/modules/storages/internal/providers"
	"databasus-backend/internal/modules/storages/internal/providers/azureblob"
	"databasus-backend/internal/modules/storages/internal/providers/ftp"
	"databasus-backend/internal/modules/storages/internal/providers/googledrive"
	"databasus-backend/internal/modules/storages/internal/providers/local"
	"databasus-backend/internal/modules/storages/internal/providers/nas"
	"databasus-backend/internal/modules/storages/internal/providers/rclone"
	"databasus-backend/internal/modules/storages/internal/providers/s3"
	"databasus-backend/internal/modules/storages/internal/providers/sftp"
)

// toProviderSettings picks the provider's conversion by the discriminator of the settings. A
// provider cannot convert itself, because it would import the generated types of this package.
func toProviderSettings(body StorageSettings) (providers.Settings, error) {
	settingsBody, err := body.ValueByDiscriminator()
	if err != nil {
		return nil, errUnknownSettingsValue
	}

	switch typedBody := settingsBody.(type) {
	case LocalStorageSettings:
		return local.Settings{}, nil
	case S3StorageSettings:
		return toS3Settings(typedBody), nil
	case GoogleDriveStorageSettings:
		return toGoogleDriveSettings(typedBody), nil
	case NasStorageSettings:
		return toNasSettings(typedBody), nil
	case AzureBlobStorageSettings:
		return toAzureBlobSettings(typedBody), nil
	case FtpStorageSettings:
		return toFtpSettings(typedBody), nil
	case SftpStorageSettings:
		return toSftpSettings(typedBody), nil
	case RcloneStorageSettings:
		return toRcloneSettings(typedBody), nil
	}

	return nil, errUnknownSettingsValue
}

// fromProviderSettings leaves every secret out, because the contract marks them writeOnly.
func fromProviderSettings(settings providers.Settings) (StorageSettings, error) {
	var body StorageSettings

	var err error

	switch typedSettings := settings.(type) {
	case local.Settings:
		err = body.FromLocalStorageSettings(LocalStorageSettings{Type: LocalStorageSettingsTypeLOCAL})
	case s3.Settings:
		err = body.FromS3StorageSettings(fromS3Settings(typedSettings))
	case googledrive.Settings:
		err = body.FromGoogleDriveStorageSettings(fromGoogleDriveSettings(typedSettings))
	case nas.Settings:
		err = body.FromNasStorageSettings(fromNasSettings(typedSettings))
	case azureblob.Settings:
		err = body.FromAzureBlobStorageSettings(fromAzureBlobSettings(typedSettings))
	case ftp.Settings:
		err = body.FromFtpStorageSettings(fromFtpSettings(typedSettings))
	case sftp.Settings:
		err = body.FromSftpStorageSettings(fromSftpSettings(typedSettings))
	case rclone.Settings:
		err = body.FromRcloneStorageSettings(fromRcloneSettings(typedSettings))
	default:
		return StorageSettings{}, fmt.Errorf("%w: %T", errUnknownSettingsType, settings)
	}

	if err != nil {
		return StorageSettings{}, fmt.Errorf("encode the storage settings: %w", err)
	}

	return body, nil
}
