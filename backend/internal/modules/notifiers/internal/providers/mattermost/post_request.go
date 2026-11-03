package mattermost

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"databasus-backend/internal/modules/notifiers/internal/providers/redaction"
)

// maxErrorBodyBytes bounds the response body a failed send reports, so a misbehaving server cannot
// push an arbitrarily large text into the last send error.
const maxErrorBodyBytes = 8 * 1024

type postRequest struct {
	url string
	// isURLSecret is set for an incoming webhook, whose URL is itself a credential. A bot mode server
	// URL is not a secret, so it stays in errors to keep failures diagnosable.
	isURLSecret          bool
	botToken             string
	payload              map[string]any
	isInsecureSkipVerify bool
}

func (p *Provider) sendViaIncomingWebhook(ctx context.Context, mattermostSettings Settings, postText string) error {
	webhookURL, err := p.fieldEncryptor.Decrypt(mattermostSettings.WebhookURL)
	if err != nil {
		return fmt.Errorf("decrypt the incoming webhook URL: %w", err)
	}

	payload := map[string]any{"text": postText}

	if mattermostSettings.TargetChannelName != "" {
		payload["channel"] = mattermostSettings.TargetChannelName
	}

	if mattermostSettings.OverrideUsername != "" {
		payload["username"] = mattermostSettings.OverrideUsername
	}

	if mattermostSettings.OverrideIconURL != "" {
		payload["icon_url"] = mattermostSettings.OverrideIconURL
	}

	return p.post(ctx, postRequest{
		url:                  webhookURL,
		isURLSecret:          true,
		botToken:             "",
		payload:              payload,
		isInsecureSkipVerify: mattermostSettings.IsInsecureSkipVerify,
	})
}

func (p *Provider) sendViaBotAPI(ctx context.Context, mattermostSettings Settings, postText string) error {
	botToken, err := p.fieldEncryptor.Decrypt(mattermostSettings.BotToken)
	if err != nil {
		return fmt.Errorf("decrypt the bot token: %w", err)
	}

	payload := map[string]any{"channel_id": mattermostSettings.TargetChannelID, "message": postText}
	props := map[string]any{}

	if mattermostSettings.OverrideUsername != "" {
		props["override_username"] = mattermostSettings.OverrideUsername
	}

	if mattermostSettings.OverrideIconURL != "" {
		props["override_icon_url"] = mattermostSettings.OverrideIconURL
	}

	if len(props) > 0 {
		payload["props"] = props
	}

	return p.post(ctx, postRequest{
		url:                  strings.TrimSuffix(mattermostSettings.ServerURL, "/") + createPostPath,
		isURLSecret:          false,
		botToken:             botToken,
		payload:              payload,
		isInsecureSkipVerify: mattermostSettings.IsInsecureSkipVerify,
	})
}

func (p *Provider) post(ctx context.Context, request postRequest) error {
	jsonPayload, err := json.Marshal(request.payload)
	if err != nil {
		return fmt.Errorf("marshal the mattermost payload: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, request.url, bytes.NewReader(jsonPayload))
	if err != nil {
		return fmt.Errorf("create the request: %w", request.withoutSecretURL(err))
	}

	httpRequest.Header.Set("Content-Type", "application/json")

	if request.botToken != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+request.botToken)
	}

	client := p.verifyingClient
	if request.isInsecureSkipVerify {
		client = p.skippingClient
	}

	response, err := client.Do(httpRequest)
	if err != nil {
		return fmt.Errorf("send the mattermost message: %w", request.withoutSecretURL(err))
	}

	defer func() {
		_ = response.Body.Close()
	}()

	responseBody, _ := io.ReadAll(io.LimitReader(response.Body, maxErrorBodyBytes))
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf(
			"mattermost returned non-OK status: %s. Error: %s",
			response.Status,
			redaction.RedactSecretsInText(describeError(responseBody), request.getSecrets()...),
		)
	}

	return nil
}

func (r postRequest) getSecrets() []string {
	if r.isURLSecret {
		return append(redaction.GetSecretURLForms(r.url), r.botToken)
	}

	return []string{r.botToken}
}

func (r postRequest) withoutSecretURL(err error) error {
	if !r.isURLSecret {
		return err
	}

	return redaction.ErrorWithoutURLCredentials(err)
}

func describeError(responseBody []byte) string {
	var mattermostError struct {
		Message string `json:"message"`
	}

	err := json.Unmarshal(responseBody, &mattermostError)
	if err == nil && mattermostError.Message != "" {
		return mattermostError.Message
	}

	return string(responseBody)
}

// newTLSSkippingTransport serves only the notifiers whose user turned on IsInsecureSkipVerify.
func newTLSSkippingTransport() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		// nosemgrep: problem-based-packs.insecure-transport.go-stdlib.bypass-tls-verification.bypass-tls-verification
		TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: true,
		},
	}
}
