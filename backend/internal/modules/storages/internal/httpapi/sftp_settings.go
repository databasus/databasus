package httpapi

import "databasus-backend/internal/modules/storages/internal/providers/sftp"

func toSftpSettings(body SftpStorageSettings) sftp.Settings {
	return sftp.Settings{
		Host:                body.Host,
		Port:                body.Port,
		Username:            body.Username,
		Password:            getStringOrEmpty(body.Password),
		PrivateKey:          getStringOrEmpty(body.PrivateKey),
		Path:                getStringOrEmpty(body.Path),
		IsSkipHostKeyVerify: body.IsSkipHostKeyVerify,
	}
}

func fromSftpSettings(settings sftp.Settings) SftpStorageSettings {
	return SftpStorageSettings{
		Type:                SftpStorageSettingsTypeSFTP,
		Host:                settings.Host,
		Port:                settings.Port,
		Username:            settings.Username,
		Password:            nil,
		PrivateKey:          nil,
		Path:                toOptionalString(settings.Path),
		IsSkipHostKeyVerify: settings.IsSkipHostKeyVerify,
	}
}
