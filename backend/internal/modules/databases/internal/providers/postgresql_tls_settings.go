package providers

import (
	"fmt"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/crypto"
)

// PostgresqlTLSSettings are the TLS settings of a PostgreSQL connection. ClientKey is a secret; the
// certificates are public and returned as they were saved.
type PostgresqlTLSSettings struct {
	SslMode    pgshared.SslMode
	ClientCert string
	ClientKey  string
	RootCert   string
}

func ValidatePostgresqlTLS(settings PostgresqlTLSSettings) error {
	err := pgshared.TLS{
		Mode:       settings.SslMode,
		ClientCert: settings.ClientCert,
		ClientKey:  settings.ClientKey,
		RootCert:   settings.RootCert,
	}.Validate()
	if err != nil {
		return NewInvalidSettingError("sslMode", err.Error())
	}

	return nil
}

// MergeStoredPostgresqlTLS keeps the stored client key when the incoming settings leave it out and the
// database's resource address stays the stored one.
func MergeStoredPostgresqlTLS(incoming, stored PostgresqlTLSSettings, isAddressChanged bool) PostgresqlTLSSettings {
	merged := incoming
	if merged.ClientKey == "" && !isAddressChanged {
		merged.ClientKey = stored.ClientKey
	}

	return merged
}

// DecryptPostgresqlTLS decrypts every value, because releases before the module encrypted the
// certificates too; a value in clear text passes through.
func DecryptPostgresqlTLS(fieldEncryptor *crypto.FieldEncryptor, settings PostgresqlTLSSettings) (pgshared.TLS, error) {
	tls := pgshared.TLS{Mode: settings.SslMode}

	for _, field := range []struct {
		stored    string
		decrypted *string
	}{
		{settings.ClientCert, &tls.ClientCert},
		{settings.ClientKey, &tls.ClientKey},
		{settings.RootCert, &tls.RootCert},
	} {
		decrypted, err := DecryptSecret(fieldEncryptor, field.stored, "an SSL certificate or key")
		if err != nil {
			return pgshared.TLS{}, err
		}

		*field.decrypted = decrypted
	}

	return tls, nil
}

// PostgresqlTLSColumns are the TLS columns of the PostgreSQL provider tables. A row type embeds them
// with gorm:"embedded".
type PostgresqlTLSColumns struct {
	SslMode    pgshared.SslMode `gorm:"column:ssl_mode"`
	ClientCert string           `gorm:"column:ssl_client_cert"`
	ClientKey  string           `gorm:"column:ssl_client_key"`
	RootCert   string           `gorm:"column:ssl_root_cert"`
}

// NewPostgresqlTLSColumns encrypts the client key when it is not encrypted yet and stores the
// certificates as they are.
func NewPostgresqlTLSColumns(
	fieldEncryptor *crypto.FieldEncryptor,
	settings PostgresqlTLSSettings,
) (PostgresqlTLSColumns, error) {
	encryptedClientKey, err := fieldEncryptor.Encrypt(settings.ClientKey)
	if err != nil {
		return PostgresqlTLSColumns{}, fmt.Errorf("encrypt the SSL client key: %w", err)
	}

	sslMode := settings.SslMode
	if sslMode == "" {
		sslMode = pgshared.SslModeDisable
	}

	return PostgresqlTLSColumns{
		SslMode:    sslMode,
		ClientCert: settings.ClientCert,
		ClientKey:  encryptedClientKey,
		RootCert:   settings.RootCert,
	}, nil
}

// ToSettings returns the certificates in clear text, which older releases stored encrypted, and
// keeps the client key encrypted.
func (c PostgresqlTLSColumns) ToSettings(fieldEncryptor *crypto.FieldEncryptor) (PostgresqlTLSSettings, error) {
	clientCert, err := DecryptSecret(fieldEncryptor, c.ClientCert, "the SSL client certificate")
	if err != nil {
		return PostgresqlTLSSettings{}, err
	}

	rootCert, err := DecryptSecret(fieldEncryptor, c.RootCert, "the SSL root certificate")
	if err != nil {
		return PostgresqlTLSSettings{}, err
	}

	return PostgresqlTLSSettings{
		SslMode:    c.SslMode,
		ClientCert: clientCert,
		ClientKey:  c.ClientKey,
		RootCert:   rootCert,
	}, nil
}
