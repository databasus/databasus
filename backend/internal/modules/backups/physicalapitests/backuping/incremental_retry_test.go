package backuping_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestPhysicalIncrementalRetryBudget(t *testing.T) {
	t.Parallel()

	t.Run("keeps the chain extendable after two automatic failures and breaks it on the third", func(t *testing.T) {
		t.Parallel()

		team := newBackupTeam(t, backupTeamSpec{})
		full := team.runBackup(t, "full")
		require.Equal(t, "COMPLETED", full.Status)

		inbox := &webhookInbox{}
		team.attachWebhook(t, inbox, apiclient.NotificationCategoryBACKUPFAILED)
		team.setNotifications(t, apiclient.NotificationTriggerCHAINBROKEN)
		team.setSchedules(t, false)

		team.App.ExecuteSQL(
			t,
			"UPDATE backups_physical SET created_at=?,completed_at=? WHERE id=?",
			time.Now().UTC().Add(-5*time.Hour),
			time.Now().UTC().Add(-5*time.Hour),
			full.ID,
		)
		team.App.ExecuteSQL(
			t,
			"UPDATE databases_postgresql_physical SET host='127.0.0.1',port=1 WHERE database_id=?",
			team.Database.Id,
		)

		first := team.runAutomaticBackup(t)
		require.Equal(t, "ERROR", first.Status)
		team.assertTerminalDetails(t, first, "NETWORK_FAILURE")

		team.App.ExecuteSQL(
			t,
			"UPDATE backups_physical SET created_at=?,completed_at=? WHERE id=?",
			time.Now().UTC().Add(-3*time.Hour),
			time.Now().UTC().Add(-3*time.Hour),
			first.ID,
		)

		second := team.runAutomaticBackup(t)
		require.Equal(t, "ERROR", second.Status)
		team.assertTerminalDetails(t, second, "NETWORK_FAILURE")

		team.App.ExecuteSQL(
			t,
			"UPDATE backups_physical SET created_at=?,completed_at=? WHERE id=?",
			time.Now().UTC().Add(-2*time.Hour),
			time.Now().UTC().Add(-2*time.Hour),
			second.ID,
		)

		third := team.runAutomaticBackup(t)
		team.App.WaitForPhysicalBackups(t)
		require.Equal(t, "CHAIN_BROKEN", third.Status)
		team.assertTerminalDetails(t, third, "NETWORK_FAILURE")
		team.assertNoRunningBackup(t)

		team.App.WaitForNotificationDeliveries(t)
		require.Len(t, inbox.getMessages(), 1)
		assert.Contains(t, inbox.getMessages()[0], "chain-broken")
	})
	t.Run("keeps the retry budget when a completed incremental separates the failures", func(t *testing.T) {
		t.Parallel()

		team := newBackupTeam(t, backupTeamSpec{})
		full := team.runBackup(t, "full")
		require.Equal(t, "COMPLETED", full.Status)

		first := team.runBackup(t, "incremental")
		require.Equal(t, "COMPLETED", first.Status)

		team.setSchedules(t, false)
		team.App.ExecuteSQL(
			t,
			"UPDATE backups_physical SET created_at=?,completed_at=? WHERE id=?",
			time.Now().UTC().Add(-7*time.Hour),
			time.Now().UTC().Add(-7*time.Hour),
			full.ID,
		)
		team.App.ExecuteSQL(
			t,
			"UPDATE backups_physical SET created_at=?,completed_at=? WHERE id=?",
			time.Now().UTC().Add(-4*time.Hour),
			time.Now().UTC().Add(-4*time.Hour),
			first.ID,
		)

		for _, age := range []time.Duration{5 * time.Hour, 2 * time.Hour} {
			team.App.InsertPhysicalIncrementalBackup(
				t,
				apptest.PhysicalIncrementalBackupSeed{
					DatabaseID:       team.Database.Id,
					StorageID:        team.StorageID,
					RootFullBackupID: full.ID,
					TimelineID:       1,
					Status:           "ERROR",
					ErrorReason:      new("NETWORK_FAILURE"),
					CreatedAt:        time.Now().UTC().Add(-age),
				},
			)
		}

		team.App.ExecuteSQL(
			t,
			"UPDATE databases_postgresql_physical SET host='127.0.0.1',port=1 WHERE database_id=?",
			team.Database.Id,
		)

		backup := team.runAutomaticBackup(t)
		assert.Equal(t, "ERROR", backup.Status)
		team.assertNoRunningBackup(t)
	})
}
