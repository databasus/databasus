package walstreaming_test

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

func TestPhysicalBackupTimeAfterWal(t *testing.T) {
	t.Parallel()

	t.Run("keeps the archived WAL time when an older full finishes after it", func(t *testing.T) {
		t.Parallel()

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
		var backupLockOwner int
		backupLock.QuerySQL(t, &backupLockOwner, "SELECT pg_backend_pid()")

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
			team.App.QuerySQL(t, &waiting, `SELECT count(*) FROM pg_locks waiting
				JOIN pg_locks held USING (locktype, database, classid, objid, objsubid)
				WHERE NOT waiting.granted AND held.granted AND held.pid=?
				AND waiting.locktype='advisory'`, backupLockOwner)
			if waiting == 0 {
				return fmt.Errorf("the full has not reached the backup lock")
			}
			return nil
		})

		fullWaitingAt := time.Now().UTC()
		rotateWal(t, source)
		testwait.Until(t, "a real WAL upload commits after the full chose its start time", func(context.Context) error {
			database, err := team.Owner.Client.GetDatabaseWithResponse(t.Context(), team.Database.Id)
			if err != nil {
				return err
			}
			require.NotNil(t, database.JSON200)
			if database.JSON200.LastBackupTime == nil || !database.JSON200.LastBackupTime.After(fullWaitingAt) {
				return fmt.Errorf("no WAL received after the full began waiting has committed")
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
		assert.Equal(t, walTime, *database.JSON200.LastBackupTime)
		for _, backup := range team.listBackups(t) {
			if backup.Type == "FULL" {
				assert.Equal(t, "COMPLETED", backup.Status)
			}
		}
	})
}
