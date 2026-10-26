package upgrade

import (
	"context"
	"fmt"
	"maps"

	"databasus-backend/e2e/upgrade/upgradecheck"
)

// This file speaks the notifier API of the previous release, like previous_release_client_test.go.

const databaseNotifiersKind = "database notifiers"

// previousNotifierRequest carries the settings of one notifier type, in the field the previous
// release names after that type.
type previousNotifierRequest struct {
	WorkspaceID        string                     `json:"workspaceId"`
	Name               string                     `json:"name"`
	NotifierType       string                     `json:"notifierType"`
	EmailNotifier      *previousEmailRequest      `json:"emailNotifier,omitempty"`
	TelegramNotifier   *previousTelegramRequest   `json:"telegramNotifier,omitempty"`
	SlackNotifier      *previousSlackRequest      `json:"slackNotifier,omitempty"`
	DiscordNotifier    *previousDiscordRequest    `json:"discordNotifier,omitempty"`
	TeamsNotifier      *previousTeamsRequest      `json:"teamsNotifier,omitempty"`
	MattermostNotifier *previousMattermostRequest `json:"mattermostNotifier,omitempty"`
	WebhookNotifier    *previousWebhookRequest    `json:"webhookNotifier,omitempty"`
}

type previousEmailRequest struct {
	TargetEmail          string `json:"targetEmail"`
	SMTPHost             string `json:"smtpHost"`
	SMTPPort             int    `json:"smtpPort"`
	SMTPUser             string `json:"smtpUser"`
	SMTPPassword         string `json:"smtpPassword"`
	From                 string `json:"from"`
	IsInsecureSkipVerify bool   `json:"isInsecureSkipVerify"`
	Security             string `json:"security"`
	HeloName             string `json:"heloName"`
}

type previousTelegramRequest struct {
	BotToken       string `json:"botToken"`
	TargetChatID   string `json:"targetChatId"`
	ThreadID       int64  `json:"threadId"`
	IsProxyEnabled bool   `json:"isProxyEnabled"`
	ProxyURL       string `json:"proxyUrl"`
	BotAPIURL      string `json:"botApiUrl"`
}

type previousSlackRequest struct {
	BotToken     string `json:"botToken"`
	TargetChatID string `json:"targetChatId"`
}

type previousDiscordRequest struct {
	ChannelWebhookURL string `json:"channelWebhookUrl"`
}

type previousTeamsRequest struct {
	PowerAutomateURL string `json:"powerAutomateUrl"`
}

type previousMattermostRequest struct {
	DeliveryMode         string `json:"deliveryMode"`
	WebhookURL           string `json:"webhookUrl"`
	ServerURL            string `json:"serverUrl"`
	BotToken             string `json:"botToken"`
	TargetChannelID      string `json:"targetChannelId"`
	TargetChannelName    string `json:"targetChannelName"`
	OverrideUsername     string `json:"overrideUsername"`
	OverrideIconURL      string `json:"overrideIconUrl"`
	IsInsecureSkipVerify bool   `json:"isInsecureSkipVerify"`
}

type previousWebhookRequest struct {
	WebhookURL              string                  `json:"webhookUrl"`
	WebhookMethod           string                  `json:"webhookMethod"`
	BodyTemplate            string                  `json:"bodyTemplate"`
	Headers                 []previousWebhookHeader `json:"headers"`
	AcceptNotificationTypes []string                `json:"acceptNotificationTypes"`
}

type previousWebhookHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// previousNotifierReference names a notifier a new database uses.
type previousNotifierReference struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspaceId"`
}

// writtenNotifier is a notifier the previous release saved: its request and the values the current
// code is expected to list for it. Secrets are not among the values, because the current code
// never returns them.
type writtenNotifier struct {
	request        previousNotifierRequest
	expectedValues map[string]string
}

