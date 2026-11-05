package httpapi

import "databasus-backend/internal/modules/storages/internal/providers/ftp"

func toFtpSettings(body FtpStorageSettings) ftp.Settings {
	return ftp.Settings{
		Host:            body.Host,
		Port:            body.Port,
		Username:        body.Username,
		Password:        getStringOrEmpty(body.Password),
		Path:            getStringOrEmpty(body.Path),
		IsSSL:           body.IsSsl,
		IsSkipTLSVerify: body.IsSkipTlsVerify,
	}
}

func fromFtpSettings(settings ftp.Settings) FtpStorageSettings {
	return FtpStorageSettings{
		Type:            FtpStorageSettingsTypeFTP,
		Host:            settings.Host,
		Port:            settings.Port,
		Username:        settings.Username,
		Password:        nil,
		Path:            toOptionalString(settings.Path),
		IsSsl:           settings.IsSSL,
		IsSkipTlsVerify: settings.IsSkipTLSVerify,
	}
}
