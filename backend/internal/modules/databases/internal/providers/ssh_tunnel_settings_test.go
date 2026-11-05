package providers_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/modules/databases/internal/providers"
	"databasus-backend/internal/modules/databases/internal/providers/providertest"
)

func TestValidateSshTunnel(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		change       func(settings *providers.SSHTunnelSettings)
		refusedField string
	}{
		{name: "accepts complete password settings", change: func(*providers.SSHTunnelSettings) {}},
		{
			name:   "accepts empty settings when the tunnel is disabled",
			change: func(settings *providers.SSHTunnelSettings) { *settings = providers.SSHTunnelSettings{} },
		},
		{
			name:         "refuses settings without a host",
			change:       func(settings *providers.SSHTunnelSettings) { settings.Host = "" },
			refusedField: "settings.sshTunnel.host",
		},
		{
			name:         "refuses settings with a port out of range",
			change:       func(settings *providers.SSHTunnelSettings) { settings.Port = 65536 },
			refusedField: "settings.sshTunnel.port",
		},
		{
			name:         "refuses settings without a username",
			change:       func(settings *providers.SSHTunnelSettings) { settings.Username = "" },
			refusedField: "settings.sshTunnel.username",
		},
		{
			name:         "refuses password authentication without a password",
			change:       func(settings *providers.SSHTunnelSettings) { settings.Password = "" },
			refusedField: "settings.sshTunnel.password",
		},
		{
			name:         "refuses password authentication that also carries a private key",
			change:       func(settings *providers.SSHTunnelSettings) { settings.PrivateKey = "private-key" },
			refusedField: "settings.sshTunnel.privateKey",
		},
		{
			name: "refuses private key authentication that also carries a password",
			change: func(settings *providers.SSHTunnelSettings) {
				settings.AuthType = providers.SSHAuthTypePrivateKey
				settings.PrivateKey = "private-key"
			},
			refusedField: "settings.sshTunnel.password",
		},
		{
			name:         "refuses an unknown authentication type",
			change:       func(settings *providers.SSHTunnelSettings) { settings.AuthType = "CERTIFICATE" },
			refusedField: "settings.sshTunnel.authType",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			tunnelSettings := newPasswordTunnelSettings()
			testCase.change(&tunnelSettings)

			err := providers.ValidateSSHTunnel(tunnelSettings)

			if testCase.refusedField == "" {
				require.NoError(t, err)

				return
			}

			assert.Equal(t, testCase.refusedField, providertest.GetRefusedField(t, err))
		})
	}
}

