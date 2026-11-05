package databases_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

// unansweredPort is a port where no test server listens, so a connection to it is refused at once.
const unansweredPort = 1

//nolint:tparallel // The subtests share one source server.
func TestCreateDatabase(t *testing.T) {
	t.Parallel()

	server := containers.StartPostgres(t, "postgres:16")

	t.Run("returns the saved PostgreSQL database with its detected version and without its secrets",
		func(t *testing.T) {
			tm := newTeam(t)
			logical := newLogicalSettings(server)
			logical.SshTunnel = apiclient.SshTunnelSettings{
				IsEnabled:            false,
				AuthType:             new(apiclient.SshAuthTypePRIVATEKEY),
				Host:                 new("bastion.example.com"),
				Port:                 new(22),
				Username:             new("tunneluser"),
				Password:             nil,
				PrivateKey:           new("tunnel-private-key-value"),
				PrivateKeyPassphrase: new("tunnel-passphrase-value"),
			}

			response, err := tm.owner.Client.CreateDatabaseWithResponse(t.Context(), apiclient.DatabaseCreate{
				WorkspaceId: tm.workspace.Id,
				Name:        "Orders",
				NotifierIds: []uuid.UUID{},
				Settings:    fromLogicalSettings(t, logical),
			})

			require.NoError(t, err)
			require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))
			assert.NotContains(t, string(response.Body), containers.PostgresPassword)
			assert.NotContains(t, string(response.Body), "tunnel-private-key-value")
			assert.NotContains(t, string(response.Body), "tunnel-passphrase-value")
			assert.NotContains(t, string(response.Body), "enc:")
			assert.NotEqual(t, uuid.Nil, response.JSON201.Id)
			assert.Equal(t, "Orders", response.JSON201.Name)
			assert.Equal(t, tm.workspace.Id, response.JSON201.WorkspaceId)
			assert.Empty(t, response.JSON201.NotifierIds)
			assert.Nil(t, response.JSON201.LastBackupTime)
			assert.Nil(t, response.JSON201.LastBackupErrorMessage)

			createdLogical := getLogicalSettings(t, *response.JSON201)
			assert.Equal(t, server.Host, createdLogical.Host)
			assert.Equal(t, server.Port, createdLogical.Port)
			assert.Equal(t, containers.PostgresUsername, createdLogical.Username)
			assert.Equal(t, new(apiclient.PostgresqlVersionN16), createdLogical.Version)
			assert.Equal(t, new("bastion.example.com"), createdLogical.SshTunnel.Host)
			assert.Equal(t, "Orders", getDatabase(t, tm.owner, response.JSON201.Id).Name)
		})

	t.Run("returns 400 invalid_request naming the host and creates nothing without a host", func(t *testing.T) {
		tm := newTeam(t)
		logical := newLogicalSettings(server)
		logical.Host = ""

		response, err := tm.owner.Client.CreateDatabaseWithResponse(t.Context(), apiclient.DatabaseCreate{
			WorkspaceId: tm.workspace.Id,
			Name:        "Without a host",
			NotifierIds: []uuid.UUID{},
			Settings:    fromLogicalSettings(t, logical),
		})

		require.NoError(t, err)
		requireInvalidField(t, response.StatusCode(), response.Body, "settings.host")
		assert.Empty(t, listDatabases(t, tm.owner, tm.workspace.Id))
	})

	t.Run("returns 400 invalid_request and creates nothing when it points at the internal database",
		func(t *testing.T) {
			tm := newTeam(t)
			recorder := startConnectionRecorder(t)
			logical := newLogicalSettings(server)
			logical.Host = recorder.host
			logical.Port = recorder.port
			logical.Database = new("databasus")

			response, err := tm.owner.Client.CreateDatabaseWithResponse(t.Context(), apiclient.DatabaseCreate{
				WorkspaceId: tm.workspace.Id,
				Name:        "Databasus itself",
				NotifierIds: []uuid.UUID{},
				Settings:    fromLogicalSettings(t, logical),
			})

			require.NoError(t, err)
			requireInvalidField(t, response.StatusCode(), response.Body, "settings.host")
			assert.Empty(t, listDatabases(t, tm.owner, tm.workspace.Id))
			assert.Zero(t, recorder.getAcceptedCount())
		})

	t.Run("returns 400 database_connection_failed and creates nothing when no server answers", func(t *testing.T) {
		tm := newTeam(t)
		logical := newLogicalSettings(server)
		logical.Port = unansweredPort

		response, err := tm.owner.Client.CreateDatabaseWithResponse(t.Context(), apiclient.DatabaseCreate{
			WorkspaceId: tm.workspace.Id,
			Name:        "Unreachable",
			NotifierIds: []uuid.UUID{},
			Settings:    fromLogicalSettings(t, logical),
		})

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeDatabaseConnectionFailed)
		assert.NotEmpty(t, response.ApplicationproblemJSON400.Detail)
		assert.Empty(t, listDatabases(t, tm.owner, tm.workspace.Id))
	})

	t.Run("creates a new database in the member's workspace when the body names another team's database",
		func(t *testing.T) {
			tm := newTeam(t)
			stranger := tm.signUpNonMember(t)
			strangerWorkspace := apptest.CreateWorkspace(t, stranger, "Stranger "+uuid.NewString()[:8])
			otherDatabaseID := tm.app.InsertDatabase(t, strangerWorkspace.Id)
			otherDatabase := getDatabase(t, stranger, otherDatabaseID)
			otherBackupID := insertCompletedLogicalBackupRow(t, tm, stranger, strangerWorkspace.Id, otherDatabaseID)
			body := newCreationBodyWithID(t, apiclient.DatabaseCreate{
				WorkspaceId: tm.workspace.Id,
				Name:        "Takeover",
				NotifierIds: []uuid.UUID{},
				Settings:    fromLogicalSettings(t, newLogicalSettings(server)),
			}, otherDatabaseID)

			response, err := tm.owner.Client.CreateDatabaseWithBodyWithResponse(
				t.Context(), "application/json", bytes.NewReader(body))

			require.NoError(t, err)
			require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))
			assert.NotEqual(t, otherDatabaseID, response.JSON201.Id)
			assert.Equal(t, tm.workspace.Id, response.JSON201.WorkspaceId)
			untouchedDatabase := getDatabase(t, stranger, otherDatabaseID)
			assert.Equal(t, strangerWorkspace.Id, untouchedDatabase.WorkspaceId)
			assert.Equal(t, otherDatabase.Name, untouchedDatabase.Name)
			assert.Equal(t, getLogicalSettings(t, otherDatabase), getLogicalSettings(t, untouchedDatabase))
			assert.Equal(t, []uuid.UUID{otherBackupID}, listLogicalBackupIDs(t, tm, otherDatabaseID))
		})
}

