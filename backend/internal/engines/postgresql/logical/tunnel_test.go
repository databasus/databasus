package logical

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/sshtunnel"
	containers "databasus-backend/internal/lib/testcontainers"
)

const unreachableBastionTimeout = 2 * time.Second

func newTunnelSettings() Settings {
	return Settings{
		Host:     "db.internal",
		HostAddr: "",
		Port:     5432,
		Username: "testuser",
		Password: "testpassword",
		Database: "testdb",
		TLS:      pgshared.TLS{Mode: pgshared.SslModeDisable},
		SSHTunnel: &sshtunnel.Settings{
			Host:                 "bastion.example.com",
			Port:                 22,
			Username:             "tunneluser",
			AuthType:             sshtunnel.AuthTypePassword,
			Password:             "tunnelpassword",
			PrivateKey:           "",
			PrivateKeyPassphrase: "",
		},
	}
}

func TestOpenTunnel(t *testing.T) {
	t.Parallel()

	t.Run("returns the same source without a local endpoint when no tunnel is set", func(t *testing.T) {
		t.Parallel()
		settings := newTunnelSettings()
		settings.SSHTunnel = nil
		source := NewSource(settings)

		tunnel, err := source.OpenTunnel(t.Context(), newTestLogger(t))
		require.NoError(t, err)

		defer tunnel.Close()

		_, hasLocalEndpoint := tunnel.GetLocalEndpoint()
		assert.Same(t, source, tunnel.GetSource())
		assert.False(t, hasLocalEndpoint)
	})

	t.Run("returns error when target is the embedded database", func(t *testing.T) {
		t.Parallel()
		settings := newTunnelSettings()
		settings.Host = "/tmp"
		settings.Port = 5437
		settings.Database = "databasus"
		settings.SSHTunnel = nil

		_, err := NewSource(settings).OpenTunnel(t.Context(), newTestLogger(t))

		require.ErrorContains(t, err, "backing up Databasus internal database is not allowed")
	})

	t.Run("returns error when bastion is unreachable", func(t *testing.T) {
		t.Parallel()
		settings := newTunnelSettings()
		// RFC 5737 TEST-NET-1: routable nowhere, so this fails to connect rather than resolving.
		settings.SSHTunnel.Host = "192.0.2.1"

		// A blackholing network would otherwise burn the forwarder's full 30s dial timeout here.
		ctx, cancel := context.WithTimeout(t.Context(), unreachableBastionTimeout)
		defer cancel()

		_, err := NewSource(settings).OpenTunnel(ctx, newTestLogger(t))

		assert.Error(t, err)
	})

	// The database publishes no port, so a successful connection proves the traffic went through
	// the tunnel. Host stays the real one: sslmode=verify-full matches the certificate against it,
	// and libpq looks up .pgpass by host and port.
	t.Run("reaches database through bastion and keeps the real host", func(t *testing.T) {
		t.Parallel()
		bastionedDatabase := containers.StartPostgresBehindSSHBastion(t, "postgres:17")
		tunnelSettings := sshtunnel.Settings{
			Host:                 bastionedDatabase.Bastion.Host,
			Port:                 bastionedDatabase.Bastion.Port,
			Username:             containers.SSHBastionUsername,
			AuthType:             sshtunnel.AuthTypePassword,
			Password:             containers.SSHBastionPassword,
			PrivateKey:           "",
			PrivateKeyPassphrase: "",
		}

		settings := Settings{
			Host:      bastionedDatabase.Database.Host,
			HostAddr:  "",
			Port:      bastionedDatabase.Database.Port,
			Username:  containers.PostgresUsername,
			Password:  containers.PostgresPassword,
			Database:  containers.PostgresDatabase,
			TLS:       pgshared.TLS{Mode: pgshared.SslModeDisable},
			SSHTunnel: &tunnelSettings,
		}
		source := NewSource(settings)

		tunnel, err := source.OpenTunnel(t.Context(), newTestLogger(t))
		require.NoError(t, err)

		defer tunnel.Close()

		localEndpoint, hasLocalEndpoint := tunnel.GetLocalEndpoint()
		require.True(t, hasLocalEndpoint)

		settingsThroughTunnel := tunnel.GetSource().settings
		assert.Equal(t, bastionedDatabase.Database.Host, settingsThroughTunnel.Host)
		assert.Equal(t, localEndpoint.Host, settingsThroughTunnel.HostAddr)
		assert.Equal(t, localEndpoint.Port, settingsThroughTunnel.Port)
		assert.Nil(t, settingsThroughTunnel.SSHTunnel, "a source through the tunnel must not open a second one")
		assert.Equal(t, settings, source.settings, "the original source must keep its settings")

		detectedVersion, err := tunnel.GetSource().DetectVersion(t.Context(), newTestLogger(t))
		require.NoError(t, err)
		assert.Equal(t, pgshared.Version("17"), detectedVersion)
	})
}
