package postgresql_physical

import (
	"fmt"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/lib/sshtunnel"
	"databasus-backend/internal/util/encryption"
)

// decryptSettings builds the engine settings with every secret in clear text. A nil encryptor
// passes the values through.
func (p *PostgresqlPhysicalDatabase) decryptSettings(encryptor encryption.FieldEncryptor) (pgphysical.Settings, error) {
	password, err := encryption.DecryptFieldIfNeeded(p.Password, encryptor)
	if err != nil {
		return pgphysical.Settings{}, fmt.Errorf("failed to decrypt password: %w", err)
	}

	credentialSpec, err := p.DecryptCredentialSpec(encryptor)
	if err != nil {
		return pgphysical.Settings{}, err
	}

	var tunnelSettings *sshtunnel.Settings

	if p.SshTunnel.IsEnabled {
		decryptedTunnelSettings, err := p.SshTunnel.DecryptSettings(encryptor)
		if err != nil {
			return pgphysical.Settings{}, err
		}

		tunnelSettings = &decryptedTunnelSettings
	}

	return pgphysical.Settings{
		Host:                credentialSpec.Host,
		HostAddr:            credentialSpec.HostAddr,
		Port:                credentialSpec.Port,
		Username:            credentialSpec.Username,
		Password:            password,
		TLS:                 credentialSpec.TLS,
		SSHTunnel:           tunnelSettings,
		ReplicationSlotName: p.ReplicationSlotName,
	}, nil
}

func (p *PostgresqlPhysicalDatabase) newSource(encryptor encryption.FieldEncryptor) (*pgphysical.Source, error) {
	settings, err := p.decryptSettings(encryptor)
	if err != nil {
		return nil, err
	}

	return pgphysical.NewSource(settings), nil
}