// insertCompletedLogicalBackupRow writes a backup record directly, because no contract operation creates
// a backup yet.
func insertCompletedLogicalBackupRow(
	t *testing.T,
	tm team,
	member apptest.Session,
	workspaceID, databaseID uuid.UUID,
) uuid.UUID {
	t.Helper()

	backupID := uuid.New()
	storageID := tm.app.CreateLocalStorage(t, member, workspaceID)
	tm.app.InsertBackupConfigUsingStorage(t, apptest.StorageUse{DatabaseID: databaseID, StorageID: storageID})
	tm.app.ExecuteSQL(t, `INSERT INTO backups_logical (id, database_id, storage_id, status, file_name)
		VALUES (?, ?, ?, 'COMPLETED', ?)`, backupID, databaseID, storageID, "backup-"+backupID.String())

	return backupID
}

func listLogicalBackupIDs(t *testing.T, tm team, databaseID uuid.UUID) []uuid.UUID {
	t.Helper()

	var backupIDs []uuid.UUID
	tm.app.QuerySQL(t, &backupIDs, "SELECT id FROM backups_logical WHERE database_id = ? ORDER BY id", databaseID)

	return backupIDs
}

//nolint:tparallel // The subtests share one source server.
func TestUpdateDatabase(t *testing.T) {
	t.Parallel()

	server := containers.StartPostgres(t, "postgres:16")

	t.Run("renames the database and keeps the stored password when the request leaves it out",
		func(t *testing.T) {
			tm := newTeam(t)
			createdDatabase := tm.createDatabaseNamed(t, "Orders", fromLogicalSettings(t, newLogicalSettings(server)))
			update := toUpdate(createdDatabase)
			update.Name = "Orders renamed"

			response := updateDatabase(t, tm.owner, createdDatabase.Id, update)

			require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
			assert.Equal(t, "Orders renamed", getDatabase(t, tm.owner, createdDatabase.Id).Name)
			assert.Equal(t, containers.PostgresPassword,
				tm.app.DecryptSecret(t, getStoredPassword(t, tm, "databases_postgresql_logical", createdDatabase.Id)))

			testResponse, err := tm.owner.Client.TestDatabaseWithResponse(t.Context(), createdDatabase.Id)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, testResponse.StatusCode(), string(testResponse.Body))
			assert.True(t, testResponse.JSON200.IsConnected, string(testResponse.Body))
		})

	t.Run("returns 400 database_connection_failed and keeps the stored port when no server answers",
		func(t *testing.T) {
			tm := newTeam(t)
			createdDatabase := tm.createDatabaseNamed(t, "Orders", fromLogicalSettings(t, newLogicalSettings(server)))
			movedLogical := getLogicalSettings(t, createdDatabase)
			movedLogical.Port = unansweredPort
			movedLogical.Password = new(containers.PostgresPassword)
			update := toUpdate(createdDatabase)
			update.Name = "Orders moved"
			update.Settings = fromLogicalSettings(t, movedLogical)

			response := updateDatabase(t, tm.owner, createdDatabase.Id, update)

			requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
				apiclient.ProblemCodeDatabaseConnectionFailed)
			assert.NotEmpty(t, response.ApplicationproblemJSON400.Detail)
			unchangedDatabase := getDatabase(t, tm.owner, createdDatabase.Id)
			assert.Equal(t, "Orders", unchangedDatabase.Name)
			assert.Equal(t, server.Port, getLogicalSettings(t, unchangedDatabase).Port)
		})

	t.Run("returns 400 invalid_request naming the host and keeps the database without a host", func(t *testing.T) {
		tm := newTeam(t)
		createdDatabase := tm.createDatabaseNamed(t, "Orders", fromLogicalSettings(t, newLogicalSettings(server)))
		logical := getLogicalSettings(t, createdDatabase)
		logical.Host = ""
		update := toUpdate(createdDatabase)
		update.Settings = fromLogicalSettings(t, logical)

		response := updateDatabase(t, tm.owner, createdDatabase.Id, update)

		requireInvalidField(t, response.StatusCode(), response.Body, "settings.host")
		assert.Equal(t, server.Host, getLogicalSettings(t, getDatabase(t, tm.owner, createdDatabase.Id)).Host)
	})

	t.Run("returns 400 invalid_request and keeps the MySQL database when it is sent with MongoDB settings",
		func(t *testing.T) {
			mysqlServer := containers.StartMysql(t, "mysql:8.0")
			tm := newTeam(t)
			createdDatabase := tm.createDatabaseNamed(t, "Shop", fromMysqlSettings(t, newMysqlSettings(mysqlServer)))
			update := toUpdate(createdDatabase)
			update.Settings = fromMongodbSettings(t, newMongodbSettings(containers.Endpoint{
				Host: "mongo.example.com",
				Port: 27017,
			}))

			response := updateDatabase(t, tm.owner, createdDatabase.Id, update)

			requireInvalidField(t, response.StatusCode(), response.Body, "settings.type")
			unchangedDatabase := getDatabase(t, tm.owner, createdDatabase.Id)
			discriminator, err := unchangedDatabase.Settings.Discriminator()
			require.NoError(t, err)
			assert.Equal(t, string(apiclient.MysqlSettingsTypeMYSQL), discriminator)
			assert.Equal(t, mysqlServer.Port, getMysqlSettings(t, unchangedDatabase).Port)
		})
}

func newCreationBodyWithID(t *testing.T, creation apiclient.DatabaseCreate, databaseID uuid.UUID) []byte {
	t.Helper()

	encodedCreation, err := json.Marshal(creation)
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, json.Unmarshal(encodedCreation, &body))
	body["id"] = databaseID

	encodedBody, err := json.Marshal(body)
	require.NoError(t, err)

	return encodedBody
}

// getStoredPassword reads the encrypted password column, which the API never returns.
func getStoredPassword(t *testing.T, tm team, settingsTable string, databaseID uuid.UUID) string {
	t.Helper()

	var storedPasswords []string
	tm.app.QuerySQL(t, &storedPasswords,
		"SELECT password FROM "+settingsTable+" WHERE database_id = ?", databaseID)
	require.Len(t, storedPasswords, 1)

	return storedPasswords[0]
}
