package notifiers_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

const (
	storedBotToken         = "123456:AAH-STORED-TOKEN"
	mattermostChannelID    = "abcdefghijklmnopqrstuvwxyz"
	storedSMTPPassword     = "stored-smtp-password"
	storedAuthorization    = "Bearer stored-value"
	storedWebhookURL       = "https://hooks.example.com/saved"
	storedBotAPIURL        = "https://gateway.example.com/telegram"
	storedProxyURL         = "socks5://proxy.example.com:1080"
	storedMattermostServer = "https://mattermost.example.com"
)

// addressChangeCase is a saved notifier and a change of its resource address that leaves the secret
// out. The moved settings point at movedAddress when the address is a URL.
type addressChangeCase struct {
	name         string
	newStored    func(t *testing.T) apiclient.NotifierSettings
	newMoved     func(t *testing.T, movedAddress string) apiclient.NotifierSettings
	refusedField string
}

func newEmail(host string, port int, password string) apiclient.EmailNotifierSettings {
	return apiclient.EmailNotifierSettings{
		Type:                 apiclient.EmailNotifierSettingsTypeEMAIL,
		TargetEmail:          "ops@example.com",
		SmtpHost:             host,
		SmtpPort:             port,
		SmtpUser:             optional("mailer"),
		SmtpPassword:         optional(password),
		From:                 nil,
		IsInsecureSkipVerify: false,
		Security:             apiclient.SmtpSecuritySTARTTLS,
		HeloName:             nil,
	}
}

func newEmailSettings(t *testing.T, email apiclient.EmailNotifierSettings) apiclient.NotifierSettings {
	t.Helper()

	return toSettings(t, func(settings *apiclient.NotifierSettings) error {
		return settings.FromEmailNotifierSettings(email)
	})
}

func newBotMattermost(serverURL, botToken string) apiclient.MattermostNotifierSettings {
	return apiclient.MattermostNotifierSettings{
		Type:                 apiclient.MattermostNotifierSettingsTypeMATTERMOST,
		DeliveryMode:         apiclient.MattermostNotifierSettingsDeliveryModeBOT,
		WebhookUrl:           nil,
		ServerUrl:            optional(serverURL),
		BotToken:             optional(botToken),
		TargetChannelName:    nil,
		TargetChannelId:      optional(mattermostChannelID),
		OverrideUsername:     nil,
		OverrideIconUrl:      nil,
		IsInsecureSkipVerify: false,
	}
}

func newMattermostSettings(
	t *testing.T,
	mattermost apiclient.MattermostNotifierSettings,
) apiclient.NotifierSettings {
	t.Helper()

	return toSettings(t, func(settings *apiclient.NotifierSettings) error {
		return settings.FromMattermostNotifierSettings(mattermost)
	})
}

func newProxiedTelegram(botToken, proxyURL string) apiclient.TelegramNotifierSettings {
	telegram := newTelegram(botToken, storedBotAPIURL)
	telegram.IsProxyEnabled = true
	telegram.ProxyUrl = optional(proxyURL)

	return telegram
}

func newStoredWebhook(t *testing.T) apiclient.NotifierSettings {
	t.Helper()

	return newWebhookSettings(t, newPostWebhook(storedWebhookURL,
		apiclient.WebhookHeader{Name: "Authorization", Value: optional(storedAuthorization)}))
}

