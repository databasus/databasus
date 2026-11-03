package verification_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

func (team verificationTeam) runScheduler(t *testing.T) {
	t.Helper()
	team.App.RunJobPass(t, "verification_scheduler")
}

func (team verificationTeam) enableSchedule(t *testing.T, scheduleType apiclient.VerificationScheduleType) {
	t.Helper()
	config := configInput(team.config(t))
	config.IsAutomaticVerificationEnabled = true
	config.ScheduleType = scheduleType
	config.Schedule = apiclient.Schedule{
		Type:           apiclient.ScheduleTypeHOURLY,
		CronExpression: nil,
		DayOfMonth:     nil,
		TimeOfDay:      nil,
		Weekday:        nil,
	}
	team.saveConfig(t, config)
}

func TestVerificationScheduler(t *testing.T) {
	t.Parallel()

	t.Run(
		"uses the newest completed backup and keeps one interval verification when the schedule is due",
		func(t *testing.T) {
			t.Parallel()

			team := newVerificationTeam(t)
			team.backup(t)
			team.App.Clock.Advance(time.Second)
			latest := team.backup(t)

			team.enableSchedule(t, apiclient.VerificationScheduleTypeINTERVAL)
			team.runScheduler(t)
			team.runScheduler(t)

			verifications := team.list(t)
			require.Len(t, verifications, 1)
			assert.Equal(t, latest.Id, verifications[0].BackupId)
			assert.Equal(t, apiclient.VerificationTriggerSCHEDULED, verifications[0].Trigger)
		},
	)

	t.Run("creates no verification when the stored cron expression has no date to run on", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		team.backup(t)
		team.enableSchedule(t, apiclient.VerificationScheduleTypeINTERVAL)
		team.App.ExecuteSQL(t, `UPDATE verification_configs
			SET interval_type = 'CRON', cron_expression = '0 0 31 2 *' WHERE database_id = ?`, team.Database.Id)

		team.runScheduler(t)

		assert.Empty(t, team.list(t))
	})

	t.Run("keeps manual work and creates scheduled work when an interval is due", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		backup := team.backup(t)
		manual := team.start(t, backup.Id)

		team.enableSchedule(t, apiclient.VerificationScheduleTypeINTERVAL)
		team.runScheduler(t)

		assert.Len(t, team.list(t), 2)
		assert.Equal(t, apiclient.VerificationStatusPENDING, team.read(t, manual.Id).Status)
	})
	t.Run("waits for the next interval after a user cancels scheduled work", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		team.backup(t)
		team.enableSchedule(t, apiclient.VerificationScheduleTypeINTERVAL)
		team.runScheduler(t)
		verifications := team.list(t)
		require.Len(t, verifications, 1)

		canceled, err := team.Owner.Client.CancelVerificationWithResponse(t.Context(), verifications[0].Id)
		require.NoError(t, err)
		require.NotNil(t, canceled.JSON200)

		team.runScheduler(t)
		assert.Len(t, team.list(t), 1)

		team.App.Clock.Advance(time.Hour)
		team.runScheduler(t)
		assert.Len(t, team.list(t), 2)
	})
	t.Run("does not create interval work when the config runs after backups", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		team.backup(t)
		team.enableSchedule(t, apiclient.VerificationScheduleTypeAFTERBACKUP)
		team.runScheduler(t)
		assert.Empty(t, team.list(t))
	})
	t.Run("fails without retry when pending work waits longer than a day", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		backup := team.backup(t)
		verification := team.start(t, backup.Id)

		team.App.Clock.Advance(24*time.Hour + time.Second)
		team.runScheduler(t)

		persisted := team.read(t, verification.Id)
		assert.Equal(t, apiclient.VerificationStatusFAILED, persisted.Status)
		assert.Equal(t, 1, persisted.AttemptCount)
		assert.NotNil(t, persisted.FinishedAt)
	})
	t.Run("requeues running work and refuses the token when its agent is deleted", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		backup := team.backup(t)
		verification := team.start(t, backup.Id)
		agent := team.App.CreateVerificationAgent(t, team.Admin)
		claim(t, team, agent)

		deleted, err := team.Admin.Client.DeleteVerificationAgentWithResponse(t.Context(), agent.Agent.ID)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, deleted.StatusCode())
		assert.Empty(t, team.App.ListVerificationAgents(t, team.Admin))
		status, _ := newProtocolAgent(t, team.App, agent).recordHeartbeat(agent.Agent.ID, heartbeatBody(nil))
		assert.Equal(t, http.StatusUnauthorized, status)

		team.runScheduler(t)
		persisted := team.read(t, verification.Id)
		assert.Equal(t, apiclient.VerificationStatusPENDING, persisted.Status)
		assert.Equal(t, 2, persisted.AttemptCount)

		team.App.Clock.Advance(24*time.Hour + time.Second)
		team.runScheduler(t)
		assert.Equal(t, apiclient.VerificationStatusFAILED, team.read(t, verification.Id).Status)
	})
	for _, attempts := range []int{1, 3} {
		t.Run(
			"requeues or fails at its retry limit when the running agent goes silent with attempts "+string(
				rune('0'+attempts),
			),
			func(t *testing.T) {
				t.Parallel()

				team := newVerificationTeam(t)
				backup := team.backup(t)
				verification := team.start(t, backup.Id)
				agent := team.App.CreateVerificationAgent(t, team.Admin)

				claim(t, team, agent)

				team.App.ExecuteSQL(
					t,
					"UPDATE verification_runs SET attempt_count=? WHERE id=?",
					attempts,
					verification.Id,
				)

				team.App.Clock.Advance(5*time.Minute + time.Second)
				team.runScheduler(t)

				persisted := team.read(t, verification.Id)
				if attempts < 3 {
					assert.Equal(t, apiclient.VerificationStatusPENDING, persisted.Status)
					assert.Equal(t, 2, persisted.AttemptCount)
				} else {
					assert.Equal(t, apiclient.VerificationStatusFAILED, persisted.Status)
				}
			},
		)
	}
	for _, trigger := range []string{"SCHEDULED", "AFTER_BACKUP"} {
		for _, status := range []string{"PENDING", "RUNNING"} {
			t.Run("cancels automatic "+trigger+" "+status+" work when its config is disabled", func(t *testing.T) {
				t.Parallel()

				team := newVerificationTeam(t)
				backup := team.backup(t)
				verification := team.start(t, backup.Id)
				team.saveConfig(t, configInput(team.config(t)))

				team.App.ExecuteSQL(
					t,
					"UPDATE verification_runs SET trigger=?, status=? WHERE id=?",
					trigger,
					status,
					verification.Id,
				)

				team.runScheduler(t)
				assert.Equal(t, apiclient.VerificationStatusCANCELED, team.read(t, verification.Id).Status)
			})
		}
	}
	for _, status := range []string{"PENDING", "RUNNING"} {
		t.Run("keeps manual "+status+" work when its config is disabled", func(t *testing.T) {
			t.Parallel()

			team := newVerificationTeam(t)
			backup := team.backup(t)
			verification := team.start(t, backup.Id)

			team.App.ExecuteSQL(t, "UPDATE verification_runs SET status=? WHERE id=?", status, verification.Id)
			team.runScheduler(t)
			assert.Equal(t, apiclient.VerificationStatus(status), team.read(t, verification.Id).Status)
		})
	}
}

