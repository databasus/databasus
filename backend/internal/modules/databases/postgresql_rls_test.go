package databases_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

type rlsConnectionFixture struct {
	admin    *pgx.Conn
	settings apiclient.PostgresqlLogicalSettings
}

func newRLSConnectionFixture(t *testing.T, server containers.Endpoint) rlsConnectionFixture {
	t.Helper()

	databaseName := newUniqueName("rls_")
	roleName := newUniqueName("rls_reader_")
	password := uuid.NewString()
	_, err := connectAsAdmin(t, server, containers.PostgresDatabase).Exec(t.Context(), `CREATE DATABASE `+databaseName)
	require.NoError(t, err)

	admin := connectAsAdmin(t, server, databaseName)
	_, err = admin.Exec(t.Context(), fmt.Sprintf(`
		CREATE ROLE %s LOGIN PASSWORD '%s';
		CREATE TABLE tenant_rows (id INT, tenant_id INT);
		INSERT INTO tenant_rows VALUES (1, 1), (2, 2);
		GRANT USAGE ON SCHEMA public TO %s;
		GRANT SELECT ON tenant_rows TO %s;
	`, roleName, password, roleName, roleName))
	require.NoError(t, err)

	settings := newLogicalSettings(server)
	settings.Database = &databaseName
	settings.Username = roleName
	settings.Password = &password

	return rlsConnectionFixture{admin: admin, settings: settings}
}

func (fixture rlsConnectionFixture) enableRLS(t *testing.T) {
	t.Helper()

	_, err := fixture.admin.Exec(t.Context(), `ALTER TABLE tenant_rows ENABLE ROW LEVEL SECURITY;
		CREATE POLICY tenant_access ON tenant_rows USING (tenant_id = 1)`)
	require.NoError(t, err)
}

func (fixture rlsConnectionFixture) requireRLSDiagnostic(t *testing.T, diagnostic *string) {
	t.Helper()

	require.NotNil(t, diagnostic)
	assert.Contains(t, *diagnostic, "RLS")
	assert.Contains(t, *diagnostic, "public.tenant_rows")
	assert.Contains(t, *diagnostic, fixture.settings.Username)
	assert.NotContains(t, *diagnostic, *fixture.settings.Password)
}

