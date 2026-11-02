package backups_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest/enginetest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

func TestGetLatestCompletedLogicalBackups(t *testing.T) {
	t.Parallel()

	t.Run("returns the latest completed backup per database when newer backups have failed", func(t *testing.T) {
		t.Parallel()

		tm := enginetest.NewTeam(t)
		server := containers.StartPostgres(t, "postgres:18")
		database := tm.CreateDatabase(t, apiclient.PostgresqlLogicalSettings{
			Type: apiclient.PostgresqlLogicalSettingsTypePOSTGRESLOGICAL,
			Host: server.Host, Port: server.Port, Username: containers.PostgresUsername,
			Password: new(containers.PostgresPassword), Database: new(containers.PostgresDatabase),
			SslMode: apiclient.PostgresqlSslModeDisable, SshTunnel: enginetest.NewDisabledSSHTunnelSettings(),
			CpuCount: 1, SslClientCert: nil, SslClientKey: nil, SslRootCert: nil,
			IncludeSchemas: nil, ExcludeTables: nil, ExcludeTableData: nil, IsSkipUserMappings: false, Version: nil,
		})

		firstBackup := tm.BackUp(t, database.Id, apiclient.BackupEncryptionNONE)
		tm.App.Clock.Advance(time.Minute)
		latestBackup := tm.BackUp(t, database.Id, apiclient.BackupEncryptionNONE)
		tm.App.ExecuteSQL(t, "UPDATE databases_postgresql_logical SET port = 1 WHERE database_id = ?", database.Id)
		tm.App.Clock.Advance(time.Minute)

		started, err := tm.Owner.Client.StartLogicalBackupWithResponse(
			t.Context(),
			apiclient.LogicalBackupStart{DatabaseId: database.Id},
		)
		require.NoError(t, err)
		require.NotNil(t, started.JSON202)
		failedBackup := tm.WaitForBackupOutcome(t, started.JSON202.Id)
		require.Equal(t, apiclient.LogicalBackupStatusFAILED, failedBackup.Status)

		emptyDatabaseID := tm.App.InsertDatabase(t, tm.Workspace.Id)
		completedBackups := tm.App.GetLatestCompletedLogicalBackups(
			t,
			[]uuid.UUID{database.Id, emptyDatabaseID, uuid.New()},
		)

		require.Len(t, completedBackups, 1)
		assert.Equal(t, latestBackup.Id, completedBackups[database.Id].ID)
		assert.NotEqual(t, firstBackup.Id, completedBackups[database.Id].ID)
		assert.Empty(t, tm.App.GetLatestCompletedLogicalBackups(t, nil))
	})
}
