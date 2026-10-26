package httpapi

import "databasus-backend/internal/modules/notifiers/internal/providers/telegram"

func toTelegramSettings(body TelegramNotifierSettings) telegram.Settings {
	return telegram.Settings{
		BotToken:       getStringOrEmpty(body.BotToken),
		TargetChatID:   body.TargetChatId,
		ThreadID:       body.ThreadId,
		IsProxyEnabled: body.IsProxyEnabled,
		ProxyURL:       getStringOrEmpty(body.ProxyUrl),
		BotAPIURL:      getStringOrEmpty(body.BotApiUrl),
	}
}

func fromTelegramSettings(settings telegram.Settings) TelegramNotifierSettings {
	threadID := settings.ThreadID
	if threadID != nil && *threadID == 0 {
		threadID = nil
	}

	return TelegramNotifierSettings{
		Type:           TelegramNotifierSettingsTypeTELEGRAM,
		BotToken:       nil,
		TargetChatId:   settings.TargetChatID,
		ThreadId:       threadID,
		IsProxyEnabled: settings.IsProxyEnabled,
		ProxyUrl:       nil,
		BotApiUrl:      toOptionalString(settings.BotAPIURL),
	}
}
