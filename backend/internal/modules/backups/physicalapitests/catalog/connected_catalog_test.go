package catalog_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

func newConnectedCatalog(t *testing.T) *catalogFixture {
	t.Helper()
	source := containers.StartPhysicalPostgres(t, "postgres:17")
	testApp := apptest.New(t)
	owner := testApp.CreateAdmin(t)
	workspace := apptest.CreateWorkspace(t, owner, "Connected physical catalog")
	storageID := testApp.CreateLocalStorage(t, owner, workspace.Id)

	password := "testpassword"
	var settings apiclient.DatabaseSettings
	require.NoError(
		t,
		settings.FromPostgresqlPhysicalSettings(
			apiclient.PostgresqlPhysicalSettings{
				Type:       apiclient.PostgresqlPhysicalSettingsTypePOSTGRESPHYSICAL,
				Host:       source.Host,
				Port:       source.Port,
				Username:   "testuser",
				Password:   &password,
				SslMode:    apiclient.PostgresqlSslModeDisable,
				SshTunnel:  apiclient.SshTunnelSettings{IsEnabled: false},
				BackupType: apiclient.PhysicalBackupTypeFULLINCREMENTALWALSTREAM,
			},
		),
	)

	database := apptest.CreateDatabase(t, owner, workspace.Id, settings)
	return &catalogFixture{
		App:          testApp,
		Owner:        owner,
		Admin:        owner,
		WorkspaceID:  workspace.Id,
		StorageID:    storageID,
		DatabaseID:   database.Id,
		DatabaseName: database.Name,
	}
}

func TestStopWalStreamingAfterBackupTypeChange(t *testing.T) {
	t.Parallel()

	t.Run("deletes the streamer and keeps the running full when the database stops streaming WAL", func(t *testing.T) {
		t.Parallel()

		catalog := newConnectedCatalog(t)
		catalog.saveStorageConfig(t, catalog.StorageID)
		catalog.seedRunningBackupAndStreamer(t)

		response, err := catalog.Owner.Client.GetDatabaseWithResponse(t.Context(), catalog.DatabaseID)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, response.StatusCode())
		database := response.JSON200
		physicalSettings, err := database.Settings.AsPostgresqlPhysicalSettings()
		require.NoError(t, err)
		physicalSettings.BackupType = apiclient.PhysicalBackupTypeFULLINCREMENTAL
		var settings apiclient.DatabaseSettings
		require.NoError(t, settings.FromPostgresqlPhysicalSettings(physicalSettings))

		updated, err := catalog.Owner.Client.UpdateDatabaseWithResponse(
			t.Context(),
			catalog.DatabaseID,
			apiclient.DatabaseUpdate{Name: database.Name, NotifierIds: database.NotifierIds, Settings: settings},
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, updated.StatusCode(), "%s", updated.Body)

		catalog.assertStreamerAbsent(t)
		assert.NotNil(t, catalog.getRunningBackupID(t))
	})
}
