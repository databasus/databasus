package databases_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/modules/databases"
)

//nolint:tparallel // The subtests share one source server.
func TestCopyDatabase(t *testing.T) {
	t.Parallel()

	server := containers.StartPostgres(t, "postgres:16")

	t.Run("creates a logical copy with the settings, notifiers and backup config but no backup history",
		func(t *testing.T) {
			tm := newTeam(t)
			attachedNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id, newWebhookNotifierSettings(t))
			logical := newLogicalSettings(server)
			logical.ExcludeTables = []string{"audit_tmp"}
			originalDatabase := createDatabase(t, tm.owner, apiclient.DatabaseCreate{
				WorkspaceId: tm.workspace.Id,
				Name:        "Orders",
				NotifierIds: []uuid.UUID{attachedNotifier.Id},
				Settings:    fromLogicalSettings(t, logical),
			})
			storageID := tm.app.CreateLocalStorage(t, tm.owner, tm.workspace.Id)
			apptest.SaveBackupConfigWithStorage(t, tm.owner, apptest.StorageUse{
				DatabaseID: originalDatabase.Id,
				StorageID:  storageID,
			})

			require.NoError(t, tm.app.RecordBackupSuccess(
				t.Context(), originalDatabase.Id, time.Now().UTC()))
			unavailable := databases.HealthStatusUnavailable
			require.NoError(t, tm.app.SetHealthStatus(
				t.Context(), originalDatabase.Id, &unavailable))

			copiedDatabase := copyDatabase(t, tm.owner, originalDatabase.Id)

			assert.NotEqual(t, originalDatabase.Id, copiedDatabase.Id)
			assert.Equal(t, "Orders (Copy)", copiedDatabase.Name)
			assert.Equal(t, tm.workspace.Id, copiedDatabase.WorkspaceId)
			assert.Equal(t, []uuid.UUID{attachedNotifier.Id}, copiedDatabase.NotifierIds)
			assert.Equal(t, getLogicalSettings(t, originalDatabase), getLogicalSettings(t, copiedDatabase))
			assert.Nil(t, copiedDatabase.LastBackupTime)
			assert.Nil(t, copiedDatabase.LastBackupErrorMessage)
			assert.Equal(t, new(apiclient.DatabaseHealthStatusUNAVAILABLE), copiedDatabase.HealthStatus)
			assert.ElementsMatch(t, []string{"Orders", "Orders (Copy)"},
				getListedDatabaseNames(t, tm.owner, tm.workspace.Id))
			assert.Equal(t, &storageID, apptest.GetBackupConfig(t, tm.owner, copiedDatabase.Id).StorageId)

			testResponse, err := tm.owner.Client.TestDatabaseWithResponse(t.Context(), copiedDatabase.Id)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, testResponse.StatusCode(), string(testResponse.Body))
			assert.True(t, testResponse.JSON200.IsConnected, string(testResponse.Body))
		})

	t.Run("creates a physical copy with its own replication slot and no backup config", func(t *testing.T) {
		physicalServer := containers.StartPhysicalPostgres(t, "postgres:17")
		tm := newTeam(t)
		originalDatabase := tm.createDatabaseNamed(t, "Cluster",
			fromPhysicalSettings(t, newPhysicalSettings(physicalServer, apiclient.PhysicalBackupTypeFULL)))
		storageID := tm.app.CreateLocalStorage(t, tm.owner, tm.workspace.Id)
		tm.app.InsertPhysicalBackupConfigUsingStorage(t, apptest.StorageUse{
			DatabaseID: originalDatabase.Id,
			StorageID:  storageID,
		})

		copiedDatabase := copyDatabase(t, tm.owner, originalDatabase.Id)

		assert.Equal(t, "Cluster (Copy)", copiedDatabase.Name)
		originalPhysical := getPhysicalSettings(t, originalDatabase)
		copiedPhysical := getPhysicalSettings(t, copiedDatabase)
		assert.Equal(t, originalPhysical.Host, copiedPhysical.Host)
		assert.Equal(t, originalPhysical.Port, copiedPhysical.Port)
		assert.Equal(t, originalPhysical.Username, copiedPhysical.Username)
		assert.Equal(t, originalPhysical.BackupType, copiedPhysical.BackupType)
		assert.Equal(t, originalPhysical.Version, copiedPhysical.Version)
		assert.Nil(t, copiedDatabase.LastBackupTime)

		originalSlots := getReplicationSlotNames(t, tm, originalDatabase.Id)
		copiedSlots := getReplicationSlotNames(t, tm, copiedDatabase.Id)
		assert.NotEmpty(t, copiedSlots.ReplicationSlotName)
		assert.NotEqual(t, originalSlots.ReplicationSlotName, copiedSlots.ReplicationSlotName)
		assert.NotEqual(t, originalSlots.BackupSlotName, copiedSlots.BackupSlotName)

		var copiedConfigCount int
		tm.app.QuerySQL(t, &copiedConfigCount,
			"SELECT COUNT(*) FROM backups_configs WHERE database_id = ?", copiedDatabase.Id)
		assert.Zero(t, copiedConfigCount)
		assert.Nil(t, apptest.GetBackupConfig(t, tm.owner, copiedDatabase.Id).StorageId)
	})
}

type replicationSlotNames struct {
	ReplicationSlotName string
	BackupSlotName      string
}

// getReplicationSlotNames reads the slot names, which the API does not return.
func getReplicationSlotNames(t *testing.T, tm team, databaseID uuid.UUID) replicationSlotNames {
	t.Helper()

	var slotNames []replicationSlotNames
	tm.app.QuerySQL(t, &slotNames,
		`SELECT replication_slot_name, backup_slot_name FROM databases_postgresql_physical WHERE database_id = ?`,
		databaseID)
	require.Len(t, slotNames, 1)

	return slotNames[0]
}
