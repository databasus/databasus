package notifiers_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestTelegramBotAPIURL(t *testing.T) {
	t.Parallel()

	t.Run("returns the bot API URL in clear and clears it when a change leaves it out", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		telegram := newTelegram("123456:AAH-TOKEN", "https://gateway.example.com/telegram")
		telegram.IsProxyEnabled = true
		telegram.ProxyUrl = optional("socks5://user:password@proxy.example.com:1080")
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id, newTelegramSettings(t, telegram))

		readTelegram, err := getNotifier(t, tm.owner, createdNotifier.Id).Settings.AsTelegramNotifierSettings()
		require.NoError(t, err)
		assert.Equal(t, optional("https://gateway.example.com/telegram"), readTelegram.BotApiUrl)
		assert.Nil(t, readTelegram.BotToken)
		assert.Nil(t, readTelegram.ProxyUrl)

		readTelegram.BotApiUrl = nil
		readTelegram.BotToken = optional("123456:AAH-TOKEN")
		response := updateNotifier(t, tm.owner, createdNotifier.Id, apiclient.NotifierUpdate{
			Name:     createdNotifier.Name,
			Settings: newTelegramSettings(t, readTelegram),
		})
		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

		updatedTelegram, err := getNotifier(t, tm.owner, createdNotifier.Id).Settings.AsTelegramNotifierSettings()
		require.NoError(t, err)
		assert.Nil(t, updatedTelegram.BotApiUrl)
	})

	t.Run("refuses an unusable bot API URL and keeps the stored notifier", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		createdNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newTelegramSettings(t, newTelegram("123456:AAH-TOKEN", "https://gateway.example.com/telegram")))

		for _, unusableBotAPIURL := range []string{
			"ftp://telegram.example.com",
			"telegram-bot-api:8081",
			"https://user:password@gateway.example.com",
			"https://gateway.example.com/telegram?key=1",
			"https://gateway.example.com/#bot",
		} {
			createResponse, err := tm.owner.Client.CreateNotifierWithResponse(t.Context(), apiclient.NotifierCreate{
				WorkspaceId: tm.workspace.Id,
				Name:        "Unusable",
				Settings:    newTelegramSettings(t, newTelegram("123456:AAH-TOKEN", unusableBotAPIURL)),
			})
			require.NoError(t, err)
			requireProblem(t, createResponse.StatusCode(), createResponse.Body, http.StatusBadRequest,
				apiclient.ProblemCodeInvalidRequest)

			updateResponse := updateNotifier(t, tm.owner, createdNotifier.Id, apiclient.NotifierUpdate{
				Name:     createdNotifier.Name,
				Settings: newTelegramSettings(t, newTelegram("", unusableBotAPIURL)),
			})
			requireProblem(t, updateResponse.StatusCode(), updateResponse.Body, http.StatusBadRequest,
				apiclient.ProblemCodeInvalidRequest)
		}

		storedTelegram, err := getNotifier(t, tm.owner, createdNotifier.Id).Settings.AsTelegramNotifierSettings()
		require.NoError(t, err)
		assert.Equal(t, optional("https://gateway.example.com/telegram"), storedTelegram.BotApiUrl)
		assert.Len(t, listNotifiers(t, tm.owner, tm.workspace.Id), 1)
	})

	t.Run("accepts a plain http bot API URL with a port or a path", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)

		for _, botAPIURL := range []string{
			"http://localhost:8081",
			"http://telegram-bot-api:8081",
			"https://gateway.example.com/telegram/",
		} {
			response, err := tm.owner.Client.CreateNotifierWithResponse(t.Context(), apiclient.NotifierCreate{
				WorkspaceId: tm.workspace.Id,
				Name:        botAPIURL,
				Settings:    newTelegramSettings(t, newTelegram("123456:AAH-TOKEN", botAPIURL)),
			})
			require.NoError(t, err)
			require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))
		}
	})
}
