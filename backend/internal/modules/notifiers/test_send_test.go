package notifiers_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

const telegramBotToken = "123456:AAH-TEST-TOKEN"

func TestTestNotifier(t *testing.T) {
	t.Parallel()

	t.Run("reports the test message as delivered when a viewer tests a working webhook", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		viewer := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
		webhookEndpoint := startEndpoint(t, http.StatusOK)
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newWebhookSettings(t, newPostWebhook(webhookEndpoint.URL)))

		testResult := testNotifier(t, viewer, createdNotifier.Id)

		assert.True(t, testResult.IsDelivered)
		assert.Nil(t, testResult.Error)
		assert.Len(t, webhookEndpoint.getRequests(), 1)
	})

	t.Run("reports the channel's error and records it when the webhook answers 500", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		webhookEndpoint := startEndpoint(t, http.StatusInternalServerError)
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newWebhookSettings(t, newPostWebhook(webhookEndpoint.URL)))

		testResult := testNotifier(t, tm.owner, createdNotifier.Id)

		assert.False(t, testResult.IsDelivered)
		require.NotNil(t, testResult.Error)
		assert.Contains(t, *testResult.Error, "500")
		lastSendError := getNotifier(t, tm.owner, createdNotifier.Id).LastSendError
		require.NotNil(t, lastSendError)
		assert.Equal(t, *testResult.Error, *lastSendError)
	})

	t.Run("reports a slack rate limit at once when retry-after does not fit a test send", func(t *testing.T) {
		t.Parallel()

		slackAPI := startRateLimitedSlack(t, "60")
		tm := newTeam(t, apptest.WithNotifierEndpoints(apptest.NotifierEndpoints{SlackPostMessageURL: slackAPI.URL}))
		slackNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id, newSlackNotifierSettings(t))
		testedAt := time.Now()

		testResult := testNotifier(t, tm.owner, slackNotifier.Id)

		assert.False(t, testResult.IsDelivered)
		require.NotNil(t, testResult.Error)
		assert.Contains(t, *testResult.Error, "rate-limit")
		assert.Less(t, time.Since(testedAt), 20*time.Second)

		lastSendError := getNotifier(t, tm.owner, slackNotifier.Id).LastSendError
		require.NotNil(t, lastSendError)
		assert.Contains(t, *lastSendError, "rate-limit")
	})

	t.Run("ends a slack test send at its own limit when the endpoint never answers", func(t *testing.T) {
		t.Parallel()

		silentSlack := startSilentEndpoint(t)
		tm := newTeam(t, apptest.WithNotificationDeliveryTimeout(time.Second),
			apptest.WithNotifierEndpoints(apptest.NotifierEndpoints{SlackPostMessageURL: silentSlack.URL}))
		slackNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id, newSlackNotifierSettings(t))
		testedAt := time.Now()

		testResult := testNotifier(t, tm.owner, slackNotifier.Id)

		assert.False(t, testResult.IsDelivered)
		assert.Less(t, time.Since(testedAt), 20*time.Second)
	})

	t.Run("delivers the telegram message under the path prefix of the bot API URL", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		botAPI := startEndpoint(t, http.StatusOK)
		telegram := newTelegram(telegramBotToken, botAPI.URL+"/telegram/")
		telegram.TargetChatId = "-1001234567890"
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id, newTelegramSettings(t, telegram))

		testResult := testNotifier(t, tm.owner, createdNotifier.Id)

		assert.True(t, testResult.IsDelivered, "%v", testResult.Error)
		botAPIRequests := botAPI.getRequests()
		require.Len(t, botAPIRequests, 1)
		assert.Equal(t, "/telegram/bot"+telegramBotToken+"/sendMessage", botAPIRequests[0].Path)
		assert.Equal(t, "-1001234567890", getFormValue(t, botAPIRequests[0].Body, "chat_id"))
	})

	t.Run("records the failure without the header value the webhook echoes", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		webhookEndpoint := startEndpointAnswering(t, http.StatusInternalServerError,
			"rejected Authorization: Bearer header-secret")
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id, newWebhookSettings(t,
			newPostWebhook(webhookEndpoint.URL,
				apiclient.WebhookHeader{Name: "Authorization", Value: optional("Bearer header-secret")})))

		testResult := testNotifier(t, tm.owner, createdNotifier.Id)

		require.NotNil(t, testResult.Error)
		assert.Contains(t, *testResult.Error, "rejected Authorization: redacted")
		lastSendError := getNotifier(t, tm.owner, createdNotifier.Id).LastSendError
		require.NotNil(t, lastSendError)
		assert.NotContains(t, *lastSendError, "header-secret")
	})

	t.Run("records the failure without the discord webhook URL the API echoes", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		discordAPI := startEndpointAnswering(t, http.StatusBadRequest, "unknown webhook {path}")
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			toSettings(t, func(settings *apiclient.NotifierSettings) error {
				return settings.FromDiscordNotifierSettings(apiclient.DiscordNotifierSettings{
					Type:              apiclient.DiscordNotifierSettingsTypeDISCORD,
					ChannelWebhookUrl: optional(discordAPI.URL + "/api/webhooks/1/discord-secret"),
				})
			}))

		testResult := testNotifier(t, tm.owner, createdNotifier.Id)

		require.NotNil(t, testResult.Error)
		lastSendError := getNotifier(t, tm.owner, createdNotifier.Id).LastSendError
		require.NotNil(t, lastSendError)
		assert.NotContains(t, *lastSendError, "discord-secret")
	})

	t.Run("returns 404 not_found and sends nothing when a user tests a notifier of another team",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			stranger := tm.stranger(t)
			webhookEndpoint := startEndpoint(t, http.StatusOK)
			createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
				newWebhookSettings(t, newPostWebhook(webhookEndpoint.URL)))

			response, err := stranger.Client.TestNotifierWithResponse(t.Context(), createdNotifier.Id)

			require.NoError(t, err)
			requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound,
				apiclient.ProblemCodeNotFound)
			assert.Empty(t, webhookEndpoint.getRequests())
		})
}

