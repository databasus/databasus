package backuping_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/modules/storages"
)

func TestRecoverPhysicalBackupAfterRestart(t *testing.T) {
	for _, testCase := range []struct {
		name        string
		hasArtifact bool
	}{
		{name: "fails the interrupted full and releases its claim when modules restart", hasArtifact: false},
		{name: "removes its named artifact after restart recovery fails an interrupted full", hasArtifact: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			team := newBackupState(t)
			full := team.App.InsertPhysicalFullBackup(
				t,
				apptest.PhysicalFullBackupSeed{
					DatabaseID: team.Database.Id,
					StorageID:  team.StorageID,
					TimelineID: 1,
					Status:     "IN_PROGRESS",
					CreatedAt:  time.Now().UTC().Add(-time.Hour),
				},
			)
			team.App.InsertPhysicalInFlightBackup(
				t,
				apptest.PhysicalInFlightBackupSeed{DatabaseID: team.Database.Id, BackupID: full.ID, BackupType: "FULL"},
			)
			fileName := "interrupted-" + full.ID.String()
			if testCase.hasArtifact {
				team.App.ExecuteSQL(t, "UPDATE backups_physical_full SET file_name=? WHERE id=?", fileName, full.ID)
				team.App.PublishStorageFile(
					t,
					storages.StoredFileReference{StorageID: team.StorageID, FileName: fileName},
					strings.NewReader("partial physical backup"),
				)
			}
			team.App.RestartModules(t)
			backups := team.listBackups(t)
			require.Len(t, backups, 1)
			assert.Equal(t, "ERROR", backups[0].Status)
			team.assertTerminalDetails(t, backups[0], "APPLICATION_RESTART")
			team.assertClaimReleased(t)
			var stored apptest.PhysicalFullBackupSeed
			team.App.QuerySQL(t, &stored, "SELECT * FROM backups_physical_full WHERE id=?", full.ID)
			require.NotNil(t, stored.FailMessage)
			if testCase.hasArtifact {
				require.NotNil(t, stored.FileName)
				assert.Equal(t, fileName, *stored.FileName)
				team.App.RunJobPass(t, "storage_file_deletion")
				assert.False(t, team.App.IsLocalStorageFilePresent(t, fileName))
			} else {
				assert.Nil(t, stored.FileName)
			}
		})
	}
}
