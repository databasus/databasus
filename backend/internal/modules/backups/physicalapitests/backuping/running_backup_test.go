package backuping_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/lib/testwait"
)

func TestCancelRunningPhysicalBackup(t *testing.T) {
	t.Parallel()

	t.Run("keeps its slot during the real full and drops it after HTTP cancellation", func(t *testing.T) {
		t.Parallel()

		team := newBackupTeam(t, backupTeamSpec{})
		source := team.openSource(t)
		releaseStorage := holdStorageWrites(t, team)

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
		require.Equal(t, http.StatusOK, status, "%s", body)

		var cancelledBackup backupRow
		require.NoError(t, json.Unmarshal(body, &cancelledBackup))
		assert.Equal(t, "CANCELED", cancelledBackup.Status)
		assert.Equal(t, []backupRow{cancelledBackup}, team.listBackups(t))

		releaseStorage()
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

		// A canceled upload can retain its unspent receipt until the file commit window expires.
		team.App.Clock.Advance(3 * time.Second)
		team.App.RunJobPass(t, "storage_file_deletion")

		var files []struct{ FileName *string }
		team.App.QuerySQL(t, &files, "SELECT file_name FROM backups_physical WHERE id=?", backup.ID)
		if files[0].FileName != nil {
			assert.False(t, team.App.IsLocalStorageFilePresent(t, *files[0].FileName))
		}
	})
}
