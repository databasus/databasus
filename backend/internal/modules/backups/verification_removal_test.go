package backups_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestRemoveBackupVerifications(t *testing.T) {
	server := startRestoreServer(t, restoreServerImage)

	t.Run("removes completed running and pending verifications when a member deletes their backup", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)
		tm.app.EnqueueVerification(t, tm.owner, database.backup.Id)
		for _, status := range []string{"COMPLETED", "RUNNING"} {
			tm.app.ExecuteSQL(t, `INSERT INTO verification_runs (database_id, backup_id, trigger, status)
    VALUES (?, ?, 'MANUAL', ?)`, database.databaseID, database.backup.Id, status)
		}
		require.Len(t, listDatabaseVerifications(t, tm, database.databaseID), 3)

		response := requestLogicalBackupDeletion(t, tm.owner, database.backup.Id)

		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Empty(t, listDatabaseVerifications(t, tm, database.databaseID))
		assert.Empty(t, listLogicalBackupIDs(t, tm.owner, database.databaseID))
	})

	t.Run("removes expired verifications when retention drops their backup", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)
		settings := newLogicalBackupSettings()
		settings.Retention.Policy = apiclient.LogicalRetentionPolicyCOUNT
		settings.Retention.Count = 1
		saveLogicalBackupSettings(
			t,
			tm.owner,
			database.databaseID,
			database.storageID,
			settings,
			apiclient.BackupEncryptionNONE,
		)
		tm.app.Clock.Advance(2 * time.Hour)
		keptBackup := completeRestorableBackup(t, tm.owner, database.databaseID)
		keptVerificationID := tm.app.EnqueueVerification(t, tm.owner, keptBackup.Id)
		tm.app.ExecuteSQL(t, `INSERT INTO verification_runs (database_id, backup_id, trigger, status)
   VALUES (?, ?, 'MANUAL', 'PENDING')`, database.databaseID, database.backup.Id)
		require.Len(t, listDatabaseVerifications(t, tm, database.databaseID), 2)

		tm.app.RunJobPass(t, logicalBackupRetentionJobName)

		assert.Equal(t, []uuid.UUID{keptBackup.Id}, listLogicalBackupIDs(t, tm.owner, database.databaseID))
		verifications := listDatabaseVerifications(t, tm, database.databaseID)
		require.Len(t, verifications, 1)
		assert.Equal(t, keptVerificationID, verifications[0].ID)
	})

	t.Run("removes verifications when the backup config changes storage", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)
		tm.app.EnqueueVerification(t, tm.owner, database.backup.Id)
		replacementStorageID := tm.createLocalStorage(t)

		saveLogicalBackupConfig(t, tm.owner, database.databaseID, replacementStorageID)

		assert.Empty(t, listLogicalBackupIDs(t, tm.owner, database.databaseID))
		assert.Empty(t, listDatabaseVerifications(t, tm, database.databaseID))
	})

	t.Run("removes verifications and their config when their database is deleted", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)
		tm.app.EnqueueVerification(t, tm.owner, database.backup.Id)
		readVerificationConfig(t, tm, database.databaseID)

		response, err := tm.owner.Client.DeleteDatabaseWithResponse(t.Context(), database.databaseID)

		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))
		assert.Zero(
			t,
			countRows(t, tm, "SELECT COUNT(*) FROM verification_runs WHERE database_id = ?", database.databaseID),
		)
		assert.Zero(
			t,
			countRows(t, tm, "SELECT COUNT(*) FROM verification_configs WHERE database_id = ?", database.databaseID),
		)
		assert.Zero(
			t,
			countRows(t, tm, "SELECT COUNT(*) FROM backups_logical WHERE database_id = ?", database.databaseID),
		)
	})

	t.Run("keeps verifications and their config when a running backup refuses database deletion", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)
		verificationID := tm.app.EnqueueVerification(t, tm.owner, database.backup.Id)
		configBeforeDeletion := readVerificationConfig(t, tm, database.databaseID)
		startBlockedBackup(t, tm, server, database)

		response, err := tm.owner.Client.DeleteDatabaseWithResponse(t.Context(), database.databaseID)

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict, apiclient.ProblemCodeInUse)
		assert.Contains(t, listLogicalBackupIDs(t, tm.owner, database.databaseID), database.backup.Id)
		verifications := listDatabaseVerifications(t, tm, database.databaseID)
		require.Len(t, verifications, 1)
		assert.Equal(t, verificationID, verifications[0].ID)
		assert.Equal(t, configBeforeDeletion, readVerificationConfig(t, tm, database.databaseID))
	})
}

func listDatabaseVerifications(t *testing.T, tm team, databaseID uuid.UUID) []apptest.Verification {
	t.Helper()
	status, body := tm.app.CallOldAPI(t, tm.owner, apptest.OldAPICall{
		Method: http.MethodGet, Path: "/verifications/by-database/" + databaseID.String(), Body: nil,
	})
	require.Equal(t, http.StatusOK, status, string(body))
	var page struct {
		Verifications []apptest.Verification `json:"verifications"`
	}
	require.NoError(t, json.Unmarshal(body, &page))
	return page.Verifications
}

func readVerificationConfig(t *testing.T, tm team, databaseID uuid.UUID) map[string]any {
	t.Helper()
	status, body := tm.app.CallOldAPI(t, tm.owner, apptest.OldAPICall{
		Method: http.MethodGet, Path: "/verification-config/" + databaseID.String(), Body: nil,
	})
	require.Equal(t, http.StatusOK, status, string(body))
	var config map[string]any
	require.NoError(t, json.Unmarshal(body, &config))
	return config
}
