package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/modules/notifiers/internal/notification"
	"databasus-backend/internal/modules/notifiers/internal/providers"
	"databasus-backend/internal/modules/notifiers/internal/providers/redaction"
)

// httpTokenPattern matches an RFC 9110 token, the only header name an HTTP client sends.
var httpTokenPattern = regexp.MustCompile("^[!#$%&'*+\\-.^_`|~0-9A-Za-z]+$")

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
	return notification.NotifierTypeWebhook
}

func (p *Provider) GetRepository() providers.SettingsRepository {
	return p.repository
}

func (p *Provider) Validate(settings providers.Settings) error {
	webhookSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	if webhookSettings.URL == "" {
		return providers.NewInvalidSettingError("webhookUrl", "is required")
	}

	if webhookSettings.Method != MethodPOST && webhookSettings.Method != MethodGET {
		return providers.NewInvalidSettingError("webhookMethod", "must be POST or GET")
	}

	for headerIndex, header := range webhookSettings.Headers {
		if !httpTokenPattern.MatchString(header.Name) {
			return providers.NewInvalidSettingError(
				fmt.Sprintf("headers[%d].name", headerIndex),
				"must be an HTTP token",
			)
		}
	}

	return nil
}

// MergeStoredSecrets gives a header sent without a value the stored value of the header with the
// same name, and refuses such a header at another URL, because Validate accepts a header without a
// value. A header the incoming settings leave out is removed.
func (p *Provider) MergeStoredSecrets(merge providers.SecretMerge) (providers.Settings, error) {
	incomingSettings, err := providers.CastSettings[Settings](merge.Incoming)
	if err != nil {
		return nil, err
	}

	storedSettings, err := providers.CastSettings[Settings](merge.Stored)
	if err != nil {
		return nil, err
	}

	isAddressChanged := incomingSettings.URL != storedSettings.URL

	mergedHeaders := make([]Header, 0, len(incomingSettings.Headers))
	for headerIndex, incomingHeader := range incomingSettings.Headers {
		if incomingHeader.Value == "" {
			storedHeaderIndex := slices.IndexFunc(storedSettings.Headers, func(storedHeader Header) bool {
				return strings.EqualFold(storedHeader.Name, incomingHeader.Name) && storedHeader.Value != ""
			})
			if storedHeaderIndex >= 0 {
				if isAddressChanged {
					return nil, providers.NewInvalidSettingError(
						fmt.Sprintf("headers[%d].value", headerIndex), "must be entered again for the new URL",
					)
				}

				incomingHeader.Value = storedSettings.Headers[storedHeaderIndex].Value
			}
		}

		mergedHeaders = append(mergedHeaders, incomingHeader)
	}

	incomingSettings.Headers = mergedHeaders

	return incomingSettings, nil
}

// Send skips a notification of a category the webhook does not accept, which counts as delivered.
func (p *Provider) Send(ctx context.Context, settings providers.Settings, message notification.Notification) error {
	webhookSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	if !isCategoryAccepted(webhookSettings.AcceptedCategories, message.Category) {
		return nil
	}

	if webhookSettings.storedFieldsDecryptionErr != nil {
		return webhookSettings.storedFieldsDecryptionErr
	}

	headers, err := p.decryptHeaders(webhookSettings.Headers)
	if err != nil {
		return err
	}

	request, err := buildRequest(ctx, webhookSettings, headers, message)
	if err != nil {
		return err
	}

	response, err := p.client.Do(request)
	if err != nil {
		return fmt.Errorf("send the %s webhook: %w", webhookSettings.Method, redaction.ErrorWithoutURLCredentials(err))
	}

	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		responseBody, _ := io.ReadAll(response.Body)

		headerValues := make([]string, 0, len(headers))
		for _, header := range headers {
			headerValues = append(headerValues, header.Value)
		}

		return fmt.Errorf(
			"webhook %s returned status: %s, body: %s",
			webhookSettings.Method,
			response.Status,
			redaction.RedactSecretsInText(string(responseBody), headerValues...),
		)
	}

	return nil
}

func (p *Provider) decryptHeaders(storedHeaders []Header) ([]Header, error) {
	headers := make([]Header, 0, len(storedHeaders))
	for _, storedHeader := range storedHeaders {
		value, err := p.fieldEncryptor.Decrypt(storedHeader.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt webhook header %q: %w", storedHeader.Name, err)
		}

		headers = append(headers, Header{Name: storedHeader.Name, Value: value})
	}

	return headers, nil
}

// isCategoryAccepted treats CategoryAll as a wildcard on both sides: on the accepted list it accepts
// every category, and a test notification carries it, so a test always goes out.
func isCategoryAccepted(acceptedCategories []notification.Category, category notification.Category) bool {
	if len(acceptedCategories) == 0 || category == notification.CategoryAll {
		return true
	}

	return slices.Contains(acceptedCategories, notification.CategoryAll) ||
		slices.Contains(acceptedCategories, category)
}

func buildRequest(
	ctx context.Context,
	webhookSettings Settings,
	headers []Header,
	message notification.Notification,
) (*http.Request, error) {
	var request *http.Request

	var err error

	if webhookSettings.Method == MethodGET {
		requestURL := fmt.Sprintf("%s?heading=%s&message=%s",
			webhookSettings.URL, url.QueryEscape(message.Heading), url.QueryEscape(message.Message))
		request, err = http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	} else {
		request, err = http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			webhookSettings.URL,
			bytes.NewReader(buildBody(webhookSettings.BodyTemplate, message)),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("create the request: %w", redaction.ErrorWithoutURLCredentials(err))
	}

	if webhookSettings.Method == MethodPOST {
		request.Header.Set("Content-Type", "application/json")
	}

	for _, header := range headers {
		request.Header.Set(header.Name, header.Value)
	}

	return request, nil
}

func buildBody(bodyTemplate string, message notification.Notification) []byte {
	if bodyTemplate != "" {
		body := strings.ReplaceAll(bodyTemplate, "{{heading}}", escapeJSONString(message.Heading))
		body = strings.ReplaceAll(body, "{{message}}", escapeJSONString(message.Message))

		return []byte(body)
	}

	body, _ := json.Marshal(map[string]string{"heading": message.Heading, "message": message.Message})

	return body
}

// escapeJSONString escapes text for a place inside a JSON string of the body template.
func escapeJSONString(text string) string {
	encoded, _ := json.Marshal(text)

	return string(encoded[1 : len(encoded)-1])
}
