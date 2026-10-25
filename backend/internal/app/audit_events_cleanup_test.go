package app_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
)

func TestAuditEventsCleanup(t *testing.T) {
	t.Run("deletes an event older than one year and keeps a younger one", func(t *testing.T) {
		testApp := apptest.New(t)
		token := signUpThroughOldAPI(t, testApp)
		expiringWorkspaceID := createWorkspaceThroughOldAPI(t, testApp, token, "expiring-"+uuid.NewString())

		testApp.Clock.Advance(366 * 24 * time.Hour)
		keptWorkspaceID := createWorkspaceThroughOldAPI(t, testApp, token, "kept-"+uuid.NewString())

		testApp.RunJobPass(t, "audit_events_cleanup")

		assert.Empty(t, listWorkspaceAuditMessages(t, testApp, token, expiringWorkspaceID))
		assert.NotEmpty(t, listWorkspaceAuditMessages(t, testApp, token, keptWorkspaceID))
	})
}

func signUpThroughOldAPI(t *testing.T, testApp *apptest.App) string {
	t.Helper()

	var signedUp struct {
		Token string `json:"token"`
	}
	callOldAPI(t, testApp, oldAPICall{
		method: http.MethodPost,
		path:   "/api/v1/users/signup",
		token:  "",
		body: map[string]string{
			"email":    "owner-" + uuid.NewString() + "@example.com",
			"password": "correct-horse-battery",
			"name":     "Owner",
		},
	}, &signedUp)
	require.NotEmpty(t, signedUp.Token)

	return signedUp.Token
}

func createWorkspaceThroughOldAPI(t *testing.T, testApp *apptest.App, token, name string) uuid.UUID {
	t.Helper()

	var createdWorkspace struct {
		ID uuid.UUID `json:"id"`
	}
	callOldAPI(t, testApp, oldAPICall{
		method: http.MethodPost,
		path:   "/api/v1/workspaces",
		token:  token,
		body:   map[string]string{"name": name},
	}, &createdWorkspace)

	return createdWorkspace.ID
}

func listWorkspaceAuditMessages(t *testing.T, testApp *apptest.App, token string, workspaceID uuid.UUID) []string {
	t.Helper()

	var listedLogs struct {
		AuditLogs []struct {
			Message string `json:"message"`
		} `json:"auditLogs"`
	}
	callOldAPI(t, testApp, oldAPICall{
		method: http.MethodGet,
		path:   "/api/v1/workspaces/" + workspaceID.String() + "/audit-logs",
		token:  token,
		body:   nil,
	}, &listedLogs)

	var messages []string
	for _, listedLog := range listedLogs.AuditLogs {
		messages = append(messages, listedLog.Message)
	}

	return messages
}

type oldAPICall struct {
	method string
	path   string
	token  string
	body   any
}

// callOldAPI calls an operation the contract does not describe yet, so it cannot go through the
// contract-checking client.
func callOldAPI(t *testing.T, testApp *apptest.App, call oldAPICall, response any) {
	t.Helper()

	var requestBody bytes.Buffer
	if call.body != nil {
		require.NoError(t, json.NewEncoder(&requestBody).Encode(call.body))
	}

	request, err := http.NewRequestWithContext(t.Context(), call.method, testApp.URL+call.path, &requestBody)
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")

	if call.token != "" {
		request.Header.Set("Authorization", "Bearer "+call.token)
	}

	httpResponse, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer func() { _ = httpResponse.Body.Close() }()

	require.Less(t, httpResponse.StatusCode, http.StatusMultipleChoices, "%s %s", call.method, call.path)
	require.NoError(t, json.NewDecoder(httpResponse.Body).Decode(response))
}
