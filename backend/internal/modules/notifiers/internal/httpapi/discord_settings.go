package httpapi

import "databasus-backend/internal/modules/notifiers/internal/providers/discord"

func toDiscordSettings(body DiscordNotifierSettings) discord.Settings {
	return discord.Settings{ChannelWebhookURL: getStringOrEmpty(body.ChannelWebhookUrl)}
}

func fromDiscordSettings(discord.Settings) DiscordNotifierSettings {
	return DiscordNotifierSettings{Type: DiscordNotifierSettingsTypeDISCORD, ChannelWebhookUrl: nil}
}
