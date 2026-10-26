package mysql

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseServerVersion(t *testing.T) {
	t.Parallel()

	t.Run("returns the release line with every supported server", func(t *testing.T) {
		t.Parallel()
		source := NewSource(Settings{}, getClientTools(t))
		serverCases := []struct {
			reportedVersion string
			expectedVersion Version
		}{
			{"8.0.46", Version80},
			{"8.4.11", Version84},
			{"9.7.2", Version9},
			{"26.7.0", Version26},
			// A release later than the client we ship, still inside the line.
			{"26.10.1", Version26},
		}

		for _, serverCase := range serverCases {
			version, err := source.parseServerVersion(serverCase.reportedVersion)

			require.NoError(t, err, serverCase.reportedVersion)
			assert.Equal(t, serverCase.expectedVersion, version, serverCase.reportedVersion)
		}
	})

	// Issue #786: Oracle MySQL HeatWave answers VERSION() with a vendor suffix.
	t.Run("accepts the server with a vendor suffix", func(t *testing.T) {
		t.Parallel()

		version, err := NewSource(Settings{}, getClientTools(t)).parseServerVersion("26.7.0-cloud")

		require.NoError(t, err)
		assert.Equal(t, Version26, version)
	})

	t.Run("names the supported versions with a line no client serves", func(t *testing.T) {
		t.Parallel()

		_, err := NewSource(Settings{}, getClientTools(t)).parseServerVersion("27.1.0")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "supported")
	})

	t.Run("reports the value received with an unreadable version", func(t *testing.T) {
		t.Parallel()

		_, err := NewSource(Settings{}, getClientTools(t)).parseServerVersion("unknown")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown")
	})

	t.Run("names the architecture when it ships no client for the line", func(t *testing.T) {
		t.Parallel()

		if runtime.GOARCH != "arm64" {
			t.Skip("MySQL 5.7 ships a client on amd64; only arm64 refuses it")
		}

		_, err := NewSource(Settings{}, getClientTools(t)).parseServerVersion("5.7.44")

		require.ErrorIs(t, err, ErrNoClientForArchitecture)
		assert.Contains(t, err.Error(), "arm")
	})
}
