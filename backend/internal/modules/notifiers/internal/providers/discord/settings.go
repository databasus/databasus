// Package discord sends notifications to a Discord channel through its webhook.
package discord

import "databasus-backend/internal/modules/notifiers/internal/notification"

type Settings struct {
	// ChannelWebhookURL is a secret, because the URL itself authorizes the post.
	ChannelWebhookURL string
}

func (Settings) GetType() notification.NotifierType {
	return notification.NotifierTypeDiscord
}
