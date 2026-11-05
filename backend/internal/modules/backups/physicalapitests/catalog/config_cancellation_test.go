package catalog_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/testwait"
)

func TestCancelPhysicalWorkAfterConfigSave(t *testing.T) {
	t.Parallel()

	t.Run("keeps completed files when a full finishes while disabling backups waits on its row", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		catalog.saveStorageConfig(t, catalog.StorageID)
		full := catalog.insertFull(t, newInProgressFull(inProgressFullSpec{
			DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID, TimelineID: 1,
		}))

		fileName := "completed-" + full.ID.String()
		catalog.App.ExecuteSQL(t, "UPDATE backups_physical SET file_name=? WHERE id=?", fileName, full.ID)
		catalog.publishNames(t, fileName, fileName+".metadata", fileName+".manifest")
		config := apptest.GetBackupConfig(t, catalog.Owner, catalog.DatabaseID)

		transaction := catalog.App.BeginSQLTransaction(t)
		transaction.ExecuteSQL(t, "SELECT id FROM backups_physical WHERE id=? FOR UPDATE", full.ID)
		type saveOutcome struct {
			response *apiclient.UpdateBackupConfigResponse
			err      error
		}
		outcomes := make(chan saveOutcome, 1)
		go func() {
			response, err := catalog.Owner.Client.UpdateBackupConfigWithResponse(t.Context(), catalog.DatabaseID,
				apiclient.BackupConfigInput{
					IsBackupsEnabled: false, StorageId: &catalog.StorageID,
					Encryption: config.Encryption, Settings: config.Settings,
				})
			outcomes <- saveOutcome{response: response, err: err}
		}()

		testwait.Until(t, "disabling backups waits on the backup row", func(context.Context) error {
			var waiting int64
			catalog.App.QuerySQL(t, &waiting, `SELECT count(*) FROM pg_stat_activity
				WHERE datname=current_database() AND wait_event_type='Lock' AND pid<>pg_backend_pid()`)
			if waiting == 0 {
				return fmt.Errorf("the save has not reached the backup row")
			}
			return nil
		})

		transaction.ExecuteSQL(t, `UPDATE backups_physical
			SET status='COMPLETED', start_lsn='0/1000000', stop_lsn='0/2000000', completed_at=? WHERE id=?`,
			catalog.App.Clock.Now(), full.ID)
		transaction.Commit(t)

		select {
		case outcome := <-outcomes:
			require.NoError(t, outcome.err)
			require.Equal(t, http.StatusOK, outcome.response.StatusCode(), "%s", outcome.response.Body)
		case <-t.Context().Done():
			t.Fatal("the configuration save did not finish after the backup completed")
		}

		page := getBackups(t, catalog, "")
		require.Len(t, page.Backups, 1)
		assert.Equal(t, "COMPLETED", page.Backups[0].Status)

		var pendingFiles []string
		catalog.App.QuerySQL(
			t,
			&pendingFiles,
			"SELECT file_name FROM storages_pending_file_deletions WHERE storage_id=?",
			catalog.StorageID,
		)
		for _, suffix := range []string{"", ".metadata", ".manifest"} {
			assert.NotContains(t, pendingFiles, fileName+suffix)
		}
	})
	t.Run("cancels the running backup and deletes the streamer when backups are disabled", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		catalog.saveStorageConfig(t, catalog.StorageID)
		catalog.seedRunningBackupAndStreamer(t)
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

		assert.Nil(t, catalog.getRunningBackupID(t))
		page := getBackups(t, catalog, "")
		require.Len(t, page.Backups, 1)
		assert.Equal(t, "CANCELED", page.Backups[0].Status)
		catalog.assertStreamerAbsent(t)
	})
}

func (catalog *catalogFixture) seedRunningBackupAndStreamer(t *testing.T) {
	t.Helper()
	catalog.insertFull(
		t,
		newInProgressFull(
			inProgressFullSpec{DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID, TimelineID: 1},
		),
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
