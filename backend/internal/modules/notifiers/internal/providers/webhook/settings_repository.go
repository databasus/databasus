package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/notifiers/internal/notification"
	"databasus-backend/internal/modules/notifiers/internal/providers"
)

// settingsRepository keeps the settings of webhook notifiers in notifiers_webhook.
type settingsRepository struct {
	db             *gorm.DB
	fieldEncryptor *crypto.FieldEncryptor
}

// Repositories are the repositories of the package. module.go builds them from the pool, so the
// provider never holds it.
type Repositories struct {
	settings *settingsRepository
}

func NewRepositories(db *gorm.DB, fieldEncryptor *crypto.FieldEncryptor) Repositories {
	return Repositories{settings: &settingsRepository{db: db, fieldEncryptor: fieldEncryptor}}
}

// storedHeader is the shape of a header in the headers column, which releases before this one
// wrote with the key "key".
type storedHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type settingsRow struct {
	NotifierID         uuid.UUID
	WebhookURL         string
	WebhookMethod      Method
	BodyTemplate       *string
	Headers            string
	AcceptedCategories pq.StringArray `gorm:"type:text[]"`
}

func (r *settingsRepository) Upsert(ctx context.Context, notifierID uuid.UUID, settings providers.Settings) error {
	webhookSettings, err := providers.CastSettings[Settings](settings)
	if err != nil {
		return err
	}

	storedHeaders := make([]storedHeader, 0, len(webhookSettings.Headers))
	for _, header := range webhookSettings.Headers {
		encryptedValue, err := r.fieldEncryptor.Encrypt(header.Value)
		if err != nil {
			return fmt.Errorf("encrypt the value of a webhook header: %w", err)
		}

		storedHeaders = append(storedHeaders, storedHeader{Key: header.Name, Value: encryptedValue})
	}

	headersJSON, err := json.Marshal(storedHeaders)
	if err != nil {
		return fmt.Errorf("marshal the webhook headers: %w", err)
	}

	acceptedCategories := make(pq.StringArray, 0, len(webhookSettings.AcceptedCategories))
	for _, category := range webhookSettings.AcceptedCategories {
		acceptedCategories = append(acceptedCategories, string(category))
	}

	if len(acceptedCategories) == 0 {
		acceptedCategories = pq.StringArray{string(notification.CategoryAll)}
	}

	err = database.Conn(ctx, r.db).Exec(
		`INSERT INTO notifiers_webhook
			(notifier_id, webhook_url, webhook_method, body_template, headers, accepted_categories)
		VALUES (?, ?, ?, ?, ?::jsonb, ?::text[])
		ON CONFLICT (notifier_id) DO UPDATE SET
			webhook_url = EXCLUDED.webhook_url,
			webhook_method = EXCLUDED.webhook_method,
			body_template = EXCLUDED.body_template,
			headers = EXCLUDED.headers,
			accepted_categories = EXCLUDED.accepted_categories`,
		notifierID, webhookSettings.URL, webhookSettings.Method, webhookSettings.BodyTemplate, string(headersJSON),
		acceptedCategories,
	).Error
	if err != nil {
		return fmt.Errorf("upsert the webhook settings: %w", err)
	}

	return nil
}

func (r *settingsRepository) ListByNotifierIDs(
	ctx context.Context,
	notifierIDs []uuid.UUID,
) (map[uuid.UUID]providers.Settings, error) {
	var rows []settingsRow

	err := database.Conn(ctx, r.db).Raw(
		`SELECT notifier_id, webhook_url, webhook_method, body_template, headers::text AS headers, accepted_categories
		FROM notifiers_webhook
		WHERE notifier_id IN ?`,
		notifierIDs,
	).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("list the webhook settings: %w", err)
	}

	settingsByNotifierID := make(map[uuid.UUID]providers.Settings, len(rows))
	for _, row := range rows {
		settings, err := r.toSettings(row)
		if err != nil {
			return nil, err
		}

		settingsByNotifierID[row.NotifierID] = settings
	}

	return settingsByNotifierID, nil
}

// toSettings decrypts the URL and the body template that releases before this one stored
// encrypted. A value it cannot decrypt is left empty, so the notifier still loads and the user can
// enter the value again.
func (r *settingsRepository) toSettings(row settingsRow) (Settings, error) {
	var storedHeaders []storedHeader

	err := json.Unmarshal([]byte(row.Headers), &storedHeaders)
	if err != nil {
		return Settings{}, fmt.Errorf("unmarshal the webhook headers: %w", err)
	}

	headers := make([]Header, 0, len(storedHeaders))
	for _, header := range storedHeaders {
		headers = append(headers, Header{Name: header.Key, Value: header.Value})
	}

	acceptedCategories := make([]notification.Category, 0, len(row.AcceptedCategories))
	for _, category := range row.AcceptedCategories {
		acceptedCategories = append(acceptedCategories, notification.Category(category))
	}

	settings := Settings{
		URL:                       "",
		Method:                    row.WebhookMethod,
		BodyTemplate:              "",
		Headers:                   headers,
		AcceptedCategories:        acceptedCategories,
		storedFieldsDecryptionErr: nil,
	}

	webhookURL, err := r.fieldEncryptor.Decrypt(row.WebhookURL)
	if err != nil {
		settings.storedFieldsDecryptionErr = fmt.Errorf("failed to decrypt webhook URL: %w", err)
	} else {
		settings.URL = webhookURL
	}

	if row.BodyTemplate != nil {
		bodyTemplate, err := r.fieldEncryptor.Decrypt(*row.BodyTemplate)
		if err != nil {
			settings.storedFieldsDecryptionErr = errors.Join(
				settings.storedFieldsDecryptionErr, fmt.Errorf("failed to decrypt webhook body template: %w", err),
			)
		} else {
			settings.BodyTemplate = bodyTemplate
		}
	}

	return settings, nil
}
