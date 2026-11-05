package mattermost

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/modules/notifiers/internal/notification"
	"databasus-backend/internal/modules/notifiers/internal/providers"
)

const (
	// maxPostLength is the longest post the REST API accepts; incoming webhooks silently split longer ones.
	maxPostLength = 16383
	// channelIDLength is the length of every Mattermost ID, which is how a pasted channel name is caught.
	channelIDLength = 26
	createPostPath  = "/api/v4/posts"
)

type Deps struct {
	Repositories   Repositories
	FieldEncryptor *crypto.FieldEncryptor
}

type Provider struct {
	repository      *settingsRepository
	fieldEncryptor  *crypto.FieldEncryptor
	verifyingClient *http.Client
	skippingClient  *http.Client
}

func New(deps Deps) *Provider {
	return &Provider{
		repository:      deps.Repositories.settings,
		fieldEncryptor:  deps.FieldEncryptor,
		verifyingClient: providers.NewHTTPClient(nil),
		skippingClient:  providers.NewHTTPClient(newTLSSkippingTransport()),
	}
}

func (p *Provider) GetType() notification.NotifierType {
	return notification.NotifierTypeMattermost
}

func (p *Provider) GetRepository() providers.SettingsRepository {
	return p.repository
}

func (p *Provider) Validate(settings providers.Settings) error {
	mattermostSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	if mattermostSettings.OverrideIconURL != "" && !isHTTPURL(mattermostSettings.OverrideIconURL) {
		return providers.NewInvalidSettingError("overrideIconUrl", "must be an http or https URL")
	}

	switch mattermostSettings.DeliveryMode {
	case DeliveryModeWebhook:
		return p.validateWebhookMode(mattermostSettings)
	case DeliveryModeBot:
		return validateBotMode(mattermostSettings)
	}

	return providers.NewInvalidSettingError(
		"deliveryMode", fmt.Sprintf("must be %s or %s", DeliveryModeWebhook, DeliveryModeBot),
	)
}

func (p *Provider) MergeStoredSecrets(merge providers.SecretMerge) (providers.Settings, error) {
	incomingSettings, err := providers.CastSettings[Settings](merge.Incoming)
	if err != nil {
		return nil, err
	}

	storedSettings, err := providers.CastSettings[Settings](merge.Stored)
	if err != nil {
		return nil, err
	}

	if incomingSettings.WebhookURL == "" {
		incomingSettings.WebhookURL = storedSettings.WebhookURL
	}

	if incomingSettings.BotToken == "" && incomingSettings.ServerURL == storedSettings.ServerURL {
		incomingSettings.BotToken = storedSettings.BotToken
	}

	return incomingSettings, nil
}

func (p *Provider) Send(ctx context.Context, settings providers.Settings, message notification.Notification) error {
	mattermostSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	postText := buildPost(message)

	switch mattermostSettings.DeliveryMode {
	case DeliveryModeWebhook:
		return p.sendViaIncomingWebhook(ctx, mattermostSettings, postText)
	case DeliveryModeBot:
		return p.sendViaBotAPI(ctx, mattermostSettings, postText)
	}

	return fmt.Errorf("%w: %s", errUnknownDeliveryMode, mattermostSettings.DeliveryMode)
}

func (p *Provider) validateWebhookMode(mattermostSettings Settings) error {
	if mattermostSettings.WebhookURL == "" {
		return providers.NewInvalidSettingError("webhookUrl", "is required in the webhook mode")
	}

	webhookURL, err := p.fieldEncryptor.Decrypt(mattermostSettings.WebhookURL)
	if err != nil {
		return fmt.Errorf("decrypt the incoming webhook URL: %w", err)
	}

	if !isHTTPURL(webhookURL) {
		return providers.NewInvalidSettingError("webhookUrl", "must be an http or https URL")
	}

	return nil
}

func validateBotMode(mattermostSettings Settings) error {
	if mattermostSettings.ServerURL == "" {
		return providers.NewInvalidSettingError("serverUrl", "is required in the bot mode")
	}

	if !isHTTPURL(mattermostSettings.ServerURL) {
		return providers.NewInvalidSettingError("serverUrl", "must be an http or https URL")
	}

	if mattermostSettings.BotToken == "" {
		return providers.NewInvalidSettingError("botToken", "is required in the bot mode")
	}

	if mattermostSettings.TargetChannelID == "" {
		return providers.NewInvalidSettingError("targetChannelId", "is required in the bot mode")
	}

	if len([]rune(mattermostSettings.TargetChannelID)) != channelIDLength {
		return providers.NewInvalidSettingError("targetChannelId", fmt.Sprintf(
			"must be the %d-character Mattermost channel ID, not the channel name", channelIDLength,
		))
	}

	return nil
}

func isHTTPURL(candidate string) bool {
	parsed, err := url.Parse(candidate)
	if err != nil {
		return false
	}

	return (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

func buildPost(message notification.Notification) string {
	postText := fmt.Sprintf("**%s**", message.Heading)
	if message.Message != "" {
		postText = fmt.Sprintf("%s\n\n%s", postText, message.Message)
	}

	postRunes := []rune(postText)
	if len(postRunes) > maxPostLength {
		return string(postRunes[:maxPostLength])
	}

	return postText
}
