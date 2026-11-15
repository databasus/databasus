package e2e

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

// accountPassword is the password of every account the scenarios sign up.
const accountPassword = "recovery-cycle-password-123"

// accountSession is a signed-in account of one running instance.
type accountSession struct {
	Client *apiclient.ClientWithResponses
	UserID uuid.UUID
	Email  string
	Token  string
}

func newAccountEmail(label string) string {
	return label + "-" + uuid.NewString()[:8] + "@example.com"
}

func signUp(t *testing.T, instance runningInstance, email string) accountSession {
	t.Helper()

	response, err := instance.newClient(t).SignUpWithResponse(t.Context(), apiclient.SignUpRequest{
		Email:                    openapi_types.Email(email),
		Password:                 new(accountPassword),
		Name:                     "Recovery cycle account",
		CloudflareTurnstileToken: nil,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))

	return accountSession{
		Client: instance.newSessionClient(t, response.JSON201.Token),
		UserID: response.JSON201.UserId,
		Email:  email,
		Token:  response.JSON201.Token,
	}
}

func signIn(t *testing.T, instance runningInstance, email string) accountSession {
	t.Helper()

	response := requestSignIn(t, instance, email)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	completed, err := response.JSON200.AsSignInCompleted()
	require.NoError(t, err)

	return accountSession{
		Client: instance.newSessionClient(t, completed.Token),
		UserID: completed.UserId,
		Email:  email,
		Token:  completed.Token,
	}
}

func requestSignIn(t *testing.T, instance runningInstance, email string) *apiclient.SignInResponse {
	t.Helper()

	response, err := instance.newClient(t).SignInWithResponse(t.Context(), apiclient.SignInRequest{
		Email:                    email,
		Password:                 new(accountPassword),
		CloudflareTurnstileToken: nil,
	})
	require.NoError(t, err)

	return response
}

// getOwnAccount reads the session's own account, which answers 401 once the session ended.
func (s accountSession) getOwnAccount(t *testing.T) *apiclient.GetUserResponse {
	t.Helper()

	response, err := s.Client.GetUserWithResponse(t.Context(), s.UserID)
	require.NoError(t, err)

	return response
}

func (s accountSession) getInstanceBackupStatus(t *testing.T) apiclient.InstanceBackupStatus {
	t.Helper()

	response, err := s.Client.GetInstanceBackupStatusWithResponse(t.Context())
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return *response.JSON200
}

func (s accountSession) activateRecovery(t *testing.T, recoveryID uuid.UUID) apiclient.InstanceRecovery {
	t.Helper()

	response, err := s.Client.ActivateInstanceRecoveryWithResponse(t.Context(), recoveryID,
		apiclient.InstanceRecoveryActivation{IsOldInstanceStopped: true})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return *response.JSON200
}

func (s accountSession) listWorkspaceNames(t *testing.T) []string {
	t.Helper()

	response, err := s.Client.ListWorkspacesWithResponse(t.Context(), nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	workspaceNames := []string{}
	for _, workspace := range response.JSON200.Items {
		workspaceNames = append(workspaceNames, workspace.Name)
	}

	return workspaceNames
}

func (s accountSession) createWorkspace(t *testing.T, name string) apiclient.Workspace {
	t.Helper()

	response, err := s.Client.CreateWorkspaceWithResponse(t.Context(), apiclient.WorkspaceCreate{Name: name})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))

	return *response.JSON201
}

// requireProblem checks the status and the stable code of a refused request.
func requireProblem(t *testing.T, status int, body []byte, expectedStatus int, code apiclient.ProblemCode) {
	t.Helper()

	require.Equal(t, expectedStatus, status, string(body))
	require.Contains(t, string(body), `"code":"`+string(code)+`"`)
}
