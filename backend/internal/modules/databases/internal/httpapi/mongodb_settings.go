package httpapi

import "databasus-backend/internal/modules/databases/internal/providers/mongodb"

func toMongodbSettings(body MongodbSettings) mongodb.Settings {
	return mongodb.Settings{
		Version:            "",
		Host:               body.Host,
		Port:               getValueOrZero(body.Port),
		Username:           body.Username,
		Password:           getValueOrZero(body.Password),
		Database:           body.Database,
		AuthDatabase:       body.AuthDatabase,
		IsHTTPS:            body.IsHttps,
		IsSrv:              body.IsSrv,
		IsDirectConnection: body.IsDirectConnection,
		CPUCount:           body.CpuCount,
		SSHTunnel:          toSSHTunnelSettings(body.SshTunnel),
		ExcludeCollections: body.ExcludeCollections,
	}
}

func fromMongodbSettings(settings mongodb.Settings) MongodbSettings {
	return MongodbSettings{
		Type:               MongodbSettingsTypeMONGODB,
		Version:            toOptionalValue(MongodbVersion(settings.Version)),
		Host:               settings.Host,
		Port:               toOptionalValue(settings.Port),
		Username:           settings.Username,
		Password:           nil,
		Database:           settings.Database,
		AuthDatabase:       settings.AuthDatabase,
		IsHttps:            settings.IsHTTPS,
		IsSrv:              settings.IsSrv,
		IsDirectConnection: settings.IsDirectConnection,
		CpuCount:           settings.CPUCount,
		SshTunnel:          fromSSHTunnelSettings(settings.SSHTunnel),
		ExcludeCollections: toNonNilStrings(settings.ExcludeCollections),
	}
}
