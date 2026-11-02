package apptest_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/app/apptest/enginetest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

func TestTelemetryCollectionCaps(t *testing.T) {
	t.Parallel()

	t.Run("limits every array to 200 when the instance contains more entries", func(t *testing.T) {
		t.Parallel()

		receiver := &telemetryReceiver{status: http.StatusOK}
		server := httptest.NewServer(receiver)
		t.Cleanup(server.Close)

		team := enginetest.NewTeam(
			t,
			apptest.WithTelemetry(
				apptest.TelemetrySettings{
					ReceiverURL:  server.URL,
					IdentityPath: filepath.Join(t.TempDir(), "identity.json"),
				},
			),
		)

		source := containers.StartPostgres(t, "postgres:18")
		var notifierSettings apiclient.NotifierSettings
		require.NoError(t, notifierSettings.FromWebhookNotifierSettings(apiclient.WebhookNotifierSettings{
			Type:               apiclient.WebhookNotifierSettingsTypeWEBHOOK,
			WebhookMethod:      apiclient.WebhookNotifierSettingsWebhookMethodPOST,
			WebhookUrl:         "http://localhost/private-destination",
			Headers:            []apiclient.WebhookHeader{},
			AcceptedCategories: []apiclient.NotificationCategory{apiclient.NotificationCategoryALL},
		}))

		for index := range 201 {
			database := team.CreateDatabase(t, apiclient.PostgresqlLogicalSettings{
				Host:      source.Host,
				Port:      source.Port,
				Username:  containers.PostgresUsername,
				Password:  new(containers.PostgresPassword),
				Database:  new(containers.PostgresDatabase),
				SslMode:   apiclient.PostgresqlSslModeDisable,
				SshTunnel: enginetest.NewDisabledSSHTunnelSettings(),
				CpuCount:  1,
			})
			team.App.ExecuteSQL(t, "UPDATE databases SET health_status = 'AVAILABLE' WHERE id = ?", database.Id)
			team.App.CreateLocalStorage(t, team.Owner, team.Workspace.Id)
			apptest.CreateNotifier(t, team.Owner, team.Workspace.Id, notifierSettings)
			agent, err := team.Admin.Client.CreateVerificationAgentWithResponse(
				t.Context(),
				apiclient.VerificationAgentInput{Name: fmt.Sprintf("private agent %d", index)},
			)
			require.NoError(t, err)
			require.NotNil(t, agent.JSON201, string(agent.Body))
		}

		team.App.RunJobPass(t, "telemetry_ping")
		team.App.Clock.Advance(time.Minute)
		team.App.RunJobPass(t, "telemetry_ping")

		reports := receiver.getReports()
		require.Len(t, reports, 1)
		for _, collection := range []string{"databases", "storages", "notifiers", "verificationAgents"} {
			assert.Len(t, reports[0][collection], 200)
		}
		storageTypes, isStorageArray := reports[0]["storages"].([]any)
		require.True(t, isStorageArray)
		for _, storageType := range storageTypes {
			assert.Equal(t, "LOCAL", storageType)
		}
		notifierTypes, isNotifierArray := reports[0]["notifiers"].([]any)
		require.True(t, isNotifierArray)
		for _, notifierType := range notifierTypes {
			assert.Equal(t, "WEBHOOK", notifierType)
		}
	})
}