func newWrittenNotifiers(workspaceID string) []writtenNotifier {
	return []writtenNotifier{
		{
			request: previousNotifierRequest{
				WorkspaceID: workspaceID, Name: "upgrade email", NotifierType: "EMAIL",
				EmailNotifier: &previousEmailRequest{
					TargetEmail:          "upgrade-alerts@databasus.local",
					SMTPHost:             "smtp.upgrade.invalid",
					SMTPPort:             2525,
					SMTPUser:             "upgrade-smtp-user",
					SMTPPassword:         "upgrade-smtp-password",
					From:                 "upgrade-sender@databasus.local",
					IsInsecureSkipVerify: true,
					Security:             "starttls",
					HeloName:             "upgrade.databasus.local",
				},
			},
			expectedValues: map[string]string{
				"type":                 "EMAIL",
				"targetEmail":          "upgrade-alerts@databasus.local",
				"smtpHost":             "smtp.upgrade.invalid",
				"smtpPort":             "2525",
				"smtpUser":             "upgrade-smtp-user",
				"from":                 "upgrade-sender@databasus.local",
				"isInsecureSkipVerify": "true",
				"security":             "STARTTLS",
				"heloName":             "upgrade.databasus.local",
			},
		},
		{
			request: previousNotifierRequest{
				WorkspaceID: workspaceID, Name: "upgrade telegram", NotifierType: "TELEGRAM",
				TelegramNotifier: &previousTelegramRequest{
					BotToken:       "123456:AAH-UPGRADE-TOKEN",
					TargetChatID:   "-1001234567890",
					ThreadID:       42,
					IsProxyEnabled: true,
					ProxyURL:       "socks5://proxy.upgrade.invalid:1080",
					BotAPIURL:      "https://telegram-bot-api.upgrade.invalid/telegram",
				},
			},
			expectedValues: map[string]string{
				"type":           "TELEGRAM",
				"targetChatId":   "-1001234567890",
				"threadId":       "42",
				"isProxyEnabled": "true",
				"botApiUrl":      "https://telegram-bot-api.upgrade.invalid/telegram",
			},
		},
		{
			request: previousNotifierRequest{
				WorkspaceID: workspaceID, Name: "upgrade slack", NotifierType: "SLACK",
				SlackNotifier: &previousSlackRequest{BotToken: "xoxb-upgrade-token", TargetChatID: "C0UPGRADE"},
			},
			expectedValues: map[string]string{"type": "SLACK", "targetChatId": "C0UPGRADE"},
		},
		{
			request: previousNotifierRequest{
				WorkspaceID: workspaceID, Name: "upgrade discord", NotifierType: "DISCORD",
				DiscordNotifier: &previousDiscordRequest{
					ChannelWebhookURL: "https://discord.upgrade.invalid/api/webhooks/1/token",
				},
			},
			expectedValues: map[string]string{"type": "DISCORD"},
		},
		{
			request: previousNotifierRequest{
				WorkspaceID: workspaceID, Name: "upgrade teams", NotifierType: "TEAMS",
				TeamsNotifier: &previousTeamsRequest{PowerAutomateURL: "https://teams.upgrade.invalid/workflow"},
			},
			expectedValues: map[string]string{"type": "TEAMS"},
		},
		{
			request: previousNotifierRequest{
				WorkspaceID: workspaceID, Name: "upgrade mattermost", NotifierType: "MATTERMOST",
				MattermostNotifier: &previousMattermostRequest{
					DeliveryMode:         "WEBHOOK",
					WebhookURL:           "https://mattermost.upgrade.invalid/hooks/upgrade",
					TargetChannelName:    "upgrade-alerts",
					OverrideUsername:     "Upgrade Bot",
					OverrideIconURL:      "https://mattermost.upgrade.invalid/icon.png",
					IsInsecureSkipVerify: true,
				},
			},
			expectedValues: map[string]string{
				"type":                 "MATTERMOST",
				"deliveryMode":         "WEBHOOK",
				"targetChannelName":    "upgrade-alerts",
				"overrideUsername":     "Upgrade Bot",
				"overrideIconUrl":      "https://mattermost.upgrade.invalid/icon.png",
				"isInsecureSkipVerify": "true",
			},
		},
		{
			request: previousNotifierRequest{
				WorkspaceID: workspaceID, Name: "upgrade mattermost bot", NotifierType: "MATTERMOST",
				MattermostNotifier: &previousMattermostRequest{
					DeliveryMode:    "BOT",
					ServerURL:       "https://mattermost-bot.upgrade.invalid",
					BotToken:        "upgrade-mattermost-bot-token",
					TargetChannelID: "upgradebotchannel000000001",
				},
			},
			expectedValues: map[string]string{
				"type":            "MATTERMOST",
				"deliveryMode":    "BOT",
				"serverUrl":       "https://mattermost-bot.upgrade.invalid",
				"targetChannelId": "upgradebotchannel000000001",
			},
		},
		{
			request: previousNotifierRequest{
				WorkspaceID: workspaceID, Name: "upgrade webhook", NotifierType: "WEBHOOK",
				WebhookNotifier: &previousWebhookRequest{
					WebhookURL:    "http://upgrade-webhook.invalid/hook",
					WebhookMethod: "POST",
					BodyTemplate:  `{"text":"{{heading}}: {{message}}"}`,
					Headers: []previousWebhookHeader{
						{Key: "X-Upgrade-Token", Value: "upgrade-header-secret"},
						{Key: "X-Upgrade-Source", Value: "previous-release"},
					},
					AcceptNotificationTypes: []string{"BACKUP_FAILED", "HEALTHCHECK_FAILED"},
				},
			},
			expectedValues: map[string]string{
				"type":               "WEBHOOK",
				"webhookUrl":         "http://upgrade-webhook.invalid/hook",
				"webhookMethod":      "POST",
				"bodyTemplate":       `{"text":"{{heading}}: {{message}}"}`,
				"headers":            "X-Upgrade-Token,X-Upgrade-Source",
				"acceptedCategories": "BACKUP_FAILED,HEALTHCHECK_FAILED",
			},
		},
	}
}

