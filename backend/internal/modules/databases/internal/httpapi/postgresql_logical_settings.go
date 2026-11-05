package httpapi

import (
	"databasus-backend/internal/modules/databases/internal/providers/postgresqllogical"
)

func toPostgresqlLogicalSettings(body PostgresqlLogicalSettings) postgresqllogical.Settings {
	return postgresqllogical.Settings{
		Version:  "",
		Host:     body.Host,
		Port:     body.Port,
		Username: body.Username,
		Password: getValueOrZero(body.Password),
		Database: getValueOrZero(body.Database),
		TLS: toPostgresqlTLSSettings(postgresqlTLSBody{
			sslMode:       body.SslMode,
			sslClientCert: body.SslClientCert,
			sslClientKey:  body.SslClientKey,
			sslRootCert:   body.SslRootCert,
		}),
		SSHTunnel:          toSSHTunnelSettings(body.SshTunnel),
		IncludeSchemas:     body.IncludeSchemas,
		ExcludeTables:      body.ExcludeTables,
		ExcludeTableData:   body.ExcludeTableData,
		CPUCount:           body.CpuCount,
		IsSkipUserMappings: body.IsSkipUserMappings,
	}
}

func fromPostgresqlLogicalSettings(settings postgresqllogical.Settings) PostgresqlLogicalSettings {
	tlsBody := fromPostgresqlTLSSettings(settings.TLS)

	return PostgresqlLogicalSettings{
		Type:               PostgresqlLogicalSettingsTypePOSTGRESLOGICAL,
		Version:            toOptionalValue(PostgresqlVersion(settings.Version)),
		Host:               settings.Host,
		Port:               settings.Port,
		Username:           settings.Username,
		Password:           nil,
		Database:           toOptionalValue(settings.Database),
		SslMode:            tlsBody.sslMode,
		SslClientCert:      tlsBody.sslClientCert,
		SslClientKey:       tlsBody.sslClientKey,
		SslRootCert:        tlsBody.sslRootCert,
		SshTunnel:          fromSSHTunnelSettings(settings.SSHTunnel),
		IncludeSchemas:     toNonNilStrings(settings.IncludeSchemas),
		ExcludeTables:      toNonNilStrings(settings.ExcludeTables),
		ExcludeTableData:   toNonNilStrings(settings.ExcludeTableData),
		CpuCount:           settings.CPUCount,
		IsSkipUserMappings: settings.IsSkipUserMappings,
	}
}
