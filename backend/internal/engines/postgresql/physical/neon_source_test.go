package physical

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	containers "databasus-backend/internal/lib/testcontainers"
)

func TestOpenTunnelForNeonSource(t *testing.T) {
	t.Parallel()

	neonHosts := []string{
		"ep-young-dew-123456.c-8.us-east-2.aws.neon.tech",
		"ep-young-dew-123456-pooler.c-8.us-east-2.aws.neon.tech",
		"EP-YOUNG-DEW-123456.US-EAST-2.AWS.NEON.TECH.",
		"neon.tech",
	}

	for _, neonHost := range neonHosts {
		t.Run("refuses physical replication for "+neonHost, func(t *testing.T) {
			t.Parallel()
			settings := newUnreachableSettings()
			settings.Host = neonHost

			_, err := NewSource(settings).OpenTunnel(t.Context(), newTestLogger(t))

			require.Error(t, err)
			assert.Equal(t, pgshared.ConnErrPhysicalReplicationUnsupported, getConnectionTestErrorCode(t, err))
			assert.Contains(t, err.Error(), "Neon supports only logical backups")
		})
	}

	t.Run("refuses a Neon source behind an SSH tunnel without dialing the bastion", func(t *testing.T) {
		t.Parallel()
		settings := newUnreachableSettings()
		settings.Host = "ep-young-dew-123456.c-8.us-east-2.aws.neon.tech"
		// RFC 5737 TEST-NET-1: a dial attempt would fail with a different error or hit the deadline.
		settings.SSHTunnel = newBastionSettings(containers.Endpoint{Host: "192.0.2.1", Port: 22})

		ctx, cancel := context.WithTimeout(t.Context(), unreachableBastionTimeout)
		defer cancel()

		_, err := NewSource(settings).OpenTunnel(ctx, newTestLogger(t))

		require.Error(t, err)
		assert.Equal(t, pgshared.ConnErrPhysicalReplicationUnsupported, getConnectionTestErrorCode(t, err))
		assert.NoError(t, ctx.Err(), "the refusal must come before any dial")
	})

	lookAlikeHosts := []string{"neon.tech.example.com", "myneon.tech", "ep-young-dew-123456.neon.technology"}

	for _, lookAlikeHost := range lookAlikeHosts {
		t.Run("handles "+lookAlikeHost+" like any other host", func(t *testing.T) {
			t.Parallel()
			settings := newUnreachableSettings()
			settings.Host = lookAlikeHost

			tunnel, err := NewSource(settings).OpenTunnel(t.Context(), newTestLogger(t))
			require.NoError(t, err)

			defer tunnel.Close()
		})
	}
}
