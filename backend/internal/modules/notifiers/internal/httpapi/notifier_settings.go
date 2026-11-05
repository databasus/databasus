package httpapi

import (
	"fmt"

	"databasus-backend/internal/modules/notifiers/internal/providers"
	"databasus-backend/internal/modules/notifiers/internal/providers/discord"
	"databasus-backend/internal/modules/notifiers/internal/providers/email"
	"databasus-backend/internal/modules/notifiers/internal/providers/mattermost"
	"databasus-backend/internal/modules/notifiers/internal/providers/slack"
	"databasus-backend/internal/modules/notifiers/internal/providers/teams"
	"databasus-backend/internal/modules/notifiers/internal/providers/telegram"
	"databasus-backend/internal/modules/notifiers/internal/providers/webhook"
)

// toProviderSettings picks the provider's conversion by the discriminator of the settings. A
// provider cannot convert itself, because it would import the generated types of this package.
func toProviderSettings(body NotifierSettings) (providers.Settings, error) {
	settingsBody, err := body.ValueByDiscriminator()
	if err != nil {
		return nil, errUnknownSettingsValue
	}

	switch typedBody := settingsBody.(type) {
	case EmailNotifierSettings:
		return toEmailSettings(typedBody), nil
	case TelegramNotifierSettings:
		return toTelegramSettings(typedBody), nil
	case SlackNotifierSettings:
		return toSlackSettings(typedBody), nil
	case DiscordNotifierSettings:
		return toDiscordSettings(typedBody), nil
	case TeamsNotifierSettings:
		return toTeamsSettings(typedBody), nil
	case MattermostNotifierSettings:
		return toMattermostSettings(typedBody), nil
	case WebhookNotifierSettings:
		return toWebhookSettings(typedBody), nil
	}

	return nil, errUnknownSettingsValue
}

// fromProviderSettings leaves every secret out, because the contract marks them writeOnly.
func fromProviderSettings(settings providers.Settings) (NotifierSettings, error) {
	var body NotifierSettings

	var err error

	switch typedSettings := settings.(type) {
	case email.Settings:
		err = body.FromEmailNotifierSettings(fromEmailSettings(typedSettings))
	case telegram.Settings:
		err = body.FromTelegramNotifierSettings(fromTelegramSettings(typedSettings))
	case slack.Settings:
		err = body.FromSlackNotifierSettings(fromSlackSettings(typedSettings))
	case discord.Settings:
		err = body.FromDiscordNotifierSettings(fromDiscordSettings(typedSettings))
	case teams.Settings:
		err = body.FromTeamsNotifierSettings(fromTeamsSettings(typedSettings))
	case mattermost.Settings:
		err = body.FromMattermostNotifierSettings(fromMattermostSettings(typedSettings))
	case webhook.Settings:
		err = body.FromWebhookNotifierSettings(fromWebhookSettings(typedSettings))
	default:
		return NotifierSettings{}, fmt.Errorf("%w: %T", errUnknownSettingsType, settings)
	}

	if err != nil {
		return NotifierSettings{}, fmt.Errorf("encode the notifier settings: %w", err)
	}

	return body, nil
}
