package walstreaming_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func (team *walTeam) addDatabase(t *testing.T) *walTeam {
	t.Helper()
	settings, err := team.Database.Settings.AsPostgresqlPhysicalSettings()
	require.NoError(t, err)
	settings.Password = new("testpassword")

	var databaseSettings apiclient.DatabaseSettings
	require.NoError(t, databaseSettings.FromPostgresqlPhysicalSettings(settings))
	database := apptest.CreateDatabase(t, team.Owner, team.Workspace.Id, databaseSettings)

	second := *team
	second.Database = database
	return &second
}

func TestIndependentPhysicalWalStreams(t *testing.T) {
	t.Parallel()

	t.Run("archives both databases into independent catalog rows and storage keys on one source", func(t *testing.T) {
		t.Parallel()

		first := newWalTeam(t, walTeamSpec{})
		second := first.addDatabase(t)

		first.enable(t)
		second.enable(t)
		first.waitForReceiver(t)
		second.waitForReceiver(t)

		source := first.openSource(t)
		for range 3 {
			rotateWal(t, source)
		}
		first.waitForWal(t, 1)
		second.waitForWal(t, 1)

		var databaseIDs []string
		first.App.QuerySQL(
			t,
			&databaseIDs,
			"SELECT DISTINCT database_id FROM backups_physical_wal_segments WHERE file_name IS NOT NULL",
		)
		require.ElementsMatch(t, []string{first.Database.Id.String(), second.Database.Id.String()}, databaseIDs)
	})
}
