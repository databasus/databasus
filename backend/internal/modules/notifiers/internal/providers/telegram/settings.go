// Package telegram sends notifications as messages of a Telegram bot, through the official Bot API
// or a Bot API server the notifier names, optionally through a proxy.
package telegram

import "databasus-backend/internal/modules/notifiers/internal/notification"

type Settings struct {
	// BotToken is a secret.
	BotToken     string
	TargetChatID string
	// ThreadID is nil, or zero, for a chat without topics.
	ThreadID       *int64
	IsProxyEnabled bool
	// ProxyURL is a secret, because it may carry the proxy's credentials. It is empty while the
	// proxy is off.
	ProxyURL string
	// BotAPIURL is empty for the official Bot API.
	BotAPIURL string
}

func (Settings) GetType() notification.NotifierType {
	return notification.NotifierTypeTelegram
}
