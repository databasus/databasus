package mongodb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/sshtunnel"
	containers "databasus-backend/internal/lib/testcontainers"
)

// TestOpenTunnel is not parallel, so its server never runs beside the version matrix's one.
func TestOpenTunnel(t *testing.T) {
	t.Run("returns the source itself without a tunnel", func(t *testing.T) {
		source := NewSource(newURISettings())

		tunnel, err := source.OpenTunnel(t.Context(), newTestLogger())
		require.NoError(t, err)

		defer tunnel.Close()

		_, isTunneled := tunnel.GetLocalEndpoint()

		assert.Same(t, source, tunnel.GetSource())
		assert.False(t, isTunneled)
	})

	t.Run("returns an error with a tunnel and without a port", func(t *testing.T) {
		settings := newURISettings()
		settings.Port = 0
		settings.SSHTunnel = &sshtunnel.Settings{
			Host:                 "bastion.example.com",
			Port:                 22,
			Username:             "tunneluser",
			AuthType:             sshtunnel.AuthTypePassword,
			Password:             "tunnelpassword",
			PrivateKey:           "",
			PrivateKeyPassphrase: "",
		}

		_, err := NewSource(settings).OpenTunnel(t.Context(), newTestLogger())

		require.ErrorIs(t, err, ErrTunnelPortRequired)
	})

	t.Run("detects the version through the bastion with a direct connection to the local end", func(t *testing.T) {
		topology := containers.StartMongodbBehindSSHBastion(t, "mongo:7.0")
		settings := Settings{
			Host:               topology.Database.Host,
			Port:               topology.Database.Port,
			Username:           containers.MongodbUsername,
			Password:           containers.MongodbPassword,
			Database:           containers.MongodbDatabase,
			AuthDatabase:       containers.MongodbAuthDatabase,
			IsTLS:              false,
			IsSrv:              false,
			IsDirectConnection: false,
			SSHTunnel: &sshtunnel.Settings{
				Host:                 topology.Bastion.Host,
				Port:                 topology.Bastion.Port,
				Username:             containers.SSHBastionUsername,
				AuthType:             sshtunnel.AuthTypePassword,
				Password:             containers.SSHBastionPassword,
				PrivateKey:           "",
				PrivateKeyPassphrase: "",
			},
		}

		tunnel, err := NewSource(settings).OpenTunnel(t.Context(), newTestLogger())
		require.NoError(t, err)

		defer tunnel.Close()

		localEndpoint, isTunneled := tunnel.GetLocalEndpoint()
		require.True(t, isTunneled)

		settingsThroughTunnel := tunnel.GetSource().settings
		assert.Equal(t, localEndpoint.Host, settingsThroughTunnel.Host)
		assert.Equal(t, localEndpoint.Port, settingsThroughTunnel.Port)
		assert.True(t, settingsThroughTunnel.IsDirectConnection)
		assert.Nil(t, settingsThroughTunnel.SSHTunnel)

		detectedVersion, err := tunnel.GetSource().DetectVersion(t.Context(), newTestLogger())

		require.NoError(t, err)
		assert.Equal(t, Version7, detectedVersion)
	})
}
