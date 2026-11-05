// Package slack sends notifications as messages of a Slack bot through chat.postMessage.
package slack

import "databasus-backend/internal/modules/notifiers/internal/notification"

type Settings struct {
	// BotToken is a secret.
	BotToken string
	// TargetChatID is a channel ID, starting with C, G or D, or a user ID, starting with U.
	TargetChatID string
}

func (Settings) GetType() notification.NotifierType {
	return notification.NotifierTypeSlack
}