func TestAfterBackupVerification(t *testing.T) {
	t.Parallel()

	t.Run("supersedes pending automatic work and keeps manual work when a new backup completes", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		team.enableSchedule(t, apiclient.VerificationScheduleTypeAFTERBACKUP)
		first := team.backup(t)
		team.App.WaitForBackupRuns(t)
		firstVerifications := team.list(t)
		require.Len(t, firstVerifications, 1)

		manual := team.start(t, first.Id)
		team.App.Clock.Advance(time.Second)
		latest := team.backup(t)
		team.App.WaitForBackupRuns(t)

		verifications := team.list(t)
		require.Len(t, verifications, 3)
		assert.Equal(t, apiclient.VerificationStatusCANCELED, team.read(t, firstVerifications[0].Id).Status)
		assert.Equal(t, apiclient.VerificationStatusPENDING, team.read(t, manual.Id).Status)

		var pendingAutomatic []apiclient.Verification
		for _, verification := range verifications {
			if verification.Trigger == apiclient.VerificationTriggerAFTERBACKUP &&
				verification.Status == apiclient.VerificationStatusPENDING {
				pendingAutomatic = append(pendingAutomatic, verification)
			}
		}
		require.Len(t, pendingAutomatic, 1)
		assert.Equal(t, latest.Id, pendingAutomatic[0].BackupId)
	})
	for _, isEnabled := range []bool{false, true} {
		t.Run(
			"creates no after-backup work when the config is disabled or interval based with enabled="+map[bool]string{false: "false", true: "true"}[isEnabled],
			func(t *testing.T) {
				t.Parallel()

				team := newVerificationTeam(t)
				config := configInput(team.config(t))
				config.IsAutomaticVerificationEnabled = isEnabled
				config.ScheduleType = apiclient.VerificationScheduleTypeINTERVAL
				team.saveConfig(t, config)

				team.backup(t)
				team.App.WaitForBackupRuns(t)

				assert.Empty(t, team.list(t))
			},
		)
	}
}
