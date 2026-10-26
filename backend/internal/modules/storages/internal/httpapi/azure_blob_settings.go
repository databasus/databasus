package httpapi

import "databasus-backend/internal/modules/storages/internal/providers/azureblob"

func toAzureBlobSettings(body AzureBlobStorageSettings) azureblob.Settings {
	return azureblob.Settings{
		AuthMethod:       azureblob.AuthMethod(body.AuthMethod),
		ConnectionString: getStringOrEmpty(body.ConnectionString),
		AccountName:      getStringOrEmpty(body.AccountName),
		AccountKey:       getStringOrEmpty(body.AccountKey),
		ContainerName:    body.ContainerName,
		Endpoint:         getStringOrEmpty(body.Endpoint),
		Prefix:           getStringOrEmpty(body.Prefix),
	}
}

func fromAzureBlobSettings(settings azureblob.Settings) AzureBlobStorageSettings {
	return AzureBlobStorageSettings{
		Type:             AzureBlobStorageSettingsTypeAZUREBLOB,
		AuthMethod:       AzureBlobAuthMethod(settings.AuthMethod),
		ConnectionString: nil,
		AccountName:      toOptionalString(settings.AccountName),
		AccountKey:       nil,
		ContainerName:    settings.ContainerName,
		Endpoint:         toOptionalString(settings.Endpoint),
		Prefix:           toOptionalString(settings.Prefix),
	}
}
