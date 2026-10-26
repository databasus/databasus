package mysql

import (
	"fmt"

	"databasus-backend/internal/config"
	mysql_engine "databasus-backend/internal/engines/mysql"
	"databasus-backend/internal/lib/sshtunnel"
	"databasus-backend/internal/util/encryption"
)

// decryptSettings builds the engine settings with every secret in clear text. A nil encryptor
// passes the values through, which is how a restore target that was never persisted arrives.
func (m *MysqlDatabase) decryptSettings(encryptor encryption.FieldEncryptor) (mysql_engine.Settings, error) {
	password, err := encryption.DecryptFieldIfNeeded(m.Password, encryptor)
	if err != nil {
		return mysql_engine.Settings{}, fmt.Errorf("failed to decrypt password: %w", err)
	}

	var tunnelSettings *sshtunnel.Settings

	if m.SshTunnel.IsEnabled {
		decryptedTunnelSettings, err := m.SshTunnel.DecryptSettings(encryptor)
		if err != nil {
			return mysql_engine.Settings{}, err
		}

		tunnelSettings = &decryptedTunnelSettings
	}

	databaseName := ""
	if m.Database != nil {
		databaseName = *m.Database
	}

	return mysql_engine.Settings{
		Host:      m.Host,
		Port:      m.Port,
		Username:  m.Username,
		Password:  password,
		Database:  databaseName,
		IsHTTPS:   m.IsHttps,
		SSHTunnel: tunnelSettings,
	}, nil
}

func (m *MysqlDatabase) newSource(encryptor encryption.FieldEncryptor) (*mysql_engine.Source, error) {
	settings, err := m.decryptSettings(encryptor)
	if err != nil {
		return nil, err
	}

	return mysql_engine.NewSource(settings, mysql_engine.ClientTools{Root: config.GetEnv().ClientToolsDir}), nil
}

func (m *MysqlDatabase) writeInspection(inspection mysql_engine.Inspection) {
	m.Version = inspection.Version
	m.Privileges = inspection.Privileges.GetEffectivePrivilegesCsv()
}
