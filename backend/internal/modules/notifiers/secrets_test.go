package notifiers_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestCreateNotifier(t *testing.T) {
	t.Parallel()

	t.Run("returns the saved telegram notifier without its bot token and proxy URL", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		telegram := newTelegram("123456:AAH-SECRET-TOKEN", "")
		telegram.IsProxyEnabled = true
		telegram.ProxyUrl = optional("socks5://user:proxy-secret@proxy.example.com:1080")

		response, err := tm.owner.Client.CreateNotifierWithResponse(t.Context(), apiclient.NotifierCreate{
			WorkspaceId: tm.workspace.Id,
			Name:        "Ops chat",
			Settings:    newTelegramSettings(t, telegram),
		})

		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))
		assert.NotContains(t, string(response.Body), "AAH-SECRET-TOKEN")
		assert.NotContains(t, string(response.Body), "proxy-secret")
		assert.Equal(t, "Ops chat", response.JSON201.Name)
		assert.Equal(t, tm.workspace.Id, response.JSON201.WorkspaceId)
		assert.Equal(t, "Ops chat", getNotifier(t, tm.owner, response.JSON201.Id).Name)

		createdTelegram, err := response.JSON201.Settings.AsTelegramNotifierSettings()
		require.NoError(t, err)
		assert.Equal(t, apiclient.TelegramNotifierSettingsTypeTELEGRAM, createdTelegram.Type)
		assert.Equal(t, "123456", createdTelegram.TargetChatId)
		assert.True(t, createdTelegram.IsProxyEnabled)
	})

	t.Run("returns 400 invalid_request and creates nothing when the webhook has no URL", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)

		response, err := tm.owner.Client.CreateNotifierWithResponse(t.Context(), apiclient.NotifierCreate{
			WorkspaceId: tm.workspace.Id,
			Name:        "Hook",
			Settings:    newWebhookSettings(t, newPostWebhook("")),
		})

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest)
		assert.Empty(t, listNotifiers(t, tm.owner, tm.workspace.Id))
	})

	t.Run("returns 400 invalid_request and creates nothing when a webhook header name is not an HTTP token",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)

			for _, invalidHeaderName := range getInvalidWebhookHeaderNames() {
				response, err := tm.owner.Client.CreateNotifierWithResponse(t.Context(), apiclient.NotifierCreate{
					WorkspaceId: tm.workspace.Id,
					Name:        "Hook",
					Settings: newWebhookSettings(t, newPostWebhook("https://hooks.example.com/databasus",
						apiclient.WebhookHeader{Name: invalidHeaderName, Value: optional("payments")})),
				})

				require.NoError(t, err)
				requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
					apiclient.ProblemCodeInvalidRequest)
			}

			assert.Empty(t, listNotifiers(t, tm.owner, tm.workspace.Id))
		})

	t.Run("saves the webhook and sends the header when its name holds letters, digits and token punctuation",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)

			for _, validHeaderName := range []string{"X-Api_Key.1", "X~Team!1"} {
				webhookEndpoint := startEndpoint(t, http.StatusOK)
				createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id, newWebhookSettings(t,
					newPostWebhook(webhookEndpoint.URL,
						apiclient.WebhookHeader{Name: validHeaderName, Value: optional("payments")})))

				testResult := testNotifier(t, tm.owner, createdNotifier.Id)

				assert.True(t, testResult.IsDelivered, "%v", testResult.Error)
				webhookRequests := webhookEndpoint.getRequests()
				require.Len(t, webhookRequests, 1)
				assert.Equal(t, "payments", webhookRequests[0].Headers.Get(validHeaderName))
			}
		})
}

func TestGetNotifier(t *testing.T) {
	t.Parallel()

	t.Run("lists the webhook header names without their values and shows the URL", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id, newWebhookSettings(t,
			newPostWebhook("https://hooks.example.com/databasus",
				apiclient.WebhookHeader{Name: "Authorization", Value: optional("Bearer header-secret")}),
		))

		response, err := tm.owner.Client.GetNotifierWithResponse(t.Context(), createdNotifier.Id)

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.NotContains(t, string(response.Body), "header-secret")
		readWebhook, err := response.JSON200.Settings.AsWebhookNotifierSettings()
		require.NoError(t, err)
		assert.Equal(t, "https://hooks.example.com/databasus", readWebhook.WebhookUrl)
		assert.Equal(t, []apiclient.WebhookHeader{{Name: "Authorization", Value: nil}}, readWebhook.Headers)
	})

	t.Run("lists no headers when the webhook has none", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newWebhookSettings(t, newPostWebhook("https://hooks.example.com/databasus")))

		readNotifier := getNotifier(t, tm.owner, createdNotifier.Id)

		readWebhook, err := readNotifier.Settings.AsWebhookNotifierSettings()
		require.NoError(t, err)
		assert.Empty(t, readWebhook.Headers)
	})

	t.Run("returns the URL in clear when an older release stored it encrypted", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newWebhookSettings(t, newPostWebhook("https://hooks.example.com/placeholder")))
		tm.app.ExecuteSQL(t, "UPDATE notifiers_webhook SET webhook_url = ? WHERE notifier_id = ?",
			tm.app.EncryptSecret(t, "https://hooks.example.com/legacy"), createdNotifier.Id)

		readNotifier := getNotifier(t, tm.owner, createdNotifier.Id)

		readWebhook, err := readNotifier.Settings.AsWebhookNotifierSettings()
		require.NoError(t, err)
		assert.Equal(t, "https://hooks.example.com/legacy", readWebhook.WebhookUrl)
	})

	t.Run("lists the notifier with an empty URL when its stored URL cannot be decrypted", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newWebhookSettings(t, newPostWebhook("https://hooks.example.com/placeholder")))
		tm.app.ExecuteSQL(t, "UPDATE notifiers_webhook SET webhook_url = 'enc:not-a-ciphertext' WHERE notifier_id = ?",
			createdNotifier.Id)

		listedNotifiers := listNotifiers(t, tm.owner, tm.workspace.Id)

		require.Len(t, listedNotifiers, 1)
		listedWebhook, err := listedNotifiers[0].Settings.AsWebhookNotifierSettings()
		require.NoError(t, err)
		assert.Empty(t, listedWebhook.WebhookUrl)
	})
}

