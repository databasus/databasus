package teams

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/modules/notifiers/internal/notification"
	"databasus-backend/internal/modules/notifiers/internal/providers"
	"databasus-backend/internal/modules/notifiers/internal/providers/redaction"
)

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

type cardAttachment struct {
	ContentType string `json:"contentType"`
	Content     any    `json:"content"`
}

type payload struct {
	Title       string           `json:"title"`
	Text        string           `json:"text"`
	Attachments []cardAttachment `json:"attachments,omitzero"`
}

func (p *Provider) GetType() notification.NotifierType {
	return notification.NotifierTypeTeams
}

func (p *Provider) GetRepository() providers.SettingsRepository {
	return p.repository
}

func (p *Provider) Validate(settings providers.Settings) error {
	teamsSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	if teamsSettings.WebhookURL == "" {
		return providers.NewInvalidSettingError("powerAutomateUrl", "is required")
	}

	webhookURL, err := p.fieldEncryptor.Decrypt(teamsSettings.WebhookURL)
	if err != nil {
		return fmt.Errorf("decrypt the webhook URL: %w", err)
	}

	parsedURL, err := url.Parse(webhookURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return providers.NewInvalidSettingError("powerAutomateUrl", "must be an http or https URL")
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

	if incomingSettings.WebhookURL == "" {
		incomingSettings.WebhookURL = storedSettings.WebhookURL
	}

	return incomingSettings, nil
}

func (p *Provider) Send(ctx context.Context, settings providers.Settings, message notification.Notification) error {
	teamsSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	webhookURL, err := p.fieldEncryptor.Decrypt(teamsSettings.WebhookURL)
	if err != nil {
		return fmt.Errorf("decrypt the webhook URL: %w", err)
	}

	card := map[string]any{
		"type":    "AdaptiveCard",
		"version": "1.4",
		"body": []any{
			map[string]any{"type": "TextBlock", "size": "Medium", "weight": "Bolder", "text": message.Heading},
			map[string]any{"type": "TextBlock", "wrap": true, "text": message.Message},
		},
	}

	body, err := json.Marshal(payload{
		Title:       message.Heading,
		Text:        message.Message,
		Attachments: []cardAttachment{{ContentType: "application/vnd.microsoft.card.adaptive", Content: card}},
	})
	if err != nil {
		return fmt.Errorf("marshal the teams card: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create the request: %w", redaction.ErrorWithoutURLCredentials(err))
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := p.client.Do(request)
	if err != nil {
		return fmt.Errorf("send the teams message: %w", redaction.ErrorWithoutURLCredentials(err))
	}

	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("teams webhook returned status %d", response.StatusCode)
	}

	return nil
}