func TestTestNotifierSettings(t *testing.T) {
	t.Parallel()

	t.Run("sends to the entered discord webhook and creates no notifier", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		discordWebhook := startEndpoint(t, http.StatusNoContent)

		testResult := testSettings(t, tm.owner, apiclient.NotifierSettingsTest{
			WorkspaceId: tm.workspace.Id,
			NotifierId:  nil,
			Settings: toSettings(t, func(settings *apiclient.NotifierSettings) error {
				return settings.FromDiscordNotifierSettings(apiclient.DiscordNotifierSettings{
					Type:              apiclient.DiscordNotifierSettingsTypeDISCORD,
					ChannelWebhookUrl: optional(discordWebhook.URL),
				})
			}),
		})

		assert.True(t, testResult.IsDelivered, "%v", testResult.Error)
		assert.Len(t, discordWebhook.getRequests(), 1)
		assert.Empty(t, listNotifiers(t, tm.owner, tm.workspace.Id))
	})

	t.Run("sends to the edited chat with the stored bot token and keeps the saved chat", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		botAPI := startEndpoint(t, http.StatusOK)
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newTelegramSettings(t, newTelegram(telegramBotToken, botAPI.URL)))
		editedTelegram := newTelegram("", botAPI.URL)
		editedTelegram.TargetChatId = "777"

		testResult := testSettings(t, tm.owner, apiclient.NotifierSettingsTest{
			WorkspaceId: tm.workspace.Id,
			NotifierId:  &createdNotifier.Id,
			Settings:    newTelegramSettings(t, editedTelegram),
		})

		assert.True(t, testResult.IsDelivered, "%v", testResult.Error)
		botAPIRequests := botAPI.getRequests()
		require.Len(t, botAPIRequests, 1)
		assert.Equal(t, "/bot"+telegramBotToken+"/sendMessage", botAPIRequests[0].Path)
		assert.Equal(t, "777", getFormValue(t, botAPIRequests[0].Body, "chat_id"))

		storedTelegram, err := getNotifier(t, tm.owner, createdNotifier.Id).Settings.AsTelegramNotifierSettings()
		require.NoError(t, err)
		assert.Equal(t, "123456", storedTelegram.TargetChatId)
		assert.Nil(t, getNotifier(t, tm.owner, createdNotifier.Id).LastSendError)
	})

	t.Run("reports a failed send without writing its error to the named notifier", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		botAPI := startEndpoint(t, http.StatusInternalServerError)
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newTelegramSettings(t, newTelegram(telegramBotToken, botAPI.URL)))

		testResult := testSettings(t, tm.owner, apiclient.NotifierSettingsTest{
			WorkspaceId: tm.workspace.Id,
			NotifierId:  &createdNotifier.Id,
			Settings:    newTelegramSettings(t, newTelegram("", botAPI.URL)),
		})

		assert.False(t, testResult.IsDelivered)
		assert.NotNil(t, testResult.Error)
		assert.Len(t, botAPI.getRequests(), 1)
		assert.Nil(t, getNotifier(t, tm.owner, createdNotifier.Id).LastSendError)
	})

	t.Run("returns 404 not_found and sends nothing when a user tests settings in a workspace of another team",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			stranger := tm.stranger(t)
			webhookEndpoint := startEndpoint(t, http.StatusOK)

			response, err := stranger.Client.TestNotifierSettingsWithResponse(
				t.Context(),
				apiclient.NotifierSettingsTest{
					WorkspaceId: tm.workspace.Id,
					NotifierId:  nil,
					Settings:    newWebhookSettings(t, newPostWebhook(webhookEndpoint.URL)),
				},
			)

			require.NoError(t, err)
			requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound,
				apiclient.ProblemCodeNotFound)
			assert.Empty(t, webhookEndpoint.getRequests())
		})

	t.Run("returns 403 permission_denied and sends nothing when a viewer tests settings with a saved notifier's id",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			viewer := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
			viewerEndpoint := startEndpoint(t, http.StatusOK)
			createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id, newWebhookSettings(t,
				newPostWebhook("https://hooks.example.com/saved",
					apiclient.WebhookHeader{Name: "Authorization", Value: optional("Bearer saved-value")})))

			response, err := viewer.Client.TestNotifierSettingsWithResponse(t.Context(),
				apiclient.NotifierSettingsTest{
					WorkspaceId: tm.workspace.Id,
					NotifierId:  &createdNotifier.Id,
					Settings: newWebhookSettings(t, newPostWebhook(viewerEndpoint.URL,
						apiclient.WebhookHeader{Name: "Authorization", Value: nil})),
				})

			require.NoError(t, err)
			requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
				apiclient.ProblemCodePermissionDenied)
			assert.Empty(t, viewerEndpoint.getRequests())
		})

	t.Run("returns 403 permission_denied when a viewer tests new settings", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		viewer := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
		viewerEndpoint := startEndpoint(t, http.StatusOK)

		response, err := viewer.Client.TestNotifierSettingsWithResponse(t.Context(), apiclient.NotifierSettingsTest{
			WorkspaceId: tm.workspace.Id,
			NotifierId:  nil,
			Settings:    newWebhookSettings(t, newPostWebhook(viewerEndpoint.URL)),
		})

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		assert.Empty(t, viewerEndpoint.getRequests())
	})

	t.Run("reports the outcome when a member tests unsaved settings", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
		webhookEndpoint := startEndpoint(t, http.StatusOK)

		testResult := testSettings(t, member, apiclient.NotifierSettingsTest{
			WorkspaceId: tm.workspace.Id,
			NotifierId:  nil,
			Settings:    newWebhookSettings(t, newPostWebhook(webhookEndpoint.URL)),
		})

		assert.True(t, testResult.IsDelivered, "%v", testResult.Error)
		assert.Len(t, webhookEndpoint.getRequests(), 1)
	})

	t.Run("returns 404 not_found and sends nothing when the named notifier is of another workspace",
		func(t *testing.T) {
			t.Parallel()

			slackAPI := startEndpoint(t, http.StatusOK)
			tm := newTeam(t, apptest.WithNotifierEndpoints(apptest.NotifierEndpoints{
				SlackPostMessageURL: slackAPI.URL,
				TelegramBotAPIURL:   "http://telegram.invalid",
			}))
			stranger := tm.stranger(t)
			strangerWorkspace := apptest.CreateWorkspace(t, stranger, "Stranger "+uuid.NewString()[:8])
			slack := apiclient.SlackNotifierSettings{
				Type:         apiclient.SlackNotifierSettingsTypeSLACK,
				BotToken:     optional("xoxb-team-token"),
				TargetChatId: "C0123",
			}
			teamSlackNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id, newSlackSettings(t, slack))
			slack.BotToken = nil

			response, err := stranger.Client.TestNotifierSettingsWithResponse(t.Context(),
				apiclient.NotifierSettingsTest{
					WorkspaceId: strangerWorkspace.Id,
					NotifierId:  &teamSlackNotifier.Id,
					Settings:    newSlackSettings(t, slack),
				})

			require.NoError(t, err)
			requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound,
				apiclient.ProblemCodeNotFound)
			assert.Empty(t, slackAPI.getRequests())
		})

	t.Run("returns 400 invalid_request and sends nothing when a webhook header name is not an HTTP token",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			webhookEndpoint := startEndpoint(t, http.StatusOK)

			for _, invalidHeaderName := range getInvalidWebhookHeaderNames() {
				response, err := tm.owner.Client.TestNotifierSettingsWithResponse(t.Context(),
					apiclient.NotifierSettingsTest{
						WorkspaceId: tm.workspace.Id,
						NotifierId:  nil,
						Settings: newWebhookSettings(t, newPostWebhook(webhookEndpoint.URL,
							apiclient.WebhookHeader{Name: invalidHeaderName, Value: optional("payments")})),
					})

				require.NoError(t, err)
				requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
					apiclient.ProblemCodeInvalidRequest)
			}

			assert.Empty(t, webhookEndpoint.getRequests())
		})

	t.Run("returns 400 invalid_request and sends nothing when the bot API URL is unusable", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		botAPI := startEndpoint(t, http.StatusOK)
		botAPIHost := strings.TrimPrefix(botAPI.URL, "http://")

		for _, unusableBotAPIURL := range []string{
			"ftp://" + botAPIHost + "/telegram",
			"http://user:password@" + botAPIHost + "/telegram",
		} {
			response, err := tm.owner.Client.TestNotifierSettingsWithResponse(t.Context(),
				apiclient.NotifierSettingsTest{
					WorkspaceId: tm.workspace.Id,
					NotifierId:  nil,
					Settings:    newTelegramSettings(t, newTelegram(telegramBotToken, unusableBotAPIURL)),
				})

			require.NoError(t, err)
			requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
				apiclient.ProblemCodeInvalidRequest)
		}

		assert.Empty(t, botAPI.getRequests())
	})

	t.Run("sends the telegram message through the proxy when the proxy is on", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		proxy := startEndpoint(t, http.StatusOK)
		telegram := newTelegram(telegramBotToken, "http://telegram-bot-api.invalid:8081/telegram")
		telegram.IsProxyEnabled = true
		telegram.ProxyUrl = optional(proxy.URL)

		testResult := testSettings(t, tm.owner, apiclient.NotifierSettingsTest{
			WorkspaceId: tm.workspace.Id,
			NotifierId:  nil,
			Settings:    newTelegramSettings(t, telegram),
		})

		assert.True(t, testResult.IsDelivered, "%v", testResult.Error)
		proxyRequests := proxy.getRequests()
		require.Len(t, proxyRequests, 1)
		assert.Equal(t, "http://telegram-bot-api.invalid:8081/telegram/bot"+telegramBotToken+"/sendMessage",
			proxyRequests[0].RequestURI)
	})

	t.Run("names the host without the bot token when the bot API cannot be reached", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		closedServer := httptest.NewServer(http.NotFoundHandler())
		closedServer.Close()

		testResult := testSettings(t, tm.owner, apiclient.NotifierSettingsTest{
			WorkspaceId: tm.workspace.Id,
			NotifierId:  nil,
			Settings:    newTelegramSettings(t, newTelegram(telegramBotToken, closedServer.URL+"/telegram")),
		})

		assert.False(t, testResult.IsDelivered)
		require.NotNil(t, testResult.Error)
		assert.Contains(t, *testResult.Error, strings.TrimPrefix(closedServer.URL, "http://"))
		assert.NotContains(t, *testResult.Error, "AAH-TEST-TOKEN")
	})

	t.Run("keeps the body without the bot token when the bot API echoes the request path", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		botAPI := startEndpointAnswering(t, http.StatusNotFound, "Cannot POST {path}")

		testResult := testSettings(t, tm.owner, apiclient.NotifierSettingsTest{
			WorkspaceId: tm.workspace.Id,
			NotifierId:  nil,
			Settings:    newTelegramSettings(t, newTelegram(telegramBotToken, botAPI.URL+"/telegram")),
		})

		require.NotNil(t, testResult.Error)
		assert.Contains(t, *testResult.Error, "Cannot POST /telegram/botredacted/sendMessage")
		assert.NotContains(t, *testResult.Error, "AAH-TEST-TOKEN")
	})
}

