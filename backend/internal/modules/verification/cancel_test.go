package verification_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestCancelVerification(t *testing.T) {
	t.Parallel()

	t.Run("cancels a pending verification when a member asks", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		backup := team.backup(t)
		verification := team.start(t, backup.Id)
		agent := team.App.CreateVerificationAgent(t, team.Admin)

		refused, err := team.App.NewSessionClient(t, agent.Token).
			CancelVerificationWithResponse(t.Context(), verification.Id)
		require.NoError(t, err)
		requireProblem(
			t,
			refused.StatusCode(),
			refused.Body,
			http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated,
		)
		assert.Equal(t, apiclient.VerificationStatusPENDING, team.read(t, verification.Id).Status)

		response, err := team.Owner.Client.CancelVerificationWithResponse(t.Context(), verification.Id)
		require.NoError(t, err)
		require.NotNil(t, response.JSON200, string(response.Body))
		assert.Equal(t, apiclient.VerificationStatusCANCELED, response.JSON200.Status)
		assert.Equal(t, response.JSON200.Status, team.read(t, verification.Id).Status)
	})
	for _, status := range []apiclient.VerificationStatus{apiclient.VerificationStatusCOMPLETED, apiclient.VerificationStatusFAILED, apiclient.VerificationStatusCANCELED} {
		t.Run("returns not_running when the verification is "+string(status), func(t *testing.T) {
			t.Parallel()

			team := newVerificationTeam(t)
			backup := team.backup(t)
			verification := team.start(t, backup.Id)

			team.App.ExecuteSQL(
				t,
				"UPDATE verification_runs SET status=?,finished_at=? WHERE id=?",
				status,
				team.App.Clock.Now(),
				verification.Id,
			)

			response, err := team.Owner.Client.CancelVerificationWithResponse(t.Context(), verification.Id)
			require.NoError(t, err)
			requireProblem(
				t,
				response.StatusCode(),
				response.Body,
				http.StatusConflict,
				apiclient.ProblemCodeNotRunning,
			)
			assert.Equal(t, status, team.read(t, verification.Id).Status)
		})
	}
}
