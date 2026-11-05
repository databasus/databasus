package verification_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestVerificationAgents(t *testing.T) {
	t.Parallel()

	t.Run("returns a token once and pages agents newest first when an admin creates agents", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		createdAgents := make([]apiclient.VerificationAgent, 3)
		for index := range createdAgents {
			created, err := admin.Client.CreateVerificationAgentWithResponse(
				t.Context(),
				apiclient.VerificationAgentInput{Name: "agent"},
			)
			require.NoError(t, err)
			require.Equal(t, http.StatusCreated, created.StatusCode(), string(created.Body))
			require.Len(t, created.JSON201.Token, 32)
			createdAgents[index] = created.JSON201.Agent
			app.Clock.Advance(time.Second)
		}

		first, err := admin.Client.ListVerificationAgentsWithResponse(
			t.Context(),
			&apiclient.ListVerificationAgentsParams{Limit: new(2), After: nil},
		)
		require.NoError(t, err)
		require.NotNil(t, first.JSON200, string(first.Body))
		require.Len(t, first.JSON200.Items, 2)
		require.NotNil(t, first.JSON200.NextCursor)
		assert.Equal(t, createdAgents[2].Id, first.JSON200.Items[0].Id)
		assert.Equal(t, createdAgents[1].Id, first.JSON200.Items[1].Id)
		assert.NotContains(t, string(first.Body), "token")
		assert.NotContains(t, string(first.Body), "deletedAt")

		last, err := admin.Client.ListVerificationAgentsWithResponse(
			t.Context(),
			&apiclient.ListVerificationAgentsParams{Limit: new(2), After: first.JSON200.NextCursor},
		)
		require.NoError(t, err)
		require.NotNil(t, last.JSON200)
		require.Len(t, last.JSON200.Items, 1)
		assert.Equal(t, createdAgents[0].Id, last.JSON200.Items[0].Id)
		assert.Nil(t, last.JSON200.NextCursor)

		rotated, err := admin.Client.RotateVerificationAgentTokenWithResponse(t.Context(), createdAgents[0].Id)
		require.NoError(t, err)
		require.NotNil(t, rotated.JSON200)
		assert.Len(t, rotated.JSON200.Token, 32)

		deleted, err := admin.Client.DeleteVerificationAgentWithResponse(t.Context(), createdAgents[0].Id)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, deleted.StatusCode())

		rotatedDeleted, err := admin.Client.RotateVerificationAgentTokenWithResponse(t.Context(), createdAgents[0].Id)
		require.NoError(t, err)
		requireProblem(
			t,
			rotatedDeleted.StatusCode(),
			rotatedDeleted.Body,
			http.StatusNotFound,
			apiclient.ProblemCodeNotFound,
		)

		repeated, err := admin.Client.DeleteVerificationAgentWithResponse(t.Context(), createdAgents[0].Id)
		require.NoError(t, err)
		requireProblem(t, repeated.StatusCode(), repeated.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
	})
	t.Run("refuses every admin operation when a workspace owner is not an instance admin", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		workspaceOwner := app.CreateNonAdminUser(t, admin)
		apptest.CreateWorkspace(t, workspaceOwner, "Agent administration")
		agent := app.CreateVerificationAgent(t, admin)

		created, err := workspaceOwner.Client.CreateVerificationAgentWithResponse(
			t.Context(),
			apiclient.VerificationAgentInput{Name: "forbidden"},
		)
		require.NoError(t, err)
		requireProblem(
			t,
			created.StatusCode(),
			created.Body,
			http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied,
		)

		listed, err := workspaceOwner.Client.ListVerificationAgentsWithResponse(t.Context(), nil)
		require.NoError(t, err)
		requireProblem(t, listed.StatusCode(), listed.Body, http.StatusForbidden, apiclient.ProblemCodePermissionDenied)

		rotated, err := workspaceOwner.Client.RotateVerificationAgentTokenWithResponse(t.Context(), agent.Agent.ID)
		require.NoError(t, err)
		requireProblem(
			t,
			rotated.StatusCode(),
			rotated.Body,
			http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied,
		)

		deleted, err := workspaceOwner.Client.DeleteVerificationAgentWithResponse(t.Context(), agent.Agent.ID)
		require.NoError(t, err)
		requireProblem(
			t,
			deleted.StatusCode(),
			deleted.Body,
			http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied,
		)

		assert.Len(t, app.ListVerificationAgents(t, admin), 1)
		status, body := newProtocolAgent(t, app, agent).recordHeartbeat(agent.Agent.ID, heartbeatBody(nil))
		require.Equal(t, http.StatusOK, status, string(body))
	})
	t.Run("returns unauthenticated when creating or listing without a session", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		agent := app.CreateVerificationAgent(t, admin)
		client := app.NewClient(t)

		created, err := client.CreateVerificationAgentWithResponse(
			t.Context(),
			apiclient.VerificationAgentInput{Name: "no session"},
		)
		require.NoError(t, err)
		requireProblem(
			t,
			created.StatusCode(),
			created.Body,
			http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated,
		)

		listed, err := client.ListVerificationAgentsWithResponse(t.Context(), nil)
		require.NoError(t, err)
		requireProblem(
			t,
			listed.StatusCode(),
			listed.Body,
			http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated,
		)

		rotated, err := client.RotateVerificationAgentTokenWithResponse(t.Context(), agent.Agent.ID)
		require.NoError(t, err)
		requireProblem(
			t,
			rotated.StatusCode(),
			rotated.Body,
			http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated,
		)

		deleted, err := client.DeleteVerificationAgentWithResponse(t.Context(), agent.Agent.ID)
		require.NoError(t, err)
		requireProblem(
			t,
			deleted.StatusCode(),
			deleted.Body,
			http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated,
		)

		assert.Equal(t, []apptest.VerificationAgent{agent.Agent}, app.ListVerificationAgents(t, admin))
	})
	t.Run("refuses invalid input and a foreign cursor without changing agents", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		created, err := admin.Client.CreateVerificationAgentWithResponse(
			t.Context(),
			apiclient.VerificationAgentInput{Name: ""},
		)
		require.NoError(t, err)
		requireProblem(
			t,
			created.StatusCode(),
			created.Body,
			http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest,
		)

		listed, err := admin.Client.ListVerificationAgentsWithResponse(
			t.Context(),
			&apiclient.ListVerificationAgentsParams{Limit: nil, After: new("foreign")},
		)
		require.NoError(t, err)
		requireProblem(t, listed.StatusCode(), listed.Body, http.StatusBadRequest, apiclient.ProblemCodeInvalidRequest)

		rotated, err := admin.Client.RotateVerificationAgentTokenWithResponse(t.Context(), uuid.New())
		require.NoError(t, err)
		requireProblem(t, rotated.StatusCode(), rotated.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
		deleted, err := admin.Client.DeleteVerificationAgentWithResponse(t.Context(), uuid.New())
		require.NoError(t, err)
		requireProblem(t, deleted.StatusCode(), deleted.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
	})
}