//nolint:tparallel // The scenarios share a bounded source server and use isolated databases.
func TestPostgresqlRLSConnectionValidation(t *testing.T) {
	t.Parallel()
	server := containers.StartPostgres(t, "postgres:16")

	t.Run("refuses creation when role is restricted by rls", func(t *testing.T) {
		tm := newTeam(t)
		fixture := newRLSConnectionFixture(t, server)
		fixture.enableRLS(t)

		response, err := tm.owner.Client.CreateDatabaseWithResponse(t.Context(), apiclient.DatabaseCreate{
			WorkspaceId: tm.workspace.Id,
			Name:        "Restricted",
			NotifierIds: []uuid.UUID{},
			Settings:    fromLogicalSettings(t, fixture.settings),
		})

		require.NoError(t, err)
		requireProblem(
			t,
			response.StatusCode(),
			response.Body,
			http.StatusBadRequest,
			apiclient.ProblemCodeDatabaseConnectionFailed,
		)
		assert.Empty(t, tm.listDatabaseIDs(t))
		assert.Contains(t, string(response.Body), "RLS")
		assert.Contains(t, string(response.Body), "public.tenant_rows")
		assert.Contains(t, string(response.Body), fixture.settings.Username)
		assert.NotContains(t, string(response.Body), *fixture.settings.Password)
	})

	t.Run("keeps stored configuration when an update is refused by rls", func(t *testing.T) {
		tm := newTeam(t)
		fixture := newRLSConnectionFixture(t, server)
		authorizedSettings := fixture.settings
		authorizedSettings.Username = containers.PostgresUsername
		authorizedSettings.Password = new(containers.PostgresPassword)
		savedDatabase := tm.createDatabase(t, fromLogicalSettings(t, authorizedSettings))
		databaseBeforeUpdate := tm.readDatabase(t, savedDatabase.Id)
		fixture.enableRLS(t)
		update := toUpdate(savedDatabase)
		update.Name = "Refused update"
		update.Settings = fromLogicalSettings(t, fixture.settings)

		response := updateDatabase(t, tm.owner, savedDatabase.Id, update)

		requireProblem(
			t,
			response.StatusCode(),
			response.Body,
			http.StatusBadRequest,
			apiclient.ProblemCodeDatabaseConnectionFailed,
		)
		assert.Contains(t, string(response.Body), "RLS")
		assert.Contains(t, string(response.Body), fixture.settings.Username)
		assert.Contains(t, string(response.Body), "public.tenant_rows")
		assert.NotContains(t, string(response.Body), *fixture.settings.Password)
		assert.Equal(t, databaseBeforeUpdate, tm.readDatabase(t, savedDatabase.Id))
		assert.True(t, testSavedDatabase(t, tm.owner, savedDatabase.Id).IsConnected)
	})

	t.Run("leaves every database field unchanged when saved test detects rls drift", func(t *testing.T) {
		tm := newTeam(t)
		fixture := newRLSConnectionFixture(t, server)
		savedDatabase := tm.createDatabase(t, fromLogicalSettings(t, fixture.settings))
		fixture.enableRLS(t)

		// The connection API cannot set health or backup outcomes; seed nonempty values to detect resets.
		tm.app.ExecuteSQL(t, `UPDATE databases SET health_status = 'UNAVAILABLE',
            last_backup_time = '2026-01-02T03:04:05Z', last_backup_error_message = 'previous backup failed'
            WHERE id = ?`, savedDatabase.Id)
		databaseBeforeTest := tm.readDatabase(t, savedDatabase.Id)

		connection := testSavedDatabase(t, tm.owner, savedDatabase.Id)

		assert.False(t, connection.IsConnected)
		fixture.requireRLSDiagnostic(t, connection.Error)
		assert.Equal(t, databaseBeforeTest, tm.readDatabase(t, savedDatabase.Id))
	})

	t.Run("reports failed connection when unsaved role is restricted by rls", func(t *testing.T) {
		tm := newTeam(t)
		fixture := newRLSConnectionFixture(t, server)
		fixture.enableRLS(t)

		response, err := tm.owner.Client.TestDatabaseSettingsWithResponse(
			t.Context(),
			newSettingsTarget(tm.workspace.Id, nil, fromLogicalSettings(t, fixture.settings)),
		)

		require.NoError(t, err)
		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.False(t, response.JSON200.IsConnected)
		fixture.requireRLSDiagnostic(t, response.JSON200.Error)
		assert.Empty(t, tm.listDatabaseIDs(t))
	})

	t.Run("accepts authorized role when included table has forced rls", func(t *testing.T) {
		tm := newTeam(t)
		fixture := newRLSConnectionFixture(t, server)
		fixture.enableRLS(t)
		_, err := fixture.admin.Exec(
			t.Context(),
			fmt.Sprintf(
				`ALTER ROLE %s BYPASSRLS; ALTER TABLE tenant_rows FORCE ROW LEVEL SECURITY`,
				fixture.settings.Username,
			),
		)
		require.NoError(t, err)

		savedDatabase := tm.createDatabase(t, fromLogicalSettings(t, fixture.settings))

		assert.True(t, testSavedDatabase(t, tm.owner, savedDatabase.Id).IsConnected)
	})

	t.Run("requires select when a data exclusion removes the rls requirement", func(t *testing.T) {
		tm := newTeam(t)
		fixture := newRLSConnectionFixture(t, server)
		fixture.enableRLS(t)
		fixture.settings.ExcludeTableData = []string{"public.tenant_rows"}
		savedDatabase := tm.createDatabase(t, fromLogicalSettings(t, fixture.settings))
		require.True(t, testSavedDatabase(t, tm.owner, savedDatabase.Id).IsConnected)
		_, err := fixture.admin.Exec(t.Context(), `REVOKE SELECT ON tenant_rows FROM `+fixture.settings.Username)
		require.NoError(t, err)

		connection := testSavedDatabase(t, tm.owner, savedDatabase.Id)

		assert.False(t, connection.IsConnected)
		require.NotNil(t, connection.Error)
		assert.Contains(t, *connection.Error, "Missing SELECT")
	})
}
