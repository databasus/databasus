package httpapi

import (
	"databasus-backend/internal/modules/databases/internal/providers"
	"databasus-backend/internal/modules/databases/internal/providers/postgresqlphysical"
)

// toPostgresqlPhysicalSettings ignores the detected values a client sends back: the provider owns
// them and keeps the stored ones.
func toPostgresqlPhysicalSettings(body PostgresqlPhysicalSettings) postgresqlphysical.Settings {
	return postgresqlphysical.Settings{
		Version:  "",
		Host:     body.Host,
		Port:     body.Port,
		Username: body.Username,
		Password: getValueOrZero(body.Password),
		TLS: toPostgresqlTLSSettings(postgresqlTLSBody{
			sslMode:       body.SslMode,
			sslClientCert: body.SslClientCert,
			sslClientKey:  body.SslClientKey,
			sslRootCert:   body.SslRootCert,
		}),
		SSHTunnel:           toSSHTunnelSettings(body.SshTunnel),
		BackupType:          providers.PhysicalBackupType(body.BackupType),
		ReplicationSlotName: "",
		SystemIdentifier:    nil,
		WalSegmentSizeBytes: nil,
	}
}

func fromPostgresqlPhysicalSettings(settings postgresqlphysical.Settings) PostgresqlPhysicalSettings {
	tlsBody := fromPostgresqlTLSSettings(settings.TLS)

	return PostgresqlPhysicalSettings{
		Type:                PostgresqlPhysicalSettingsTypePOSTGRESPHYSICAL,
		Version:             toOptionalValue(PostgresqlVersion(settings.Version)),
		Host:                settings.Host,
		Port:                settings.Port,
		Username:            settings.Username,
		Password:            nil,
		SslMode:             tlsBody.sslMode,
		SslClientCert:       tlsBody.sslClientCert,
		SslClientKey:        tlsBody.sslClientKey,
		SslRootCert:         tlsBody.sslRootCert,
		SshTunnel:           fromSSHTunnelSettings(settings.SSHTunnel),
		BackupType:          PhysicalBackupType(settings.BackupType),
		SystemIdentifier:    settings.SystemIdentifier,
		WalSegmentSizeBytes: settings.WalSegmentSizeBytes,
	}
}
