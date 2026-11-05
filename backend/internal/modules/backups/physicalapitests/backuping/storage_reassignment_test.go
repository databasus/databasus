package backuping_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

type storageConfigSave struct {
	StorageID *uuid.UUID
	IsEnabled bool
}

func (team *backupTeam) saveStorage(t *testing.T, save storageConfigSave) *apiclient.UpdateBackupConfigResponse {
	t.Helper()
	backupConfig := apptest.GetBackupConfig(t, team.Owner, team.Database.Id)
	response, err := team.Owner.Client.UpdateBackupConfigWithResponse(
		t.Context(),
		team.Database.Id,
		apiclient.BackupConfigInput{
			IsBackupsEnabled: save.IsEnabled,
			StorageId:        save.StorageID,
			Encryption:       backupConfig.Encryption,
			Settings:         backupConfig.Settings,
		},
	)
	require.NoError(t, err)
	return response
}

func TestReassignPhysicalStorage(t *testing.T) {
	t.Parallel()
	for _, isSameStorage := range []bool{false, true} {
		for _, isDisabled := range []bool{false, true} {
			scenario := "starts a fresh full with another storage after clearing"
			if isSameStorage {
				scenario = "starts a fresh full with the historical storage after clearing"
			}
			if isDisabled {
				scenario += " when backups are enabled later"
			}
			t.Run(scenario, func(t *testing.T) {
				t.Parallel()
				team := newBackupTeam(t, backupTeamSpec{})
				full := team.runBackup(t, "full")
				require.Equal(t, "COMPLETED", full.Status)

				cleared := team.saveStorage(t, storageConfigSave{IsEnabled: !isDisabled})
				require.Equal(t, http.StatusOK, cleared.StatusCode(), "%s", cleared.Body)
				require.Len(t, team.listBackups(t), 1)
				if !isSameStorage {
					team.StorageID = apptest.CreateStorageNamespace(t, team.Owner, team.Database.WorkspaceId).StorageID
				}

				assigned := team.saveStorage(t, storageConfigSave{StorageID: &team.StorageID, IsEnabled: !isDisabled})
				require.Equal(t, http.StatusOK, assigned.StatusCode(), "%s", assigned.Body)
				require.Empty(t, team.listBackups(t))

				if isDisabled {
					assert.False(t, apptest.GetBackupConfig(t, team.Owner, team.Database.Id).IsBackupsEnabled)
					team.App.RunJobPass(t, "backups_physical_scheduler")
					require.Empty(t, team.listBackups(t))
					setBackupsEnabled(t, team, true)
				}

				replacement := team.runAutomaticBackup(t)
				assert.Equal(t, "FULL", replacement.Type)
				assert.Equal(t, "COMPLETED", replacement.Status)
				assert.NotEqual(t, full.ID, replacement.ID)
			})
		}
	}
}

func TestEnableSwitchedPhysicalStorage(t *testing.T) {
	t.Parallel()
	t.Run("starts a fresh full when a disabled switch preserves a target chain", func(t *testing.T) {
		t.Parallel()

		team := newBackupTeam(t, backupTeamSpec{})
		full := team.runBackup(t, "full")

		targetStorageID := team.App.CreateLocalStorage(t, team.Owner, team.Database.WorkspaceId)
		team.App.ExecuteSQL(t, "UPDATE backups_physical SET storage_id=? WHERE id=?", targetStorageID, full.ID)

		switched := team.saveStorage(t, storageConfigSave{StorageID: &targetStorageID, IsEnabled: false})
		require.Equal(t, http.StatusOK, switched.StatusCode(), "%s", switched.Body)

		team.App.RunJobPass(t, "backups_physical_scheduler")
		require.Len(t, team.listBackups(t), 1)

		setBackupsEnabled(t, team, true)
		team.App.RunJobPass(t, "backups_physical_scheduler")
		team.App.WaitForPhysicalBackups(t)

		backups := team.listBackups(t)
		require.Len(t, backups, 2)
		for _, backup := range backups {
			assert.Equal(t, "FULL", backup.Type)
			assert.Equal(t, "COMPLETED", backup.Status)
		}
	})
}

func TestRollbackRunningPhysicalStorageSwitch(t *testing.T) {
	t.Parallel()
	t.Run("keeps old execution alive when cleanup persistence fails", func(t *testing.T) {
		t.Parallel()

		team := newBackupTeam(t, backupTeamSpec{})
		team.App.ExecuteSQL(
			t,
			`CREATE FUNCTION refuse_switch_files() RETURNS trigger AS $$ BEGIN IF NEW.cleanup_database_id IS NOT NULL THEN RAISE EXCEPTION 'cleanup refused'; END IF; RETURN NEW; END $$ LANGUAGE plpgsql`,
		)
		team.App.ExecuteSQL(
			t,
			`CREATE TRIGGER refuse_switch_files BEFORE INSERT ON storages_pending_file_deletions FOR EACH ROW EXECUTE FUNCTION refuse_switch_files()`,
		)

		backup, releaseStorage := startHeldPhysicalStream(t, team, "full")

		targetStorageID := team.App.CreateLocalStorage(t, team.Owner, team.Database.WorkspaceId)
		response := team.saveStorage(t, storageConfigSave{StorageID: &targetStorageID, IsEnabled: true})
		require.Equal(t, http.StatusInternalServerError, response.StatusCode(), "%s", response.Body)
		assert.Equal(t, &team.StorageID, apptest.GetBackupConfig(t, team.Owner, team.Database.Id).StorageId)
		assert.Equal(t, "IN_PROGRESS", findPhysicalBackup(t, team, backup).Status)

		releaseStorage()
		team.App.WaitForPhysicalBackups(t)
		assert.Equal(t, "COMPLETED", findPhysicalBackup(t, team, backup).Status)
	})
}
