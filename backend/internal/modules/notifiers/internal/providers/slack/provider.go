package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/modules/notifiers/internal/notification"
	"databasus-backend/internal/modules/notifiers/internal/providers"
)

const (
	maxAttempts = 5
	// defaultBackoff is the wait before a retry when Slack sends no Retry-After header.
	defaultBackoff    = 2 * time.Second
	backoffMultiplier = 1.5
	// deliveryTimeout leaves room for four waits of a typical 30-second Retry-After. Each request stays
	// limited by the HTTP client, so an endpoint that does not answer still fails within 30 seconds.
	deliveryTimeout = 3 * time.Minute
)

type Deps struct {
	Repositories   Repositories
	FieldEncryptor *crypto.FieldEncryptor
	// PostMessageURL is the chat.postMessage method of the Slack Web API.
	PostMessageURL string
}

type Provider struct {
	repository     *settingsRepository
	fieldEncryptor *crypto.FieldEncryptor
	postMessageURL string
	client         *http.Client
}

func New(deps Deps) *Provider {
	return &Provider{
		repository:     deps.Repositories.settings,
		fieldEncryptor: deps.FieldEncryptor,
		postMessageURL: deps.PostMessageURL,
		client:         providers.NewHTTPClient(nil),
	}
}

func (p *Provider) GetType() notification.NotifierType {
	return notification.NotifierTypeSlack
}

func (p *Provider) GetRepository() providers.SettingsRepository {
	return p.repository
}

func (p *Provider) Validate(settings providers.Settings) error {
	slackSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	if slackSettings.BotToken == "" {
		return providers.NewInvalidSettingError("botToken", "is required")
	}

	if slackSettings.TargetChatID == "" {
		return providers.NewInvalidSettingError("targetChatId", "is required")
	}

	if !strings.HasPrefix(slackSettings.TargetChatID, "C") && !strings.HasPrefix(slackSettings.TargetChatID, "G") &&
		!strings.HasPrefix(slackSettings.TargetChatID, "D") && !strings.HasPrefix(slackSettings.TargetChatID, "U") {
		return providers.NewInvalidSettingError(
			"targetChatId",
			"must be a Slack channel ID, starting with C, G or D, or a user ID, starting with U",
		)
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

	if incomingSettings.BotToken == "" {
		incomingSettings.BotToken = storedSettings.BotToken
	}

	return incomingSettings, nil
}

func (p *Provider) GetDeliveryTimeout() time.Duration {
	return deliveryTimeout
}

// Send retries while Slack rate-limits the message, waiting as long as its Retry-After header asks,
// and gives up at once when that wait does not fit the time left.
func (p *Provider) Send(ctx context.Context, settings providers.Settings, message notification.Notification) error {
	slackSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	botToken, err := p.fieldEncryptor.Decrypt(slackSettings.BotToken)
	if err != nil {
		return fmt.Errorf("decrypt the bot token: %w", err)
	}

	text := fmt.Sprintf("*%s*", message.Heading)
	if message.Message != "" {
		text = fmt.Sprintf("%s\n\n%s", text, message.Message)
	}

	payload, err := json.Marshal(map[string]any{"channel": slackSettings.TargetChatID, "text": text, "mrkdwn": true})
	if err != nil {
		return fmt.Errorf("marshal the slack message: %w", err)
	}

	backoff := defaultBackoff

	for attempt := 1; ; attempt++ {
		retryAfter, isRateLimited, err := p.postMessage(ctx, botToken, payload)
		if err != nil || !isRateLimited {
			return err
		}

		if attempt >= maxAttempts {
			return fmt.Errorf("%w after %d attempts", errRateLimited, attempt)
		}

		if retryAfter == 0 {
			retryAfter = backoff
		}

		err = waitForRetry(ctx, retryAfter)
		if err != nil {
			return err
		}

		backoff = time.Duration(float64(backoff) * backoffMultiplier)
	}
}

// postMessage reports a rate-limited attempt with the wait Slack asked for, zero when it named none.
func (p *Provider) postMessage(ctx context.Context, botToken string, payload []byte) (time.Duration, bool, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, p.postMessageURL, bytes.NewReader(payload))
	if err != nil {
		return 0, false, fmt.Errorf("create the request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json; charset=utf-8")
	request.Header.Set("Authorization", "Bearer "+botToken)

	response, err := p.client.Do(request)
	if err != nil {
		return 0, false, fmt.Errorf("send the slack message: %w", err)
	}

	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode == http.StatusTooManyRequests {
		retryAfterSeconds, _ := strconv.Atoi(response.Header.Get("Retry-After"))

		return time.Duration(max(retryAfterSeconds, 0)) * time.Second, true, nil
	}

	// Slack answers 200 to a refused message too, so the body tells whether it was sent.
	var responseBody struct {
		IsOK  bool   `json:"ok"`
		Error string `json:"error"`
	}

	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return 0, false, fmt.Errorf("decode the slack response with status %s: %w", response.Status, err)
	}

	if !responseBody.IsOK {
		return 0, false, fmt.Errorf("slack API error: %s", responseBody.Error)
	}

	return 0, false, nil
}

func waitForRetry(ctx context.Context, wait time.Duration) error {
	deadline, hasDeadline := ctx.Deadline()
	if hasDeadline && time.Until(deadline) < wait {
		return fmt.Errorf("%w: it asked to wait %s", errRateLimited, wait)
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return fmt.Errorf("wait to retry the slack message: %w", ctx.Err())
	case <-timer.C:
		return nil
	}
}
