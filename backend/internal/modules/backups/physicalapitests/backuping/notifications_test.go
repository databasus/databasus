package backuping_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestPhysicalBackupNotifications(t *testing.T) {
	t.Parallel()

	t.Run("sends no success notification when its trigger is disabled", func(t *testing.T) {
		t.Parallel()

		team := newBackupTeam(t, backupTeamSpec{})
		inbox := &webhookInbox{}
		team.attachWebhook(t, inbox, apiclient.NotificationCategoryBACKUPSUCCESS)
		team.setNotifications(t, apiclient.NotificationTriggerCHAINBROKEN)

		backup := team.runBackup(t, "full")
		require.Equal(t, "COMPLETED", backup.Status)

		team.App.WaitForNotificationDeliveries(t)
		assert.Empty(t, inbox.getMessages())
	})
	t.Run("notifies every attached webhook when a full succeeds", func(t *testing.T) {
		t.Parallel()

		team := newBackupTeam(t, backupTeamSpec{})
		first := &webhookInbox{}
		second := &webhookInbox{}
		team.attachWebhook(t, first, apiclient.NotificationCategoryBACKUPSUCCESS)
		team.attachWebhook(t, second, apiclient.NotificationCategoryBACKUPSUCCESS)
		team.setNotifications(t, apiclient.NotificationTriggerBACKUPSUCCESS)

		backup := team.runBackup(t, "full")
		require.Equal(t, "COMPLETED", backup.Status)

		team.App.WaitForNotificationDeliveries(t)
		require.Len(t, first.getMessages(), 1)
		require.Len(t, second.getMessages(), 1)
		assert.Contains(t, first.getMessages()[0], "completed")
	})
	t.Run("sends a failed notification when a full loses its source", func(t *testing.T) {
		t.Parallel()

		team := newBackupTeam(t, backupTeamSpec{})
		inbox := &webhookInbox{}
		team.attachWebhook(t, inbox, apiclient.NotificationCategoryBACKUPFAILED)
		team.setNotifications(t, apiclient.NotificationTriggerBACKUPFAILED)
		team.App.ExecuteSQL(
			t,
			"UPDATE databases_postgresql_physical SET host='127.0.0.1',port=1 WHERE database_id=?",
			team.Database.Id,
		)

		backup := team.runBackup(t, "full")
		require.Equal(t, "ERROR", backup.Status)

		team.App.WaitForNotificationDeliveries(t)
		require.Len(t, inbox.getMessages(), 1)
		assert.Contains(t, inbox.getMessages()[0], "failed")
	})
	t.Run("sends a chain broken notification when a full sees another cluster identity", func(t *testing.T) {
		t.Parallel()

		team := newBackupTeam(t, backupTeamSpec{})
		inbox := &webhookInbox{}
		team.attachWebhook(t, inbox, apiclient.NotificationCategoryBACKUPFAILED)
		team.setNotifications(t, apiclient.NotificationTriggerCHAINBROKEN)
		team.App.ExecuteSQL(
			t,
			"UPDATE databases_postgresql_physical SET system_identifier='1' WHERE database_id=?",
			team.Database.Id,
		)

		backup := team.runBackup(t, "full")
		require.Equal(t, "CHAIN_BROKEN", backup.Status)

		team.App.WaitForNotificationDeliveries(t)
		require.Len(t, inbox.getMessages(), 1)
		assert.Contains(t, inbox.getMessages()[0], "chain-broken")
	})
	t.Run("sends a failed notification when an incremental loses its source", func(t *testing.T) {
		t.Parallel()

		team := newBackupTeam(t, backupTeamSpec{})
		full := team.runBackup(t, "full")
		require.Equal(t, "COMPLETED", full.Status)
		inbox := &webhookInbox{}
		team.attachWebhook(t, inbox, apiclient.NotificationCategoryBACKUPFAILED)
		team.setNotifications(t, apiclient.NotificationTriggerBACKUPFAILED)
		team.App.ExecuteSQL(
			t,
			"UPDATE databases_postgresql_physical SET host='127.0.0.1',port=1 WHERE database_id=?",
			team.Database.Id,
		)

		backup := team.runBackup(t, "incremental")
		require.Equal(t, "ERROR", backup.Status)

		team.App.WaitForNotificationDeliveries(t)
		require.Len(t, inbox.getMessages(), 1)
		assert.Contains(t, inbox.getMessages()[0], "INCR failed")
	})
}