func listAddressChangeCases() []addressChangeCase {
	newStoredEmail := func(t *testing.T) apiclient.NotifierSettings {
		t.Helper()

		return newEmailSettings(t, newEmail("smtp.example.com", 587, storedSMTPPassword))
	}
	newStoredTelegram := func(t *testing.T) apiclient.NotifierSettings {
		t.Helper()

		return newTelegramSettings(t, newTelegram(storedBotToken, storedBotAPIURL))
	}
	newStoredProxiedTelegram := func(t *testing.T) apiclient.NotifierSettings {
		t.Helper()

		return newTelegramSettings(t, newProxiedTelegram(storedBotToken, storedProxyURL))
	}

	return []addressChangeCase{
		{
			name:      "the smtp host",
			newStored: newStoredEmail,
			newMoved: func(t *testing.T, _ string) apiclient.NotifierSettings {
				t.Helper()

				return newEmailSettings(t, newEmail("moved.example.com", 587, ""))
			},
			refusedField: "settings.smtpPassword",
		},
		{
			name:      "the smtp port",
			newStored: newStoredEmail,
			newMoved: func(t *testing.T, _ string) apiclient.NotifierSettings {
				t.Helper()

				return newEmailSettings(t, newEmail("smtp.example.com", 2525, ""))
			},
			refusedField: "settings.smtpPassword",
		},
		{
			name:      "the telegram bot api url",
			newStored: newStoredTelegram,
			newMoved: func(t *testing.T, movedAddress string) apiclient.NotifierSettings {
				t.Helper()

				return newTelegramSettings(t, newTelegram("", movedAddress))
			},
			refusedField: "settings.botToken",
		},
		{
			name:      "a telegram proxy url in the request",
			newStored: newStoredProxiedTelegram,
			newMoved: func(t *testing.T, movedAddress string) apiclient.NotifierSettings {
				t.Helper()

				return newTelegramSettings(t, newProxiedTelegram("", movedAddress))
			},
			refusedField: "settings.botToken",
		},
		{
			name:      "the telegram proxy turned on",
			newStored: newStoredTelegram,
			newMoved: func(t *testing.T, movedAddress string) apiclient.NotifierSettings {
				t.Helper()

				return newTelegramSettings(t, newProxiedTelegram("", movedAddress))
			},
			refusedField: "settings.botToken",
		},
		{
			name:      "the telegram proxy turned off",
			newStored: newStoredProxiedTelegram,
			newMoved: func(t *testing.T, _ string) apiclient.NotifierSettings {
				t.Helper()

				return newTelegramSettings(t, newTelegram("", storedBotAPIURL))
			},
			refusedField: "settings.botToken",
		},
		{
			name: "the mattermost server url in the bot mode",
			newStored: func(t *testing.T) apiclient.NotifierSettings {
				t.Helper()

				return newMattermostSettings(t, newBotMattermost(storedMattermostServer, storedBotToken))
			},
			newMoved: func(t *testing.T, movedAddress string) apiclient.NotifierSettings {
				t.Helper()

				return newMattermostSettings(t, newBotMattermost(movedAddress, ""))
			},
			refusedField: "settings.botToken",
		},
		{
			name:      "the webhook url",
			newStored: newStoredWebhook,
			newMoved: func(t *testing.T, movedAddress string) apiclient.NotifierSettings {
				t.Helper()

				return newWebhookSettings(t, newPostWebhook(movedAddress,
					apiclient.WebhookHeader{Name: "Authorization", Value: nil}))
			},
			refusedField: "settings.headers[0].value",
		},
		{
			name:      "a trailing slash on the webhook url",
			newStored: newStoredWebhook,
			newMoved: func(t *testing.T, _ string) apiclient.NotifierSettings {
				t.Helper()

				return newWebhookSettings(t, newPostWebhook(storedWebhookURL+"/",
					apiclient.WebhookHeader{Name: "Authorization", Value: nil}))
			},
			refusedField: "settings.headers[0].value",
		},
	}
}

func marshalSettings(t *testing.T, settings apiclient.NotifierSettings) string {
	t.Helper()

	marshaledSettings, err := json.Marshal(settings)
	require.NoError(t, err)

	return string(marshaledSettings)
}

