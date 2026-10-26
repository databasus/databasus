package postgresql_logical

import (
	"fmt"

	pglogical "databasus-backend/internal/engines/postgresql/logical"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/sshtunnel"
	"databasus-backend/internal/util/encryption"
)

// decryptSettings builds the engine settings with every secret in clear text. A nil encryptor
// passes the values through, which is how a restore target that was never persisted arrives.
func (p *PostgresqlLogicalDatabase) decryptSettings(encryptor encryption.FieldEncryptor) (pglogical.Settings, error) {
	password, err := encryption.DecryptFieldIfNeeded(p.Password, encryptor)
	if err != nil {
		return pglogical.Settings{}, fmt.Errorf("failed to decrypt password: %w", err)
	}

	credentialSpec, err := p.DecryptCredentialSpec(encryptor)
	if err != nil {
		return pglogical.Settings{}, err
	}

	var tunnelSettings *sshtunnel.Settings

	if p.SshTunnel.IsEnabled {
		decryptedTunnelSettings, err := p.SshTunnel.DecryptSettings(encryptor)
		if err != nil {
			return pglogical.Settings{}, err
		}

		tunnelSettings = &decryptedTunnelSettings
	}

	databaseName := ""
	if p.Database != nil {
		databaseName = *p.Database
	}

	return pglogical.Settings{
		Host:      credentialSpec.Host,
		HostAddr:  credentialSpec.HostAddr,
		Port:      credentialSpec.Port,
		Username:  credentialSpec.Username,
		Password:  password,
		Database:  databaseName,
		TLS:       credentialSpec.TLS,
		SSHTunnel: tunnelSettings,
	}, nil
}

func (p *PostgresqlLogicalDatabase) newSource(encryptor encryption.FieldEncryptor) (*pglogical.Source, error) {
	settings, err := p.decryptSettings(encryptor)
	if err != nil {
		return nil, err
	}

	return pglogical.NewSource(settings), nil
}

func (p *PostgresqlLogicalDatabase) getDumpFilter() pglogical.DumpFilter {
	return pglogical.DumpFilter{
		IncludeSchemas:           p.IncludeSchemas,
		ExcludeTablePatterns:     p.ExcludeTables,
		ExcludeTableDataPatterns: p.ExcludeTableData,
	}
}

// DecryptCredentialSpec maps this database into the credential inputs of pg_dump and pg_restore in
// the backup and restore usecases.
func (p *PostgresqlLogicalDatabase) DecryptCredentialSpec(
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
func (p *PostgresqlLogicalDatabase) WriteCredentialFiles(
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
func (p *PostgresqlLogicalDatabase) GetHostAddr() string {
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
