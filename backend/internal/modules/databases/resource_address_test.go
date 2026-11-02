package databases_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

// addressedEngineCase builds the settings of one engine at an address, with or without the password.
type addressedEngineCase struct {
	name           string
	storedPassword string
	startServer    func(t *testing.T) containers.Endpoint
	newSettings    func(
		t *testing.T, server containers.Endpoint, host string, port int, password *string,
	) apiclient.DatabaseSettings
}

func listAddressedEngineCases() []addressedEngineCase {
	return []addressedEngineCase{
		{
			name:           "PostgreSQL logical",
			storedPassword: containers.PostgresPassword,
			startServer: func(t *testing.T) containers.Endpoint {
				t.Helper()

				return containers.StartPostgres(t, "postgres:16")
			},
			newSettings: func(
				t *testing.T, server containers.Endpoint, host string, port int, password *string,
			) apiclient.DatabaseSettings {
				t.Helper()

				logical := newLogicalSettings(server)
				logical.Host, logical.Port, logical.Password = host, port, password

				return fromLogicalSettings(t, logical)
			},
		},
		{
			name:           "PostgreSQL physical",
			storedPassword: containers.PostgresPassword,
			startServer: func(t *testing.T) containers.Endpoint {
				t.Helper()

				return containers.StartPhysicalPostgres(t, "postgres:17")
			},
			newSettings: func(
				t *testing.T, server containers.Endpoint, host string, port int, password *string,
			) apiclient.DatabaseSettings {
				t.Helper()

				physical := newPhysicalSettings(server, apiclient.PhysicalBackupTypeFULL)
				physical.Host, physical.Port, physical.Password = host, port, password

				return fromPhysicalSettings(t, physical)
			},
		},
		{
			name:           "MySQL",
			storedPassword: containers.MysqlPassword,
			startServer: func(t *testing.T) containers.Endpoint {
				t.Helper()

				return containers.StartMysql(t, "mysql:8.0")
			},
			newSettings: func(
				t *testing.T, server containers.Endpoint, host string, port int, password *string,
			) apiclient.DatabaseSettings {
				t.Helper()

				mysql := newMysqlSettings(server)
				mysql.Host, mysql.Port, mysql.Password = host, port, password

				return fromMysqlSettings(t, mysql)
			},
		},
		{
			name:           "MariaDB",
			storedPassword: containers.MariadbPassword,
			startServer: func(t *testing.T) containers.Endpoint {
				t.Helper()

				return containers.StartMariadb(t, "mariadb:10.11")
			},
			newSettings: func(
				t *testing.T, server containers.Endpoint, host string, port int, password *string,
			) apiclient.DatabaseSettings {
				t.Helper()

				mariadb := newMariadbSettings(server)
				mariadb.Host, mariadb.Port, mariadb.Password = host, port, password

				return fromMariadbSettings(t, mariadb)
			},
		},
		{
			name:           "MongoDB",
			storedPassword: containers.MongodbPassword,
			startServer: func(t *testing.T) containers.Endpoint {
				t.Helper()

				return containers.StartMongodb(t, "mongo:7.0")
			},
			newSettings: func(
				t *testing.T, server containers.Endpoint, host string, port int, password *string,
			) apiclient.DatabaseSettings {
				t.Helper()

				mongodb := newMongodbSettings(server)
				mongodb.Host, mongodb.Port, mongodb.Password = host, new(port), password

				return fromMongodbSettings(t, mongodb)
			},
		},
	}
}

func marshalDatabaseSettings(t *testing.T, settings apiclient.DatabaseSettings) string {
	t.Helper()

	marshaledSettings, err := json.Marshal(settings)
	require.NoError(t, err)

	return string(marshaledSettings)
}

