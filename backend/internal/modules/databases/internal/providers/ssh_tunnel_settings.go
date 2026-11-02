package providers

import (
	"fmt"

	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/sshtunnel"
)

const maxPort = 65535

// SSHAuthType is how Databasus signs in to the SSH bastion.
type SSHAuthType string

const (
	SSHAuthTypePassword   SSHAuthType = "PASSWORD"
	SSHAuthTypePrivateKey SSHAuthType = "PRIVATE_KEY"
)

// SSHTunnelSettings reach a database through a bastion. When the tunnel is enabled, the host and the
// port of the database address it as the bastion sees it. Password, PrivateKey and
// PrivateKeyPassphrase are secrets.
type SSHTunnelSettings struct {
	IsEnabled            bool
	Host                 string
	Port                 int
	Username             string
	AuthType             SSHAuthType
	Password             string
	PrivateKey           string
	PrivateKeyPassphrase string
}

// ValidateSSHTunnel refuses the secret of the other authentication type rather than ignoring it, so a
// save never keeps a second, invisible way into the bastion.
func ValidateSSHTunnel(settings SSHTunnelSettings) error {
	if !settings.IsEnabled {
		return nil
	}

	switch {
	case settings.Host == "":
		return NewInvalidSettingError("sshTunnel.host", "is required")
	case settings.Port <= 0 || settings.Port > maxPort:
		return NewInvalidSettingError("sshTunnel.port", "must be between 1 and 65535")
	case settings.Username == "":
		return NewInvalidSettingError("sshTunnel.username", "is required")
	}

	switch settings.AuthType {
	case SSHAuthTypePassword:
		if settings.Password == "" {
			return NewInvalidSettingError("sshTunnel.password", "is required")
		}

		if settings.PrivateKey != "" || settings.PrivateKeyPassphrase != "" {
			return NewInvalidSettingError("sshTunnel.privateKey", "must be empty with password authentication")
		}
	case SSHAuthTypePrivateKey:
		if settings.PrivateKey == "" {
			return NewInvalidSettingError("sshTunnel.privateKey", "is required")
		}

		if settings.Password != "" {
			return NewInvalidSettingError("sshTunnel.password", "must be empty with private key authentication")
		}
	default:
		return NewInvalidSettingError("sshTunnel.authType", "must be PASSWORD or PRIVATE_KEY")
	}

	return nil
}

// IsSSHTunnelAddressChanged reports a change of where the tunnel connects. A disabled tunnel connects
// nowhere, so turning a tunnel on or off counts as a change and moving a disabled one does not.
func IsSSHTunnelAddressChanged(incoming, stored SSHTunnelSettings) bool {
	if incoming.IsEnabled != stored.IsEnabled {
		return true
	}

	return incoming.IsEnabled && (incoming.Host != stored.Host || incoming.Port != stored.Port)
}

// MergeStoredSSHTunnel keeps the stored secrets the incoming settings leave out, because the edit form
// never receives them back, while the tunnel's address stays the stored one. At another address a
// secret stays empty, so a user who may not see it cannot send it to a bastion of their choice.
// Switching the authentication type drops the secret of the type left behind, so the previous way into
// the bastion does not stay behind unseen.
func MergeStoredSSHTunnel(incoming, stored SSHTunnelSettings) SSHTunnelSettings {
	merged := incoming

	if merged.AuthType == "" {
		merged.AuthType = stored.AuthType
	}

	if !IsSSHTunnelAddressChanged(incoming, stored) {
		if merged.Password == "" {
			merged.Password = stored.Password
		}

		if merged.PrivateKey == "" {
			merged.PrivateKey = stored.PrivateKey
		}

		if merged.PrivateKeyPassphrase == "" {
			merged.PrivateKeyPassphrase = stored.PrivateKeyPassphrase
		}
	}

	switch merged.AuthType {
	case SSHAuthTypePassword:
		merged.PrivateKey = ""
		merged.PrivateKeyPassphrase = ""
	case SSHAuthTypePrivateKey:
		merged.Password = ""
	}

	return merged
}

// DecryptSSHTunnel returns nil when the tunnel is disabled, which is how the engines tell a direct
// connection.
func DecryptSSHTunnel(fieldEncryptor *crypto.FieldEncryptor, settings SSHTunnelSettings) (*sshtunnel.Settings, error) {
	if !settings.IsEnabled {
		return nil, nil
	}

	password, err := DecryptSecret(fieldEncryptor, settings.Password, "the SSH tunnel password")
	if err != nil {
		return nil, err
	}

	privateKey, err := DecryptSecret(fieldEncryptor, settings.PrivateKey, "the SSH tunnel private key")
	if err != nil {
		return nil, err
	}

	passphrase, err := DecryptSecret(fieldEncryptor, settings.PrivateKeyPassphrase, "the SSH tunnel passphrase")
	if err != nil {
		return nil, err
	}

	return &sshtunnel.Settings{
		Host:                 settings.Host,
		Port:                 settings.Port,
		Username:             settings.Username,
		AuthType:             sshtunnel.AuthType(settings.AuthType),
		Password:             password,
		PrivateKey:           privateKey,
		PrivateKeyPassphrase: passphrase,
	}, nil
}

// SSHTunnelColumns are the tunnel columns every provider table carries with the prefix ssh_. A row
// type embeds them with gorm:"embedded;embeddedPrefix:ssh_".
type SSHTunnelColumns struct {
	IsEnabled            bool        `gorm:"column:is_enabled"`
	Host                 string      `gorm:"column:host"`
	Port                 int         `gorm:"column:port"`
	Username             string      `gorm:"column:username"`
	AuthType             SSHAuthType `gorm:"column:auth_type"`
	Password             string      `gorm:"column:password"`
	PrivateKey           string      `gorm:"column:private_key"`
	PrivateKeyPassphrase string      `gorm:"column:private_key_passphrase"`
}

// NewSSHTunnelColumns encrypts the secrets that are not encrypted yet.
func NewSSHTunnelColumns(fieldEncryptor *crypto.FieldEncryptor, settings SSHTunnelSettings) (SSHTunnelColumns, error) {
	encryptedSecrets := make([]string, 0, 3)

	for _, secret := range []string{settings.Password, settings.PrivateKey, settings.PrivateKeyPassphrase} {
		encryptedSecret, err := fieldEncryptor.Encrypt(secret)
		if err != nil {
			return SSHTunnelColumns{}, fmt.Errorf("encrypt an SSH tunnel secret: %w", err)
		}

		encryptedSecrets = append(encryptedSecrets, encryptedSecret)
	}

	authType := settings.AuthType
	if authType == "" {
		authType = SSHAuthTypePassword
	}

	port := settings.Port
	if port == 0 {
		port = defaultSSHPort
	}

	return SSHTunnelColumns{
		IsEnabled:            settings.IsEnabled,
		Host:                 settings.Host,
		Port:                 port,
		Username:             settings.Username,
		AuthType:             authType,
		Password:             encryptedSecrets[0],
		PrivateKey:           encryptedSecrets[1],
		PrivateKeyPassphrase: encryptedSecrets[2],
	}, nil
}

func (c SSHTunnelColumns) ToSettings() SSHTunnelSettings {
	return SSHTunnelSettings(c)
}

const defaultSSHPort = 22
