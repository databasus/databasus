package backups_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

const logicalRetentionJobName = "backups_logical_retention"

//nolint:tparallel // The subtests share one source server.
func TestWorkWhileRestoreRuns(t *testing.T) {
	t.Parallel()

	server := startRestoreServer(t, restoreServerImage)

	t.Run("returns 409 in_use and keeps the backup when a restore reads it on delete", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)
		startBlockedRestore(t, tm.owner, server, database.backup.Id)

		response, err := tm.owner.Client.DeleteLogicalBackupWithResponse(t.Context(), database.backup.Id)

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict, apiclient.ProblemCodeInUse)
		assert.Contains(t, listLogicalBackupIDs(t, tm.owner, database.databaseID), database.backup.Id)
	})

	t.Run("keeps an expired backup while a restore reads it and removes it after the restore", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)
		keepBackupsForOneDay(t, tm, database)
		runningRestore, _ := startBlockedRestore(t, tm.owner, server, database.backup.Id)
		tm.app.Clock.Advance(48 * time.Hour)

		tm.app.RunJobPass(t, logicalRetentionJobName)

		assert.Contains(t, listLogicalBackupIDs(t, tm.owner, database.databaseID), database.backup.Id)

		response := cancelRestore(t, tm.owner, runningRestore.Id)
		require.Equal(t, http.StatusAccepted, response.StatusCode(), string(response.Body))
		waitForRestoreStatus(t, tm.owner, runningRestore.Id, apiclient.RestoreStatusCANCELED)

		tm.app.RunJobPass(t, logicalRetentionJobName)

		assert.NotContains(t, listLogicalBackupIDs(t, tm.owner, database.databaseID), database.backup.Id)
	})

	t.Run("returns 409 in_use and keeps the database when a restore of its backup runs", func(t *testing.T) {
		tm := newTeam(t)
		database := newBackedUpDatabase(t, tm, server)
		startBlockedRestore(t, tm.owner, server, database.backup.Id)

		response, err := tm.owner.Client.DeleteDatabaseWithResponse(t.Context(), database.databaseID)

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict, apiclient.ProblemCodeInUse)

		databaseResponse, err := tm.owner.Client.GetDatabaseWithResponse(t.Context(), database.databaseID)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, databaseResponse.StatusCode(), string(databaseResponse.Body))
		assert.Contains(t, listLogicalBackupIDs(t, tm.owner, database.databaseID), database.backup.Id)
	})
}

// keepBackupsForOneDay saves the database's config with the time period retention of one day.
func keepBackupsForOneDay(t *testing.T, tm team, database backedUpDatabase) {
	t.Helper()

	logicalSettings := getLogicalSettings(t, database.backupConfig)
	logicalSettings.Retention.Policy = apiclient.LogicalRetentionPolicyTIMEPERIOD
	logicalSettings.Retention.Period = apiclient.RetentionPeriodDAY

	response := updateBackupConfig(t, tm.owner, database.databaseID,
		newLogicalBackupConfigInput(t, &database.storageID, logicalSettings))
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
}
