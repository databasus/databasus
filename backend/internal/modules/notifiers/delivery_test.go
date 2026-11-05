package notifiers_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	mailpit "databasus-backend/internal/lib/mailpittest"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/modules/notifiers"
)

var (
	errNoRequest        = errors.New("the endpoint has received no request")
	errNoSendError      = errors.New("the notifier has no last send error")
	errSendErrorPresent = errors.New("the notifier still has a last send error")
	errNoMail           = errors.New("the message has not arrived")
)

func getBackupFailedNotification() notifiers.Notification {
	return notifiers.Notification{
		Category: notifiers.CategoryBackupFailed,
		Heading:  "Backup failed",
		Message:  "The disk of the storage is full",
	}
}

func TestNotify(t *testing.T) {
	t.Parallel()

	t.Run("delivers to discord without waiting for a webhook that never answers, which then times out",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t, apptest.WithNotificationDeliveryTimeout(time.Second))
			silentWebhook := startSilentEndpoint(t)
			discordWebhook := startEndpoint(t, http.StatusNoContent)
			webhookNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
				newWebhookSettings(t, newPostWebhook(silentWebhook.URL)))
			discordNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
				newDiscordSettings(t, discordWebhook.URL))
			notifiedAt := time.Now()

			tm.app.Notify(t, []uuid.UUID{webhookNotifier.Id, discordNotifier.Id}, getBackupFailedNotification())

			assert.Less(t, time.Since(notifiedAt), 500*time.Millisecond)
			waitForRequest(t, discordWebhook)
			assert.Contains(t, discordWebhook.getRequests()[0].Body, getBackupFailedNotification().Message)
			lastSendError := waitForSendError(t, tm.owner, webhookNotifier.Id)
			assert.Contains(t, lastSendError, "timed out")
		})

	t.Run("retries slack when retry-after outlasts the default delivery timeout", func(t *testing.T) {
		t.Parallel()

		slackAPI := startRateLimitedSlack(t, "1")
		tm := newTeam(t, apptest.WithNotificationDeliveryTimeout(200*time.Millisecond),
			apptest.WithNotifierEndpoints(apptest.NotifierEndpoints{SlackPostMessageURL: slackAPI.URL}))
		slackNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id, newSlackNotifierSettings(t))

		tm.app.Notify(t, []uuid.UUID{slackNotifier.Id}, getBackupFailedNotification())
		tm.app.WaitForNotificationDeliveries(t)

		assert.Equal(t, int64(2), slackAPI.getRequestCount())
		assert.Nil(t, getNotifier(t, tm.owner, slackNotifier.Id).LastSendError)
	})

	t.Run("fails the slack delivery at once when retry-after does not fit", func(t *testing.T) {
		t.Parallel()

		slackAPI := startRateLimitedSlack(t, "3600")
		tm := newTeam(t, apptest.WithNotifierEndpoints(apptest.NotifierEndpoints{SlackPostMessageURL: slackAPI.URL}))
		slackNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id, newSlackNotifierSettings(t))
		notifiedAt := time.Now()

		tm.app.Notify(t, []uuid.UUID{slackNotifier.Id}, getBackupFailedNotification())

		assert.Contains(t, waitForSendError(t, tm.owner, slackNotifier.Id), "rate-limit")
		assert.Less(t, time.Since(notifiedAt), 20*time.Second)
		assert.Equal(t, int64(1), slackAPI.getRequestCount())
	})

	t.Run("fails the slack delivery when the endpoint never answers", func(t *testing.T) {
		t.Parallel()

		silentSlack := startSilentEndpoint(t)
		tm := newTeam(t, apptest.WithNotifierEndpoints(apptest.NotifierEndpoints{SlackPostMessageURL: silentSlack.URL}))
		slackNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id, newSlackNotifierSettings(t))

		tm.app.Notify(t, []uuid.UUID{slackNotifier.Id}, getBackupFailedNotification())

		assert.Contains(t, waitForSendError(t, tm.owner, slackNotifier.Id), "Timeout")
	})

	t.Run("sends the first 2000 characters of a longer message", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		webhookEndpoint := startEndpoint(t, http.StatusOK)
		webhookNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newWebhookSettings(t, newPostWebhook(webhookEndpoint.URL)))
		longMessage := strings.Repeat("0123456789", 250)

		tm.app.Notify(t, []uuid.UUID{webhookNotifier.Id}, notifiers.Notification{
			Category: notifiers.CategoryBackupSuccess,
			Heading:  "Backup completed",
			Message:  longMessage,
		})

		waitForRequest(t, webhookEndpoint)
		var webhookBody struct {
			Message string `json:"message"`
		}
		require.NoError(t, json.Unmarshal([]byte(webhookEndpoint.getRequests()[0].Body), &webhookBody))
		assert.Equal(t, longMessage[:2000], webhookBody.Message)
	})

	t.Run("delivers to the other notifier when one points at an address that refuses connections",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			webhookEndpoint := startEndpoint(t, http.StatusOK)
			refusingNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
				newWebhookSettings(t, newPostWebhook(reserveRefusingAddress(t))))
			workingNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
				newWebhookSettings(t, newPostWebhook(webhookEndpoint.URL)))

			tm.app.Notify(t, []uuid.UUID{refusingNotifier.Id, workingNotifier.Id}, notifiers.Notification{
				Category: notifiers.CategoryHealthcheckFailed,
				Heading:  "Database is unavailable",
				Message:  "The healthcheck could not connect",
			})

			waitForRequest(t, webhookEndpoint)
			waitForSendError(t, tm.owner, refusingNotifier.Id)
		})
}

