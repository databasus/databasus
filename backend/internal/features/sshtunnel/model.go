// Package sshtunnel holds the stored SSH tunnel settings of the old database models. The tunnel
// itself lives in lib/sshtunnel.
package sshtunnel

import (
	"errors"
	"fmt"

	libsshtunnel "databasus-backend/internal/lib/sshtunnel"
	"databasus-backend/internal/util/encryption"
)

// Consumers must embed this with gorm:"embedded;embeddedPrefix:ssh_" so each engine table owns its
// own bastion columns; the column names below assume that prefix.
type Config struct {
	IsEnabled            bool     `json:"isEnabled"            gorm:"column:is_enabled;type:boolean;not null;default:false"`
	Host                 string   `json:"host"                 gorm:"column:host;type:text;not null;default:''"`
	Port                 int      `json:"port"                 gorm:"column:port;type:integer;not null;default:22"`
	Username             string   `json:"username"             gorm:"column:username;type:text;not null;default:''"`
	AuthType             AuthType `json:"authType"             gorm:"column:auth_type;type:text;not null;default:'PASSWORD'"`
	Password             string   `json:"password"             gorm:"column:password;type:text;not null;default:''"`
	PrivateKey           string   `json:"privateKey"           gorm:"column:private_key;type:text;not null;default:''"`
	PrivateKeyPassphrase string   `json:"privateKeyPassphrase" gorm:"column:private_key_passphrase;type:text;not null;default:''"`
}

func (c *Config) Validate() error {
	if c == nil || !c.IsEnabled {
		return nil
	}

	if c.Host == "" {
		return errors.New("SSH tunnel host is required")
	}

	if c.Port <= 0 || c.Port > 65535 {
		return errors.New("SSH tunnel port must be between 1 and 65535")
	}

	if c.Username == "" {
		return errors.New("SSH tunnel username is required")
	}

	// Rejecting the other type's secret rather than ignoring it keeps a submission from persisting a
	// second, invisible way into the bastion; on update the same invariant is reached by clearing.
	switch c.AuthType {
	case AuthTypePassword:
		if c.Password == "" {
			return errors.New("SSH tunnel password is required")
		}

		if c.PrivateKey != "" || c.PrivateKeyPassphrase != "" {
			return errors.New("SSH tunnel password auth must not carry a private key")
		}
	case AuthTypePrivateKey:
		if c.PrivateKey == "" {
			return errors.New("SSH tunnel private key is required")
		}

		if c.Password != "" {
			return errors.New("SSH tunnel private key auth must not carry a password")
		}
	case "":
		return errors.New("SSH tunnel auth type is required")
	default:
		return fmt.Errorf("invalid SSH tunnel auth type: %s", c.AuthType)
	}

	return nil
}

func (c *Config) HideSensitiveData() {
	if c == nil {
		return
	}

	c.Password = ""
	c.PrivateKey = ""
	c.PrivateKeyPassphrase = ""
}

func (c *Config) Update(incomingConfig *Config) {
	if c == nil || incomingConfig == nil {
		return
	}

	c.IsEnabled = incomingConfig.IsEnabled
	c.Host = incomingConfig.Host
	c.Port = incomingConfig.Port
	c.Username = incomingConfig.Username

	// A blank auth type keeps the stored one for the same reason the secrets below do, and with a
	// sharper consequence: switching it also discards the secret of the type left behind.
	if incomingConfig.AuthType != "" {
		c.AuthType = incomingConfig.AuthType
	}

	// A blank secret means "keep the stored one" - the edit form never receives
	// them back from the API, so overwriting on blank would wipe them.
	if incomingConfig.Password != "" {
		c.Password = incomingConfig.Password
	}

	if incomingConfig.PrivateKey != "" {
		c.PrivateKey = incomingConfig.PrivateKey
	}

	if incomingConfig.PrivateKeyPassphrase != "" {
		c.PrivateKeyPassphrase = incomingConfig.PrivateKeyPassphrase
	}

	c.clearSecretsOfUnusedAuthType()
}

func (c *Config) EncryptSensitiveFields(encryptor encryption.FieldEncryptor) error {
	if c == nil {
		return nil
	}

	for _, field := range []*string{
		&c.Password,
		&c.PrivateKey,
		&c.PrivateKeyPassphrase,
	} {
		if *field == "" {
			continue
		}

		encrypted, err := encryptor.Encrypt(*field)
		if err != nil {
			return err
		}

		*field = encrypted
	}

	return nil
}

// DecryptSettings builds the tunnel settings with the secrets in clear text. A restore target
// config is never persisted, so it arrives in plaintext with a nil encryptor.
func (c *Config) DecryptSettings(encryptor encryption.FieldEncryptor) (libsshtunnel.Settings, error) {
	password, err := decryptIfNeeded(c.Password, encryptor)
	if err != nil {
		return libsshtunnel.Settings{}, fmt.Errorf("failed to decrypt the SSH tunnel password: %w", err)
	}

	privateKey, err := decryptIfNeeded(c.PrivateKey, encryptor)
	if err != nil {
		return libsshtunnel.Settings{}, fmt.Errorf("failed to decrypt the SSH tunnel private key: %w", err)
	}

	passphrase, err := decryptIfNeeded(c.PrivateKeyPassphrase, encryptor)
	if err != nil {
		return libsshtunnel.Settings{}, fmt.Errorf(
			"failed to decrypt the SSH tunnel private key passphrase: %w", err,
		)
	}

	return libsshtunnel.Settings{
		Host:                 c.Host,
		Port:                 c.Port,
		Username:             c.Username,
		AuthType:             libsshtunnel.AuthType(c.AuthType),
		Password:             password,
		PrivateKey:           privateKey,
		PrivateKeyPassphrase: passphrase,
	}, nil
}

// Without this the "blank keeps the stored one" rule above would leave the previous way of logging
// in behind forever once the user switches the auth type.
func (c *Config) clearSecretsOfUnusedAuthType() {
	switch c.AuthType {
	case AuthTypePassword:
		c.PrivateKey = ""
		c.PrivateKeyPassphrase = ""
	case AuthTypePrivateKey:
		c.Password = ""
	}
}

func decryptIfNeeded(value string, encryptor encryption.FieldEncryptor) (string, error) {
	if encryptor == nil || value == "" {
		return value, nil
	}

	return encryptor.Decrypt(value)
}
