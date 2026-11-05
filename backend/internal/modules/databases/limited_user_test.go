package databases_test

import (
	"database/sql"
	"fmt"
	"net/http"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

const mysqlRootUsername = "root"

//nolint:tparallel // The subtests share one source server.
func TestMysqlLimitedUser(t *testing.T) {
	t.Parallel()

	mysqlServer := containers.StartMysql(t, "mysql:8.0")
	rootSettings := newMysqlServerSettings(t, mysqlServer, mysqlRootUsername, containers.MysqlRootPassword)

	t.Run("reports the write privileges when a member checks a user that may write", func(t *testing.T) {
		tm := newTeam(t)
		member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)

		limitedUserCheck := checkLimitedUser(t, member, newSettingsTarget(tm.workspace.Id, nil, rootSettings))

		assert.True(t, limitedUserCheck.IsAdvisable)
		assert.NotEmpty(t, limitedUserCheck.Privileges)
	})

	t.Run("creates a user that reads the database and cannot write to it when a member asks", func(t *testing.T) {
		tm := newTeam(t)
		member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
		tableName := newUniqueName("items_")
		rootConn := openMysql(t, mysqlServer, mysqlRootUsername, containers.MysqlRootPassword)
		_, err := rootConn.ExecContext(t.Context(), fmt.Sprintf("CREATE TABLE %s (id INT)", tableName))
		require.NoError(t, err)
		_, err = rootConn.ExecContext(t.Context(), fmt.Sprintf("INSERT INTO %s VALUES (1)", tableName))
		require.NoError(t, err)

		createdUser := createLimitedUser(t, member, newSettingsTarget(tm.workspace.Id, nil, rootSettings))

		require.NotEmpty(t, createdUser.Username)
		require.NotEmpty(t, createdUser.Password)

		limitedConn := openMysql(t, mysqlServer, createdUser.Username, createdUser.Password)

		var rowCount int
		require.NoError(t, limitedConn.QueryRowContext(t.Context(),
			fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&rowCount))
		assert.Equal(t, 1, rowCount)

		_, err = limitedConn.ExecContext(t.Context(), fmt.Sprintf("INSERT INTO %s VALUES (2)", tableName))
		assert.Error(t, err, "the limited user must not write")
	})
}

