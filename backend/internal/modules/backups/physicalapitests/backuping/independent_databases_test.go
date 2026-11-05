package backuping_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/testwait"
)

func TestIndependentPhysicalBackupSlots(t *testing.T) {
	t.Parallel()

	t.Run("runs two database backups with distinct slots on the same source", func(t *testing.T) {
		t.Parallel()

		first := newBackupTeam(t, backupTeamSpec{})
		settings, err := first.Database.Settings.AsPostgresqlPhysicalSettings()
		require.NoError(t, err)
		settings.Password = new("testpassword")

		var databaseSettings apiclient.DatabaseSettings
		require.NoError(t, databaseSettings.FromPostgresqlPhysicalSettings(settings))
		database := apptest.CreateDatabase(t, first.Owner, first.Database.WorkspaceId, databaseSettings)
		second := &backupTeam{
			App:       first.App,
			Owner:     first.Owner,
			Database:  database,
			StorageID: first.StorageID,
			Source:    first.Source,
		}
		apptest.SaveBackupConfigWithStorage(
			t,
			second.Owner,
			apptest.StorageUse{DatabaseID: second.Database.Id, StorageID: second.StorageID},
		)

		firstSlot, _ := first.slotNames(t)
		secondSlot, _ := second.slotNames(t)
		require.NotEqual(t, firstSlot, secondSlot)

		source := first.openSource(t)
		lock := first.App.BeginSQLTransaction(t)
		// Each backup creates its slot before registering its first file write, so this lock
		// keeps both slots alive until the test observes them, regardless of backup speed.
		lock.ExecuteSQL(t, "LOCK TABLE storages_pending_file_deletions IN SHARE MODE")

		first.trigger(t, "full")
		second.trigger(t, "full")
		first.App.RunJobPass(t, "backups_physical_scheduler")

		testwait.Until(t, "both backups reserve their own physical replication slot", func(ctx context.Context) error {
			var count int
			err := source.QueryRow(ctx, "SELECT count(*) FROM pg_replication_slots WHERE slot_name=ANY($1)", []string{firstSlot, secondSlot}).
				Scan(&count)
			if err != nil {
				return err
			}
			if count != 2 {
				return fmt.Errorf("%d slots instead of two", count)
			}
			return nil
		})

		lock.Rollback(t)
		first.App.WaitForPhysicalBackups(t)

		for _, team := range []*backupTeam{first, second} {
			rows := team.listBackups(t)
			require.Len(t, rows, 1)
			assert.Equal(t, "COMPLETED", rows[0].Status)
			team.assertNoRunningBackup(t)
		}

		var count int
		require.NoError(
			t,
			source.QueryRow(t.Context(), "SELECT count(*) FROM pg_replication_slots WHERE slot_name=ANY($1)", []string{firstSlot, secondSlot}).
				Scan(&count),
		)
		assert.Zero(t, count)
	})
}
