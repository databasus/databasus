package httpapi

import "databasus-backend/internal/modules/storages/internal/providers/googledrive"

func toGoogleDriveSettings(body GoogleDriveStorageSettings) googledrive.Settings {
	return googledrive.Settings{
		ClientID:     body.ClientId,
		ClientSecret: getStringOrEmpty(body.ClientSecret),
		TokenJSON:    getStringOrEmpty(body.TokenJson),
	}
}

func fromGoogleDriveSettings(settings googledrive.Settings) GoogleDriveStorageSettings {
	return GoogleDriveStorageSettings{
		Type:         GoogleDriveStorageSettingsTypeGOOGLEDRIVE,
		ClientId:     settings.ClientID,
		ClientSecret: nil,
		TokenJson:    nil,
	}
}
