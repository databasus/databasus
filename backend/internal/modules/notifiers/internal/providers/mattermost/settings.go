// Package mattermost sends notifications to Mattermost through an incoming webhook or as posts of a
// bot account.
package mattermost

import "databasus-backend/internal/modules/notifiers/internal/notification"

type DeliveryMode string

const (
	DeliveryModeWebhook DeliveryMode = "WEBHOOK"
	DeliveryModeBot     DeliveryMode = "BOT"
)

type Settings struct {
	DeliveryMode DeliveryMode
	// WebhookURL is the incoming webhook of the webhook mode, a secret because the URL itself
	// authorizes the post.
	WebhookURL string
	// ServerURL and BotToken are the server and the secret token of the bot mode.
	ServerURL string
	BotToken  string
	// TargetChannelName overrides the channel of the incoming webhook; empty keeps the webhook's own.
	TargetChannelName string
	// TargetChannelID is the 26-character channel ID the bot posts to.
	TargetChannelID      string
	OverrideUsername     string
	OverrideIconURL      string
	IsInsecureSkipVerify bool
}

func (Settings) GetType() notification.NotifierType {
	return notification.NotifierTypeMattermost
}