func requireProblem(t *testing.T, status int, body []byte, expectedStatus int, expectedCode apiclient.ProblemCode) {
	t.Helper()
	require.Equal(t, expectedStatus, status, string(body))
	var problem apiclient.Problem
	require.NoError(t, json.Unmarshal(body, &problem))
	assert.Equal(t, expectedCode, problem.Code)
}

func TestVerificationAgentAudit(t *testing.T) {
	t.Parallel()

	t.Run("records agent changes without a workspace and returns a hex token when an admin acts", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		created, err := admin.Client.CreateVerificationAgentWithResponse(
			t.Context(),
			apiclient.VerificationAgentInput{Name: "audited-agent"},
		)
		require.NoError(t, err)
		require.NotNil(t, created.JSON201)
		assert.Regexp(t, "^[0-9a-f]{32}$", created.JSON201.Token)

		rotated, err := admin.Client.RotateVerificationAgentTokenWithResponse(t.Context(), created.JSON201.Agent.Id)
		require.NoError(t, err)
		require.NotNil(t, rotated.JSON200)
		deleted, err := admin.Client.DeleteVerificationAgentWithResponse(t.Context(), created.JSON201.Agent.Id)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, deleted.StatusCode())

		events := apptest.ListAllAuditEvents(
			t,
			admin,
			apiclient.ListAuditEventsParams{WorkspaceId: nil, AuthorUserId: &admin.UserID, Limit: new(100), After: nil},
		)

		var agentEventCount int
		for _, event := range events {
			if strings.Contains(event.Message, "audited-agent") {
				agentEventCount++
				assert.Nil(t, event.Workspace)
			}
		}
		assert.Equal(t, 3, agentEventCount)
	})
}
