package mariadb

import (
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
			{"5.5.68-MariaDB", Version55},
			{"10.1.48-MariaDB", Version101},
			{"10.11.14-MariaDB", Version1011},
			{"12.0.2-MariaDB", Version120},
			{"13.0.2-MariaDB-1:13.0.2+maria~ubu2204", Version130},
			// A minor later than the client we ship, still inside the 13 line.
			{"13.2.0-MariaDB", Version130},
		}

		for _, serverCase := range serverCases {
			version, err := source.parseServerVersion(serverCase.reportedVersion)

			require.NoError(t, err, serverCase.reportedVersion)
			assert.Equal(t, serverCase.expectedVersion, version, serverCase.reportedVersion)
		}
	})

	t.Run("names the supported versions with a line no client serves", func(t *testing.T) {
		t.Parallel()

		_, err := NewSource(Settings{}, getClientTools(t)).parseServerVersion("14.1.0-MariaDB")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "supported")
	})

	t.Run("directs to the MySQL type with a MySQL server", func(t *testing.T) {
		t.Parallel()

		_, err := NewSource(Settings{}, getClientTools(t)).parseServerVersion("8.0.46")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "MySQL")
	})
}
