package sshtunnel

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	libsshtunnel "databasus-backend/internal/lib/sshtunnel"
)

// The real encryptor resolves its key through the secret-key service and a database; these tests
// only cover which fields get walked and which are skipped. It prefixes unconditionally, including
// the empty string, so that dropping the skip-empty guard in EncryptSensitiveFields is visible.
type prefixingEncryptor struct{}

func (prefixingEncryptor) Encrypt(plaintext string) (string, error) {
	return "enc:" + plaintext, nil
}

func (prefixingEncryptor) Decrypt(ciphertext string) (string, error) {
	return strings.TrimPrefix(ciphertext, "enc:"), nil
}

func enabledConfig() Config {
	return Config{
		IsEnabled: true,
		Host:      "bastion.example.com",
		Port:      22,
		Username:  "tunneluser",
		AuthType:  AuthTypePassword,
		Password:  "tunnelpassword",
	}
}

func enabledPrivateKeyConfig() Config {
	config := enabledConfig()
	config.AuthType = AuthTypePrivateKey
	config.Password = ""
	config.PrivateKey = "stored-private-key"
	config.PrivateKeyPassphrase = "stored-passphrase"

	return config
}

func Test_Validate_WhenTunnelIsDisabled_IgnoresEmptyFields(t *testing.T) {
	config := Config{}

	assert.NoError(t, config.Validate())
}

func Test_Validate_WhenTunnelIsEnabledAndComplete_ReturnsNoError(t *testing.T) {
	config := enabledConfig()

	assert.NoError(t, config.Validate())
}

func Test_Validate_WhenHostIsMissing_ReturnsError(t *testing.T) {
	config := enabledConfig()
	config.Host = ""

	assert.Error(t, config.Validate())
}

func Test_Validate_WhenUsernameIsMissing_ReturnsError(t *testing.T) {
	config := enabledConfig()
	config.Username = ""

	assert.Error(t, config.Validate())
}

func Test_Validate_WhenPortIsOutOfRange_ReturnsError(t *testing.T) {
	for _, port := range []int{0, -1, 65536} {
		config := enabledConfig()
		config.Port = port

		assert.Error(t, config.Validate(), "port %d must be rejected", port)
	}
}

func Test_Validate_WhenAuthTypeIsPasswordAndOnlyPrivateKeyIsSet_ReturnsError(t *testing.T) {
	config := enabledConfig()
	config.Password = ""
	config.PrivateKey = "-----BEGIN OPENSSH PRIVATE KEY-----"

	assert.Error(t, config.Validate())
}

func Test_Validate_WhenAuthTypeIsPrivateKeyAndOnlyPasswordIsSet_ReturnsError(t *testing.T) {
	config := enabledConfig()
	config.AuthType = AuthTypePrivateKey

	assert.Error(t, config.Validate())
}

func Test_Validate_WhenAuthTypeIsPrivateKeyAndTheKeyIsSet_ReturnsNoError(t *testing.T) {
	config := enabledPrivateKeyConfig()

	assert.NoError(t, config.Validate())
}

func Test_Validate_WhenAuthTypeIsBlank_ReturnsError(t *testing.T) {
	config := enabledConfig()
	config.AuthType = ""

	assert.EqualError(t, config.Validate(), "SSH tunnel auth type is required")
}

func Test_Validate_WhenAuthTypeIsUnknown_ReturnsError(t *testing.T) {
	config := enabledConfig()
	config.AuthType = "CERTIFICATE"

	assert.EqualError(t, config.Validate(), "invalid SSH tunnel auth type: CERTIFICATE")
}

// A second secret alongside the chosen one is a second way into the bastion that nothing in the UI
// would ever show again.
func Test_Validate_WhenAuthTypeIsPasswordAndAPrivateKeyIsAlsoSet_ReturnsError(t *testing.T) {
	config := enabledConfig()
	config.PrivateKey = "-----BEGIN OPENSSH PRIVATE KEY-----"

	assert.Error(t, config.Validate())
}

