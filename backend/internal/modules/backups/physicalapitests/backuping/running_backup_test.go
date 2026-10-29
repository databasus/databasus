package backuping_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/lib/testwait"
)

func TestCancelRunningPhysicalBackup(t *testing.T) {
	t.Run("keeps its slot during the real full and drops it after HTTP cancellation", func(t *testing.T) {
		team := newBackupTeam(t, backupTeamSpec{})
		source := team.openSource(t)
		storageLock := team.App.BeginSQLTransaction(t)
		var lockedStorageIDs []string
		storageLock.QuerySQL(
			t,
			&lockedStorageIDs,
			"SELECT id FROM storages WHERE id=? FOR NO KEY UPDATE",
			team.StorageID,
		)
		require.Len(t, lockedStorageIDs, 1)
		team.trigger(t, "full")
		team.App.RunJobPass(t, "backups_physical_scheduler")
		var backup backupRow
		var slotName string
		team.App.QuerySQL(
			t,
			&slotName,
			"SELECT backup_slot_name FROM databases_postgresql_physical WHERE database_id=?",
			team.Database.Id,
		)
		testwait.Until(t, "the full starts while its storage write waits", func(context.Context) error {
			rows := team.listBackups(t)
			if len(rows) != 1 || rows[0].Status != "IN_PROGRESS" {
				return fmt.Errorf("no running full")
			}
			var exists bool
			if err := source.QueryRow(t.Context(), "SELECT EXISTS(SELECT 1 FROM pg_replication_slots WHERE slot_name=$1)", slotName).
				Scan(&exists); err != nil {
				return err
			}
			if !exists {
				return fmt.Errorf("the backup slot does not exist yet")
			}
			backup = rows[0]
			return nil
		})
		var runningBackupID string
		team.App.QuerySQL(
			t,
			&runningBackupID,
			"SELECT id FROM backups_physical WHERE status = 'IN_PROGRESS' AND database_id=?",
			team.Database.Id,
		)
		assert.Equal(t, backup.ID.String(), runningBackupID)
		status, body := team.App.CallPhysicalAPI(
			t,
			team.Owner,
			apptest.PhysicalCancelRequest(backup.ID),
		)
		require.Equal(t, http.StatusAccepted, status, "%s", body)
		storageLock.Rollback(t)
		team.App.WaitForPhysicalBackups(t)
		rows := team.listBackups(t)
		require.Len(t, rows, 1)
		assert.Equal(t, "CANCELED", rows[0].Status)
		team.assertNoRunningBackup(t)
		var count int
		require.NoError(
			t,
			source.QueryRow(t.Context(), "SELECT count(*) FROM pg_replication_slots WHERE slot_name=$1", slotName).
				Scan(&count),
		)
		assert.Zero(t, count)
		team.App.RunJobPass(t, "storage_file_deletion")
		var files []struct{ FileName *string }
		team.App.QuerySQL(t, &files, "SELECT file_name FROM backups_physical WHERE id=?", backup.ID)
		if files[0].FileName != nil {
			assert.False(t, team.App.IsLocalStorageFilePresent(t, *files[0].FileName))
		}
	})
}