//nolint:tparallel // The subtests share one source server.
func TestPostgresqlLimitedUser(t *testing.T) {
	t.Parallel()

	postgresServer := containers.StartPostgres(t, "postgres:16")

	t.Run("lets a member check and then create a limited user of a saved database", func(t *testing.T) {
		tm := newTeam(t)
		member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
		savedDatabase := tm.createDatabase(
			t,
			newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer)),
		)
		targetWithoutPassword := newPostgresqlContainerTarget(postgresServer)
		targetWithoutPassword.password = nil
		settingsTarget := newSettingsTarget(tm.workspace.Id, &savedDatabase.Id,
			newPostgresqlLogicalSettings(t, targetWithoutPassword))

		limitedUserCheck := checkLimitedUser(t, member, settingsTarget)
		createdUser := createLimitedUser(t, member, settingsTarget)

		assert.True(t, limitedUserCheck.IsAdvisable)
		require.NotEmpty(t, createdUser.Username)
		connectToPostgres(t, postgresServer, containers.PostgresDatabase, createdUser.Username, createdUser.Password)
	})

	t.Run("returns 400 invalid_request without connecting when a member checks a saved database at another host",
		func(t *testing.T) {
			tm := newTeam(t)
			member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
			recorder := startConnectionRecorder(t)
			savedDatabase := tm.createDatabase(t,
				newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer)))

			response, err := member.Client.CheckLimitedUserWithResponse(t.Context(), newSettingsTarget(
				tm.workspace.Id, &savedDatabase.Id, newPostgresqlLogicalSettings(t, recorder.getPostgresqlTarget())))

			require.NoError(t, err)
			requireInvalidField(t, response.StatusCode(), response.Body, "settings.password")
			assert.Zero(t, recorder.getAcceptedCount())
		})

	t.Run("returns 400 invalid_request without connecting when a member creates a limited user at another host",
		func(t *testing.T) {
			tm := newTeam(t)
			member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
			recorder := startConnectionRecorder(t)
			savedDatabase := tm.createDatabase(t,
				newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer)))

			response, err := member.Client.CreateLimitedUserWithResponse(t.Context(), newSettingsTarget(
				tm.workspace.Id, &savedDatabase.Id, newPostgresqlLogicalSettings(t, recorder.getPostgresqlTarget())))

			require.NoError(t, err)
			requireInvalidField(t, response.StatusCode(), response.Body, "settings.password")
			assert.Zero(t, recorder.getAcceptedCount())
		})

	t.Run("returns 403 permission_denied without connecting when a viewer checks a saved database at another host",
		func(t *testing.T) {
			tm := newTeam(t)
			viewer := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
			recorder := startConnectionRecorder(t)
			savedDatabase := tm.createDatabase(t,
				newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer)))

			response, err := viewer.Client.CheckLimitedUserWithResponse(t.Context(), newSettingsTarget(
				tm.workspace.Id, &savedDatabase.Id, newPostgresqlLogicalSettings(t, recorder.getPostgresqlTarget())))

			require.NoError(t, err)
			requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
				apiclient.ProblemCodePermissionDenied)
			assert.Zero(t, recorder.getAcceptedCount())
		})

	t.Run("returns 403 permission_denied without connecting when a viewer creates a limited user", func(t *testing.T) {
		tm := newTeam(t)
		viewer := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
		recorder := startConnectionRecorder(t)
		savedDatabase := tm.createDatabase(
			t,
			newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer)),
		)

		response, err := viewer.Client.CreateLimitedUserWithResponse(t.Context(), newSettingsTarget(
			tm.workspace.Id, &savedDatabase.Id, newPostgresqlLogicalSettings(t, recorder.getPostgresqlTarget())))

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		assert.Zero(t, recorder.getAcceptedCount())
	})

	t.Run("returns 404 not_found when an outsider checks a database of another team", func(t *testing.T) {
		tm := newTeam(t)
		outsider := tm.signUpNonMember(t)
		outsiderWorkspace := apptest.CreateWorkspace(t, outsider, "Outsider")
		savedDatabase := tm.createDatabase(
			t,
			newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer)),
		)

		response, err := outsider.Client.CheckLimitedUserWithResponse(t.Context(), newSettingsTarget(
			outsiderWorkspace.Id, &savedDatabase.Id,
			newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer))))

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
	})

	t.Run("returns 401 unauthenticated without a session", func(t *testing.T) {
		tm := newTeam(t)

		response, err := tm.app.NewClient(t).CreateLimitedUserWithResponse(t.Context(), newSettingsTarget(
			tm.workspace.Id, nil, newPostgresqlLogicalSettings(t, newPostgresqlContainerTarget(postgresServer))))

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated)
	})
}

//nolint:tparallel // The subtests share one physical PostgreSQL server.
func TestPhysicalLimitedUser(t *testing.T) {
	t.Parallel()

	physicalServer := containers.StartPhysicalPostgres(t, "postgres:17")

	t.Run("creates a replication-only user that may force a WAL switch when the source allows it", func(t *testing.T) {
		tm := newTeam(t)

		createdUser := createLimitedUser(t, tm.owner, newSettingsTarget(tm.workspace.Id, nil,
			newPostgresqlPhysicalSettings(t, physicalServer, apiclient.PhysicalBackupTypeFULL)))

		require.NotEmpty(t, createdUser.Username)
		assert.True(t, createdUser.IsForcedWalRotationAvailable)

		var isReplicationRole, isSuperuser bool
		require.NoError(t, connectAsAdmin(t, physicalServer, containers.PostgresDatabase).QueryRow(t.Context(),
			"SELECT rolreplication, rolsuper FROM pg_roles WHERE rolname = $1", createdUser.Username,
		).Scan(&isReplicationRole, &isSuperuser))
		assert.True(t, isReplicationRole)
		assert.False(t, isSuperuser)
	})
}

func checkLimitedUser(
	t *testing.T,
	session apptest.Session,
	target apiclient.DatabaseSettingsTarget,
) apiclient.LimitedUserCheck {
	t.Helper()

	response, err := session.Client.CheckLimitedUserWithResponse(t.Context(), target)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return *response.JSON200
}

func createLimitedUser(
	t *testing.T,
	session apptest.Session,
	target apiclient.DatabaseSettingsTarget,
) apiclient.LimitedUser {
	t.Helper()

	response, err := session.Client.CreateLimitedUserWithResponse(t.Context(), target)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))

	return *response.JSON201
}

func openMysql(t *testing.T, server containers.Endpoint, username, password string) *sql.DB {
	t.Helper()

	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		username, password, server.Host, server.Port, containers.MysqlDatabase))
	require.NoError(t, err)

	t.Cleanup(func() { _ = db.Close() })

	require.NoError(t, db.PingContext(t.Context()))

	return db
}
