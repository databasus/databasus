package backuping_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/testwait"
)

func TestPhysicalBackupRunOverlap(t *testing.T) {
	t.Parallel()

	for _, action := range []string{"cancel", "delete", "disable", "switch", "delete then switch", "delete then clear then switch"} {
		t.Run("waits for the old process before completing the next full after "+action, func(t *testing.T) {
			t.Parallel()

			isStorageSwitch := action == "switch" || action == "delete then switch" ||
				action == "delete then clear then switch"

			team := newBackupTeam(t, backupTeamSpec{})
			sourceStorageID := team.StorageID

			// The target comes first: a cancelled process is killed after 10 seconds, and starting the
			// target's storage between the cancellation and the scheduler pass can outlast that.
			var targetWorkspace apiclient.Workspace
			var target apptest.StorageNamespace
			if isStorageSwitch {
				targetWorkspace = apptest.CreateWorkspace(t, team.Owner, "Switched running backup")
				target = apptest.CreateStorageNamespace(t, team.Owner, targetWorkspace.Id)
			}

			backup, releaseStorage := startHeldPhysicalStream(t, team, "full")
			resume := apptest.SuspendPhysicalBackupProcess(t, backup.ID)
			releaseStorage()
			var attemptFiles []string
			team.App.QuerySQL(t, &attemptFiles, "SELECT file_name FROM backups_physical WHERE id=?", backup.ID)
			require.Len(t, attemptFiles, 1)
			require.NotEmpty(t, attemptFiles[0])

			switch action {
			case "switch", "delete then switch", "delete then clear then switch":
				if action != "switch" {
					status, body := team.App.CallPhysicalAPI(t, team.Owner, apptest.PhysicalDeleteRequest(backup.ID))
					require.Equal(t, http.StatusNoContent, status, "%s", body)
					require.Empty(t, team.listBackups(t))
				}
				if action == "delete then clear then switch" {
					cleared := team.saveStorage(t, storageConfigSave{IsEnabled: true})
					require.Equal(t, http.StatusOK, cleared.StatusCode(), "%s", cleared.Body)
					require.Nil(t, apptest.GetBackupConfig(t, team.Owner, team.Database.Id).StorageId)
				}

				operation := apptest.TransferToStorage(
					t,
					team.Owner,
					apptest.StorageSwitchTransfer{
						DatabaseID:  team.Database.Id,
						WorkspaceID: targetWorkspace.Id,
						StorageID:   target.StorageID,
					},
				)
				assert.Positive(t, operation.Cleanup.PendingFileCount)

				moved, err := team.Owner.Client.GetDatabaseWithResponse(t.Context(), team.Database.Id)
				require.NoError(t, err)
				require.NotNil(t, moved.JSON200)
				assert.Equal(t, targetWorkspace.Id, moved.JSON200.WorkspaceId)
				assert.Equal(t, &target.StorageID, apptest.GetBackupConfig(t, team.Owner, team.Database.Id).StorageId)

				team.StorageID = target.StorageID
				require.Empty(t, team.listBackups(t))

				var capturedFiles []string
				team.App.QuerySQL(
					t,
					&capturedFiles,
					`SELECT file_name FROM storages_pending_file_deletions WHERE cleanup_database_id=?`,
					team.Database.Id,
				)

				for _, suffix := range []string{"", ".metadata", ".manifest"} {
					assert.Contains(t, capturedFiles, attemptFiles[0]+suffix)
				}
			case "cancel":
				cancelPhysicalBackup(t, team, backup)
				assert.Equal(t, "CANCELED", findPhysicalBackup(t, team, backup).Status)
				team.trigger(t, "full")
			case "delete":
				status, body := team.App.CallPhysicalAPI(t, team.Owner, apptest.PhysicalDeleteRequest(backup.ID))
				require.Equal(t, http.StatusNoContent, status, "%s", body)
				require.Empty(t, team.listBackups(t))
			case "disable":
				setBackupsEnabled(t, team, false)
				assert.Equal(t, "CANCELED", findPhysicalBackup(t, team, backup).Status)
				team.App.Clock.Advance(48 * time.Hour)
				setBackupsEnabled(t, team, true)
			}

			beforePass := team.listBackups(t)
			team.App.RunJobPass(t, "backups_physical_scheduler")
			require.Equal(
				t,
				beforePass,
				team.listBackups(t),
				"the scheduler must insert nothing while the stopped process is alive",
			)

			resume()
			var nextBackup backupRow
			testwait.Until(
				t,
				"the next scheduler pass completes a full after the old run ends",
				func(context.Context) error {
					team.App.RunJobPass(t, "backups_physical_scheduler")
					for _, listed := range team.listBackups(t) {
						if listed.ID != backup.ID {
							if listed.Status == "IN_PROGRESS" {
								return fmt.Errorf("the next full is still running")
							}
							require.Equal(t, "COMPLETED", listed.Status)
							nextBackup = listed
							return nil
						}
					}
					return fmt.Errorf("the old run has not ended yet")
				},
			)

			assert.Equal(t, "FULL", nextBackup.Type)
			if isStorageSwitch {
				var oldHistory int
				team.App.QuerySQL(t, &oldHistory, "SELECT count(*) FROM backups_physical WHERE id=?", backup.ID)
				assert.Zero(t, oldHistory)
			}
			if action != "delete" && !isStorageSwitch {
				assert.Equal(t, "CANCELED", findPhysicalBackup(t, team, backup).Status)
			}

			var pendingFiles []string
			team.App.QuerySQL(
				t,
				&pendingFiles,
				"SELECT file_name FROM storages_pending_file_deletions WHERE storage_id=?",
				sourceStorageID,
			)
			for _, suffix := range []string{"", ".metadata", ".manifest"} {
				assert.Contains(t, pendingFiles, attemptFiles[0]+suffix)
			}
		})
	}
}

func setBackupsEnabled(t *testing.T, team *backupTeam, isEnabled bool) {
	t.Helper()
	config := apptest.GetBackupConfig(t, team.Owner, team.Database.Id)
	response, err := team.Owner.Client.UpdateBackupConfigWithResponse(t.Context(), team.Database.Id,
		apiclient.BackupConfigInput{
			IsBackupsEnabled: isEnabled, StorageId: config.StorageId,
			Encryption: config.Encryption, Settings: config.Settings,
		})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), "%s", response.Body)
}
