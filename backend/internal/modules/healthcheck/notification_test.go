package healthcheck_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func recordNotifications(
	t *testing.T,
	team healthcheckTeam,
	categories ...apiclient.NotificationCategory,
) <-chan string {
	t.Helper()
	if len(categories) == 0 {
		categories = []apiclient.NotificationCategory{apiclient.NotificationCategoryALL}
	}

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
				AcceptedCategories: categories,
			},
		),
	)

	notifier := apptest.CreateNotifier(t, team.Owner, team.Workspace.Id, settings)
	current, err := team.Owner.Client.GetDatabaseWithResponse(t.Context(), team.Database.Id)
	require.NoError(t, err)
	require.NotNil(t, current.JSON200, string(current.Body))

	updated, err := team.Owner.Client.UpdateDatabaseWithResponse(
		t.Context(),
		team.Database.Id,
		apiclient.DatabaseUpdate{
			Name:        team.Database.Name,
			NotifierIds: append(current.JSON200.NotifierIds, notifier.Id),
			Settings:    team.Database.Settings,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, updated.JSON200, string(updated.Body))
	return messages
}
