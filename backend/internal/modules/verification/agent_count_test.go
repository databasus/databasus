package verification_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestVerificationAgentCount(t *testing.T) {
	t.Parallel()

	t.Run("returns only the number of live agents when a member asks", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.CreateNonAdminUser(t, admin)
		empty, err := member.Client.GetVerificationAgentCountWithResponse(t.Context())
		require.NoError(t, err)
		require.NotNil(t, empty.JSON200)
		assert.Zero(t, empty.JSON200.AgentCount)

		unauthenticated, err := app.NewClient(t).GetVerificationAgentCountWithResponse(t.Context())
		require.NoError(t, err)
		requireProblem(
			t,
			unauthenticated.StatusCode(),
			unauthenticated.Body,
			http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated,
		)

		deletedAgent := app.CreateVerificationAgent(t, admin)
		app.CreateVerificationAgent(t, admin)
		app.CreateVerificationAgent(t, admin)
		deleted, err := admin.Client.DeleteVerificationAgentWithResponse(t.Context(), deletedAgent.Agent.ID)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, deleted.StatusCode())

		count, err := member.Client.GetVerificationAgentCountWithResponse(t.Context())
		require.NoError(t, err)
		require.NotNil(t, count.JSON200)
		assert.Equal(t, int64(2), count.JSON200.AgentCount)
		assert.JSONEq(t, `{"agentCount":2}`, string(count.Body))
	})
}
