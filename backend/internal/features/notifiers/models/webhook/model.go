package webhook_notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	notifier_models "databasus-backend/internal/features/notifiers/models"
	"databasus-backend/internal/util/encryption"
)

type WebhookHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type WebhookNotifier struct {
	NotifierID    uuid.UUID     `json:"notifierId"    gorm:"primaryKey;column:notifier_id"`
	WebhookURL    string        `json:"webhookUrl"    gorm:"not null;column:webhook_url"`
	WebhookMethod WebhookMethod `json:"webhookMethod" gorm:"not null;column:webhook_method"`
	BodyTemplate  *string       `json:"bodyTemplate"  gorm:"column:body_template;type:text"`

	HeadersJSON string          `json:"-"       gorm:"column:headers;type:text"`
	Headers     []WebhookHeader `json:"headers" gorm:"-"`

	AcceptNotificationTypes     []notifier_models.NotificationType `json:"acceptNotificationTypes" gorm:"-"`
	AcceptNotificationTypesJSON string                             `json:"-"                       gorm:"column:accept_notification_types;type:text"`

	// AfterFind cannot fail the query without breaking every list that loads this notifier, so
	// it keeps the error here and Send refuses to deliver with a field it could not decrypt.
	storedFieldsDecryptionErr error `gorm:"-"`
}

func (t *WebhookNotifier) TableName() string {
	return "webhook_notifiers"
}

func (t *WebhookNotifier) BeforeSave(_ *gorm.DB) error {
	if len(t.Headers) > 0 {
		data, err := json.Marshal(t.Headers)
		if err != nil {
			return err
		}

		t.HeadersJSON = string(data)
	} else {
		t.HeadersJSON = "[]"
	}

	if len(t.AcceptNotificationTypes) == 0 {
		t.AcceptNotificationTypes = []notifier_models.NotificationType{
			notifier_models.NotificationTypeAll,
		}
	}

	serializedAcceptTypes, err := json.Marshal(t.AcceptNotificationTypes)
	if err != nil {
		return err
	}

	t.AcceptNotificationTypesJSON = string(serializedAcceptTypes)

	return nil
}

func (t *WebhookNotifier) AfterFind(_ *gorm.DB) error {
	if t.HeadersJSON != "" {
		if err := json.Unmarshal([]byte(t.HeadersJSON), &t.Headers); err != nil {
			return err
		}
	}

	if t.AcceptNotificationTypesJSON != "" {
		if err := json.Unmarshal([]byte(t.AcceptNotificationTypesJSON), &t.AcceptNotificationTypes); err != nil {
			return err
		}
	}

	encryptor := encryption.GetFieldEncryptor()
	t.storedFieldsDecryptionErr = nil

	if t.WebhookURL != "" {
		decrypted, err := encryptor.Decrypt(t.WebhookURL)
		if err != nil {
			t.storedFieldsDecryptionErr = fmt.Errorf("failed to decrypt webhook URL: %w", err)
		} else {
			t.WebhookURL = decrypted
		}
	}

	if t.BodyTemplate != nil && *t.BodyTemplate != "" {
		decrypted, err := encryptor.Decrypt(*t.BodyTemplate)
		if err != nil {
			t.storedFieldsDecryptionErr = errors.Join(
				t.storedFieldsDecryptionErr,
				fmt.Errorf("failed to decrypt webhook body template: %w", err),
			)
		} else {
			t.BodyTemplate = &decrypted
		}
	}

	return nil
}

func (t *WebhookNotifier) Validate(encryptor encryption.FieldEncryptor) error {
	if t.WebhookURL == "" {
		return errors.New("webhook URL is required")
	}

	if t.WebhookMethod == "" {
		return errors.New("webhook method is required")
	}

	return nil
}

func (t *WebhookNotifier) Send(
	encryptor encryption.FieldEncryptor,
	logger *slog.Logger,
	notification notifier_models.Notification,
) error {
	if !t.isNotificationTypeAccepted(notification.Type) {
		return nil
	}

	if t.storedFieldsDecryptionErr != nil {
		return t.storedFieldsDecryptionErr
	}

	if err := t.decryptHeadersForSending(encryptor); err != nil {
		return err
	}

	switch t.WebhookMethod {
	case WebhookMethodGET:
		return t.sendGET(t.WebhookURL, notification.Heading, notification.Message, logger)
	case WebhookMethodPOST:
		return t.sendPOST(t.WebhookURL, notification.Heading, notification.Message, logger)
	default:
		return fmt.Errorf("unsupported webhook method: %s", t.WebhookMethod)
	}
}

