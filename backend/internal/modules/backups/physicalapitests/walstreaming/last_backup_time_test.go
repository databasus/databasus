package walstreaming_test

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

func TestPhysicalBackupTimeAfterWal(t *testing.T) {
	t.Parallel()

	t.Run("keeps the archived WAL time when an older full finishes after it", func(t *testing.T) {
		team := newWalTeam(t, walTeamSpec{})
		team.enable(t)
		team.waitForReceiver(t)
		source := team.openSource(t)
		rotateWal(t, source)
		team.waitForWal(t, 1)
		team.App.WaitForPhysicalBackups(t)
		backupLock := team.App.BeginSQLTransaction(t)
		backupLock.ExecuteSQL(
			t,
			"SELECT pg_advisory_xact_lock(hashtextextended('physical-backup:' || ?::text,0))",
			team.Database.Id,
		)
		team.call(
			t,
			apptest.PhysicalStartRequest(team.Database.Id, "FULL"),
			http.StatusAccepted,
			nil,
		)
		scheduled := make(chan struct{}, 1)
		go func() { team.App.RunJobPass(t, "backups_physical_scheduler"); scheduled <- struct{}{} }()
		testwait.Until(t, "the next full waits after choosing its start time", func(context.Context) error {
			var waiting int
			team.App.QuerySQL(t, &waiting, `SELECT count(*) FROM pg_locks
				WHERE NOT granted AND locktype='advisory'
				AND database=(SELECT oid FROM pg_database WHERE datname=current_database())`)
			if waiting == 0 {
				return fmt.Errorf("the full has not reached the backup lock")
			}
			return nil
		})
		var firstWalCount int
		team.App.QuerySQL(
			t,
			&firstWalCount,
			"SELECT count(*) FROM backups_physical_wal_segments WHERE database_id=? AND file_name IS NOT NULL",
			team.Database.Id,
		)
		rotateWal(t, source)
		testwait.Until(t, "a real WAL upload commits after the full chose its start time", func(context.Context) error {
			var count int
			team.App.QuerySQL(
				t,
				&count,
				"SELECT count(*) FROM backups_physical_wal_segments WHERE database_id=? AND file_name IS NOT NULL",
				team.Database.Id,
			)
			if count <= firstWalCount {
				return fmt.Errorf("no later WAL commit")
			}
			return nil
		})
		settings, err := team.Database.Settings.AsPostgresqlPhysicalSettings()
		require.NoError(t, err)
		settings.Password = new("testpassword")
		settings.BackupType = apiclient.PhysicalBackupTypeFULLINCREMENTAL
		var updatedSettings apiclient.DatabaseSettings
		require.NoError(t, updatedSettings.FromPostgresqlPhysicalSettings(settings))
		updated, err := team.Owner.Client.UpdateDatabaseWithResponse(
			t.Context(),
			team.Database.Id,
			apiclient.DatabaseUpdate{Name: team.Database.Name, Settings: updatedSettings},
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, updated.StatusCode(), "%s", updated.Body)
		testwait.Until(t, "the receiver stops after WAL archiving is turned off", func(context.Context) error {
			if team.live(t).IsStreamerRunning {
				return fmt.Errorf("streamer has not stopped")
			}
			return nil
		})
		database, err := team.Owner.Client.GetDatabaseWithResponse(t.Context(), team.Database.Id)
		require.NoError(t, err)
		require.NotNil(t, database.JSON200)
		require.NotNil(t, database.JSON200.LastBackupTime)
		walTime := *database.JSON200.LastBackupTime
		backupLock.Commit(t)
		select {
		case <-scheduled:
		case <-t.Context().Done():
			t.Fatal("scheduler did not finish after the backup lock was released")
		}
		team.App.WaitForPhysicalBackups(t)
		database, err = team.Owner.Client.GetDatabaseWithResponse(t.Context(), team.Database.Id)
		require.NoError(t, err)
		require.NotNil(t, database.JSON200)
		require.NotNil(t, database.JSON200.LastBackupTime)
		assert.True(t, walTime.Equal(*database.JSON200.LastBackupTime))
		for _, backup := range team.listBackups(t) {
			if backup.Type == "FULL" {
				assert.Equal(t, "COMPLETED", backup.Status)
			}
		}
	})
}
