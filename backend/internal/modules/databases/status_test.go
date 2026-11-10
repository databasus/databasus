package databases_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/modules/databases"
)

//nolint:tparallel // The subtests share one source server.
func TestDatabaseStatusUpdates(t *testing.T) {
	t.Parallel()

	server := containers.StartPostgres(t, "postgres:16")

	t.Run("keeps the password a user saved before a backup finishes", func(t *testing.T) {
		tm := newTeam(t)
		databaseID := tm.app.InsertDatabase(t, tm.workspace.Id)
		tm.app.ExecuteSQL(t, "UPDATE databases_postgresql_logical SET password = ? WHERE database_id = ?",
			tm.app.EncryptSecret(t, "password-saved-meanwhile"), databaseID)
		backupTime := time.Now().UTC().Truncate(time.Second)

		require.NoError(t, tm.app.RecordBackupSuccess(t.Context(), databaseID, backupTime))

		assert.Equal(t, "password-saved-meanwhile",
			tm.app.DecryptSecret(t, getStoredPassword(t, tm, "databases_postgresql_logical", databaseID)))
		readDatabase := getDatabase(t, tm.owner, databaseID)
		require.NotNil(t, readDatabase.LastBackupTime)
		assert.True(t, backupTime.Equal(*readDatabase.LastBackupTime), readDatabase.LastBackupTime)
	})

	t.Run("keeps the new name when a health check records its result after a rename", func(t *testing.T) {
		tm := newTeam(t)
		createdDatabase := tm.createDatabaseNamed(t, "Orders", fromLogicalSettings(t, newLogicalSettings(server)))
		update := toUpdate(createdDatabase)
		update.Name = "Orders renamed"

		response := updateDatabase(t, tm.owner, createdDatabase.Id, update)
		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		unavailable := databases.HealthStatusUnavailable

		require.NoError(t, tm.app.SetHealthStatus(t.Context(), createdDatabase.Id, &unavailable))

		readDatabase := getDatabase(t, tm.owner, createdDatabase.Id)
		assert.Equal(t, "Orders renamed", readDatabase.Name)
		assert.Equal(t, new(apiclient.DatabaseHealthStatusUNAVAILABLE), readDatabase.HealthStatus)
	})

	t.Run("shows the error and changes no setting when a backup failure is recorded", func(t *testing.T) {
		tm := newTeam(t)
		databaseID := tm.app.InsertDatabase(t, tm.workspace.Id)
		databaseBefore := getDatabase(t, tm.owner, databaseID)
		passwordBefore := getStoredPassword(t, tm, "databases_postgresql_logical", databaseID)

		require.NoError(t, tm.app.RecordBackupFailure(
			t.Context(), databaseID, "connection refused",
		))

		databaseAfter := getDatabase(t, tm.owner, databaseID)
		assert.Equal(t, new("connection refused"), databaseAfter.LastBackupErrorMessage)
		assert.Equal(t, databaseBefore.Name, databaseAfter.Name)
		assert.Equal(t, getLogicalSettings(t, databaseBefore), getLogicalSettings(t, databaseAfter))
		assert.Equal(t, passwordBefore, getStoredPassword(t, tm, "databases_postgresql_logical", databaseID))
	})

	t.Run("returns ErrDatabaseNotFound for every status write when the database no longer exists",
		func(t *testing.T) {
			tm := newTeam(t)
			missingDatabaseID := uuid.New()
			available := databases.HealthStatusAvailable

			successErr := tm.app.RecordBackupSuccess(
				t.Context(), missingDatabaseID, time.Now().UTC(),
			)
			failureErr := tm.app.RecordBackupFailure(
				t.Context(), missingDatabaseID, "connection refused",
			)
			healthErr := tm.app.SetHealthStatus(t.Context(), missingDatabaseID, &available)

			require.ErrorIs(t, successErr, databases.ErrDatabaseNotFound)
			require.ErrorIs(t, failureErr, databases.ErrDatabaseNotFound)
			require.ErrorIs(t, healthErr, databases.ErrDatabaseNotFound)
		})
}
