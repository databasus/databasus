package notifiers_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func requireProblem(t *testing.T, status int, body []byte, expectedStatus int, expectedCode apiclient.ProblemCode) {
	t.Helper()

	require.Equal(t, expectedStatus, status, string(body))

	var problem apiclient.Problem
	require.NoError(t, json.Unmarshal(body, &problem), string(body))
	require.Equal(t, expectedCode, problem.Code, string(body))
}

func requireFieldError(t *testing.T, body []byte, expectedField string) {
	t.Helper()

	var problem apiclient.Problem
	require.NoError(t, json.Unmarshal(body, &problem), string(body))
	require.NotNil(t, problem.Errors, string(body))

	refusedFields := make([]string, 0, len(*problem.Errors))
	for _, fieldError := range *problem.Errors {
		refusedFields = append(refusedFields, fieldError.Field)
	}

	require.Contains(t, refusedFields, expectedField, string(body))
}

// team is an instance with a workspace owned by a user who is not an instance admin.
type team struct {
	app       *apptest.App
	admin     apptest.Session
	owner     apptest.Session
	workspace apiclient.Workspace
}

func newTeam(t *testing.T, options ...apptest.Option) team {
	t.Helper()

	app := apptest.New(t, options...)
	admin := app.CreateAdmin(t)
	owner := app.CreateNonAdminUser(t, admin)

	return team{
		app:       app,
		admin:     admin,
		owner:     owner,
		workspace: apptest.CreateWorkspace(t, owner, "Team "+uuid.NewString()[:8]),
	}
}

// addMember signs up a user and adds them to the team's workspace with the role.
func (tm team) addMember(t *testing.T, role apiclient.AssignableWorkspaceRole) apptest.Session {
	t.Helper()

	user := tm.app.SignUp(t, apptest.NewTestEmail("member"))
	apptest.AddWorkspaceMember(t, tm.owner, tm.workspace.Id, user.Email, role)

	return user
}

// stranger signs up a user who is a member of no workspace of the team.
func (tm team) stranger(t *testing.T) apptest.Session {
	t.Helper()

	return tm.app.SignUp(t, apptest.NewTestEmail("stranger"))
}

func getNotifier(t *testing.T, session apptest.Session, notifierID uuid.UUID) apiclient.Notifier {
	t.Helper()

	response, err := session.Client.GetNotifierWithResponse(t.Context(), notifierID)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return *response.JSON200
}

func listNotifiers(t *testing.T, session apptest.Session, workspaceID uuid.UUID) []apiclient.Notifier {
	t.Helper()

	response, err := session.Client.ListNotifiersWithResponse(t.Context(), &apiclient.ListNotifiersParams{
		WorkspaceId: workspaceID,
		Limit:       nil,
		After:       nil,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return response.JSON200.Items
}

func updateNotifier(
	t *testing.T,
	session apptest.Session,
	notifierID uuid.UUID,
	update apiclient.NotifierUpdate,
) *apiclient.UpdateNotifierResponse {
	t.Helper()

	response, err := session.Client.UpdateNotifierWithResponse(t.Context(), notifierID, update)
	require.NoError(t, err)

	return response
}

func testNotifier(t *testing.T, session apptest.Session, notifierID uuid.UUID) apiclient.NotifierTestResult {
	t.Helper()

	response, err := session.Client.TestNotifierWithResponse(t.Context(), notifierID)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return *response.JSON200
}

func toSettings(t *testing.T, convert func(settings *apiclient.NotifierSettings) error) apiclient.NotifierSettings {
	t.Helper()

	var settings apiclient.NotifierSettings
	require.NoError(t, convert(&settings))

	return settings
}

func newWebhookSettings(t *testing.T, webhook apiclient.WebhookNotifierSettings) apiclient.NotifierSettings {
	t.Helper()

	return toSettings(t, func(settings *apiclient.NotifierSettings) error {
		return settings.FromWebhookNotifierSettings(webhook)
	})
}

func getInvalidWebhookHeaderNames() []string {
	return []string{"X Team", " ", "", "X:Team", "X-Caf\u00e9"}
}

func newPostWebhook(webhookURL string, headers ...apiclient.WebhookHeader) apiclient.WebhookNotifierSettings {
	return apiclient.WebhookNotifierSettings{
		Type:               apiclient.WebhookNotifierSettingsTypeWEBHOOK,
		WebhookUrl:         webhookURL,
		WebhookMethod:      apiclient.WebhookNotifierSettingsWebhookMethodPOST,
		BodyTemplate:       nil,
		Headers:            append([]apiclient.WebhookHeader{}, headers...),
		AcceptedCategories: []apiclient.NotificationCategory{apiclient.NotificationCategoryALL},
	}
}

func newSlackSettings(t *testing.T, slack apiclient.SlackNotifierSettings) apiclient.NotifierSettings {
	t.Helper()

	return toSettings(t, func(settings *apiclient.NotifierSettings) error {
		return settings.FromSlackNotifierSettings(slack)
	})
}

func newTelegramSettings(t *testing.T, telegram apiclient.TelegramNotifierSettings) apiclient.NotifierSettings {
	t.Helper()

	return toSettings(t, func(settings *apiclient.NotifierSettings) error {
		return settings.FromTelegramNotifierSettings(telegram)
	})
}

func newTelegram(botToken, botAPIURL string) apiclient.TelegramNotifierSettings {
	return apiclient.TelegramNotifierSettings{
		Type:           apiclient.TelegramNotifierSettingsTypeTELEGRAM,
		BotToken:       optional(botToken),
		TargetChatId:   "123456",
		ThreadId:       nil,
		IsProxyEnabled: false,
		ProxyUrl:       nil,
		BotApiUrl:      optional(botAPIURL),
	}
}

func optional(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}

// endpoint is an HTTP server standing in for a channel. It answers every request with its status
// and keeps the requests.
type endpoint struct {
	URL      string
	mutex    sync.Mutex
	requests []receivedRequest
}

type receivedRequest struct {
	Path string
	// RequestURI is the absolute URL of the target when the endpoint serves as a forward proxy.
	RequestURI string
	Headers    http.Header
	Body       string
}

func startEndpoint(t *testing.T, status int) *endpoint {
	t.Helper()

	return startEndpointAnswering(t, status, `{"ok":true}`)
}

// startEndpointAnswering answers with the body, where {path} becomes the path of the request.
func startEndpointAnswering(t *testing.T, status int, responseBody string) *endpoint {
	t.Helper()

	started := &endpoint{URL: "", mutex: sync.Mutex{}, requests: nil}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, _ := io.ReadAll(request.Body)

		started.mutex.Lock()
		started.requests = append(started.requests, receivedRequest{
			Path:       request.URL.Path,
			RequestURI: request.RequestURI,
			Headers:    request.Header.Clone(),
			Body:       string(body),
		})
		started.mutex.Unlock()

		writer.WriteHeader(status)
		_, _ = io.WriteString(writer, strings.ReplaceAll(responseBody, "{path}", request.URL.Path))
	}))
	t.Cleanup(server.Close)
	started.URL = server.URL

	return started
}

func (e *endpoint) getRequests() []receivedRequest {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	return append([]receivedRequest{}, e.requests...)
}
