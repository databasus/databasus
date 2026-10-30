package verification_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestVerificationConfig(t *testing.T) {
	t.Parallel()

	t.Run("returns a disabled after-backup default without storing a row when no config is saved", func(t *testing.T) {
		team := newVerificationTeam(t)
		config := team.config(t)
		assert.False(t, config.IsAutomaticVerificationEnabled)
		assert.Equal(t, apiclient.VerificationScheduleTypeAFTERBACKUP, config.ScheduleType)
		assert.Equal(t, apiclient.ScheduleTypeWEEKLY, config.Schedule.Type)
		assert.Equal(t, "04:00", *config.Schedule.TimeOfDay)
		assert.Equal(t, 0, *config.Schedule.Weekday)
		assert.Equal(
			t,
			[]apiclient.VerificationNotificationTrigger{apiclient.VerificationNotificationTriggerVERIFICATIONFAILED},
			config.NotificationTriggers,
		)
		var configCount int64
		team.App.QuerySQL(
			t,
			&configCount,
			"SELECT count(*) FROM verification_configs WHERE database_id = ?",
			team.Database.Id,
		)
		assert.Zero(t, configCount)
	})
	t.Run("persists all editable fields when a member saves a valid interval config", func(t *testing.T) {
		team := newVerificationTeam(t)
		config := configInput(team.config(t))
		config.IsAutomaticVerificationEnabled = true
		config.ScheduleType = apiclient.VerificationScheduleTypeINTERVAL
		config.Schedule = apiclient.Schedule{
			Type:           apiclient.ScheduleTypeWEEKLY,
			TimeOfDay:      new("09:30"),
			Weekday:        new(3),
			DayOfMonth:     nil,
			CronExpression: nil,
		}
		config.NotificationTriggers = []apiclient.VerificationNotificationTrigger{
			apiclient.VerificationNotificationTriggerVERIFICATIONSUCCESS,
			apiclient.VerificationNotificationTriggerVERIFICATIONFAILED,
		}
		saved := team.saveConfig(t, config)
		assert.Equal(t, config, configInput(saved))
		assert.Equal(t, config, configInput(team.config(t)))
	})
	t.Run("keeps the saved config when an enabled interval schedule is invalid", func(t *testing.T) {
		team := newVerificationTeam(t)
		original := team.config(t)
		config := configInput(original)
		config.IsAutomaticVerificationEnabled = true
		config.ScheduleType = apiclient.VerificationScheduleTypeINTERVAL
		config.Schedule = apiclient.Schedule{
			Type:           apiclient.ScheduleTypeCRON,
			CronExpression: new("invalid cron"),
			DayOfMonth:     nil,
			TimeOfDay:      nil,
			Weekday:        nil,
		}
		response, err := team.Owner.Client.UpdateVerificationConfigWithResponse(t.Context(), team.Database.Id, config)
		require.NoError(t, err)
		requireProblem(
			t,
			response.StatusCode(),
			response.Body,
			http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest,
		)
		assert.Contains(t, string(response.Body), "schedule")
		assert.Equal(t, original, team.config(t))
	})
	for _, scheduleType := range []apiclient.VerificationScheduleType{apiclient.VerificationScheduleTypeINTERVAL, apiclient.VerificationScheduleTypeAFTERBACKUP} {
		t.Run(
			"saves empty notifications without validating an unused schedule with "+string(scheduleType),
			func(t *testing.T) {
				team := newVerificationTeam(t)
				config := configInput(team.config(t))
				config.ScheduleType = scheduleType
				config.IsAutomaticVerificationEnabled = scheduleType == apiclient.VerificationScheduleTypeAFTERBACKUP
				config.Schedule = apiclient.Schedule{
					Type:           apiclient.ScheduleTypeCRON,
					CronExpression: new("not a cron expression"),
					DayOfMonth:     nil,
					TimeOfDay:      nil,
					Weekday:        nil,
				}
				config.NotificationTriggers = []apiclient.VerificationNotificationTrigger{}
				saved := team.saveConfig(t, config)
				assert.Empty(t, saved.NotificationTriggers)
			},
		)
	}
	t.Run("refuses unknown schedule types and notifications without storing changes", func(t *testing.T) {
		team := newVerificationTeam(t)
		original := team.config(t)
		for _, field := range []string{"scheduleType", "notificationTriggers"} {
			config := configInput(original)
			if field == "scheduleType" {
				config.ScheduleType = "UNKNOWN"
			} else {
				config.NotificationTriggers = []apiclient.VerificationNotificationTrigger{"UNKNOWN"}
			}
			response, err := team.Owner.Client.UpdateVerificationConfigWithResponse(
				t.Context(),
				team.Database.Id,
				config,
			)
			require.NoError(t, err)
			requireProblem(
				t,
				response.StatusCode(),
				response.Body,
				http.StatusBadRequest,
				apiclient.ProblemCodeInvalidRequest,
			)
			assert.Contains(t, string(response.Body), field)
		}
		assert.Equal(t, original, team.config(t))
	})
}