// createNotifiers saves one notifier of every type with every setting filled in, and returns the
// items to find after the upgrade together with the references a new database uses them by.
func createNotifiers(
	ctx context.Context,
	api *apiClient,
	workspaceID string,
) ([]upgradecheck.Item, []previousNotifierReference, error) {
	writtenNotifiers := newWrittenNotifiers(workspaceID)
	notifierItems := make([]upgradecheck.Item, 0, len(writtenNotifiers))
	notifierReferences := make([]previousNotifierReference, 0, len(writtenNotifiers))

	for _, notifier := range writtenNotifiers {
		var createdNotifier previousCreatedItem

		err := api.post(ctx, "/notifiers", notifier.request, &createdNotifier)
		if err != nil {
			return nil, nil, fmt.Errorf("create the notifier %q: %w", notifier.request.Name, err)
		}

		notifierValues := map[string]string{"id": createdNotifier.ID}
		maps.Copy(notifierValues, notifier.expectedValues)

		notifierItems = append(notifierItems, upgradecheck.Item{
			Kind: "notifier", Name: notifier.request.Name, Values: notifierValues,
		})
		notifierReferences = append(notifierReferences, previousNotifierReference{
			ID: createdNotifier.ID, WorkspaceID: workspaceID,
		})
	}

	return notifierItems, notifierReferences, nil
}

// newDatabaseNotifiersItem expects the database to use every notifier, each listed by its name with
// its ID.
func newDatabaseNotifiersItem(notifierItems []upgradecheck.Item) upgradecheck.Item {
	usedNotifiers := make(map[string]string, len(notifierItems))
	for _, notifierItem := range notifierItems {
		usedNotifiers[notifierItem.Name] = notifierItem.Values["id"]
	}

	return upgradecheck.Item{Kind: databaseNotifiersKind, Name: databaseName, Values: usedNotifiers}
}
