package httpapi

import "databasus-backend/internal/modules/notifiers/internal/providers/slack"

func toSlackSettings(body SlackNotifierSettings) slack.Settings {
	return slack.Settings{BotToken: getStringOrEmpty(body.BotToken), TargetChatID: body.TargetChatId}
}

func fromSlackSettings(settings slack.Settings) SlackNotifierSettings {
	return SlackNotifierSettings{
		Type:         SlackNotifierSettingsTypeSLACK,
		BotToken:     nil,
		TargetChatId: settings.TargetChatID,
	}
}