func TestDeliveryOutcome(t *testing.T) {
	t.Parallel()

	t.Run("keeps the name and URL saved during a delivery and records the delivery's error", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		blockingEndpoint := startBlockingEndpoint(t, http.StatusInternalServerError)
		newEndpoint := startEndpoint(t, http.StatusOK)
		webhookNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newWebhookSettings(t, newPostWebhook(blockingEndpoint.URL)))

		tm.app.Notify(t, []uuid.UUID{webhookNotifier.Id}, getBackupFailedNotification())
		testwait.Until(t, "the delivery reaches the endpoint", func(context.Context) error {
			if blockingEndpoint.requestCount.Load() == 0 {
				return errNoRequest
			}

			return nil
		})

		updateResponse := updateNotifier(t, tm.owner, webhookNotifier.Id, apiclient.NotifierUpdate{
			Name:     "Renamed during delivery",
			Settings: newWebhookSettings(t, newPostWebhook(newEndpoint.URL)),
		})
		require.Equal(t, http.StatusOK, updateResponse.StatusCode(), string(updateResponse.Body))
		close(blockingEndpoint.release)

		lastSendError := waitForSendError(t, tm.owner, webhookNotifier.Id)

		assert.Contains(t, lastSendError, "500")
		storedNotifier := getNotifier(t, tm.owner, webhookNotifier.Id)
		assert.Equal(t, "Renamed during delivery", storedNotifier.Name)
		storedWebhook, err := storedNotifier.Settings.AsWebhookNotifierSettings()
		require.NoError(t, err)
		assert.Equal(t, newEndpoint.URL, storedWebhook.WebhookUrl)
	})

	t.Run("shows the failure after a failed delivery and no error after the next successful one",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			var endpointStatus atomic.Int64
			endpointStatus.Store(http.StatusBadGateway)
			switchingEndpoint := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(int(endpointStatus.Load()))
			}))
			t.Cleanup(switchingEndpoint.Close)

			webhookNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
				newWebhookSettings(t, newPostWebhook(switchingEndpoint.URL)))

			tm.app.Notify(t, []uuid.UUID{webhookNotifier.Id}, getBackupFailedNotification())
			failedDeliveryError := waitForSendError(t, tm.owner, webhookNotifier.Id)
			endpointStatus.Store(http.StatusOK)
			tm.app.Notify(t, []uuid.UUID{webhookNotifier.Id}, getBackupFailedNotification())

			assert.Contains(t, failedDeliveryError, "502")
			testwait.Until(t, "the successful delivery clears the error", func(context.Context) error {
				if getNotifier(t, tm.owner, webhookNotifier.Id).LastSendError != nil {
					return errSendErrorPresent
				}

				return nil
			})
		})
}

