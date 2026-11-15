package verification_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestVerificationsDuringRecoveryPause(t *testing.T) {
	t.Parallel()

	t.Run("refuses to start a verification with recovery_paused", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t, apptest.WithRecoveryPause())
		admin := app.CreateAdmin(t)
		workspace := apptest.CreateWorkspace(t, admin, "Team "+uuid.NewString()[:8])
		databaseID := app.InsertDatabase(t, workspace.Id)
		storageID := app.CreateLocalStorage(t, admin, workspace.Id)
		backupID := uuid.New()
		app.ExecuteSQL(t, `INSERT INTO backups_logical (id, database_id, storage_id, status, file_name)
			VALUES (?, ?, ?, 'COMPLETED', 'recovered.dump')`, backupID, databaseID, storageID)

		response, err := admin.Client.StartVerificationWithResponse(t.Context(),
			apiclient.StartVerificationInput{BackupId: backupID})

		require.NoError(t, err)
		requireProblem(
			t,
			response.StatusCode(),
			response.Body,
			http.StatusConflict,
			apiclient.ProblemCodeRecoveryPaused,
		)
		var runCount int64
		app.QuerySQL(t, &runCount, "SELECT count(*) FROM verification_runs WHERE backup_id = ?", backupID)
		assert.Zero(t, runCount)
	})

	t.Run("hands an agent no verification while paused", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t, apptest.WithRecoveryPause())
		admin := app.CreateAdmin(t)
		workspace := apptest.CreateWorkspace(t, admin, "Team "+uuid.NewString()[:8])
		databaseID := app.InsertDatabase(t, workspace.Id)
		storageID := app.CreateLocalStorage(t, admin, workspace.Id)
		backupID := uuid.New()
		app.ExecuteSQL(t, `INSERT INTO backups_logical (id, database_id, storage_id, status, file_name)
			VALUES (?, ?, ?, 'COMPLETED', 'recovered.dump')`, backupID, databaseID, storageID)
		runID := uuid.New()
		app.ExecuteSQL(t, `INSERT INTO verification_runs (id, database_id, backup_id, trigger, status)
			VALUES (?, ?, ?, 'MANUAL', 'PENDING')`, runID, databaseID, backupID)
		agent := app.CreateVerificationAgent(t, admin)
		protocol := newProtocolAgent(t, app, agent)
		protocol.requireAcceptedHeartbeat(nil)

		status, body := protocol.claimVerification(claimBody(5))

		assert.Equal(t, http.StatusNoContent, status, string(body))
		var runStatuses []string
		app.QuerySQL(t, &runStatuses, "SELECT status FROM verification_runs WHERE id = ?", runID)
		assert.Equal(t, []string{"PENDING"}, runStatuses)

		activation, err := admin.Client.ActivateInstanceRecoveryWithResponse(t.Context(),
			app.GetPausedRecoveryID(), apiclient.InstanceRecoveryActivation{IsOldInstanceStopped: true})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, activation.StatusCode(), string(activation.Body))
		protocol.requireAcceptedHeartbeat(nil)
		status, body = protocol.claimVerification(claimBody(6))
		assert.Equal(t, http.StatusOK, status, string(body))
	})
}
