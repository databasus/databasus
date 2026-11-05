package walstreaming_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/testwait"
)

func TestChangePhysicalWalConfiguration(t *testing.T) {
	t.Parallel()

	t.Run("stops its receiver and removes the local queue when backups are disabled", func(t *testing.T) {
		t.Parallel()

		team := newWalTeam(t, walTeamSpec{})
		team.enable(t)
		team.waitForReceiver(t)
		team.disable(t)
		testwait.Until(t, "the disabled database has no live streamer or local queue", func(context.Context) error {
			if team.live(t).IsStreamerRunning || team.App.IsWalQueueDirectoryPresent(t, team.Database.Id) {
				return fmt.Errorf("streamer cleanup is pending")
			}
			return nil
		})
		var streamers int
		team.App.QuerySQL(
			t,
			&streamers,
			"SELECT count(*) FROM backups_physical_wal_streamers WHERE database_id=?",
			team.Database.Id,
		)
		assert.Zero(t, streamers)
	})
	t.Run("archives new segments in the new storage after its configured storage changes", func(t *testing.T) {
		t.Parallel()

		team := newWalTeam(t, walTeamSpec{})
		team.enable(t)
		team.waitForReceiver(t)
		source := team.openSource(t)
		rotateWal(t, source)
		team.waitForWal(t, 1)

		oldStorage := team.StorageID
		var previousSegmentIDs []string
		team.App.QuerySQL(
			t,
			&previousSegmentIDs,
			"SELECT id FROM backups_physical_wal_segments WHERE database_id=? AND storage_id=?",
			team.Database.Id,
			oldStorage,
		)

		team.StorageID = team.App.CreateLocalStorage(t, team.Owner, team.Workspace.Id)
		team.enable(t)
		team.waitForReceiver(t)
		for range 2 {
			rotateWal(t, source)
		}
		team.waitForWal(t, 1)
		testwait.Until(t, "a committed WAL segment belongs to the new storage", func(context.Context) error {
			var count int
			team.App.QuerySQL(
				t,
				&count,
				"SELECT count(*) FROM backups_physical_wal_segments WHERE database_id=? AND storage_id=? AND file_name IS NOT NULL",
				team.Database.Id,
				team.StorageID,
			)
			if count == 0 {
				return fmt.Errorf("new storage has no WAL yet")
			}
			return nil
		})

		var remainingSegmentIDs []string
		team.App.QuerySQL(
			t,
			&remainingSegmentIDs,
			"SELECT id FROM backups_physical_wal_segments WHERE database_id=? AND storage_id=?",
			team.Database.Id,
			oldStorage,
		)
		assert.Subset(t, previousSegmentIDs, remainingSegmentIDs)
	})
}

func TestFailingPhysicalWalSessions(t *testing.T) {
	t.Parallel()

	for _, isNotified := range []bool{false, true} {
		scenario := "keeps the first failure across sessions without notifications when its trigger is disabled"
		if isNotified {
			scenario = "sends the chain failure webhook and keeps the first failure time across sessions"
		}
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()

			team := newWalTeam(t, walTeamSpec{})
			inbox := &webhookInbox{}
			team.attachWebhook(t, inbox)

			// Saving the notifications turns backups on, so the address breaks first: a streamer that
			// starts on the real address never reads it again and never fails.
			team.App.ExecuteSQL(
				t,
				"UPDATE databases_postgresql_physical SET host='127.0.0.1',port=1 WHERE database_id=?",
				team.Database.Id,
			)

			if isNotified {
				team.setNotifications(t, apiclient.NotificationTriggerCHAINBROKEN)
			} else {
				team.setNotifications(t)
			}

			team.enable(t)

			first := team.waitForFailedSession(t, streamStatistics{})
			second := team.waitForFailedSession(t, first)
			assert.True(t, first.Streamer.FailingSince.Equal(*second.Streamer.FailingSince))
			team.App.WaitForNotificationDeliveries(t)
			if isNotified {
				assert.Len(t, inbox.getMessages(), 1)
			} else {
				assert.Empty(t, inbox.getMessages())
			}

			team.App.RestartModules(t)
			third := team.waitForFailedSession(t, second)
			assert.True(t, first.Streamer.FailingSince.Equal(*third.Streamer.FailingSince))

			team.disable(t)
			team.App.RestartModules(t)
			team.App.WaitForNotificationDeliveries(t)
			if isNotified {
				assert.Len(t, inbox.getMessages(), 2)
			} else {
				assert.Empty(t, inbox.getMessages())
			}
		})
	}
}

func (team *walTeam) waitForFailedSession(t *testing.T, previous streamStatistics) streamStatistics {
	t.Helper()
	var failed streamStatistics
	testwait.Until(t, "a later failed streamer session is visible in the API", func(context.Context) error {
		failed = team.statistics(t)
		if failed.Streamer.State != "NOT_RUNNING" || failed.Streamer.FailingSince == nil ||
			failed.Streamer.SessionStartedAt == nil {
			return fmt.Errorf("no failed session")
		}
		if previous.Streamer.SessionStartedAt != nil &&
			!failed.Streamer.SessionStartedAt.After(*previous.Streamer.SessionStartedAt) {
			return fmt.Errorf("no later failed session")
		}
		return nil
	})
	return failed
}
