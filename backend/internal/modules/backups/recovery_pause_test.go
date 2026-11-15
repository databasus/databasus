package backups_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestBackupsDuringRecoveryPause(t *testing.T) {
	t.Parallel()

	t.Run("refuses a manual logical backup with recovery_paused and starts none", func(t *testing.T) {
		t.Parallel()
		tm := newTeam(t, apptest.WithRecoveryPause())
		databaseID := tm.insertLogicalDatabase(t)

		response, err := tm.owner.Client.StartLogicalBackupWithResponse(t.Context(),
			apiclient.LogicalBackupStart{DatabaseId: databaseID})

		require.NoError(t, err)
		requireProblem(
			t,
			response.StatusCode(),
			response.Body,
			http.StatusConflict,
			apiclient.ProblemCodeRecoveryPaused,
		)
		var backupCount int64
		tm.app.QuerySQL(t, &backupCount, "SELECT count(*) FROM backups_logical WHERE database_id = ?", databaseID)
		assert.Zero(t, backupCount)
	})
}
