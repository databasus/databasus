package httpapi

import "databasus-backend/internal/modules/notifiers/internal/providers/mattermost"

func toMattermostSettings(body MattermostNotifierSettings) mattermost.Settings {
	return mattermost.Settings{
		DeliveryMode:         mattermost.DeliveryMode(body.DeliveryMode),
		WebhookURL:           getStringOrEmpty(body.WebhookUrl),
		ServerURL:            getStringOrEmpty(body.ServerUrl),
		BotToken:             getStringOrEmpty(body.BotToken),
		TargetChannelName:    getStringOrEmpty(body.TargetChannelName),
		TargetChannelID:      getStringOrEmpty(body.TargetChannelId),
		OverrideUsername:     getStringOrEmpty(body.OverrideUsername),
		OverrideIconURL:      getStringOrEmpty(body.OverrideIconUrl),
		IsInsecureSkipVerify: body.IsInsecureSkipVerify,
	}
}

func fromMattermostSettings(settings mattermost.Settings) MattermostNotifierSettings {
	return MattermostNotifierSettings{
		Type:                 MattermostNotifierSettingsTypeMATTERMOST,
		DeliveryMode:         MattermostNotifierSettingsDeliveryMode(settings.DeliveryMode),
		WebhookUrl:           nil,
		ServerUrl:            toOptionalString(settings.ServerURL),
		BotToken:             nil,
		TargetChannelName:    toOptionalString(settings.TargetChannelName),
		TargetChannelId:      toOptionalString(settings.TargetChannelID),
		OverrideUsername:     toOptionalString(settings.OverrideUsername),
		OverrideIconUrl:      toOptionalString(settings.OverrideIconURL),
		IsInsecureSkipVerify: settings.IsInsecureSkipVerify,
	}
}
