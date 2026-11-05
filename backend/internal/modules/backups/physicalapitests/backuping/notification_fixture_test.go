package backuping_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

type webhookInbox struct {
	mutex    sync.Mutex
	messages []string
}

func (inbox *webhookInbox) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	body, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	inbox.mutex.Lock()
	inbox.messages = append(inbox.messages, string(body))
	inbox.mutex.Unlock()
	writer.WriteHeader(http.StatusNoContent)
}

func (inbox *webhookInbox) getMessages() []string {
	inbox.mutex.Lock()
	defer inbox.mutex.Unlock()
	return append([]string{}, inbox.messages...)
}

func (team *backupTeam) attachWebhook(
	t *testing.T,
	inbox *webhookInbox,
	category apiclient.NotificationCategory,
) uuid.UUID {
	t.Helper()
	server := httptest.NewServer(inbox)
	t.Cleanup(server.Close)

	var settings apiclient.NotifierSettings
	require.NoError(
		t,
		settings.FromWebhookNotifierSettings(
			apiclient.WebhookNotifierSettings{
				Type:               apiclient.WebhookNotifierSettingsTypeWEBHOOK,
				WebhookUrl:         server.URL,
				WebhookMethod:      apiclient.WebhookNotifierSettingsWebhookMethodPOST,
				Headers:            []apiclient.WebhookHeader{},
				AcceptedCategories: []apiclient.NotificationCategory{category},
			},
		),
	)

	notifier := apptest.CreateNotifier(t, team.Owner, team.Database.WorkspaceId, settings)
	team.Database.NotifierIds = append(team.Database.NotifierIds, notifier.Id)

	response, err := team.Owner.Client.UpdateDatabaseWithResponse(
		t.Context(),
		team.Database.Id,
		apiclient.DatabaseUpdate{
			Name:        team.Database.Name,
			NotifierIds: team.Database.NotifierIds,
			Settings:    team.Database.Settings,
		},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), "%s", response.Body)

	return notifier.Id
}

func (team *backupTeam) setNotifications(t *testing.T, triggers ...apiclient.NotificationTrigger) {
	t.Helper()
	config := apptest.GetBackupConfig(t, team.Owner, team.Database.Id)
	settings, err := config.Settings.AsPhysicalBackupSettings()
	require.NoError(t, err)
	settings.NotificationTriggers = triggers

	var backupSettings apiclient.BackupSettings
	require.NoError(t, backupSettings.FromPhysicalBackupSettings(settings))

	response, err := team.Owner.Client.UpdateBackupConfigWithResponse(
		t.Context(),
		team.Database.Id,
		apiclient.BackupConfigInput{
			IsBackupsEnabled: true,
			StorageId:        &team.StorageID,
			Encryption:       config.Encryption,
			Settings:         backupSettings,
		},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), "%s", response.Body)
}
