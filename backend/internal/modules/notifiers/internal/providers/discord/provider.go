package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/modules/notifiers/internal/notification"
	"databasus-backend/internal/modules/notifiers/internal/providers"
	"databasus-backend/internal/modules/notifiers/internal/providers/redaction"
)

// maxContentRunes is the longest message content Discord accepts.
const maxContentRunes = 2000

type Deps struct {
	Repositories   Repositories
	FieldEncryptor *crypto.FieldEncryptor
}

type Provider struct {
	repository     *settingsRepository
	fieldEncryptor *crypto.FieldEncryptor
	client         *http.Client
}

func New(deps Deps) *Provider {
	return &Provider{
		repository:     deps.Repositories.settings,
		fieldEncryptor: deps.FieldEncryptor,
		client:         providers.NewHTTPClient(nil),
	}
}

func (p *Provider) GetType() notification.NotifierType {
	return notification.NotifierTypeDiscord
}

func (p *Provider) GetRepository() providers.SettingsRepository {
	return p.repository
}

func (p *Provider) Validate(settings providers.Settings) error {
	discordSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	if discordSettings.ChannelWebhookURL == "" {
		return providers.NewInvalidSettingError("channelWebhookUrl", "is required")
	}

	return nil
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

	if incomingSettings.ChannelWebhookURL == "" {
		incomingSettings.ChannelWebhookURL = storedSettings.ChannelWebhookURL
	}

	return incomingSettings, nil
}

func (p *Provider) Send(ctx context.Context, settings providers.Settings, message notification.Notification) error {
	discordSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	webhookURL, err := p.fieldEncryptor.Decrypt(discordSettings.ChannelWebhookURL)
	if err != nil {
		return fmt.Errorf("decrypt the channel webhook URL: %w", err)
	}

	payload, err := json.Marshal(map[string]any{"content": buildContent(message)})
	if err != nil {
		return fmt.Errorf("marshal the discord payload: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create the request: %w", redaction.ErrorWithoutURLCredentials(err))
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := p.client.Do(request)
	if err != nil {
		return fmt.Errorf("send the discord message: %w", redaction.ErrorWithoutURLCredentials(err))
	}

	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		responseBody, _ := io.ReadAll(response.Body)

		return fmt.Errorf(
			"discord API returned non-OK status: %s. Error: %s",
			response.Status,
			redaction.RedactSecretsInText(string(responseBody), redaction.GetSecretURLForms(webhookURL)...),
		)
	}

	return nil
}

func buildContent(message notification.Notification) string {
	content := message.Heading
	if message.Message != "" {
		content = fmt.Sprintf("%s\n\n%s", message.Heading, message.Message)
	}

	contentRunes := []rune(content)
	if len(contentRunes) > maxContentRunes {
		return string(contentRunes[:maxContentRunes])
	}

	return content
}
