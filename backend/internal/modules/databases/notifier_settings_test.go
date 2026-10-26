package databases_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

func newWebhookNotifierSettings(t *testing.T) apiclient.NotifierSettings {
	t.Helper()

	var settings apiclient.NotifierSettings
	require.NoError(t, settings.FromWebhookNotifierSettings(apiclient.WebhookNotifierSettings{
		Type:               apiclient.WebhookNotifierSettingsTypeWEBHOOK,
		WebhookUrl:         "https://hooks.example.com",
		WebhookMethod:      apiclient.WebhookNotifierSettingsWebhookMethodPOST,
		BodyTemplate:       nil,
		Headers:            []apiclient.WebhookHeader{},
		AcceptedCategories: []apiclient.NotificationCategory{apiclient.NotificationCategoryALL},
	}))

	return settings
}
