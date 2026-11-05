package mariadb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseClientVersion(t *testing.T) {
	t.Parallel()

	t.Run("reads the server line from both output shapes", func(t *testing.T) {
		t.Parallel()
		outputCases := []struct {
			output, expectedVersion string
		}{
			// The 10.6 client prints its protocol version first; 10.19 is not a MariaDB release.
			{"mariadb-dump  Ver 10.19 Distrib 10.6.28-MariaDB, for debian-linux-gnu (x86_64)", "10.6.28"},
			{"mariadb-dump from 12.3.3-MariaDB, client 10.20 for debian-linux-gnu (x86_64)", "12.3.3"},
			{"mariadb from 13.0.2-MariaDB, client 10.20 for debian-linux-gnu (x86_64)", "13.0.2"},
		}

		for _, outputCase := range outputCases {
			reportedVersion, err := parseClientVersion(outputCase.output)

			require.NoError(t, err)
			assert.Equal(t, outputCase.expectedVersion, reportedVersion)
		}
	})

	t.Run("returns an error when the output carries no version", func(t *testing.T) {
		t.Parallel()

		_, err := parseClientVersion("mariadb-dump: command not found")

		require.ErrorIs(t, err, ErrUnreadableClientVersion)
	})
}

func TestGetClientVersionForServer(t *testing.T) {
	t.Parallel()

	t.Run("picks the designated client across the three tiers", func(t *testing.T) {
		t.Parallel()
		tierCases := []struct {
			serverVersion Version
			clientVersion ClientVersion
		}{
			{Version55, ClientLegacy},
			{Version101, ClientLegacy},
			{Version102, ClientModern},
			{Version1011, ClientModern},
			{Version120, ClientModern},
			{Version130, Client13},
		}

		for _, tierCase := range tierCases {
			assert.Equalf(t, tierCase.clientVersion, GetClientVersionForServer(tierCase.serverVersion),
				"server %s", tierCase.serverVersion)
		}
	})
}

func TestIsBackupVersionHigherThanRestoreVersion(t *testing.T) {
	t.Parallel()

	t.Run("refuses downgrades across the 13 line", func(t *testing.T) {
		t.Parallel()
		restoreCases := []struct {
			backupVersion, restoreVersion Version
			isDowngrade                   bool
		}{
			{Version130, Version120, true},
			{Version130, Version118, true},
			{Version130, Version130, false},
			{Version120, Version130, false},
			{Version1011, Version106, true},
			{Version106, Version1011, false},
		}

		for _, restoreCase := range restoreCases {
			isDowngrade, err := IsBackupVersionHigherThanRestoreVersion(
				restoreCase.backupVersion,
				restoreCase.restoreVersion,
			)

			require.NoError(t, err)
			assert.Equalf(t, restoreCase.isDowngrade, isDowngrade,
				"%s onto %s", restoreCase.backupVersion, restoreCase.restoreVersion)
		}
	})

	t.Run("returns an error with a version it cannot order", func(t *testing.T) {
		t.Parallel()

		_, err := IsBackupVersionHigherThanRestoreVersion("", Version130)

		require.Error(t, err)
	})
}
