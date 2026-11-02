package physical

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/sshtunnel"
	containers "databasus-backend/internal/lib/testcontainers"
)

const unreachableBastionTimeout = 2 * time.Second

func TestOpenTunnel(t *testing.T) {
	t.Parallel()

	t.Run("returns the source itself without a tunnel", func(t *testing.T) {
		t.Parallel()
		source := NewSource(newUnreachableSettings())

		tunnel, err := source.OpenTunnel(t.Context(), newTestLogger(t))
		require.NoError(t, err)

		defer tunnel.Close()

		_, isTunneled := tunnel.GetLocalEndpoint()

		assert.Same(t, source, tunnel.GetSource())
		assert.False(t, isTunneled)
		assert.True(t, tunnel.IsBastionReachable(t.Context()),
			"with no bastion in the path a failure is the source's own")
	})

	t.Run("returns embedded target error when settings point at the internal cluster", func(t *testing.T) {
		t.Parallel()
		settings := newUnreachableSettings()
		settings.Host = "/tmp"
		settings.Port = 5437

		_, err := NewSource(settings).OpenTunnel(t.Context(), newTestLogger(t))

		require.ErrorContains(t, err, "backing up Databasus internal PostgreSQL cluster is not allowed")
	})

	t.Run("returns error when the bastion is unreachable", func(t *testing.T) {
		t.Parallel()
		settings := newUnreachableSettings()
		// RFC 5737 TEST-NET-1: routable nowhere, so this fails to connect rather than resolving.
		settings.SSHTunnel = newBastionSettings(containers.Endpoint{Host: "192.0.2.1", Port: 22})

		// A blackholing network would otherwise burn the forwarder's full dial timeout here.
		ctx, cancel := context.WithTimeout(t.Context(), unreachableBastionTimeout)
		defer cancel()

		_, err := NewSource(settings).OpenTunnel(ctx, newTestLogger(t))

		assert.Error(t, err)
	})

	for _, image := range getSupportedVersionImages() {
		// PostgreSQL keeps the real host through a tunnel: sslmode=verify-full matches the
		// certificate against it, and libpq looks up .pgpass by host and port.
		t.Run("inspects the cluster through the bastion with the real host kept on "+image, func(t *testing.T) {
			topology := containers.StartPhysicalPostgresBehindSSHBastion(t, image)
			settings := newUnreachableSettings()
			settings.Host = topology.Database.Host
			settings.Port = topology.Database.Port
			settings.SSHTunnel = newBastionSettings(topology.Bastion)

			tunnel, err := NewSource(settings).OpenTunnel(t.Context(), newTestLogger(t))
			require.NoError(t, err)

			defer tunnel.Close()

			localEndpoint, isTunneled := tunnel.GetLocalEndpoint()
			require.True(t, isTunneled)

			settingsThroughTunnel := tunnel.GetSource().settings
			assert.Equal(t, topology.Database.Host, settingsThroughTunnel.Host)
			assert.Equal(t, localEndpoint.Host, settingsThroughTunnel.HostAddr)
			assert.Equal(t, localEndpoint.Port, settingsThroughTunnel.Port)
			assert.Nil(t, settingsThroughTunnel.SSHTunnel)
			assert.True(t, tunnel.IsBastionReachable(t.Context()))

			clusterIdentity, err := tunnel.GetSource().InspectCluster(t.Context(), newTestLogger(t))

			require.NoError(t, err)
			assert.Equal(
				t,
				pgshared.Version(strconv.Itoa(containers.PostgresMajorVersion(image))),
				clusterIdentity.Version,
			)
		})
	}
}

func newUnreachableSettings() Settings {
	return Settings{
		Host:     "cluster.internal",
		HostAddr: "",
		Port:     5432,
		Username: getAdminRole().Name,
		Password: getAdminRole().Password,
		TLS: pgshared.TLS{
			Mode:       pgshared.SslModeDisable,
			ClientCert: "",
			ClientKey:  "",
			RootCert:   "",
		},
		SSHTunnel: nil,
	}
}

func newBastionSettings(bastion containers.Endpoint) *sshtunnel.Settings {
	return &sshtunnel.Settings{
		Host:                 bastion.Host,
		Port:                 bastion.Port,
		Username:             containers.SSHBastionUsername,
		AuthType:             sshtunnel.AuthTypePassword,
		Password:             containers.SSHBastionPassword,
		PrivateKey:           "",
		PrivateKeyPassphrase: "",
	}
}
