package databases_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestDatabasesDuringRecoveryPause(t *testing.T) {
	t.Parallel()

	t.Run("refuses creating a limited user with recovery_paused without connecting", func(t *testing.T) {
		t.Parallel()
		tm := newTeam(t, apptest.WithRecoveryPause())
		recorder := startConnectionRecorder(t)
		target := recorder.getPostgresqlTarget()
		target.password = new("source-password")

		response, err := tm.owner.Client.CreateLimitedUserWithResponse(t.Context(), newSettingsTarget(
			tm.workspace.Id, nil, newPostgresqlLogicalSettings(t, target),
		))

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict,
			apiclient.ProblemCodeRecoveryPaused)
		assert.Zero(t, recorder.getAcceptedCount())
	})

	t.Run("refuses deleting a database with recovery_paused and keeps it", func(t *testing.T) {
		t.Parallel()
		tm := newTeam(t, apptest.WithRecoveryPause())
		databaseID := tm.insertMysqlDatabase(t)

		response, err := tm.owner.Client.DeleteDatabaseWithResponse(t.Context(), databaseID)

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict,
			apiclient.ProblemCodeRecoveryPaused)
		assert.Equal(t, 1, tm.countRows(t, "SELECT count(*) FROM databases WHERE id = ?", databaseID))
	})
}
