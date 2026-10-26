package httpapi

import "databasus-backend/internal/modules/storages/internal/providers/rclone"

func toRcloneSettings(body RcloneStorageSettings) rclone.Settings {
	return rclone.Settings{
		ConfigContent: getStringOrEmpty(body.ConfigContent),
		RemotePath:    getStringOrEmpty(body.RemotePath),
	}
}

func fromRcloneSettings(settings rclone.Settings) RcloneStorageSettings {
	return RcloneStorageSettings{
		Type:          RcloneStorageSettingsTypeRCLONE,
		ConfigContent: nil,
		RemotePath:    toOptionalString(settings.RemotePath),
	}
}