func TestUpdateNotifier(t *testing.T) {
	t.Parallel()

	t.Run("keeps the stored slack bot token when a rename leaves it out", func(t *testing.T) {
		t.Parallel()

		slackAPI := startEndpoint(t, http.StatusOK)
		tm := newTeam(t, apptest.WithNotifierEndpoints(apptest.NotifierEndpoints{
			SlackPostMessageURL: slackAPI.URL,
			TelegramBotAPIURL:   "http://telegram.invalid",
		}))
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id, newSlackSettings(t,
			apiclient.SlackNotifierSettings{
				Type:         apiclient.SlackNotifierSettingsTypeSLACK,
				BotToken:     optional("xoxb-stored-token"),
				TargetChatId: "C0123",
			}))

		response := updateNotifier(t, tm.owner, createdNotifier.Id, apiclient.NotifierUpdate{
			Name: "Renamed",
			Settings: newSlackSettings(t, apiclient.SlackNotifierSettings{
				Type:         apiclient.SlackNotifierSettingsTypeSLACK,
				BotToken:     nil,
				TargetChatId: "C0123",
			}),
		})
		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

		testResult := testNotifier(t, tm.owner, createdNotifier.Id)

		assert.Equal(t, "Renamed", getNotifier(t, tm.owner, createdNotifier.Id).Name)
		assert.True(t, testResult.IsDelivered, "%v", testResult.Error)
		slackRequests := slackAPI.getRequests()
		require.Len(t, slackRequests, 1)
		assert.Equal(t, "Bearer xoxb-stored-token", slackRequests[0].Headers.Get("Authorization"))
	})

	t.Run("keeps the stored value of a webhook header sent without one", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		webhookEndpoint := startEndpoint(t, http.StatusOK)
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id, newWebhookSettings(t,
			newPostWebhook(webhookEndpoint.URL,
				apiclient.WebhookHeader{Name: "Authorization", Value: optional("Bearer saved-value")}),
		))

		response := updateNotifier(t, tm.owner, createdNotifier.Id, apiclient.NotifierUpdate{
			Name: "Renamed",
			Settings: newWebhookSettings(t, newPostWebhook(webhookEndpoint.URL,
				apiclient.WebhookHeader{Name: "Authorization", Value: nil})),
		})
		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

		testNotifier(t, tm.owner, createdNotifier.Id)

		webhookRequests := webhookEndpoint.getRequests()
		require.Len(t, webhookRequests, 1)
		assert.Equal(t, "Bearer saved-value", webhookRequests[0].Headers.Get("Authorization"))
	})

	t.Run("returns 400 invalid_request and keeps the stored headers when a webhook header name is not an HTTP token",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			webhookEndpoint := startEndpoint(t, http.StatusOK)
			createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id, newWebhookSettings(
				t,
				newPostWebhook(
					webhookEndpoint.URL,
					apiclient.WebhookHeader{Name: "X-Team", Value: optional("payments")},
				),
			))

			for _, invalidHeaderName := range getInvalidWebhookHeaderNames() {
				updateResponse := updateNotifier(t, tm.owner, createdNotifier.Id, apiclient.NotifierUpdate{
					Name: createdNotifier.Name,
					Settings: newWebhookSettings(t, newPostWebhook(webhookEndpoint.URL,
						apiclient.WebhookHeader{Name: invalidHeaderName, Value: optional("changed")})),
				})
				requireProblem(t, updateResponse.StatusCode(), updateResponse.Body, http.StatusBadRequest,
					apiclient.ProblemCodeInvalidRequest)
			}

			testResult := testNotifier(t, tm.owner, createdNotifier.Id)

			assert.True(t, testResult.IsDelivered, "%v", testResult.Error)
			webhookRequests := webhookEndpoint.getRequests()
			require.Len(t, webhookRequests, 1)
			assert.Equal(t, "payments", webhookRequests[0].Headers.Get("X-Team"))
		})

	t.Run("returns 400 invalid_request when a change sends back a stored invalid header name", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newWebhookSettings(t, newPostWebhook("https://hooks.example.com/databasus")))
		tm.app.ExecuteSQL(t, "UPDATE notifiers_webhook SET headers = jsonb_build_array(jsonb_build_object("+
			"'key', 'X Team', 'value', '')) WHERE notifier_id = ?", createdNotifier.Id)

		readWebhook, err := getNotifier(t, tm.owner, createdNotifier.Id).Settings.AsWebhookNotifierSettings()
		require.NoError(t, err)
		require.Equal(t, []apiclient.WebhookHeader{{Name: "X Team", Value: nil}}, readWebhook.Headers)

		updateResponse := updateNotifier(t, tm.owner, createdNotifier.Id, apiclient.NotifierUpdate{
			Name:     createdNotifier.Name,
			Settings: newWebhookSettings(t, readWebhook),
		})

		requireProblem(t, updateResponse.StatusCode(), updateResponse.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest)
	})

	t.Run("removes a webhook header the change leaves out", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		webhookEndpoint := startEndpoint(t, http.StatusOK)
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id, newWebhookSettings(t,
			newPostWebhook(webhookEndpoint.URL, apiclient.WebhookHeader{Name: "X-Team", Value: optional("payments")}),
		))

		response := updateNotifier(t, tm.owner, createdNotifier.Id, apiclient.NotifierUpdate{
			Name:     createdNotifier.Name,
			Settings: newWebhookSettings(t, newPostWebhook(webhookEndpoint.URL)),
		})
		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

		testNotifier(t, tm.owner, createdNotifier.Id)

		webhookRequests := webhookEndpoint.getRequests()
		require.Len(t, webhookRequests, 1)
		assert.Empty(t, webhookRequests[0].Headers.Get("X-Team"))
	})

	t.Run("clears the stored proxy URL when the proxy is turned off", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		telegram := newTelegram("123456:AAH-TOKEN", "")
		telegram.IsProxyEnabled = true
		telegram.ProxyUrl = optional("socks5://proxy.example.com:1080")
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id, newTelegramSettings(t, telegram))

		response := updateNotifier(t, tm.owner, createdNotifier.Id, apiclient.NotifierUpdate{
			Name:     createdNotifier.Name,
			Settings: newTelegramSettings(t, newTelegram("123456:AAH-TOKEN", "")),
		})
		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

		var storedProxyURLs []string
		tm.app.QuerySQL(
			t,
			&storedProxyURLs,
			"SELECT COALESCE(proxy_url, '') FROM notifiers_telegram WHERE notifier_id = ?",
			createdNotifier.Id,
		)
		assert.Equal(t, []string{""}, storedProxyURLs)
	})

	t.Run("returns 400 invalid_request and keeps the slack notifier when it gets telegram settings",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id, newSlackSettings(t,
				apiclient.SlackNotifierSettings{
					Type:         apiclient.SlackNotifierSettingsTypeSLACK,
					BotToken:     optional("xoxb-token"),
					TargetChatId: "C0123",
				}))

			response := updateNotifier(t, tm.owner, createdNotifier.Id, apiclient.NotifierUpdate{
				Name:     "Now telegram",
				Settings: newTelegramSettings(t, newTelegram("123456:AAH-TOKEN", "")),
			})

			requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
				apiclient.ProblemCodeInvalidRequest)
			storedNotifier := getNotifier(t, tm.owner, createdNotifier.Id)
			assert.Equal(t, createdNotifier.Name, storedNotifier.Name)
			storedSlack, err := storedNotifier.Settings.AsSlackNotifierSettings()
			require.NoError(t, err)
			assert.Equal(t, apiclient.SlackNotifierSettingsTypeSLACK, storedSlack.Type)
			assert.Equal(t, "C0123", storedSlack.TargetChatId)
		})

	t.Run("returns 400 invalid_request and keeps the SMTP host when the change empties it", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		email := apiclient.EmailNotifierSettings{
			Type:                 apiclient.EmailNotifierSettingsTypeEMAIL,
			TargetEmail:          "ops@example.com",
			SmtpHost:             "smtp.example.com",
			SmtpPort:             587,
			SmtpUser:             nil,
			SmtpPassword:         nil,
			From:                 nil,
			IsInsecureSkipVerify: false,
			Security:             apiclient.SmtpSecuritySTARTTLS,
			HeloName:             nil,
		}
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id, toSettings(t,
			func(settings *apiclient.NotifierSettings) error { return settings.FromEmailNotifierSettings(email) }))
		email.SmtpHost = ""

		response := updateNotifier(t, tm.owner, createdNotifier.Id, apiclient.NotifierUpdate{
			Name: createdNotifier.Name,
			Settings: toSettings(t, func(settings *apiclient.NotifierSettings) error {
				return settings.FromEmailNotifierSettings(email)
			}),
		})

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest)
		storedEmail, err := getNotifier(t, tm.owner, createdNotifier.Id).Settings.AsEmailNotifierSettings()
		require.NoError(t, err)
		assert.Equal(t, "smtp.example.com", storedEmail.SmtpHost)
	})
}
