package postgresql_physical

import (
	"fmt"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/util/encryption"
)

// Every libpq path goes through here: the pgx inspection and replication connections, pg_basebackup
// and pg_receivewal.
func (p *PostgresqlPhysicalDatabase) DecryptCredentialSpec(
	encryptor encryption.FieldEncryptor,
) (pgshared.CredentialSpec, error) {
	tls, err := decryptTLS(p.SslMode, p.SslClientCert, p.SslClientKey, p.SslRootCert, encryptor)
	if err != nil {
		return pgshared.CredentialSpec{}, err
	}

	return pgshared.CredentialSpec{
		Host:     p.Host,
		HostAddr: p.GetHostAddr(),
		Port:     p.Port,
		Username: p.Username,
		TLS:      tls,
	}, nil
}

// WriteCredentialFiles decrypts the certificates and writes them with the .pgpass file for a client
// tool.
func (p *PostgresqlPhysicalDatabase) WriteCredentialFiles(
	password string,
	encryptor encryption.FieldEncryptor,
) (*pgshared.CredentialTempFiles, error) {
	spec, err := p.DecryptCredentialSpec(encryptor)
	if err != nil {
		return nil, err
	}

	return pgshared.WriteCredentialFilesToTempDir(spec, password)
}

// GetHostAddr returns the local tunnel address the connection goes to, or "" without a tunnel.
func (p *PostgresqlPhysicalDatabase) GetHostAddr() string {
	if p.LocalTunnelEndpoint == nil {
		return ""
	}

	return p.LocalTunnelEndpoint.Host
}

func decryptTLS(
	mode pgshared.SslMode,
	clientCert, clientKey, rootCert string,
	encryptor encryption.FieldEncryptor,
) (pgshared.TLS, error) {
	tls := pgshared.TLS{Mode: mode}

	for _, field := range []struct {
		encrypted string
		decrypted *string
	}{
		{clientCert, &tls.ClientCert},
		{clientKey, &tls.ClientKey},
		{rootCert, &tls.RootCert},
	} {
		if field.encrypted == "" {
			continue
		}

		decrypted, err := encryption.DecryptFieldIfNeeded(field.encrypted, encryptor)
		if err != nil {
			return pgshared.TLS{}, fmt.Errorf("failed to decrypt an SSL certificate: %w", err)
		}

		*field.decrypted = decrypted
	}

	return tls, nil
}
