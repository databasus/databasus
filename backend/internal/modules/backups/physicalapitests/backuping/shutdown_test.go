package backuping_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/testwait"
)

const shutdownFailMessage = "backup interrupted by application shutdown"

func TestPhysicalBackupShutdown(t *testing.T) {
	for _, kind := range []string{"full", "incremental"} {
		t.Run(
			"records failure and notifies subscribers when shutdown interrupts a "+kind+" stream",
			func(t *testing.T) {
				team := newBackupTeam(t, backupTeamSpec{})
				if kind == "incremental" {
					require.Equal(t, "COMPLETED", team.runBackup(t, "full").Status)
				}
				failures, successes := &webhookInbox{}, &webhookInbox{}
				team.attachWebhook(t, failures, apiclient.NotificationCategoryBACKUPFAILED)
				team.attachWebhook(t, successes, apiclient.NotificationCategoryBACKUPSUCCESS)
				team.setNotifications(t, apiclient.NotificationTriggerBACKUPFAILED)
				backup, releaseStorage := startHeldPhysicalStream(t, team, kind)
				team.App.RestartModules(t)
				releaseStorage()
				listed := findPhysicalBackup(t, team, backup)
				assert.Equal(t, "ERROR", listed.Status)
				require.NotNil(t, listed.FailMessage)
				assert.Equal(t, shutdownFailMessage, *listed.FailMessage)
				team.assertTerminalDetails(t, listed, "APPLICATION_RESTART")
				team.assertClaimReleased(t)
				team.App.WaitForNotificationDeliveries(t)
				require.Len(t, failures.getMessages(), 1)
				assert.Contains(t, failures.getMessages()[0], shutdownFailMessage)
				assert.Empty(t, successes.getMessages())
				team.App.RunJobPass(t, "storage_file_deletion")
				var artifacts []struct {
					FileName         *string
					ManifestFileName *string
				}
				table := "backups_physical_full"
				if kind == "incremental" {
					table = "backups_physical_incremental"
				}
				team.App.QuerySQL(
					t,
					&artifacts,
					"SELECT file_name, manifest_file_name FROM "+table+" WHERE id=?",
					backup.ID,
				)
				require.Len(t, artifacts, 1)
				require.NotNil(t, artifacts[0].FileName)
				assert.False(t, team.App.IsLocalStorageFilePresent(t, *artifacts[0].FileName))
				assert.False(t, team.App.IsLocalStorageFilePresent(t, *artifacts[0].FileName+".metadata"))
				if artifacts[0].ManifestFileName != nil {
					assert.False(t, team.App.IsLocalStorageFilePresent(t, *artifacts[0].ManifestFileName))
				}
			},
		)
		t.Run("keeps the user cancellation outcome when a user cancels a "+kind+" stream", func(t *testing.T) {
			team := newBackupTeam(t, backupTeamSpec{})
			if kind == "incremental" {
				require.Equal(t, "COMPLETED", team.runBackup(t, "full").Status)
			}
			backup, releaseStorage := startHeldPhysicalStream(t, team, kind)
			cancelPhysicalBackup(t, team, backup)
			releaseStorage()
			team.App.WaitForPhysicalBackups(t)
			listed := findPhysicalBackup(t, team, backup)
			assert.Equal(t, "CANCELED", listed.Status)
			require.NotNil(t, listed.FailMessage)
			assert.Equal(t, "context canceled", *listed.FailMessage)
			team.assertClaimReleased(t)
		})
	}
}

func TestCancelDispatchedPhysicalBackup(t *testing.T) {
	t.Run("ends the backup when cancellation immediately follows a scheduler pass", func(t *testing.T) {
		team := newBackupTeam(t, backupTeamSpec{})
		team.trigger(t, "full")
		team.App.RunJobPass(t, "backups_physical_scheduler")
		backups := team.listBackups(t)
		require.Len(t, backups, 1)
		cancelPhysicalBackup(t, team, backups[0])
		team.App.WaitForPhysicalBackups(t)
		listed := findPhysicalBackup(t, team, backups[0])
		assert.Contains(t, []string{"ERROR", "CANCELED"}, listed.Status)
		team.assertClaimReleased(t)
	})
}

func startHeldPhysicalStream(t *testing.T, team *backupTeam, kind string) (backupRow, func()) {
	t.Helper()
	source := team.openSource(t)
	_, err := source.Exec(
		t.Context(),
		"CREATE TABLE shutdown_backup_payload AS SELECT string_agg(md5(random()::text),'') AS payload FROM generate_series(1,1000000) entry GROUP BY entry/32",
	)
	require.NoError(t, err)
	storageLock := team.App.BeginSQLTransaction(t)
	var storageIDs []string
	storageLock.QuerySQL(t, &storageIDs, "SELECT id FROM storages WHERE id=? FOR NO KEY UPDATE", team.StorageID)
	require.Len(t, storageIDs, 1)
	team.trigger(t, kind)
	team.App.RunJobPass(t, "backups_physical_scheduler")
	var backup backupRow
	testwait.Until(t, "pg_basebackup streams the held physical backup", func(ctx context.Context) error {
		var activeCount int
		if err := source.QueryRow(ctx, "SELECT count(*) FROM pg_stat_activity WHERE application_name='pg_basebackup' AND backend_type='walsender'").
			Scan(&activeCount); err != nil {
			return err
		}
		if activeCount == 0 {
			return fmt.Errorf("pg_basebackup has not started streaming")
		}
		for _, listed := range team.listBackups(t) {
			if listed.Status == "IN_PROGRESS" {
				backup = listed
				return nil
			}
		}
		return fmt.Errorf("no in-progress physical backup")
	})
	return backup, func() { storageLock.Rollback(t) }
}

func cancelPhysicalBackup(t *testing.T, team *backupTeam, backup backupRow) {
	t.Helper()
	status, body := team.App.CallOldAPI(
		t,
		team.Owner,
		apptest.OldAPICall{
			Method: http.MethodPost,
			Path:   "/backups/physical/backups/" + backup.ID.String() + "/cancel",
		},
	)
	require.Equal(t, http.StatusNoContent, status, "%s", body)
}

func findPhysicalBackup(t *testing.T, team *backupTeam, backup backupRow) backupRow {
	t.Helper()
	for _, listed := range team.listBackups(t) {
		if listed.ID == backup.ID {
			return listed
		}
	}
	t.Fatalf("physical backup %s is absent from the list", backup.ID)
	return backupRow{}
}
