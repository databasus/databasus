package webhook_notifier

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	notifier_models "databasus-backend/internal/features/notifiers/models"
	"databasus-backend/internal/util/encryption"
)

func send(t *testing.T, notifier *WebhookNotifier, notificationType notifier_models.NotificationType) error {
	t.Helper()

	return notifier.Send(
		notifier_models.PassthroughEncryptor{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		notifier_models.Notification{
			Type:    notificationType,
			Heading: "Backup completed",
			Message: "All good",
		},
	)
}

func acceptAll() []notifier_models.NotificationType {
	return []notifier_models.NotificationType{notifier_models.NotificationTypeAll}
}

func Test_Send_WithPOSTAndNoBodyTemplate_SendsDefaultJSONBody(t *testing.T) {
	webhookURL, recorder := notifier_models.StartRecordingServer(
		t,
		notifier_models.StubResponse{StatusCode: http.StatusOK},
	)
	notifier := &WebhookNotifier{
		WebhookURL:              webhookURL,
		WebhookMethod:           WebhookMethodPOST,
		AcceptNotificationTypes: acceptAll(),
	}

	require.NoError(t, send(t, notifier, notifier_models.NotificationTypeBackupSuccess))

	require.Equal(t, 1, recorder.GetRequestCount())
	request := recorder.GetLastRequest()
	assert.Equal(t, http.MethodPost, request.Method)
	assert.Equal(t, "application/json", request.Headers.Get("Content-Type"))

	var payload map[string]string
	require.NoError(t, json.Unmarshal([]byte(request.Body), &payload))
	assert.Equal(t, "Backup completed", payload["heading"])
	assert.Equal(t, "All good", payload["message"])
}

func Test_Send_WithPOSTAndBodyTemplate_SubstitutesAndEscapesPlaceholders(t *testing.T) {
	webhookURL, recorder := notifier_models.StartRecordingServer(
		t,
		notifier_models.StubResponse{StatusCode: http.StatusOK},
	)
	template := `{"title":"{{heading}}","text":"{{message}}"}`
	notifier := &WebhookNotifier{
		WebhookURL:              webhookURL,
		WebhookMethod:           WebhookMethodPOST,
		BodyTemplate:            &template,
		AcceptNotificationTypes: acceptAll(),
	}

	err := notifier.Send(
		notifier_models.PassthroughEncryptor{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		notifier_models.Notification{
			Type:    notifier_models.NotificationTypeBackupSuccess,
			Heading: `He said "hi"`,
			Message: "line1\nline2",
		},
	)
	require.NoError(t, err)

	var payload map[string]string
	require.NoError(t, json.Unmarshal([]byte(recorder.GetLastRequest().Body), &payload))
	assert.Equal(t, `He said "hi"`, payload["title"])
	assert.Equal(t, "line1\nline2", payload["text"])
}

func Test_Send_WithCustomContentTypeHeader_DoesNotOverrideIt(t *testing.T) {
	webhookURL, recorder := notifier_models.StartRecordingServer(
		t,
		notifier_models.StubResponse{StatusCode: http.StatusOK},
	)
	notifier := &WebhookNotifier{
		WebhookURL:              webhookURL,
		WebhookMethod:           WebhookMethodPOST,
		Headers:                 []WebhookHeader{{Key: "Content-Type", Value: "application/xml"}},
		AcceptNotificationTypes: acceptAll(),
	}

	require.NoError(t, send(t, notifier, notifier_models.NotificationTypeBackupSuccess))

	assert.Equal(t, "application/xml", recorder.GetLastRequest().Headers.Get("Content-Type"))
}

func Test_Send_WithGET_SendsHeadingAndMessageAsQueryParams(t *testing.T) {
	webhookURL, recorder := notifier_models.StartRecordingServer(
		t,
		notifier_models.StubResponse{StatusCode: http.StatusOK},
	)
	notifier := &WebhookNotifier{
		WebhookURL:              webhookURL,
		WebhookMethod:           WebhookMethodGET,
		AcceptNotificationTypes: acceptAll(),
	}

	require.NoError(t, send(t, notifier, notifier_models.NotificationTypeBackupSuccess))

	request := recorder.GetLastRequest()
	assert.Equal(t, http.MethodGet, request.Method)
	assert.Equal(t, "Backup completed", request.Query.Get("heading"))
	assert.Equal(t, "All good", request.Query.Get("message"))
	assert.Empty(t, request.Body)
}

func Test_Send_WithCustomHeaders_AppliesThemToRequest(t *testing.T) {
	webhookURL, recorder := notifier_models.StartRecordingServer(
		t,
		notifier_models.StubResponse{StatusCode: http.StatusOK},
	)
	notifier := &WebhookNotifier{
		WebhookURL:              webhookURL,
		WebhookMethod:           WebhookMethodPOST,
		Headers:                 []WebhookHeader{{Key: "X-Api-Key", Value: "secret-value"}},
		AcceptNotificationTypes: acceptAll(),
	}

	require.NoError(t, send(t, notifier, notifier_models.NotificationTypeBackupSuccess))

	assert.Equal(t, "secret-value", recorder.GetLastRequest().Headers.Get("X-Api-Key"))
}

func Test_Send_WhenServerReturnsNon2xx_ReturnsError(t *testing.T) {
	webhookURL, _ := notifier_models.StartRecordingServer(
		t,
		notifier_models.StubResponse{StatusCode: http.StatusInternalServerError},
	)
	notifier := &WebhookNotifier{
		WebhookURL:              webhookURL,
		WebhookMethod:           WebhookMethodPOST,
		AcceptNotificationTypes: acceptAll(),
	}

	require.Error(t, send(t, notifier, notifier_models.NotificationTypeBackupSuccess))
}

func Test_Send_WhenURLUnreachable_ReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	unreachableURL := server.URL
	server.Close()

	notifier := &WebhookNotifier{
		WebhookURL:              unreachableURL,
		WebhookMethod:           WebhookMethodPOST,
		AcceptNotificationTypes: acceptAll(),
	}

	require.Error(t, send(t, notifier, notifier_models.NotificationTypeBackupSuccess))
}

func Test_Send_WhenAcceptTypesEmpty_SendsEveryType(t *testing.T) {
	webhookURL, recorder := notifier_models.StartRecordingServer(
		t,
		notifier_models.StubResponse{StatusCode: http.StatusOK},
	)
	notifier := &WebhookNotifier{
		WebhookURL:    webhookURL,
		WebhookMethod: WebhookMethodPOST,
	}

	require.NoError(t, send(t, notifier, notifier_models.NotificationTypeBackupSuccess))
	require.NoError(t, send(t, notifier, notifier_models.NotificationTypeHealthcheckFailed))

	assert.Equal(t, 2, recorder.GetRequestCount())
}

func Test_Send_WhenAcceptContainsAll_SendsEveryType(t *testing.T) {
	webhookURL, recorder := notifier_models.StartRecordingServer(
		t,
		notifier_models.StubResponse{StatusCode: http.StatusOK},
	)
	notifier := &WebhookNotifier{
		WebhookURL:              webhookURL,
		WebhookMethod:           WebhookMethodPOST,
		AcceptNotificationTypes: acceptAll(),
	}

	require.NoError(t, send(t, notifier, notifier_models.NotificationTypeBackupSuccess))
	require.NoError(t, send(t, notifier, notifier_models.NotificationTypeVerificationFailed))

	assert.Equal(t, 2, recorder.GetRequestCount())
}

func Test_Send_WhenNarrowedToBackupSuccess_SkipsOtherTypes(t *testing.T) {
	webhookURL, recorder := notifier_models.StartRecordingServer(
		t,
		notifier_models.StubResponse{StatusCode: http.StatusOK},
	)
	notifier := &WebhookNotifier{
		WebhookURL:              webhookURL,
		WebhookMethod:           WebhookMethodPOST,
		AcceptNotificationTypes: []notifier_models.NotificationType{notifier_models.NotificationTypeBackupSuccess},
	}

	require.NoError(t, send(t, notifier, notifier_models.NotificationTypeBackupFailed))
	require.NoError(t, send(t, notifier, notifier_models.NotificationTypeHealthcheckSuccess))
	assert.Equal(t, 0, recorder.GetRequestCount())

	require.NoError(t, send(t, notifier, notifier_models.NotificationTypeBackupSuccess))
	assert.Equal(t, 1, recorder.GetRequestCount())
}

func Test_Send_WhenNarrowed_StillSendsWildcardTestNotification(t *testing.T) {
	webhookURL, recorder := notifier_models.StartRecordingServer(
		t,
		notifier_models.StubResponse{StatusCode: http.StatusOK},
	)
	notifier := &WebhookNotifier{
		WebhookURL:              webhookURL,
		WebhookMethod:           WebhookMethodPOST,
		AcceptNotificationTypes: []notifier_models.NotificationType{notifier_models.NotificationTypeBackupFailed},
	}

	require.NoError(t, send(t, notifier, notifier_models.NotificationTypeAll))

	assert.Equal(t, 1, recorder.GetRequestCount())
}

func Test_BeforeSave_WhenAcceptTypesEmpty_DefaultsToAll(t *testing.T) {
	notifier := &WebhookNotifier{
		WebhookURL:    "https://example.com/webhook",
		WebhookMethod: WebhookMethodPOST,
	}

	require.NoError(t, notifier.BeforeSave(nil))

	assert.Equal(t, `["ALL"]`, notifier.AcceptNotificationTypesJSON)
	assert.Equal(t, acceptAll(), notifier.AcceptNotificationTypes)
}

func Test_BeforeSave_WithSingleType_SerializesOnlyThatType(t *testing.T) {
	notifier := &WebhookNotifier{
		WebhookURL:              "https://example.com/webhook",
		WebhookMethod:           WebhookMethodPOST,
		AcceptNotificationTypes: []notifier_models.NotificationType{notifier_models.NotificationTypeBackupSuccess},
	}

	require.NoError(t, notifier.BeforeSave(nil))

	assert.Equal(t, `["BACKUP_SUCCESS"]`, notifier.AcceptNotificationTypesJSON)
}

func Test_AfterFind_WithSerializedTypes_RestoresSlice(t *testing.T) {
	notifier := &WebhookNotifier{
		AcceptNotificationTypesJSON: `["VERIFICATION_FAILED"]`,
	}

	require.NoError(t, notifier.AfterFind(nil))

	assert.Equal(
		t,
		[]notifier_models.NotificationType{notifier_models.NotificationTypeVerificationFailed},
		notifier.AcceptNotificationTypes,
	)
}

func Test_Send_WithUndecryptableStoredField_FailsWithoutRequest(t *testing.T) {
	const undecryptableValue = "enc:not-a-valid-ciphertext"

	testCases := []struct {
		name               string
		buildNotifier      func(webhookURL string) *WebhookNotifier
		expectedErrorField string
	}{
		{
			name: "fails naming the URL when the stored URL cannot be decrypted",
			buildNotifier: func(_ string) *WebhookNotifier {
				return &WebhookNotifier{WebhookURL: undecryptableValue}
			},
			expectedErrorField: "webhook URL",
		},
		{
			name: "fails naming the body template when the stored template cannot be decrypted",
			buildNotifier: func(webhookURL string) *WebhookNotifier {
				bodyTemplate := undecryptableValue
				return &WebhookNotifier{WebhookURL: webhookURL, BodyTemplate: &bodyTemplate}
			},
			expectedErrorField: "webhook body template",
		},
		{
			name: "fails naming the header when a stored header value cannot be decrypted",
			buildNotifier: func(webhookURL string) *WebhookNotifier {
				return &WebhookNotifier{
					WebhookURL: webhookURL,
					Headers:    []WebhookHeader{{Key: "Authorization", Value: undecryptableValue}},
				}
			},
			expectedErrorField: `webhook header "Authorization"`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			webhookURL, recorder := notifier_models.StartRecordingServer(
				t,
				notifier_models.StubResponse{StatusCode: http.StatusOK},
			)
			loadedNotifier := testCase.buildNotifier(webhookURL)
			loadedNotifier.WebhookMethod = WebhookMethodPOST
			loadedNotifier.AcceptNotificationTypes = acceptAll()
			require.NoError(t, loadedNotifier.AfterFind(nil))

			err := loadedNotifier.Send(
				encryption.GetFieldEncryptor(),
				slog.New(slog.NewTextHandler(io.Discard, nil)),
				notifier_models.Notification{
					Type:    notifier_models.NotificationTypeBackupFailed,
					Heading: "Backup failed",
					Message: "Disk is full",
				},
			)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to decrypt "+testCase.expectedErrorField)
			assert.Equal(t, 0, recorder.GetRequestCount())
		})
	}
}