func TestChangeResourceAddress(t *testing.T) {
	t.Parallel()

	for _, changeCase := range listAddressChangeCases() {
		t.Run("returns 400 invalid_request and keeps the notifier when a member changes "+changeCase.name+
			" without the secret", func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
			movedEndpoint := startEndpoint(t, http.StatusOK)
			createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id, changeCase.newStored(t))
			settingsBefore := marshalSettings(t, getNotifier(t, member, createdNotifier.Id).Settings)

			response := updateNotifier(t, member, createdNotifier.Id, apiclient.NotifierUpdate{
				Name:     "Moved",
				Settings: changeCase.newMoved(t, movedEndpoint.URL),
			})

			requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
				apiclient.ProblemCodeInvalidRequest)
			requireFieldError(t, response.Body, changeCase.refusedField)
			storedNotifier := getNotifier(t, member, createdNotifier.Id)
			assert.Equal(t, createdNotifier.Name, storedNotifier.Name)
			assert.Equal(t, settingsBefore, marshalSettings(t, storedNotifier.Settings))
		})

		t.Run("returns 400 invalid_request and sends nothing when a member tests "+changeCase.name+
			" without the secret", func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
			movedEndpoint := startEndpoint(t, http.StatusOK)
			createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id, changeCase.newStored(t))

			response, err := member.Client.TestNotifierSettingsWithResponse(t.Context(),
				apiclient.NotifierSettingsTest{
					WorkspaceId: tm.workspace.Id,
					NotifierId:  &createdNotifier.Id,
					Settings:    changeCase.newMoved(t, movedEndpoint.URL),
				})

			require.NoError(t, err)
			requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
				apiclient.ProblemCodeInvalidRequest)
			requireFieldError(t, response.Body, changeCase.refusedField)
			assert.Empty(t, movedEndpoint.getRequests())
		})
	}

	t.Run("returns 400 invalid_request when the owner changes the telegram bot api url without the token",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			movedEndpoint := startEndpoint(t, http.StatusOK)
			createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
				newTelegramSettings(t, newTelegram(storedBotToken, storedBotAPIURL)))

			response := updateNotifier(t, tm.owner, createdNotifier.Id, apiclient.NotifierUpdate{
				Name:     createdNotifier.Name,
				Settings: newTelegramSettings(t, newTelegram("", movedEndpoint.URL)),
			})

			requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
				apiclient.ProblemCodeInvalidRequest)
			requireFieldError(t, response.Body, "settings.botToken")
			storedTelegram, err := getNotifier(t, tm.owner, createdNotifier.Id).Settings.AsTelegramNotifierSettings()
			require.NoError(t, err)
			assert.Equal(t, optional(storedBotAPIURL), storedTelegram.BotApiUrl)
		})

	t.Run("keeps the stored proxy url when the proxy is turned off without the token", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newTelegramSettings(t, newProxiedTelegram(storedBotToken, storedProxyURL)))

		response := updateNotifier(t, tm.owner, createdNotifier.Id, apiclient.NotifierUpdate{
			Name:     createdNotifier.Name,
			Settings: newTelegramSettings(t, newTelegram("", storedBotAPIURL)),
		})

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest)
		storedTelegram, err := getNotifier(t, tm.owner, createdNotifier.Id).Settings.AsTelegramNotifierSettings()
		require.NoError(t, err)
		assert.True(t, storedTelegram.IsProxyEnabled)

		var storedProxyURLs []string
		tm.app.QuerySQL(t, &storedProxyURLs,
			"SELECT COALESCE(proxy_url, '') FROM notifiers_telegram WHERE notifier_id = ?", createdNotifier.Id)
		require.Len(t, storedProxyURLs, 1)
		assert.NotEmpty(t, storedProxyURLs[0])
	})

	t.Run("keeps the stored smtp password when the target email changes at the same smtp address",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
			createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
				newEmailSettings(t, newEmail("smtp.example.com", 587, storedSMTPPassword)))
			changedEmail := newEmail("smtp.example.com", 587, "")
			changedEmail.TargetEmail = "new-target@example.com"
			changedEmail.SmtpPassword = nil

			response := updateNotifier(t, member, createdNotifier.Id, apiclient.NotifierUpdate{
				Name:     createdNotifier.Name,
				Settings: newEmailSettings(t, changedEmail),
			})

			require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
			var storedPasswords []string
			tm.app.QuerySQL(t, &storedPasswords,
				"SELECT COALESCE(smtp_password, '') FROM notifiers_email WHERE notifier_id = ?", createdNotifier.Id)
			require.Len(t, storedPasswords, 1)
			assert.Equal(t, storedSMTPPassword, tm.app.DecryptSecret(t, storedPasswords[0]))
		})

	t.Run("sends through the stored proxy with the stored token when the chat changes with the proxy staying on",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
			storedProxy := startEndpoint(t, http.StatusOK)
			// A plain HTTP Bot API URL makes the proxy see the request itself instead of a CONNECT tunnel.
			proxiedTelegram := newProxiedTelegram(telegramBotToken, storedProxy.URL)
			proxiedTelegram.BotApiUrl = optional("http://telegram.invalid")
			createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
				newTelegramSettings(t, proxiedTelegram))
			changedTelegram := proxiedTelegram
			changedTelegram.BotToken = nil
			changedTelegram.ProxyUrl = nil
			changedTelegram.TargetChatId = "-100999"

			response := updateNotifier(t, member, createdNotifier.Id, apiclient.NotifierUpdate{
				Name:     createdNotifier.Name,
				Settings: newTelegramSettings(t, changedTelegram),
			})
			require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

			testNotifier(t, member, createdNotifier.Id)

			proxiedRequests := storedProxy.getRequests()
			require.NotEmpty(t, proxiedRequests)
			assert.Contains(t, proxiedRequests[0].Path, telegramBotToken)
			assert.Contains(t, proxiedRequests[0].Body, "-100999")
		})

	t.Run("still sends the stored header value after a refused change of the webhook url", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
		webhookEndpoint := startEndpoint(t, http.StatusOK)
		movedEndpoint := startEndpoint(t, http.StatusOK)
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id, newWebhookSettings(t,
			newPostWebhook(webhookEndpoint.URL,
				apiclient.WebhookHeader{Name: "Authorization", Value: optional(storedAuthorization)})))

		response := updateNotifier(t, member, createdNotifier.Id, apiclient.NotifierUpdate{
			Name: createdNotifier.Name,
			Settings: newWebhookSettings(t, newPostWebhook(movedEndpoint.URL,
				apiclient.WebhookHeader{Name: "Authorization", Value: nil})),
		})
		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest)

		testNotifier(t, member, createdNotifier.Id)

		webhookRequests := webhookEndpoint.getRequests()
		require.Len(t, webhookRequests, 1)
		assert.Equal(t, storedAuthorization, webhookRequests[0].Headers.Get("Authorization"))
		assert.Empty(t, movedEndpoint.getRequests())
	})

	t.Run("sends with the new token when it comes with the new bot api url", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
		movedBotAPI := startEndpoint(t, http.StatusOK)
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newTelegramSettings(t, newTelegram(storedBotToken, storedBotAPIURL)))

		response := updateNotifier(t, member, createdNotifier.Id, apiclient.NotifierUpdate{
			Name:     createdNotifier.Name,
			Settings: newTelegramSettings(t, newTelegram(telegramBotToken, movedBotAPI.URL)),
		})
		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

		testResult := testNotifier(t, member, createdNotifier.Id)

		assert.True(t, testResult.IsDelivered, "%v", testResult.Error)
		botAPIRequests := movedBotAPI.getRequests()
		require.Len(t, botAPIRequests, 1)
		assert.Equal(t, "/bot"+telegramBotToken+"/sendMessage", botAPIRequests[0].Path)
	})

	t.Run("keeps the stored header value when the webhook method changes at the same url", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
		webhookEndpoint := startEndpoint(t, http.StatusOK)
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id, newWebhookSettings(t,
			newPostWebhook(webhookEndpoint.URL,
				apiclient.WebhookHeader{Name: "Authorization", Value: optional(storedAuthorization)})))
		getWebhook := newPostWebhook(webhookEndpoint.URL, apiclient.WebhookHeader{Name: "Authorization", Value: nil})
		getWebhook.WebhookMethod = apiclient.WebhookNotifierSettingsWebhookMethodGET

		response := updateNotifier(t, member, createdNotifier.Id, apiclient.NotifierUpdate{
			Name:     createdNotifier.Name,
			Settings: newWebhookSettings(t, getWebhook),
		})
		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

		testNotifier(t, member, createdNotifier.Id)

		webhookRequests := webhookEndpoint.getRequests()
		require.Len(t, webhookRequests, 1)
		assert.Equal(t, storedAuthorization, webhookRequests[0].Headers.Get("Authorization"))
	})

	t.Run("accepts a webhook header without a stored value when the url changes", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		movedEndpoint := startEndpoint(t, http.StatusOK)
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id, newWebhookSettings(t,
			newPostWebhook(storedWebhookURL, apiclient.WebhookHeader{Name: "X-Empty", Value: nil})))

		response := updateNotifier(t, tm.owner, createdNotifier.Id, apiclient.NotifierUpdate{
			Name: createdNotifier.Name,
			Settings: newWebhookSettings(t, newPostWebhook(movedEndpoint.URL,
				apiclient.WebhookHeader{Name: "X-Empty", Value: nil})),
		})

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
	})

	t.Run("keeps the secrets with their own address when the notifier is renamed", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		discordWebhook := startEndpoint(t, http.StatusNoContent)
		teamsWebhook := startEndpoint(t, http.StatusOK)
		createdDiscord := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			toSettings(t, func(settings *apiclient.NotifierSettings) error {
				return settings.FromDiscordNotifierSettings(apiclient.DiscordNotifierSettings{
					Type:              apiclient.DiscordNotifierSettingsTypeDISCORD,
					ChannelWebhookUrl: optional(discordWebhook.URL),
				})
			}))
		createdTeams := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			toSettings(t, func(settings *apiclient.NotifierSettings) error {
				return settings.FromTeamsNotifierSettings(apiclient.TeamsNotifierSettings{
					Type:             apiclient.TeamsNotifierSettingsTypeTEAMS,
					PowerAutomateUrl: optional(teamsWebhook.URL),
				})
			}))

		discordResponse := updateNotifier(t, tm.owner, createdDiscord.Id, apiclient.NotifierUpdate{
			Name:     "Renamed Discord",
			Settings: getNotifier(t, tm.owner, createdDiscord.Id).Settings,
		})
		teamsResponse := updateNotifier(t, tm.owner, createdTeams.Id, apiclient.NotifierUpdate{
			Name:     "Renamed Teams",
			Settings: getNotifier(t, tm.owner, createdTeams.Id).Settings,
		})

		require.Equal(t, http.StatusOK, discordResponse.StatusCode(), string(discordResponse.Body))
		require.Equal(t, http.StatusOK, teamsResponse.StatusCode(), string(teamsResponse.Body))
		assert.True(t, testNotifier(t, tm.owner, createdDiscord.Id).IsDelivered)
		assert.True(t, testNotifier(t, tm.owner, createdTeams.Id).IsDelivered)
	})

	t.Run(
		"removes the stored bot token when the server url changes in webhook mode, and refuses the bot mode at that url",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
			webhookEndpoint := startEndpoint(t, http.StatusOK)
			movedServer := startEndpoint(t, http.StatusCreated)
			createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
				newMattermostSettings(t, newBotMattermost(storedMattermostServer, storedBotToken)))
			parkedMattermost := newBotMattermost(movedServer.URL, "")
			parkedMattermost.DeliveryMode = apiclient.MattermostNotifierSettingsDeliveryModeWEBHOOK
			parkedMattermost.WebhookUrl = optional(webhookEndpoint.URL)
			parkedMattermost.TargetChannelId = nil

			parkResponse := updateNotifier(t, member, createdNotifier.Id, apiclient.NotifierUpdate{
				Name:     createdNotifier.Name,
				Settings: newMattermostSettings(t, parkedMattermost),
			})
			require.Equal(t, http.StatusOK, parkResponse.StatusCode(), string(parkResponse.Body))

			returnResponse := updateNotifier(t, member, createdNotifier.Id, apiclient.NotifierUpdate{
				Name:     createdNotifier.Name,
				Settings: newMattermostSettings(t, newBotMattermost(movedServer.URL, "")),
			})

			requireProblem(t, returnResponse.StatusCode(), returnResponse.Body, http.StatusBadRequest,
				apiclient.ProblemCodeInvalidRequest)
			requireFieldError(t, returnResponse.Body, "settings.botToken")
			assert.Empty(t, movedServer.getRequests())
		},
	)

	t.Run("removes the stored smtp password when the host changes with the user cleared", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newEmailSettings(t, newEmail("smtp.example.com", 587, storedSMTPPassword)))
		movedEmail := newEmail("moved.example.com", 587, "")
		movedEmail.SmtpUser = nil

		response := updateNotifier(t, member, createdNotifier.Id, apiclient.NotifierUpdate{
			Name:     createdNotifier.Name,
			Settings: newEmailSettings(t, movedEmail),
		})
		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

		var storedPasswords []string
		tm.app.QuerySQL(t, &storedPasswords,
			"SELECT COALESCE(smtp_password, '') FROM notifiers_email WHERE notifier_id = ?", createdNotifier.Id)
		require.Len(t, storedPasswords, 1)
		assert.Empty(t, strings.TrimSpace(storedPasswords[0]))
	})
}
