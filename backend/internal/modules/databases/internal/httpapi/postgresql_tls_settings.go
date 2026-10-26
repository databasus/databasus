package httpapi

import (
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/modules/databases/internal/providers"
)

// postgresqlTLSBody is the TLS part that the logical and the physical PostgreSQL schemas share.
type postgresqlTLSBody struct {
	sslMode       PostgresqlSslMode
	sslClientCert *string
	sslClientKey  *string
	sslRootCert   *string
}

func toPostgresqlTLSSettings(body postgresqlTLSBody) providers.PostgresqlTLSSettings {
	return providers.PostgresqlTLSSettings{
		SslMode:    pgshared.SslMode(body.sslMode),
		ClientCert: getValueOrZero(body.sslClientCert),
		ClientKey:  getValueOrZero(body.sslClientKey),
		RootCert:   getValueOrZero(body.sslRootCert),
	}
}

// fromPostgresqlTLSSettings leaves the client key out, because the contract marks it writeOnly.
func fromPostgresqlTLSSettings(settings providers.PostgresqlTLSSettings) postgresqlTLSBody {
	return postgresqlTLSBody{
		sslMode:       PostgresqlSslMode(settings.SslMode),
		sslClientCert: toOptionalValue(settings.ClientCert),
		sslClientKey:  nil,
		sslRootCert:   toOptionalValue(settings.RootCert),
	}
}
