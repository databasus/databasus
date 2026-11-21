package databases_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

const neonEndpointHost = "ep-young-dew-123456.c-8.us-east-2.aws.neon.tech"

func TestPhysicalDatabaseOnNeon(t *testing.T) {
	t.Parallel()

	t.Run("reports PHYSICAL_REPLICATION_UNSUPPORTED when unsaved physical settings point at Neon",
		func(t *testing.T) {
			t.Parallel()
			tm := newTeam(t)

			testResult := testUnsavedSettings(t, tm.owner, newSettingsTarget(tm.workspace.Id, nil,
				fromPhysicalSettings(t, newNeonPhysicalSettings())))

			assert.False(t, testResult.IsConnected)
			require.NotNil(t, testResult.Reason)
			assert.Equal(t, apiclient.DatabaseConnectionFailurePHYSICALREPLICATIONUNSUPPORTED, *testResult.Reason)
		})

	// The bastion does not exist, so a dial attempt would have answered with a tunnel failure instead.
	t.Run("reports PHYSICAL_REPLICATION_UNSUPPORTED without dialing the bastion when Neon sits behind a tunnel",
		func(t *testing.T) {
			t.Parallel()
			tm := newTeam(t)
			physical := newNeonPhysicalSettings()
			physical.SshTunnel = newExampleSSHTunnel()

			testResult := testUnsavedSettings(t, tm.owner, newSettingsTarget(tm.workspace.Id, nil,
				fromPhysicalSettings(t, physical)))

			assert.False(t, testResult.IsConnected)
			require.NotNil(t, testResult.Reason)
			assert.Equal(t, apiclient.DatabaseConnectionFailurePHYSICALREPLICATIONUNSUPPORTED, *testResult.Reason)
			require.NotNil(t, testResult.Error)
			assert.NotContains(t, *testResult.Error, "bastion.example.com")
		})

	t.Run("returns 400 database_connection_failed and creates nothing when a physical database is created on Neon",
		func(t *testing.T) {
			t.Parallel()
			tm := newTeam(t)

			response, err := tm.owner.Client.CreateDatabaseWithResponse(t.Context(),
				newCreation(tm, fromPhysicalSettings(t, newNeonPhysicalSettings())))

			require.NoError(t, err)
			requireNeonRefusal(t, response.StatusCode(), response.Body)
			assert.Empty(t, tm.listDatabaseIDs(t))
		})

	t.Run("returns 400 database_connection_failed and keeps the stored settings when a physical database moves to Neon",
		func(t *testing.T) {
			t.Parallel()
			physicalServer := containers.StartPhysicalPostgres(t, "postgres:17")
			tm := newTeam(t)
			savedDatabase := tm.createDatabase(t,
				newPostgresqlPhysicalSettings(t, physicalServer, apiclient.PhysicalBackupTypeFULL))
			update := toUpdate(savedDatabase)
			update.Settings = fromPhysicalSettings(t, newNeonPhysicalSettings())

			response := updateDatabase(t, tm.owner, savedDatabase.Id, update)

			requireNeonRefusal(t, response.StatusCode(), response.Body)
			storedPhysical := getPhysicalSettings(t, tm.readDatabase(t, savedDatabase.Id))
			assert.Equal(t, physicalServer.Host, storedPhysical.Host)
			assert.Equal(t, physicalServer.Port, storedPhysical.Port)
		})

	t.Run("connects to the source as for any other host when the host only resembles Neon", func(t *testing.T) {
		t.Parallel()
		tm := newTeam(t)
		physical := newNeonPhysicalSettings()
		physical.Host = "neon.tech.example.com"

		testResult := testUnsavedSettings(t, tm.owner, newSettingsTarget(tm.workspace.Id, nil,
			fromPhysicalSettings(t, physical)))

		assert.False(t, testResult.IsConnected)
		if testResult.Reason != nil {
			assert.NotEqual(t, apiclient.DatabaseConnectionFailurePHYSICALREPLICATIONUNSUPPORTED, *testResult.Reason)
		}
		require.NotNil(t, testResult.Error)
		assert.Contains(t, *testResult.Error, "neon.tech.example.com",
			"the failure comes from trying to reach the host")
	})
}

func newNeonPhysicalSettings() apiclient.PostgresqlPhysicalSettings {
	physical := newPhysicalSettings(containers.Endpoint{Host: neonEndpointHost, Port: 5432},
		apiclient.PhysicalBackupTypeFULL)
	physical.Username = "neondb_owner"
	physical.SslMode = apiclient.PostgresqlSslModeRequire

	return physical
}

func requireNeonRefusal(t *testing.T, status int, body []byte) {
	t.Helper()

	requireProblem(t, status, body, http.StatusBadRequest, apiclient.ProblemCodeDatabaseConnectionFailed)

	var problem apiclient.Problem
	require.NoError(t, json.Unmarshal(body, &problem), string(body))
	assert.Contains(t, problem.Detail, "Neon supports only logical backups")
}
