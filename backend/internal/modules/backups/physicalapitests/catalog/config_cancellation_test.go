package catalog_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestCancelPhysicalWorkAfterConfigSave(t *testing.T) {
	t.Run("releases the in flight claim and deletes the streamer when backups are disabled", func(t *testing.T) {
		catalog := newCatalog(t)
		catalog.saveStorageConfig(t, catalog.StorageID)
		catalog.seedClaimAndStreamer(t)
		config := apptest.GetBackupConfig(t, catalog.Owner, catalog.DatabaseID)
		response, err := catalog.Owner.Client.UpdateBackupConfigWithResponse(
			t.Context(),
			catalog.DatabaseID,
			apiclient.BackupConfigInput{
				IsBackupsEnabled: false,
				StorageId:        &catalog.StorageID,
				Encryption:       config.Encryption,
				Settings:         config.Settings,
			},
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, response.StatusCode(), "%s", response.Body)
		assert.Nil(t, catalog.getInFlightClaim(t))
		catalog.assertStreamerAbsent(t)
	})
	t.Run("deletes the streamer and keeps an unrelated claim when storage changes", func(t *testing.T) {
		catalog := newCatalog(t)
		catalog.saveStorageConfig(t, catalog.StorageID)
		catalog.App.InsertPhysicalInFlightBackup(
			t,
			apptest.PhysicalInFlightBackupSeed{
				DatabaseID: catalog.DatabaseID,
				BackupID:   uuid.New(),
				BackupType: "FULL",
			},
		)
		catalog.App.InsertPhysicalWalStreamer(
			t,
			apptest.PhysicalWalStreamerSeed{
				DatabaseID:      catalog.DatabaseID,
				Status:          "RUNNING",
				StartedAt:       time.Now().UTC(),
				LastHeartbeatAt: time.Now().UTC(),
			},
		)
		newStorageID := catalog.App.CreateLocalStorage(t, catalog.Owner, catalog.WorkspaceID)
		catalog.saveStorageConfig(t, newStorageID)
		catalog.assertStreamerAbsent(t)
		assert.NotNil(t, catalog.getInFlightClaim(t))
	})
}

func (catalog *catalogFixture) seedClaimAndStreamer(t *testing.T) {
	t.Helper()
	full := catalog.insertFull(
		t,
		newInProgressFull(
			inProgressFullSpec{DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID, TimelineID: 1},
		),
	)
	catalog.App.InsertPhysicalInFlightBackup(
		t,
		apptest.PhysicalInFlightBackupSeed{DatabaseID: catalog.DatabaseID, BackupID: full.ID, BackupType: "FULL"},
	)
	catalog.App.InsertPhysicalWalStreamer(
		t,
		apptest.PhysicalWalStreamerSeed{
			DatabaseID:      catalog.DatabaseID,
			Status:          "RUNNING",
			StartedAt:       time.Now().UTC(),
			LastHeartbeatAt: time.Now().UTC(),
		},
	)
}

func (catalog *catalogFixture) assertStreamerAbsent(t *testing.T) {
	t.Helper()
	var streamerIDs []uuid.UUID
	catalog.App.QuerySQL(
		t,
		&streamerIDs,
		"SELECT database_id FROM backups_physical_wal_streamers WHERE database_id=?",
		catalog.DatabaseID,
	)
	assert.Empty(t, streamerIDs)
}
