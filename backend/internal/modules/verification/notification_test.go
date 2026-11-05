package verification_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/lib/agentclient"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func recordNotifications(t *testing.T, team verificationTeam) <-chan string {
	t.Helper()
	messages := make(chan string, 10)
	endpoint := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err == nil {
			messages <- string(body)
		}
		writer.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(endpoint.Close)

	var settings apiclient.NotifierSettings
	require.NoError(
		t,
		settings.FromWebhookNotifierSettings(
			apiclient.WebhookNotifierSettings{
				Type:               apiclient.WebhookNotifierSettingsTypeWEBHOOK,
				WebhookUrl:         endpoint.URL,
				WebhookMethod:      apiclient.WebhookNotifierSettingsWebhookMethodPOST,
				BodyTemplate:       nil,
				Headers:            []apiclient.WebhookHeader{},
				AcceptedCategories: []apiclient.NotificationCategory{apiclient.NotificationCategoryALL},
			},
		),
	)

	notifier := apptest.CreateNotifier(t, team.Owner, team.Workspace.Id, settings)
	updated, err := team.Owner.Client.UpdateDatabaseWithResponse(
		t.Context(),
		team.Database.Id,
		apiclient.DatabaseUpdate{
			Name:        team.Database.Name,
			NotifierIds: []uuid.UUID{notifier.Id},
			Settings:    team.Database.Settings,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, updated.JSON200, string(updated.Body))
	return messages
}

func TestVerificationNotifications(t *testing.T) {
	t.Parallel()

	for _, scenario := range []struct {
		name      string
		report    agentclient.VerificationReport
		isSuccess bool
		isEnabled bool
		attempts  int
	}{
		{name: "sends success after a completed report commits", report: agentclient.VerificationReport{Status: "COMPLETED"}, isSuccess: true, isEnabled: true, attempts: 1},
		{name: "sends failure when the restored database is too small", report: agentclient.VerificationReport{Status: "COMPLETED", DBSizeBytesAfterRestore: new(int64(1))}, isSuccess: false, isEnabled: true, attempts: 1},
		{name: "sends failure after the backup is rejected", report: agentclient.VerificationReport{Status: "FAILED", PgRestoreExitCode: new(1), FailMessage: new("archive corrupt")}, isSuccess: false, isEnabled: true, attempts: 1},
		{name: "sends failure at the last agent attempt", report: agentclient.VerificationReport{Status: "FAILED", FailMessage: new("container unavailable")}, isSuccess: false, isEnabled: true, attempts: 3},
		{name: "sends no failure before the last agent attempt", report: agentclient.VerificationReport{Status: "FAILED"}, isSuccess: false, isEnabled: true, attempts: 1},
		{name: "sends no failure when notifications are disabled", report: agentclient.VerificationReport{Status: "FAILED", PgRestoreExitCode: new(1)}, isSuccess: false, isEnabled: false, attempts: 1},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			team := newVerificationTeam(t)
			backup := team.backup(t)
			team.App.WaitForBackupRuns(t)
			team.App.ExecuteSQL(t, "UPDATE backups_logical SET backup_raw_db_size_mb=1 WHERE id=?", backup.Id)

			messages := recordNotifications(t, team)
			config := configInput(team.config(t))
			config.NotificationTriggers = []apiclient.VerificationNotificationTrigger{}
			if scenario.isEnabled {
				trigger := apiclient.VerificationNotificationTriggerVERIFICATIONFAILED
				if scenario.isSuccess {
					trigger = apiclient.VerificationNotificationTriggerVERIFICATIONSUCCESS
				}
				config.NotificationTriggers = append(config.NotificationTriggers, trigger)
			}
			team.saveConfig(t, config)

			verification := team.start(t, backup.Id)
			agent := team.App.CreateVerificationAgent(t, team.Admin)
			claim(t, team, agent)
			team.App.ExecuteSQL(
				t,
				"UPDATE verification_runs SET attempt_count=? WHERE id=?",
				scenario.attempts,
				verification.Id,
			)

			status, body := newProtocolAgent(t, team.App, agent).reportVerification(verification.Id, scenario.report)
			require.Equal(t, http.StatusNoContent, status, string(body))
			team.App.WaitForNotificationDeliveries(t)

			persisted := team.read(t, verification.Id)
			if scenario.isEnabled && persisted.Status != apiclient.VerificationStatusPENDING {
				require.Len(t, messages, 1)
				message := <-messages
				assert.Contains(t, message, verification.Id.String())
				assert.Contains(t, message, backup.Id.String())
				if scenario.isSuccess {
					assert.Contains(t, strings.ToLower(message), "passed")
				} else {
					assert.Contains(t, strings.ToLower(message), "failed")
				}
			} else {
				assert.Empty(t, messages)
			}
		})
	}
}

func TestVerificationSilentAgentNotification(t *testing.T) {
	t.Parallel()

	t.Run("notifies only at the final attempt when the scheduler reaps a silent agent", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		backup := team.backup(t)
		team.App.WaitForBackupRuns(t)
		messages := recordNotifications(t, team)
		team.saveConfig(t, configInput(team.config(t)))
		verification := team.start(t, backup.Id)
		agent := team.App.CreateVerificationAgent(t, team.Admin)

		for attempt := 1; attempt <= 3; attempt++ {
			claim(t, team, agent)
			team.App.Clock.Advance(6 * time.Minute)
			team.runScheduler(t)
			team.App.WaitForNotificationDeliveries(t)

			if attempt < 3 {
				assert.Empty(t, messages)
			} else {
				require.Len(t, messages, 1)
				assert.Contains(t, <-messages, verification.Id.String())
			}
		}
	})
}
