package httpapi

import "databasus-backend/internal/modules/storages/internal/providers/nas"

func toNasSettings(body NasStorageSettings) nas.Settings {
	return nas.Settings{
		Host:     body.Host,
		Port:     body.Port,
		Share:    body.Share,
		Username: body.Username,
		Password: getStringOrEmpty(body.Password),
		IsSSL:    body.IsSsl,
		Domain:   getStringOrEmpty(body.Domain),
		Path:     getStringOrEmpty(body.Path),
	}
}

func fromNasSettings(settings nas.Settings) NasStorageSettings {
	return NasStorageSettings{
		Type:     NasStorageSettingsTypeNAS,
		Host:     settings.Host,
		Port:     settings.Port,
		Share:    settings.Share,
		Username: settings.Username,
		Password: nil,
		IsSsl:    settings.IsSSL,
		Domain:   toOptionalString(settings.Domain),
		Path:     toOptionalString(settings.Path),
	}
}