func requireAddressChangeRefused(
	t *testing.T,
	tm team,
	createdDatabase apiclient.Database,
	movedSettings apiclient.DatabaseSettings,
	recorder *connectionRecorder,
) {
	t.Helper()

	member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
	settingsBefore := marshalDatabaseSettings(t, getDatabase(t, member, createdDatabase.Id).Settings)
	update := toUpdate(createdDatabase)
	update.Name = "Moved"
	update.Settings = movedSettings

	updateResponse := updateDatabase(t, member, createdDatabase.Id, update)

	requireInvalidField(t, updateResponse.StatusCode(), updateResponse.Body, "settings.password")
	storedDatabase := getDatabase(t, member, createdDatabase.Id)
	assert.Equal(t, createdDatabase.Name, storedDatabase.Name)
	assert.Equal(t, settingsBefore, marshalDatabaseSettings(t, storedDatabase.Settings))

	testResponse, err := member.Client.TestDatabaseSettingsWithResponse(t.Context(),
		newSettingsTarget(tm.workspace.Id, &createdDatabase.Id, movedSettings))

	require.NoError(t, err)
	requireInvalidField(t, testResponse.StatusCode(), testResponse.Body, "settings.password")
	assert.Zero(t, recorder.getAcceptedCount())
}

func TestChangeResourceAddress(t *testing.T) {
	t.Parallel()

	for _, engineCase := range listAddressedEngineCases() {
		t.Run("refuses a moved host and port of "+engineCase.name+" without the password", func(t *testing.T) {
			t.Parallel()

			server := engineCase.startServer(t)
			storedSettings := engineCase.newSettings(
				t,
				server,
				server.Host,
				server.Port,
				new(engineCase.storedPassword),
			)
			tm := newTeam(t)
			recorder := startConnectionRecorder(t)
			createdDatabase := tm.createDatabaseNamed(t, "Stored", storedSettings)

			requireAddressChangeRefused(t, tm, createdDatabase,
				engineCase.newSettings(t, server, "moved.example.com", server.Port, nil), recorder)
			requireAddressChangeRefused(t, tm, createdDatabase,
				engineCase.newSettings(t, server, server.Host, recorder.port, nil), recorder)
		})
	}

	t.Run("refuses a host differing only in letter case without the password", func(t *testing.T) {
		t.Parallel()

		server := containers.StartPostgres(t, "postgres:16")
		tm := newTeam(t)
		recorder := startConnectionRecorder(t)
		storedLogical := newLogicalSettings(server)
		storedLogical.Host = "localhost"
		createdDatabase := tm.createDatabaseNamed(t, "Stored", fromLogicalSettings(t, storedLogical))
		movedLogical := newLogicalSettings(server)
		movedLogical.Host = strings.ToUpper(storedLogical.Host)
		movedLogical.Password = nil

		requireAddressChangeRefused(t, tm, createdDatabase, fromLogicalSettings(t, movedLogical), recorder)
	})

	t.Run("refuses a changed mongodb srv flag without the password", func(t *testing.T) {
		t.Parallel()

		server := containers.StartMongodb(t, "mongo:7.0")
		tm := newTeam(t)
		recorder := startConnectionRecorder(t)
		createdDatabase := tm.createDatabaseNamed(t, "Stored", fromMongodbSettings(t, newMongodbSettings(server)))
		movedMongodb := newMongodbSettings(server)
		movedMongodb.Password = nil
		movedMongodb.IsSrv = true

		requireAddressChangeRefused(t, tm, createdDatabase, fromMongodbSettings(t, movedMongodb), recorder)
	})

	t.Run("connects to the new server with the password sent with the new address", func(t *testing.T) {
		t.Parallel()

		firstServer := containers.StartPostgres(t, "postgres:16")
		secondServer := containers.StartPostgres(t, "postgres:16")
		tm := newTeam(t)
		member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
		createdDatabase := tm.createDatabaseNamed(t, "Stored", fromLogicalSettings(t, newLogicalSettings(firstServer)))
		update := toUpdate(createdDatabase)
		update.Settings = fromLogicalSettings(t, newLogicalSettings(secondServer))

		response := updateDatabase(t, member, createdDatabase.Id, update)

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.Equal(t, secondServer.Port, getLogicalSettings(t, getDatabase(t, member, createdDatabase.Id)).Port)
	})

	t.Run("connects with the new user name and the stored password at the same address", func(t *testing.T) {
		t.Parallel()

		server := containers.StartPostgres(t, "postgres:16")
		tm := newTeam(t)
		member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
		storedRole := newUniqueName("role_")
		newRole := newUniqueName("role_")
		adminConnection := connectAsAdmin(t, server, containers.PostgresDatabase)
		for _, roleName := range []string{storedRole, newRole} {
			_, err := adminConnection.Exec(t.Context(),
				"CREATE ROLE "+roleName+" LOGIN PASSWORD '"+containers.PostgresPassword+"'")
			require.NoError(t, err)
		}
		storedLogical := newLogicalSettings(server)
		storedLogical.Username = storedRole
		createdDatabase := tm.createDatabaseNamed(t, "Stored", fromLogicalSettings(t, storedLogical))
		// Only the new role can log in from here on, so a backend that ignored the change would fail.
		_, err := adminConnection.Exec(t.Context(), "ALTER ROLE "+storedRole+" NOLOGIN")
		require.NoError(t, err)
		changedLogical := newLogicalSettings(server)
		changedLogical.Username = newRole
		changedLogical.Password = nil

		testResult := testUnsavedSettings(t, member,
			newSettingsTarget(tm.workspace.Id, &createdDatabase.Id, fromLogicalSettings(t, changedLogical)))

		assert.True(t, testResult.IsConnected, "%v", testResult.Error)
	})

	t.Run("refuses a moved host while the client certificate is sent without its key", func(t *testing.T) {
		t.Parallel()

		server := containers.StartPostgres(t, "postgres:16")
		tm := newTeam(t)
		createdDatabase := tm.createDatabaseNamed(t, "Stored", fromLogicalSettings(t, newLogicalSettings(server)))
		// TLS with a client certificate needs a server set up for it, and this request is refused
		// before anything connects.
		tm.app.ExecuteSQL(t, "UPDATE databases_postgresql_logical SET ssl_mode = 'require', "+
			"ssl_client_cert = 'stored-cert', ssl_client_key = ? WHERE database_id = ?",
			tm.app.EncryptSecret(t, "stored-key"), createdDatabase.Id)
		recorder := startConnectionRecorder(t)
		movedLogical := newLogicalSettings(server)
		movedLogical.Host = "moved.example.com"
		movedLogical.SslMode = apiclient.PostgresqlSslModeRequire
		movedLogical.SslClientCert = new("stored-cert")
		update := toUpdate(createdDatabase)
		update.Settings = fromLogicalSettings(t, movedLogical)

		response := updateDatabase(t, tm.owner, createdDatabase.Id, update)

		requireInvalidField(t, response.StatusCode(), response.Body, "settings.sslMode")
		assert.Zero(t, recorder.getAcceptedCount())
	})

	t.Run("removes the stored client key when the host moves with the certificate cleared", func(t *testing.T) {
		t.Parallel()

		firstServer := containers.StartPostgres(t, "postgres:16")
		secondServer := containers.StartPostgres(t, "postgres:16")
		tm := newTeam(t)
		createdDatabase := tm.createDatabaseNamed(t, "Stored", fromLogicalSettings(t, newLogicalSettings(firstServer)))
		tm.app.ExecuteSQL(t, "UPDATE databases_postgresql_logical SET ssl_client_key = ? WHERE database_id = ?",
			tm.app.EncryptSecret(t, "stored-key"), createdDatabase.Id)
		update := toUpdate(createdDatabase)
		update.Settings = fromLogicalSettings(t, newLogicalSettings(secondServer))

		response := updateDatabase(t, tm.owner, createdDatabase.Id, update)

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		var storedClientKeys []string
		tm.app.QuerySQL(t, &storedClientKeys,
			"SELECT COALESCE(ssl_client_key, '') FROM databases_postgresql_logical WHERE database_id = ?",
			createdDatabase.Id)
		assert.Equal(t, []string{""}, storedClientKeys)
	})
}
