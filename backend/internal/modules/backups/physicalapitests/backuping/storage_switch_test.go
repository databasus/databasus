package backuping_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
)

func TestTransferPhysicalBackupStorage(t *testing.T) {
	t.Parallel()

	for _, isSourceUnreachable := range []bool{false, true} {
		scenario := "starts a new full in a separate namespace after transferring old chains"
		if isSourceUnreachable {
			scenario = "starts a new full after transferring chains from unreachable storage"
		}
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()
			team := newBackupTeam(t, backupTeamSpec{})
			sourceStorage := apptest.CreateStorageNamespace(t, team.Owner, team.Database.WorkspaceId)
			team.StorageID = sourceStorage.StorageID
			apptest.SaveBackupConfigWithStorage(
				t,
				team.Owner,
				apptest.StorageUse{DatabaseID: team.Database.Id, StorageID: team.StorageID},
			)

			firstFull := team.runBackup(t, "full")
			require.Equal(t, "COMPLETED", firstFull.Status)
			incremental := team.runBackup(t, "incremental")
			require.Equal(t, "COMPLETED", incremental.Status)
			secondFull := team.runBackup(t, "full")
			require.Equal(t, "COMPLETED", secondFull.Status)

			if isSourceUnreachable {
				sourceStorage.MakeUnavailable(t, team.Owner)
			}

			targetWorkspace := apptest.CreateWorkspace(t, team.Owner, "Switched physical backups")
			target := apptest.CreateStorageNamespace(t, team.Owner, targetWorkspace.Id)
			apptest.TransferToStorage(t, team.Owner, apptest.StorageSwitchTransfer{
				DatabaseID: team.Database.Id, WorkspaceID: targetWorkspace.Id, StorageID: target.StorageID,
			})

			require.Empty(t, team.listBackups(t), "old chains must leave the catalog at acceptance")
			team.StorageID = target.StorageID
			newFull := team.runAutomaticBackup(t)
			require.Equal(t, "COMPLETED", newFull.Status)
			require.Equal(t, "FULL", newFull.Type)
			assert.NotContains(t, []uuid.UUID{firstFull.ID, incremental.ID, secondFull.ID}, newFull.ID)

			var fileName string
			team.App.QuerySQL(
				t,
				&fileName,
				"SELECT file_name FROM backups_physical WHERE id=? AND storage_id=?",
				newFull.ID,
				target.StorageID,
			)

			target.RequireFile(t, fileName)
		})
	}
}