func TestDeliveryThroughEachProvider(t *testing.T) {
	t.Parallel()

	t.Run("posts the notification to the webhook", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		webhookEndpoint := startEndpoint(t, http.StatusOK)
		webhookNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newWebhookSettings(t, newPostWebhook(webhookEndpoint.URL)))

		tm.app.Notify(t, []uuid.UUID{webhookNotifier.Id}, getBackupFailedNotification())

		waitForRequest(t, webhookEndpoint)
		assert.Contains(t, webhookEndpoint.getRequests()[0].Body, getBackupFailedNotification().Heading)
		assert.Contains(t, webhookEndpoint.getRequests()[0].Body, getBackupFailedNotification().Message)
	})

	t.Run("posts the notification to the discord channel webhook", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		discordWebhook := startEndpoint(t, http.StatusNoContent)
		discordNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newDiscordSettings(t, discordWebhook.URL))

		tm.app.Notify(t, []uuid.UUID{discordNotifier.Id}, getBackupFailedNotification())

		waitForRequest(t, discordWebhook)
		assert.Contains(t, discordWebhook.getRequests()[0].Body, getBackupFailedNotification().Message)
	})

	t.Run("posts the notification to the teams workflow", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		teamsWorkflow := startEndpoint(t, http.StatusAccepted)
		teamsNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			toSettings(t, func(settings *apiclient.NotifierSettings) error {
				return settings.FromTeamsNotifierSettings(apiclient.TeamsNotifierSettings{
					Type:             apiclient.TeamsNotifierSettingsTypeTEAMS,
					PowerAutomateUrl: optional(teamsWorkflow.URL),
				})
			}))

		tm.app.Notify(t, []uuid.UUID{teamsNotifier.Id}, getBackupFailedNotification())

		waitForRequest(t, teamsWorkflow)
		assert.Contains(t, teamsWorkflow.getRequests()[0].Body, getBackupFailedNotification().Message)
	})

	t.Run("posts the notification to the mattermost incoming webhook", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		mattermostWebhook := startEndpoint(t, http.StatusOK)
		mattermostNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			toSettings(t, func(settings *apiclient.NotifierSettings) error {
				return settings.FromMattermostNotifierSettings(apiclient.MattermostNotifierSettings{
					Type:                 apiclient.MattermostNotifierSettingsTypeMATTERMOST,
					DeliveryMode:         apiclient.MattermostNotifierSettingsDeliveryModeWEBHOOK,
					WebhookUrl:           optional(mattermostWebhook.URL),
					BotToken:             nil,
					ServerUrl:            nil,
					TargetChannelId:      nil,
					TargetChannelName:    nil,
					OverrideUsername:     nil,
					OverrideIconUrl:      nil,
					IsInsecureSkipVerify: false,
				})
			}))

		tm.app.Notify(t, []uuid.UUID{mattermostNotifier.Id}, getBackupFailedNotification())

		waitForRequest(t, mattermostWebhook)
		assert.Contains(t, mattermostWebhook.getRequests()[0].Body, getBackupFailedNotification().Message)
	})

	t.Run("posts the notification to the slack channel with the bot token", func(t *testing.T) {
		t.Parallel()

		slackAPI := startEndpoint(t, http.StatusOK)
		tm := newTeam(t, apptest.WithNotifierEndpoints(apptest.NotifierEndpoints{
			SlackPostMessageURL: slackAPI.URL + "/api/chat.postMessage",
			TelegramBotAPIURL:   "",
		}))
		slackNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newSlackSettings(t, apiclient.SlackNotifierSettings{
				Type:         apiclient.SlackNotifierSettingsTypeSLACK,
				BotToken:     optional("xoxb-delivery-token"),
				TargetChatId: "C0123",
			}))

		tm.app.Notify(t, []uuid.UUID{slackNotifier.Id}, getBackupFailedNotification())

		waitForRequest(t, slackAPI)
		slackRequest := slackAPI.getRequests()[0]
		assert.Equal(t, "/api/chat.postMessage", slackRequest.Path)
		assert.Equal(t, "Bearer xoxb-delivery-token", slackRequest.Headers.Get("Authorization"))
		assert.Contains(t, slackRequest.Body, `"channel":"C0123"`)
		assert.Contains(t, slackRequest.Body, getBackupFailedNotification().Message)
	})

	t.Run("sends the notification through the official bot API when the notifier names none", func(t *testing.T) {
		t.Parallel()

		botAPI := startEndpoint(t, http.StatusOK)
		tm := newTeam(t, apptest.WithNotifierEndpoints(apptest.NotifierEndpoints{
			SlackPostMessageURL: "",
			TelegramBotAPIURL:   botAPI.URL,
		}))
		telegramNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			newTelegramSettings(t, newTelegram(telegramBotToken, "")))

		tm.app.Notify(t, []uuid.UUID{telegramNotifier.Id}, getBackupFailedNotification())

		waitForRequest(t, botAPI)
		botAPIRequest := botAPI.getRequests()[0]
		assert.Equal(t, "/bot"+telegramBotToken+"/sendMessage", botAPIRequest.Path)
		assert.Equal(t, "123456", getFormValue(t, botAPIRequest.Body, "chat_id"))
		assert.Contains(t, getFormValue(t, botAPIRequest.Body, "text"), getBackupFailedNotification().Message)
	})

	t.Run("mails the notification through the notifier's SMTP server", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		mailpitEndpoint := containers.StartMailpit(t)
		mailpitClient := mailpit.NewClient(fmt.Sprintf("%s:%d", mailpitEndpoint.HTTP.Host, mailpitEndpoint.HTTP.Port))
		emailNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id,
			toSettings(t, func(settings *apiclient.NotifierSettings) error {
				return settings.FromEmailNotifierSettings(apiclient.EmailNotifierSettings{
					Type:                 apiclient.EmailNotifierSettingsTypeEMAIL,
					TargetEmail:          "recipient@databasus.local",
					SmtpHost:             mailpitEndpoint.SMTP.Host,
					SmtpPort:             mailpitEndpoint.SMTP.Port,
					SmtpUser:             nil,
					SmtpPassword:         nil,
					From:                 optional("sender@databasus.local"),
					Security:             apiclient.SmtpSecurityNONE,
					IsInsecureSkipVerify: false,
					HeloName:             nil,
				})
			}))

		tm.app.Notify(t, []uuid.UUID{emailNotifier.Id}, getBackupFailedNotification())

		var deliveredMessages []mailpit.Message

		testwait.Until(t, "the message arrives at Mailpit", func(context.Context) error {
			messages, err := mailpitClient.FetchMessages(t.Context())
			if err != nil {
				return err
			}

			if len(messages) == 0 {
				return errNoMail
			}

			deliveredMessages = messages

			return nil
		})

		require.Len(t, deliveredMessages, 1)
		assert.Equal(t, getBackupFailedNotification().Heading, deliveredMessages[0].Subject)
		require.Len(t, deliveredMessages[0].To, 1)
		assert.Equal(t, "recipient@databasus.local", deliveredMessages[0].To[0].Address)
	})
}

