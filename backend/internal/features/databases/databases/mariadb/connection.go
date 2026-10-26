package mariadb

import (
	"fmt"

	"databasus-backend/internal/config"
	mariadb_engine "databasus-backend/internal/engines/mariadb"
	"databasus-backend/internal/lib/sshtunnel"
	"databasus-backend/internal/util/encryption"
)

// decryptSettings builds the engine settings with every secret in clear text. A nil encryptor
// passes the values through, which is how a restore target that was never persisted arrives.
func (m *MariadbDatabase) decryptSettings(encryptor encryption.FieldEncryptor) (mariadb_engine.Settings, error) {
	password, err := encryption.DecryptFieldIfNeeded(m.Password, encryptor)
	if err != nil {
		return mariadb_engine.Settings{}, fmt.Errorf("failed to decrypt password: %w", err)
	}

	var tunnelSettings *sshtunnel.Settings

	if m.SshTunnel.IsEnabled {
		decryptedTunnelSettings, err := m.SshTunnel.DecryptSettings(encryptor)
		if err != nil {
			return mariadb_engine.Settings{}, err
		}

		tunnelSettings = &decryptedTunnelSettings
	}

	databaseName := ""
	if m.Database != nil {
		databaseName = *m.Database
	}

	return mariadb_engine.Settings{
		Host:      m.Host,
		Port:      m.Port,
		Username:  m.Username,
		Password:  password,
		Database:  databaseName,
		IsHTTPS:   m.IsHttps,
		SSHTunnel: tunnelSettings,
	}, nil
}

func (m *MariadbDatabase) newSource(encryptor encryption.FieldEncryptor) (*mariadb_engine.Source, error) {
	settings, err := m.decryptSettings(encryptor)
	if err != nil {
		return nil, err
	}

	return mariadb_engine.NewSource(settings, mariadb_engine.ClientTools{Root: config.GetEnv().ClientToolsDir}), nil
}

func (m *MariadbDatabase) writeInspection(inspection mariadb_engine.Inspection) {
	m.Version = inspection.Version
	m.Privileges = inspection.Privileges.GetEffectivePrivilegesCsv()
}
