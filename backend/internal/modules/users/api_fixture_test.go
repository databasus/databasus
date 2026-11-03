package users_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

// requireProblem checks the status and the code of an error response.
func requireProblem(t *testing.T, status int, body []byte, expectedStatus int, expectedCode apiclient.ProblemCode) {
	t.Helper()

	require.Equal(t, expectedStatus, status, string(body))

	var problem apiclient.Problem
	require.NoError(t, json.Unmarshal(body, &problem), string(body))
	require.Equal(t, expectedCode, problem.Code, string(body))
}

func signIn(t *testing.T, app *apptest.App, email, password string) *apiclient.SignInResponse {
	t.Helper()

	response, err := app.NewClient(t).SignInWithResponse(t.Context(), apiclient.SignInRequest{
		Email:                    email,
		Password:                 &password,
		CloudflareTurnstileToken: nil,
	})
	require.NoError(t, err)

	return response
}

// requireCompletedSignIn checks that the sign-in answered a session and returns it.
func requireCompletedSignIn(t *testing.T, response *apiclient.SignInResponse) apiclient.SignInCompleted {
	t.Helper()

	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	completedSignIn, err := response.JSON200.AsSignInCompleted()
	require.NoError(t, err)
	require.Equal(t, apiclient.SignInCompletedOutcomeCOMPLETED, completedSignIn.Outcome, string(response.Body))

	return completedSignIn
}

func getUser(t *testing.T, client *apiclient.ClientWithResponses, userID apiclient.UserId) *apiclient.GetUserResponse {
	t.Helper()

	response, err := client.GetUserWithResponse(t.Context(), userID)
	require.NoError(t, err)

	return response
}

// sendRawRequest sends a request the contract cannot describe, such as a malformed body.
func sendRawRequest(t *testing.T, app *apptest.App, method, path, body string) (int, []byte) {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), method, app.URL+"/api/v1"+path, bytes.NewBufferString(body))
	require.NoError(t, err)

	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)

	defer func() { _ = response.Body.Close() }()

	responseBody, err := io.ReadAll(response.Body)
	require.NoError(t, err)

	return response.StatusCode, responseBody
}

func updateInstanceSettings(
	t *testing.T,
	client *apiclient.ClientWithResponses,
	update apiclient.InstanceSettingsUpdate,
) *apiclient.UpdateInstanceSettingsResponse {
	t.Helper()

	response, err := client.UpdateInstanceSettingsWithResponse(t.Context(), update)
	require.NoError(t, err)

	return response
}

// defaultSettingsUpdate is the update that keeps the settings of a new instance.
func defaultSettingsUpdate() apiclient.InstanceSettingsUpdate {
	return apiclient.InstanceSettingsUpdate{
		IsAllowExternalRegistrations:      true,
		IsAllowMemberInvitations:          true,
		IsMemberAllowedToCreateWorkspaces: true,
		IsTwoFactorAuthRequired:           false,
	}
}

func listGlobalAuditMessages(t *testing.T, admin apptest.Session) []string {
	t.Helper()

	globalEvents := apptest.ListAllAuditEvents(t, admin, apiclient.ListAuditEventsParams{})

	messages := make([]string, 0, len(globalEvents))
	for _, event := range globalEvents {
		messages = append(messages, event.Message)
	}

	return messages
}

func jsonBody(t *testing.T, body string) io.Reader {
	t.Helper()

	return bytes.NewBufferString(body)
}

var mailedCodePattern = regexp.MustCompile(`>(\d{6})<`)

func readLatestMailedCode(t *testing.T, app *apptest.App, email string) string {
	t.Helper()

	messages := app.Mail.GetMessagesTo(email)
	require.NotEmpty(t, messages, "no message went to %s", email)

	match := mailedCodePattern.FindStringSubmatch(messages[len(messages)-1].HTMLBody)
	require.Len(t, match, 2, "the latest message to %s carries no code", email)

	return match[1]
}