func (t *WebhookNotifier) HideSensitiveData() {
	for i := range t.Headers {
		t.Headers[i].Value = ""
	}
}

func (t *WebhookNotifier) Update(incoming *WebhookNotifier) {
	t.WebhookURL = incoming.WebhookURL
	t.WebhookMethod = incoming.WebhookMethod
	t.BodyTemplate = incoming.BodyTemplate
	t.Headers = incoming.Headers
	t.AcceptNotificationTypes = incoming.AcceptNotificationTypes
	t.storedFieldsDecryptionErr = nil
}

func (t *WebhookNotifier) EncryptSensitiveData(encryptor encryption.FieldEncryptor) error {
	for i := range t.Headers {
		if t.Headers[i].Value != "" {
			encrypted, err := encryptor.Encrypt(t.Headers[i].Value)
			if err != nil {
				return fmt.Errorf("failed to encrypt header value: %w", err)
			}

			t.Headers[i].Value = encrypted
		}
	}

	return nil
}

// NotificationTypeAll is a wildcard on both sides: on the accept-list it accepts every type, and as
// an incoming type (carried by test notifications) it bypasses the filter so tests always deliver.
func (t *WebhookNotifier) isNotificationTypeAccepted(notificationType notifier_models.NotificationType) bool {
	if len(t.AcceptNotificationTypes) == 0 || notificationType == notifier_models.NotificationTypeAll {
		return true
	}

	return slices.Contains(t.AcceptNotificationTypes, notifier_models.NotificationTypeAll) ||
		slices.Contains(t.AcceptNotificationTypes, notificationType)
}

func (t *WebhookNotifier) sendGET(webhookURL, heading, message string, logger *slog.Logger) error {
	reqURL := fmt.Sprintf("%s?heading=%s&message=%s",
		webhookURL,
		url.QueryEscape(heading),
		url.QueryEscape(message),
	)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create GET request: %w", err)
	}

	t.applyHeaders(req)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send GET webhook: %w", notifier_models.ErrorWithoutWebhookURLCredentials(err))
	}

	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			logger.Error("failed to close response body", "error", cerr)
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf(
			"webhook GET returned status: %s, body: %s",
			resp.Status,
			string(body),
		)
	}

	return nil
}

func (t *WebhookNotifier) sendPOST(webhookURL, heading, message string, logger *slog.Logger) error {
	body := t.buildRequestBody(heading, message)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create POST request: %w", err)
	}

	hasContentType := false

	for _, h := range t.Headers {
		if strings.EqualFold(h.Key, "Content-Type") {
			hasContentType = true
			break
		}
	}

	if !hasContentType {
		req.Header.Set("Content-Type", "application/json")
	}

	t.applyHeaders(req)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send POST webhook: %w", notifier_models.ErrorWithoutWebhookURLCredentials(err))
	}

	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			logger.Error("failed to close response body", "error", cerr)
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf(
			"webhook POST returned status: %s, body: %s",
			resp.Status,
			string(respBody),
		)
	}

	return nil
}

func (t *WebhookNotifier) buildRequestBody(heading, message string) []byte {
	if t.BodyTemplate != nil && *t.BodyTemplate != "" {
		result := *t.BodyTemplate
		result = strings.ReplaceAll(result, "{{heading}}", escapeJSONString(heading))
		result = strings.ReplaceAll(result, "{{message}}", escapeJSONString(message))
		return []byte(result)
	}

	payload := map[string]string{
		"heading": heading,
		"message": message,
	}
	body, _ := json.Marshal(payload)

	return body
}

func (t *WebhookNotifier) applyHeaders(req *http.Request) {
	for _, h := range t.Headers {
		if h.Key != "" {
			req.Header.Set(h.Key, h.Value)
		}
	}
}

func escapeJSONString(s string) string {
	b, err := json.Marshal(s)
	if err != nil || len(b) < 2 {
		escaped := strings.ReplaceAll(s, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		escaped = strings.ReplaceAll(escaped, "\n", `\n`)
		escaped = strings.ReplaceAll(escaped, "\r", `\r`)
		escaped = strings.ReplaceAll(escaped, "\t", `\t`)
		return escaped
	}

	return string(b[1 : len(b)-1])
}

func (t *WebhookNotifier) decryptHeadersForSending(encryptor encryption.FieldEncryptor) error {
	for i := range t.Headers {
		if t.Headers[i].Value != "" {
			decrypted, err := encryptor.Decrypt(t.Headers[i].Value)
			if err != nil {
				return fmt.Errorf("failed to decrypt webhook header %q: %w", t.Headers[i].Key, err)
			}

			t.Headers[i].Value = decrypted
		}
	}

	return nil
}
