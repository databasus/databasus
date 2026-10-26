package mongodb

import (
	"fmt"

	mongodb_engine "databasus-backend/internal/engines/mongodb"
	"databasus-backend/internal/util/encryption"
)

// decryptSettings builds the engine settings with every secret in clear text. A nil encryptor
// passes the values through, which is how a restore target that was never persisted arrives.
func (m *MongodbDatabase) decryptSettings(encryptor encryption.FieldEncryptor) (mongodb_engine.Settings, error) {
	password, err := encryption.DecryptFieldIfNeeded(m.Password, encryptor)
	if err != nil {
		return mongodb_engine.Settings{}, fmt.Errorf("failed to decrypt password: %w", err)
	}

	settings := m.buildSettings(password)

	if m.SshTunnel.IsEnabled {
		tunnelSettings, err := m.SshTunnel.DecryptSettings(encryptor)
		if err != nil {
			return mongodb_engine.Settings{}, err
		}

		settings.SSHTunnel = &tunnelSettings
	}

	return settings, nil
}

func (m *MongodbDatabase) newSource(encryptor encryption.FieldEncryptor) (*mongodb_engine.Source, error) {
	settings, err := m.decryptSettings(encryptor)
	if err != nil {
		return nil, err
	}

	return mongodb_engine.NewSource(settings), nil
}

// buildSettings maps the model with an already decrypted password and without its tunnel, which is
// all the connection URIs need.
func (m *MongodbDatabase) buildSettings(password string) mongodb_engine.Settings {
	port := 0
	if m.Port != nil {
		port = *m.Port
	}

	return mongodb_engine.Settings{
		Host:               m.Host,
		Port:               port,
		Username:           m.Username,
		Password:           password,
		Database:           m.Database,
		AuthDatabase:       m.AuthDatabase,
		IsTLS:              m.IsHttps,
		IsSrv:              m.IsSrv,
		IsDirectConnection: m.IsDirectConnection,
		SSHTunnel:          nil,
	}
}
