package httpapi

import "databasus-backend/internal/modules/databases/internal/providers/mariadb"

func toMariadbSettings(body MariadbSettings) mariadb.Settings {
	return mariadb.Settings{
		Version:             "",
		Host:                body.Host,
		Port:                body.Port,
		Username:            body.Username,
		Password:            getValueOrZero(body.Password),
		Database:            getValueOrZero(body.Database),
		IsHTTPS:             body.IsHttps,
		SSHTunnel:           toSSHTunnelSettings(body.SshTunnel),
		ExcludeTables:       body.ExcludeTables,
		Privileges:          nil,
		IsExcludeEvents:     body.IsExcludeEvents,
		IsSkipGaleraDisable: body.IsSkipGaleraDisable,
	}
}

func fromMariadbSettings(settings mariadb.Settings) MariadbSettings {
	return MariadbSettings{
		Type:                MariadbSettingsTypeMARIADB,
		Version:             toOptionalValue(MariadbVersion(settings.Version)),
		Host:                settings.Host,
		Port:                settings.Port,
		Username:            settings.Username,
		Password:            nil,
		Database:            toOptionalValue(settings.Database),
		IsHttps:             settings.IsHTTPS,
		SshTunnel:           fromSSHTunnelSettings(settings.SSHTunnel),
		ExcludeTables:       toNonNilStrings(settings.ExcludeTables),
		IsExcludeEvents:     settings.IsExcludeEvents,
		IsSkipGaleraDisable: settings.IsSkipGaleraDisable,
		Privileges:          new(toNonNilStrings(settings.Privileges)),
	}
}