func Test_Validate_WhenAuthTypeIsPrivateKeyAndAPasswordIsAlsoSet_ReturnsError(t *testing.T) {
	config := enabledPrivateKeyConfig()
	config.Password = "tunnelpassword"

	assert.Error(t, config.Validate())
}

func Test_HideSensitiveData_WhenCalled_ClearsSecretsAndKeepsTheRest(t *testing.T) {
	config := enabledConfig()
	config.PrivateKey = "private-key"
	config.PrivateKeyPassphrase = "passphrase"

	config.HideSensitiveData()

	assert.Empty(t, config.Password)
	assert.Empty(t, config.PrivateKey)
	assert.Empty(t, config.PrivateKeyPassphrase)
	assert.Equal(t, "bastion.example.com", config.Host)
	assert.Equal(t, "tunneluser", config.Username)
	assert.True(t, config.IsEnabled)
}

func Test_HideSensitiveData_WhenReceiverIsNil_DoesNotPanic(t *testing.T) {
	var config *Config

	config.HideSensitiveData()
}

// The edit form never receives the stored secrets back, so it submits them blank. Overwriting on
// blank would wipe the bastion credentials on every unrelated edit.
func Test_Update_WhenAuthTypeStaysPasswordAndSecretsAreBlank_KeepsTheStoredPassword(t *testing.T) {
	storedConfig := enabledConfig()

	storedConfig.Update(&Config{
		IsEnabled: true,
		Host:      "new-bastion.example.com",
		Port:      2222,
		Username:  "newuser",
		AuthType:  AuthTypePassword,
	})

	assert.Equal(t, "tunnelpassword", storedConfig.Password)
	assert.Equal(t, "new-bastion.example.com", storedConfig.Host)
	assert.Equal(t, 2222, storedConfig.Port)
	assert.Equal(t, "newuser", storedConfig.Username)
}

func Test_Update_WhenAuthTypeStaysPrivateKeyAndSecretsAreBlank_KeepsTheStoredKeyAndPassphrase(
	t *testing.T,
) {
	storedConfig := enabledPrivateKeyConfig()

	incomingConfig := enabledPrivateKeyConfig()
	incomingConfig.PrivateKey = ""
	incomingConfig.PrivateKeyPassphrase = ""

	storedConfig.Update(&incomingConfig)

	assert.Equal(t, "stored-private-key", storedConfig.PrivateKey)
	assert.Equal(t, "stored-passphrase", storedConfig.PrivateKeyPassphrase)
}

func Test_Update_WhenAuthTypeIsPasswordAndANewPasswordArrives_ReplacesTheStoredOne(t *testing.T) {
	storedConfig := enabledConfig()

	incomingConfig := enabledConfig()
	incomingConfig.Password = "new-password"

	storedConfig.Update(&incomingConfig)

	assert.Equal(t, "new-password", storedConfig.Password)
}

func Test_Update_WhenAuthTypeIsPrivateKeyAndANewKeyArrives_ReplacesTheStoredOne(t *testing.T) {
	storedConfig := enabledPrivateKeyConfig()

	incomingConfig := enabledPrivateKeyConfig()
	incomingConfig.PrivateKey = "new-private-key"
	incomingConfig.PrivateKeyPassphrase = "new-passphrase"

	storedConfig.Update(&incomingConfig)

	assert.Equal(t, "new-private-key", storedConfig.PrivateKey)
	assert.Equal(t, "new-passphrase", storedConfig.PrivateKeyPassphrase)
}

// Leaving the previous secret behind would keep a working way into the bastion that the user
// believes they have replaced.
func Test_Update_WhenAuthTypeChangesToPassword_ClearsThePrivateKeyAndItsPassphrase(t *testing.T) {
	storedConfig := enabledPrivateKeyConfig()

	incomingConfig := enabledConfig()
	incomingConfig.Password = "new-password"

	storedConfig.Update(&incomingConfig)

	assert.Empty(t, storedConfig.PrivateKey)
	assert.Empty(t, storedConfig.PrivateKeyPassphrase)
	assert.Equal(t, "new-password", storedConfig.Password)
}