func TestMergeStoredSshTunnel(t *testing.T) {
	t.Parallel()

	t.Run("keeps the stored password when the incoming settings leave it out", func(t *testing.T) {
		t.Parallel()
		incomingSettings := newPasswordTunnelSettings()
		incomingSettings.Password = ""
		incomingSettings.Username = "new-user"

		mergedSettings := providers.MergeStoredSSHTunnel(incomingSettings, newPasswordTunnelSettings())

		assert.Equal(t, "stored-password", mergedSettings.Password)
		assert.Equal(t, "new-user", mergedSettings.Username)
	})

	t.Run("leaves the password empty when the tunnel moves", func(t *testing.T) {
		t.Parallel()
		incomingSettings := newPasswordTunnelSettings()
		incomingSettings.Password = ""
		incomingSettings.Host = "new-bastion.example.com"

		mergedSettings := providers.MergeStoredSSHTunnel(incomingSettings, newPasswordTunnelSettings())

		assert.Empty(t, mergedSettings.Password)
		assert.Equal(t, "new-bastion.example.com", mergedSettings.Host)
	})

	t.Run("removes the stored passphrase when a moved tunnel gets a new key without one", func(t *testing.T) {
		t.Parallel()
		incomingSettings := newPrivateKeyTunnelSettings()
		incomingSettings.Port = 2200
		incomingSettings.PrivateKey = "new-private-key"
		incomingSettings.PrivateKeyPassphrase = ""

		mergedSettings := providers.MergeStoredSSHTunnel(incomingSettings, newPrivateKeyTunnelSettings())

		assert.Equal(t, "new-private-key", mergedSettings.PrivateKey)
		assert.Empty(t, mergedSettings.PrivateKeyPassphrase)
	})

	t.Run("removes the stored secrets when the tunnel is turned off", func(t *testing.T) {
		t.Parallel()
		incomingSettings := newPrivateKeyTunnelSettings()
		incomingSettings.IsEnabled = false
		incomingSettings.PrivateKey = ""
		incomingSettings.PrivateKeyPassphrase = ""

		mergedSettings := providers.MergeStoredSSHTunnel(incomingSettings, newPrivateKeyTunnelSettings())

		assert.Empty(t, mergedSettings.PrivateKey)
		assert.Empty(t, mergedSettings.PrivateKeyPassphrase)
	})

	t.Run("keeps the stored authentication type and its secrets when the incoming type is blank", func(t *testing.T) {
		t.Parallel()
		storedSettings := newPrivateKeyTunnelSettings()
		storedSettings.IsEnabled = false

		mergedSettings := providers.MergeStoredSSHTunnel(providers.SSHTunnelSettings{IsEnabled: false}, storedSettings)

		assert.Equal(t, providers.SSHAuthTypePrivateKey, mergedSettings.AuthType)
		assert.Equal(t, "stored-private-key", mergedSettings.PrivateKey)
		assert.Equal(t, "stored-passphrase", mergedSettings.PrivateKeyPassphrase)
	})

	t.Run("drops the private key and its passphrase after a switch to password", func(t *testing.T) {
		t.Parallel()
		incomingSettings := newPasswordTunnelSettings()
		incomingSettings.Password = "new-password"

		mergedSettings := providers.MergeStoredSSHTunnel(incomingSettings, newPrivateKeyTunnelSettings())

		assert.Equal(t, "new-password", mergedSettings.Password)
		assert.Empty(t, mergedSettings.PrivateKey)
		assert.Empty(t, mergedSettings.PrivateKeyPassphrase)
	})

	t.Run("drops the password after a switch to a private key", func(t *testing.T) {
		t.Parallel()

		mergedSettings := providers.MergeStoredSSHTunnel(newPrivateKeyTunnelSettings(), newPasswordTunnelSettings())

		assert.Empty(t, mergedSettings.Password)
		assert.Equal(t, "stored-private-key", mergedSettings.PrivateKey)
	})
}

func TestDecryptSshTunnel(t *testing.T) {
	t.Parallel()

	t.Run("returns the secrets in clear text when they were stored encrypted", func(t *testing.T) {
		t.Parallel()
		fieldEncryptor := providertest.NewFieldEncryptor(t)
		storedColumns, err := providers.NewSSHTunnelColumns(fieldEncryptor, newPrivateKeyTunnelSettings())
		require.NoError(t, err)
		require.NotEqual(t, "stored-private-key", storedColumns.PrivateKey)

		tunnelSettings, err := providers.DecryptSSHTunnel(fieldEncryptor, storedColumns.ToSettings())

		require.NoError(t, err)
		require.NotNil(t, tunnelSettings)
		assert.Equal(t, "stored-private-key", tunnelSettings.PrivateKey)
		assert.Equal(t, "stored-passphrase", tunnelSettings.PrivateKeyPassphrase)
	})

	t.Run("returns no tunnel when the tunnel is disabled", func(t *testing.T) {
		t.Parallel()
		tunnelSettings := newPasswordTunnelSettings()
		tunnelSettings.IsEnabled = false

		decryptedSettings, err := providers.DecryptSSHTunnel(providertest.NewFieldEncryptor(t), tunnelSettings)

		require.NoError(t, err)
		assert.Nil(t, decryptedSettings)
	})
}

func newPasswordTunnelSettings() providers.SSHTunnelSettings {
	return providers.SSHTunnelSettings{
		IsEnabled: true,
		Host:      "bastion.example.com",
		Port:      22,
		Username:  "tunnel-user",
		AuthType:  providers.SSHAuthTypePassword,
		Password:  "stored-password",
	}
}

func newPrivateKeyTunnelSettings() providers.SSHTunnelSettings {
	return providers.SSHTunnelSettings{
		IsEnabled:            true,
		Host:                 "bastion.example.com",
		Port:                 22,
		Username:             "tunnel-user",
		AuthType:             providers.SSHAuthTypePrivateKey,
		PrivateKey:           "stored-private-key",
		PrivateKeyPassphrase: "stored-passphrase",
	}
}
