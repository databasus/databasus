package databases_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

//nolint:tparallel // The subtests share one source server.
func TestTestDatabase(t *testing.T) {
	t.Parallel()

	postgresServer := containers.StartPostgres(t, "postgres:16")

	t.Run("reports a successful connection when a viewer tests a saved database", func(t *testing.T) {
		tm := newTeam(t)
		viewer := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
		savedDatabase := tm.createDatabase(
			t,
			newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer)),
		)

		testResult := testSavedDatabase(t, viewer, savedDatabase.Id)

		assert.True(t, testResult.IsConnected)
		assert.Nil(t, testResult.Error)
		assert.Nil(t, testResult.Reason)
	})

	t.Run("reports the failure and keeps the backup error when the server refuses the stored password",
		func(t *testing.T) {
			tm := newTeam(t)
			member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
			savedDatabase := tm.createDatabase(t,
				newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer)))
			tm.app.ExecuteSQL(t, "UPDATE databases SET last_backup_error_message = 'disk full' WHERE id = ?",
				savedDatabase.Id)
			tm.app.ExecuteSQL(t, "UPDATE databases_postgresql_logical SET password = ? WHERE database_id = ?",
				tm.app.EncryptSecret(t, "not-the-password"), savedDatabase.Id)
			databaseBeforeTest := tm.readDatabase(t, savedDatabase.Id)

			testResult := testSavedDatabase(t, member, savedDatabase.Id)

			assert.False(t, testResult.IsConnected)
			require.NotNil(t, testResult.Error)
			assert.Contains(t, *testResult.Error, "password authentication failed")

			databaseAfterTest := tm.readDatabase(t, savedDatabase.Id)
			require.NotNil(t, databaseAfterTest.LastBackupErrorMessage)
			assert.Equal(t, "disk full", *databaseAfterTest.LastBackupErrorMessage)
			assert.Equal(t, databaseBeforeTest, databaseAfterTest)
		})

	t.Run("returns 404 not_found without connecting when an outsider tests another team's database",
		func(t *testing.T) {
			tm := newTeam(t)
			outsider := tm.signUpNonMember(t)
			recorder := startConnectionRecorder(t)
			savedDatabase := tm.createDatabase(t,
				newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer)))
			tm.app.ExecuteSQL(t,
				"UPDATE databases_postgresql_logical SET host = ?, port = ? WHERE database_id = ?",
				recorder.host, recorder.port, savedDatabase.Id)

			response, err := outsider.Client.TestDatabaseWithResponse(t.Context(), savedDatabase.Id)
			require.NoError(t, err)
			requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound,
				apiclient.ProblemCodeNotFound)

			missingResponse, err := outsider.Client.TestDatabaseWithResponse(t.Context(), uuid.New())
			require.NoError(t, err)
			requireProblem(t, missingResponse.StatusCode(), missingResponse.Body, http.StatusNotFound,
				apiclient.ProblemCodeNotFound)
			assert.Zero(t, recorder.getAcceptedCount())
		})

	t.Run("returns 401 unauthenticated without a session", func(t *testing.T) {
		tm := newTeam(t)
		savedDatabase := tm.createDatabase(
			t,
			newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer)),
		)

		response, err := tm.app.NewClient(t).TestDatabaseWithResponse(t.Context(), savedDatabase.Id)

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated)
	})
}

//nolint:tparallel // The subtests share one physical PostgreSQL server.
func TestTestPhysicalDatabase(t *testing.T) {
	t.Parallel()

	physicalServer := containers.StartPhysicalPostgres(t, "postgres:17")

	t.Run("reports the missing replication privilege when the user cannot replicate", func(t *testing.T) {
		tm := newTeam(t)
		savedDatabase := tm.createDatabase(t,
			newPostgresqlPhysicalSettings(t, physicalServer, apiclient.PhysicalBackupTypeFULL))
		roleName := newUniqueName("no_replication_")
		_, err := connectAsAdmin(t, physicalServer, containers.PostgresDatabase).Exec(t.Context(),
			fmt.Sprintf("CREATE ROLE %s LOGIN PASSWORD 'plain-role'", pgx.Identifier{roleName}.Sanitize()))
		require.NoError(t, err)
		tm.app.ExecuteSQL(t,
			"UPDATE databases_postgresql_physical SET username = ?, password = ? WHERE database_id = ?",
			roleName, tm.app.EncryptSecret(t, "plain-role"), savedDatabase.Id)

		testResult := testSavedDatabase(t, tm.owner, savedDatabase.Id)

		assert.False(t, testResult.IsConnected)
		require.NotNil(t, testResult.Reason)
		assert.Equal(t, apiclient.DatabaseConnectionFailureNOREPLICATIONPRIVILEGE, *testResult.Reason)
	})
}

func testSavedDatabase(t *testing.T, session apptest.Session, databaseID uuid.UUID) apiclient.DatabaseTestResult {
	t.Helper()

	response, err := session.Client.TestDatabaseWithResponse(t.Context(), databaseID)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return *response.JSON200
}
