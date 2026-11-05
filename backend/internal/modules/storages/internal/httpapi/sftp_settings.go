package httpapi

import "databasus-backend/internal/modules/storages/internal/providers/sftp"

func toSftpSettings(body SftpStorageSettings) sftp.Settings {
	authMethods := make([]sftp.AuthMethod, len(body.AuthMethods))
	for index, method := range body.AuthMethods {
		authMethods[index] = sftp.AuthMethod(method)
	}

	return sftp.Settings{
		AuthMethods:         authMethods,
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
	authMethods := make([]SftpAuthMethod, len(settings.AuthMethods))
	for index, method := range settings.AuthMethods {
		authMethods[index] = SftpAuthMethod(method)
	}

	return SftpStorageSettings{
		AuthMethods:         authMethods,
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
