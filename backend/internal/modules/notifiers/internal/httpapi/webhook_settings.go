package httpapi

import (
	"databasus-backend/internal/modules/notifiers/internal/notification"
	"databasus-backend/internal/modules/notifiers/internal/providers/webhook"
)

func toWebhookSettings(body WebhookNotifierSettings) webhook.Settings {
	headers := make([]webhook.Header, 0, len(body.Headers))
	for _, header := range body.Headers {
		headers = append(headers, webhook.Header{Name: header.Name, Value: getStringOrEmpty(header.Value)})
	}

	acceptedCategories := make([]notification.Category, 0, len(body.AcceptedCategories))
	for _, category := range body.AcceptedCategories {
		acceptedCategories = append(acceptedCategories, notification.Category(category))
	}

	return webhook.Settings{
		URL:                body.WebhookUrl,
		Method:             webhook.Method(body.WebhookMethod),
		BodyTemplate:       getStringOrEmpty(body.BodyTemplate),
		Headers:            headers,
		AcceptedCategories: acceptedCategories,
	}
}

// fromWebhookSettings returns the header names without their values, which are secrets.
func fromWebhookSettings(settings webhook.Settings) WebhookNotifierSettings {
	headers := make([]WebhookHeader, 0, len(settings.Headers))
	for _, header := range settings.Headers {
		headers = append(headers, WebhookHeader{Name: header.Name, Value: nil})
	}

	acceptedCategories := make([]NotificationCategory, 0, len(settings.AcceptedCategories))
	for _, category := range settings.AcceptedCategories {
		acceptedCategories = append(acceptedCategories, NotificationCategory(category))
	}

	return WebhookNotifierSettings{
		Type:               WebhookNotifierSettingsTypeWEBHOOK,
		WebhookUrl:         settings.URL,
		WebhookMethod:      WebhookNotifierSettingsWebhookMethod(settings.Method),
		BodyTemplate:       toOptionalString(settings.BodyTemplate),
		Headers:            headers,
		AcceptedCategories: acceptedCategories,
	}
}