func newDiscordSettings(t *testing.T, channelWebhookURL string) apiclient.NotifierSettings {
	t.Helper()

	return toSettings(t, func(settings *apiclient.NotifierSettings) error {
		return settings.FromDiscordNotifierSettings(apiclient.DiscordNotifierSettings{
			Type:              apiclient.DiscordNotifierSettingsTypeDISCORD,
			ChannelWebhookUrl: optional(channelWebhookURL),
		})
	})
}

func waitForRequest(t *testing.T, target *endpoint) {
	t.Helper()

	testwait.Until(t, "the endpoint receives the notification", func(context.Context) error {
		if len(target.getRequests()) == 0 {
			return errNoRequest
		}

		return nil
	})
}

func waitForSendError(t *testing.T, session apptest.Session, notifierID uuid.UUID) string {
	t.Helper()

	var lastSendError string

	testwait.Until(t, "the notifier records the failed delivery", func(context.Context) error {
		recordedError := getNotifier(t, session, notifierID).LastSendError
		if recordedError == nil {
			return errNoSendError
		}

		lastSendError = *recordedError

		return nil
	})

	return lastSendError
}

// blockingEndpoint holds every request until release is closed, then answers with its status. A
// request still held when the test ends gets no answer.
type blockingEndpoint struct {
	URL          string
	release      chan struct{}
	requestCount atomic.Int64
}

func startBlockingEndpoint(t *testing.T, status int) *blockingEndpoint {
	t.Helper()

	started := &blockingEndpoint{URL: "", release: make(chan struct{}), requestCount: atomic.Int64{}}
	testEnded := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		started.requestCount.Add(1)

		select {
		case <-started.release:
			writer.WriteHeader(status)
		case <-testEnded:
		}
	}))
	t.Cleanup(server.Close)
	t.Cleanup(func() { close(testEnded) })
	started.URL = server.URL

	return started
}

// rateLimitedSlack answers its first request with a rate limit and every later one with success.
type rateLimitedSlack struct {
	URL          string
	requestCount *atomic.Int64
}

func (s rateLimitedSlack) getRequestCount() int64 {
	return s.requestCount.Load()
}

func startRateLimitedSlack(t *testing.T, retryAfter string) rateLimitedSlack {
	t.Helper()

	requestCount := &atomic.Int64{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		if requestCount.Add(1) == 1 {
			writer.Header().Set("Retry-After", retryAfter)
			writer.WriteHeader(http.StatusTooManyRequests)

			return
		}

		_, _ = writer.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(server.Close)

	return rateLimitedSlack{URL: server.URL, requestCount: requestCount}
}

func newSlackNotifierSettings(t *testing.T) apiclient.NotifierSettings {
	t.Helper()

	return newSlackSettings(t, apiclient.SlackNotifierSettings{
		Type:         apiclient.SlackNotifierSettingsTypeSLACK,
		BotToken:     optional("xoxb-test-token"),
		TargetChatId: "C012AB3CD",
	})
}

// startSilentEndpoint accepts every request and never answers it until the test ends.
func startSilentEndpoint(t *testing.T) *httptest.Server {
	t.Helper()

	testEnded := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		<-testEnded
	}))
	t.Cleanup(server.Close)
	t.Cleanup(func() { close(testEnded) })

	return server
}

func reserveRefusingAddress(t *testing.T) string {
	t.Helper()

	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	address := listener.Addr().String()
	require.NoError(t, listener.Close())

	return "http://" + address
}