func Test_Update_WhenAuthTypeChangesToPrivateKey_ClearsThePassword(t *testing.T) {
	storedConfig := enabledConfig()

	incomingConfig := enabledPrivateKeyConfig()

	storedConfig.Update(&incomingConfig)

	assert.Empty(t, storedConfig.Password)
	assert.Equal(t, "stored-private-key", storedConfig.PrivateKey)
}

// A caller that only flips the tunnel off must not silently switch how it logs back in, which
// would take the stored key down with it.
func Test_Update_WhenAuthTypeIsBlank_KeepsTheStoredOneAndItsSecrets(t *testing.T) {
	storedConfig := enabledPrivateKeyConfig()

	storedConfig.Update(&Config{IsEnabled: false})

	assert.Equal(t, AuthTypePrivateKey, storedConfig.AuthType)
	assert.Equal(t, "stored-private-key", storedConfig.PrivateKey)
	assert.Equal(t, "stored-passphrase", storedConfig.PrivateKeyPassphrase)
}

func Test_Update_WhenTunnelIsDisabledInIncoming_TurnsItOff(t *testing.T) {
	storedConfig := enabledConfig()

	incomingConfig := enabledConfig()
	incomingConfig.IsEnabled = false
	incomingConfig.Password = ""

	storedConfig.Update(&incomingConfig)

	assert.False(t, storedConfig.IsEnabled)
	assert.Equal(t, "tunnelpassword", storedConfig.Password)
}

func Test_EncryptSensitiveFields_WhenCalled_EncryptsSecretsAndSkipsEmptyOnes(t *testing.T) {
	encryptor := prefixingEncryptor{}

	config := enabledConfig()
	config.PrivateKey = "private-key"

	require.NoError(t, config.EncryptSensitiveFields(encryptor))

	assert.NotEqual(t, "tunnelpassword", config.Password)
	assert.NotEqual(t, "private-key", config.PrivateKey)
	assert.Empty(t, config.PrivateKeyPassphrase)

	decryptedPassword, err := encryptor.Decrypt(config.Password)
	require.NoError(t, err)
	assert.Equal(t, "tunnelpassword", decryptedPassword)

	decryptedPrivateKey, err := encryptor.Decrypt(config.PrivateKey)
	require.NoError(t, err)
	assert.Equal(t, "private-key", decryptedPrivateKey)
}

// Every tunnel test of lib/sshtunnel passes clear text, but a persisted database always holds
// ciphertext. Handing it straight to the tunnel would leave those tests green and the feature dead.
func Test_DecryptSettings_WithEncryptedSecrets_ReturnsThemInClearText(t *testing.T) {
	encryptor := prefixingEncryptor{}

	config := enabledPrivateKeyConfig()
	require.NoError(t, config.EncryptSensitiveFields(encryptor))
	require.NotEqual(t, "stored-passphrase", config.PrivateKeyPassphrase)

	settings, err := config.DecryptSettings(encryptor)

	require.NoError(t, err)
	assert.Equal(t, libsshtunnel.AuthTypePrivateKey, settings.AuthType)
	assert.Equal(t, "stored-private-key", settings.PrivateKey)
	assert.Equal(t, "stored-passphrase", settings.PrivateKeyPassphrase)
	assert.Equal(t, "bastion.example.com", settings.Host)
}

func Test_DecryptSettings_WithoutEncryptor_PassesSecretsThrough(t *testing.T) {
	config := enabledConfig()

	settings, err := config.DecryptSettings(nil)

	require.NoError(t, err)
	assert.Equal(t, libsshtunnel.AuthTypePassword, settings.AuthType)
	assert.Equal(t, "tunnelpassword", settings.Password)
}
