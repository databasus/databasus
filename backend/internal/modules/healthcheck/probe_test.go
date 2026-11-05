package healthcheck_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestHealthcheckProbe(t *testing.T) {
	t.Parallel()

	t.Run("marks unavailable only at the threshold and sends one notification until recovery", func(t *testing.T) {
		t.Parallel()

		team := newHealthcheckTeam(t)
		failedMessages := recordNotifications(t, team, apiclient.NotificationCategoryHEALTHCHECKFAILED)
		recoveryMessages := recordNotifications(t, team, apiclient.NotificationCategoryHEALTHCHECKSUCCESS)

		var port int
		team.App.QuerySQL(
			t,
			&port,
			"SELECT port FROM databases_postgresql_logical WHERE database_id = ?",
			team.Database.Id,
		)

		team.makeUnreachable(t)
		for attempt := 1; attempt <= 4; attempt++ {
			team.pass(t)
			require.Len(t, team.getAttempts(t), attempt)
			assert.Equal(t, apiclient.DatabaseHealthStatusUNAVAILABLE, *team.getAttempts(t)[0].Status)
			if attempt < 3 {
				assert.Equal(t, new(apiclient.DatabaseHealthStatusAVAILABLE), team.getHealthStatus(t))
				assert.Empty(t, failedMessages)
			} else {
				assert.Equal(t, new(apiclient.DatabaseHealthStatusUNAVAILABLE), team.getHealthStatus(t))
				require.Len(t, failedMessages, 1)
			}
			assert.Empty(t, recoveryMessages)
			team.App.Clock.Advance(time.Minute)
		}

		assert.Contains(t, <-failedMessages, team.Database.Name)
		team.App.ExecuteSQL(
			t,
			"UPDATE databases_postgresql_logical SET port = ? WHERE database_id = ?",
			port,
			team.Database.Id,
		)

		team.pass(t)
		assert.Equal(t, new(apiclient.DatabaseHealthStatusAVAILABLE), team.getHealthStatus(t))
		assert.Empty(t, failedMessages)
		require.Len(t, recoveryMessages, 1)
		assert.Contains(t, <-recoveryMessages, "online")

		team.App.Clock.Advance(time.Minute)
		team.pass(t)
		assert.Empty(t, failedMessages)
		assert.Empty(t, recoveryMessages)
	})
	t.Run("breaks consecutive failures when a successful probe intervenes", func(t *testing.T) {
		t.Parallel()

		team := newHealthcheckTeam(t)
		var port int
		team.App.QuerySQL(
			t,
			&port,
			"SELECT port FROM databases_postgresql_logical WHERE database_id = ?",
			team.Database.Id,
		)

		team.makeUnreachable(t)
		team.pass(t)
		team.App.Clock.Advance(time.Minute)
		team.App.ExecuteSQL(
			t,
			"UPDATE databases_postgresql_logical SET port = ? WHERE database_id = ?",
			port,
			team.Database.Id,
		)
		team.pass(t)
		team.makeUnreachable(t)
		for range 2 {
			team.App.Clock.Advance(time.Minute)
			team.pass(t)
		}

		assert.Equal(t, new(apiclient.DatabaseHealthStatusAVAILABLE), team.getHealthStatus(t))
		assert.Len(t, team.getAttempts(t), 4)
	})
	t.Run("notifies on the first success only after checks are switched off and on again", func(t *testing.T) {
		t.Parallel()

		team := newHealthcheckTeam(t)
		messages := recordNotifications(t, team)
		team.pass(t)
		assert.Empty(t, messages)

		input := configInput(team.getConfig(t))
		input.IsHealthcheckEnabled = false
		team.saveConfig(t, input)
		assert.Nil(t, team.getHealthStatus(t))
		team.App.Clock.Advance(time.Minute)
		team.pass(t)
		assert.Len(t, team.getAttempts(t), 1)

		input.IsHealthcheckEnabled = true
		team.saveConfig(t, input)
		team.pass(t)
		assert.Equal(t, new(apiclient.DatabaseHealthStatusAVAILABLE), team.getHealthStatus(t))
		require.Len(t, messages, 1)
		assert.Contains(t, <-messages, "online")
	})
	t.Run("clears an unavailable status and keeps the history when checks are switched off", func(t *testing.T) {
		t.Parallel()

		team := newHealthcheckTeam(t)
		input := configInput(team.getConfig(t))
		input.FailedAttemptsBeforeUnavailable = 1
		team.saveConfig(t, input)

		team.makeUnreachable(t)
		team.pass(t)
		assert.Equal(t, new(apiclient.DatabaseHealthStatusUNAVAILABLE), team.getHealthStatus(t))

		input.IsHealthcheckEnabled = false
		team.saveConfig(t, input)
		assert.Nil(t, team.getHealthStatus(t))
		team.App.Clock.Advance(time.Minute)
		team.pass(t)
		assert.Len(t, team.getAttempts(t), 1)
	})
	t.Run("changes status without notification when notifications are switched off", func(t *testing.T) {
		t.Parallel()

		team := newHealthcheckTeam(t)
		messages := recordNotifications(t, team)
		input := configInput(team.getConfig(t))
		input.FailedAttemptsBeforeUnavailable = 1
		input.IsNotificationEnabled = false
		team.saveConfig(t, input)

		team.makeUnreachable(t)
		team.pass(t)
		assert.Equal(t, new(apiclient.DatabaseHealthStatusUNAVAILABLE), team.getHealthStatus(t))
		assert.Empty(t, messages)
	})
	t.Run("respects the interval and its one second slack before recording the next attempt", func(t *testing.T) {
		t.Parallel()

		team := newHealthcheckTeam(t)
		input := configInput(team.getConfig(t))
		input.IntervalMinutes = 5
		team.saveConfig(t, input)
		team.pass(t)

		team.App.Clock.Advance(4 * time.Minute)
		team.pass(t)
		assert.Len(t, team.getAttempts(t), 1)

		team.App.Clock.Advance(59 * time.Second)
		team.pass(t)
		assert.Len(t, team.getAttempts(t), 2)

		team.App.Clock.Advance(5 * time.Minute)
		team.pass(t)
		assert.Len(t, team.getAttempts(t), 3)
	})
	t.Run("deletes attempts strictly older than retention from the time of a check", func(t *testing.T) {
		t.Parallel()

		team := newHealthcheckTeam(t)
		team.pass(t)
		team.App.Clock.Advance(7 * 24 * time.Hour)
		team.pass(t)
		assert.Len(t, team.getAttempts(t), 2)

		team.App.Clock.Advance(24 * time.Hour)
		team.pass(t)
		attempts := team.getAttempts(t)
		require.Len(t, attempts, 2)
		assert.WithinDuration(t, team.App.Clock.Now(), attempts[0].CreatedAt, time.Microsecond)
	})
	t.Run("rolls back attempts and sends no notification when a status update fails", func(t *testing.T) {
		t.Parallel()

		team := newHealthcheckTeam(t)
		messages := recordNotifications(t, team)
		input := configInput(team.getConfig(t))
		input.FailedAttemptsBeforeUnavailable = 1
		team.saveConfig(t, input)

		team.App.ExecuteSQL(
			t,
			"ALTER TABLE databases ADD CONSTRAINT ck_healthcheck_status_failure CHECK (health_status <> 'UNAVAILABLE')",
		)

		team.makeUnreachable(t)
		team.pass(t)
		assert.Empty(t, team.getAttempts(t))
		assert.Equal(t, new(apiclient.DatabaseHealthStatusAVAILABLE), team.getHealthStatus(t))
		assert.Empty(t, messages)
	})
}
