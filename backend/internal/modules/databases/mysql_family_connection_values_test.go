package databases_test

import (
	"fmt"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

//nolint:tparallel // The subtests share one MySQL and one MariaDB server.
func TestMysqlFamilyConnectionValues(t *testing.T) {
	t.Parallel()

	mysqlServer := containers.StartMysql(t, "mysql:8.0")
	mariadbServer := containers.StartMariadb(t, "mariadb:11.8")

	t.Run("returns 400 naming the database name and creates nothing when it starts with a dash", func(t *testing.T) {
		tm := newTeam(t)
		recorder := startConnectionRecorder(t)
		resultPath := filepath.Join(t.TempDir(), "out.sql")
		mysql := newMysqlSettings(mysqlServer)
		mysql.Host = recorder.host
		mysql.Port = recorder.port
		mysql.Database = new("--result-file=" + resultPath)

		response, err := tm.owner.Client.CreateDatabaseWithResponse(t.Context(), apiclient.DatabaseCreate{
			WorkspaceId: tm.workspace.Id,
			Name:        "Dash-led name",
			NotifierIds: []uuid.UUID{},
			Settings:    fromMysqlSettings(t, mysql),
		})

		require.NoError(t, err)
		requireInvalidField(t, response.StatusCode(), response.Body, "settings.database")
		assert.Empty(t, listDatabases(t, tm.owner, tm.workspace.Id))
		assert.Zero(t, recorder.getAcceptedCount())
		assert.NoFileExists(t, resultPath)
	})

	t.Run("returns 400 naming the user name and creates nothing when it holds a line break", func(t *testing.T) {
		tm := newTeam(t)
		recorder := startConnectionRecorder(t)
		mariadb := newMariadbSettings(mariadbServer)
		mariadb.Host = recorder.host
		mariadb.Port = recorder.port
		mariadb.Username = "x\nresult-file=" + filepath.Join(t.TempDir(), "out.sql")

		response, err := tm.owner.Client.CreateDatabaseWithResponse(t.Context(), apiclient.DatabaseCreate{
			WorkspaceId: tm.workspace.Id,
			Name:        "Line break in the user name",
			NotifierIds: []uuid.UUID{},
			Settings:    fromMariadbSettings(t, mariadb),
		})

		require.NoError(t, err)
		requireInvalidField(t, response.StatusCode(), response.Body, "settings.username")
		assert.Empty(t, listDatabases(t, tm.owner, tm.workspace.Id))
		assert.Zero(t, recorder.getAcceptedCount())
	})

	t.Run("returns 400 naming the database name and keeps the stored settings when a change makes it dash-led",
		func(t *testing.T) {
			tm := newTeam(t)
			createdDatabase := tm.createDatabaseNamed(t, "Shop", fromMysqlSettings(t, newMysqlSettings(mysqlServer)))
			recorder := startConnectionRecorder(t)
			teePath := filepath.Join(t.TempDir(), "out.log")
			changedMysql := newMysqlSettings(mysqlServer)
			changedMysql.Host = recorder.host
			changedMysql.Port = recorder.port
			changedMysql.Database = new("--tee=" + teePath)
			update := toUpdate(createdDatabase)
			update.Settings = fromMysqlSettings(t, changedMysql)

			response := updateDatabase(t, tm.owner, createdDatabase.Id, update)

			requireInvalidField(t, response.StatusCode(), response.Body, "settings.database")
			storedMysql := getMysqlSettings(t, getDatabase(t, tm.owner, createdDatabase.Id))
			assert.Equal(t, new(containers.MysqlDatabase), storedMysql.Database)
			assert.Equal(t, mysqlServer.Port, storedMysql.Port)
			assert.Zero(t, recorder.getAcceptedCount())
			assert.NoFileExists(t, teePath)
		})

	t.Run("returns 400 naming the database name without connecting when unsaved settings make it dash-led",
		func(t *testing.T) {
			tm := newTeam(t)
			recorder := startConnectionRecorder(t)
			mysql := newMysqlSettings(mysqlServer)
			mysql.Host = recorder.host
			mysql.Port = recorder.port
			mysql.Database = new("-orders")

			response := testUnsavedSettingsResponse(t, tm.owner,
				newSettingsTarget(tm.workspace.Id, nil, fromMysqlSettings(t, mysql)))

			requireInvalidField(t, response.StatusCode(), response.Body, "settings.database")
			assert.Zero(t, recorder.getAcceptedCount())
		})

	t.Run("returns 400 naming the host without connecting when a limited user is created for a host with a line break",
		func(t *testing.T) {
			tm := newTeam(t)
			recorder := startConnectionRecorder(t)
			mariadb := newMariadbSettings(mariadbServer)
			mariadb.Host = recorder.host + "\nresult-file=" + filepath.Join(t.TempDir(), "out.sql")
			mariadb.Port = recorder.port

			response, err := tm.owner.Client.CreateLimitedUserWithResponse(t.Context(),
				newSettingsTarget(tm.workspace.Id, nil, fromMariadbSettings(t, mariadb)))

			require.NoError(t, err)
			requireInvalidField(t, response.StatusCode(), response.Body, "settings.host")
			assert.Zero(t, recorder.getAcceptedCount())
		})

	t.Run("reports the whole database name as unknown when it holds driver parameters", func(t *testing.T) {
		tm := newTeam(t)
		mysql := newMysqlSettings(mysqlServer)
		mysql.Username = "root"
		mysql.Password = new(containers.MysqlRootPassword)
		mysql.Database = new("app?allowAllFiles=true&x")

		response := testUnsavedSettingsResponse(t, tm.owner,
			newSettingsTarget(tm.workspace.Id, nil, fromMysqlSettings(t, mysql)))

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.False(t, response.JSON200.IsConnected)
		require.NotNil(t, response.JSON200.Error)
		assert.Contains(t, *response.JSON200.Error, "Unknown database 'app?allowAllFiles=true&x'")
	})

	t.Run("reports a successful connection with URL separators in the password", func(t *testing.T) {
		tm := newTeam(t)
		username := "user_" + uuid.NewString()[:8]
		password := "p@ss/w?rd:1"
		rootDB := openMysql(t, mariadbServer, "root", containers.MariadbRootPassword)
		_, err := rootDB.ExecContext(t.Context(), fmt.Sprintf(
			"CREATE USER '%s'@'%%' IDENTIFIED BY '%s'", username, password,
		))
		require.NoError(t, err)
		_, err = rootDB.ExecContext(t.Context(), fmt.Sprintf(
			"GRANT SELECT, SHOW VIEW ON `%s`.* TO '%s'@'%%'", containers.MariadbDatabase, username,
		))
		require.NoError(t, err)
		mariadb := newMariadbSettings(mariadbServer)
		mariadb.Username = username
		mariadb.Password = new(password)

		response := testUnsavedSettingsResponse(t, tm.owner,
			newSettingsTarget(tm.workspace.Id, nil, fromMariadbSettings(t, mariadb)))

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.True(t, response.JSON200.IsConnected, "%v", response.JSON200.Error)
	})
}