func TestUpgradedNotifierSecrets(t *testing.T) {
	t.Parallel()

	t.Run("sends the decrypted header value an older release stored encrypted", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		webhookEndpoint := startEndpoint(t, http.StatusOK)
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newWebhookSettings(t, newPostWebhook(webhookEndpoint.URL)))
		tm.app.ExecuteSQL(t, "UPDATE notifiers_webhook SET headers = jsonb_build_array(jsonb_build_object("+
			"'key', 'Authorization', 'value', ?::text)) WHERE notifier_id = ?",
			tm.app.EncryptSecret(t, "Bearer saved-before-upgrade"), createdNotifier.Id)

		testResult := testNotifier(t, tm.owner, createdNotifier.Id)

		assert.True(t, testResult.IsDelivered, "%v", testResult.Error)
		webhookRequests := webhookEndpoint.getRequests()
		require.Len(t, webhookRequests, 1)
		assert.Equal(t, "Bearer saved-before-upgrade", webhookRequests[0].Headers.Get("Authorization"))
	})

	t.Run("sends with the bot token an older release stored encrypted", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		botAPI := startEndpoint(t, http.StatusOK)
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newTelegramSettings(t, newTelegram("placeholder:TOKEN", botAPI.URL)))
		tm.app.ExecuteSQL(t, "UPDATE notifiers_telegram SET bot_token = ? WHERE notifier_id = ?",
			tm.app.EncryptSecret(t, telegramBotToken), createdNotifier.Id)

		testResult := testNotifier(t, tm.owner, createdNotifier.Id)

		assert.True(t, testResult.IsDelivered, "%v", testResult.Error)
		botAPIRequests := botAPI.getRequests()
		require.Len(t, botAPIRequests, 1)
		assert.Equal(t, "/bot"+telegramBotToken+"/sendMessage", botAPIRequests[0].Path)
	})
}

func testSettings(
	t *testing.T,
	session apptest.Session,
	settingsTest apiclient.NotifierSettingsTest,
) apiclient.NotifierTestResult {
	t.Helper()

	response, err := session.Client.TestNotifierSettingsWithResponse(t.Context(), settingsTest)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return *response.JSON200
}

func getFormValue(t *testing.T, body, name string) string {
	t.Helper()

	form, err := url.ParseQuery(body)
	require.NoError(t, err)

	return form.Get(name)
}
