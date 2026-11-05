package backuping_test

import (
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

func mtlsDirectory(t *testing.T) string {
	t.Helper()
	_, file, _, isResolved := runtime.Caller(0)
	require.True(t, isResolved)
	return filepath.Join(filepath.Dir(file), "../testdata/mtls")
}

func readMtlsFile(t *testing.T, name string) string {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(mtlsDirectory(t), name))
	require.NoError(t, err)
	return string(contents)
}

func TestPhysicalBackupOverMtls(t *testing.T) {
	t.Parallel()

	t.Run(
		"publishes the full and manifest after replication authentication with a client certificate",
		func(t *testing.T) {
			t.Parallel()

			team := newBackupTeam(t, backupTeamSpec{IsMtls: true})
			full := team.runBackup(t, "full")
			require.Equal(t, "COMPLETED", full.Status)
			team.assertArtifacts(t, full)
			team.assertNoRunningBackup(t)
		},
	)
	t.Run("rejects a source update when its required client certificate is removed", func(t *testing.T) {
		t.Parallel()

		team := newBackupTeam(t, backupTeamSpec{IsMtls: true})
		settings, err := team.Database.Settings.AsPostgresqlPhysicalSettings()
		require.NoError(t, err)
		settings.SslClientCert = nil
		settings.SslClientKey = nil

		var update apiclient.DatabaseSettings
		require.NoError(t, update.FromPostgresqlPhysicalSettings(settings))

		response, err := team.Owner.Client.UpdateDatabaseWithResponse(
			t.Context(),
			team.Database.Id,
			apiclient.DatabaseUpdate{Name: team.Database.Name, Settings: update},
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, response.StatusCode(), "%s", response.Body)
	})
}
