package databases_test

import (
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/util/testing/containers"
)

// Without this the tunnel tests would stay green if the tunnel silently stopped being used, because
// every other assertion only proves that some route to the database exists.
func TestBastionedPostgres(t *testing.T) {
	t.Run("refuses a connection from the host without the tunnel", func(t *testing.T) {
		topology := containers.StartPostgresBehindSshBastion(t, "postgres:16")
		address := net.JoinHostPort(topology.Database.Host, strconv.Itoa(topology.Database.Port))

		dialer := net.Dialer{Timeout: 5 * time.Second}
		conn, err := dialer.DialContext(t.Context(), "tcp", address)

		if err == nil {
			_ = conn.Close()
			t.Fatalf("the bastioned database answered on %s without a tunnel", address)
		}
	})
}

func TestDatabaseSSHTunnel(t *testing.T) {
	topology := containers.StartPostgresBehindSshBastion(t, "postgres:16")
	server := containers.StartPostgres(t, "postgres:16")

	t.Run("creates a database over a password tunnel with its detected version and without the tunnel secrets",
		func(t *testing.T) {
			tm := newTeam(t)

			createdDatabase := tm.createDatabaseNamed(t, "Bastioned",
				fromLogicalSettings(t, newBastionedLogicalSettings(topology)))

			createdLogical := getLogicalSettings(t, createdDatabase)
			assert.Equal(t, new(apiclient.PostgresqlVersionN16), createdLogical.Version)
			// The stored settings address the database as the bastion sees it, not as the forwarder
			// rewrote it for the duration of the connection.
			assert.Equal(t, topology.Database.Host, createdLogical.Host)
			assert.Equal(t, topology.Database.Port, createdLogical.Port)
			assert.True(t, createdLogical.SshTunnel.IsEnabled)
			readResponse, err := tm.owner.Client.GetDatabaseWithResponse(t.Context(), createdDatabase.Id)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, readResponse.StatusCode(), string(readResponse.Body))
			assert.NotContains(t, string(readResponse.Body), containers.SshBastionPassword)
			readTunnel := getLogicalSettings(t, *readResponse.JSON200).SshTunnel
			assert.Nil(t, readTunnel.Password)
			assert.Nil(t, readTunnel.PrivateKey)
			assert.Nil(t, readTunnel.PrivateKeyPassphrase)
			assert.Equal(t, new(containers.SshBastionUsername), readTunnel.Username)
			testResponse, err := tm.owner.Client.TestDatabaseWithResponse(t.Context(), createdDatabase.Id)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, testResponse.StatusCode(), string(testResponse.Body))
			assert.True(t, testResponse.JSON200.IsConnected, string(testResponse.Body))
		})

	t.Run("creates a database over a private key tunnel", func(t *testing.T) {
		tm := newTeam(t)
		logical := newBastionedLogicalSettings(topology)
		logical.SshTunnel.AuthType = new(apiclient.SshAuthTypePRIVATEKEY)
		logical.SshTunnel.Password = nil
		logical.SshTunnel.PrivateKey = new(readBastionTestKey(t))

		createdDatabase := tm.createDatabaseNamed(t, "Bastioned by key", fromLogicalSettings(t, logical))

		assert.Equal(t, new(apiclient.PostgresqlVersionN16), getLogicalSettings(t, createdDatabase).Version)
	})

	// The stored key would otherwise stay a working way into the bastion after the user replaced it
	// with a password.
	t.Run("clears the stored private key when the auth type changes to password", func(t *testing.T) {
		tm := newTeam(t)
		logical := newBastionedLogicalSettings(topology)
		logical.SshTunnel.AuthType = new(apiclient.SshAuthTypePRIVATEKEY)
		logical.SshTunnel.Password = nil
		logical.SshTunnel.PrivateKey = new(readBastionTestKey(t))
		createdDatabase := tm.createDatabaseNamed(t, "Switching auth", fromLogicalSettings(t, logical))
		switchedLogical := getLogicalSettings(t, createdDatabase)
		switchedLogical.SshTunnel.AuthType = new(apiclient.SshAuthTypePASSWORD)
		switchedLogical.SshTunnel.Password = new(containers.SshBastionPassword)
		update := toUpdate(createdDatabase)
		update.Settings = fromLogicalSettings(t, switchedLogical)

		response := updateDatabase(t, tm.owner, createdDatabase.Id, update)

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		storedTunnel := getStoredSSHTunnel(t, tm, "databases_postgresql_logical", createdDatabase.Id)
		assert.Empty(t, storedTunnel.SSHPrivateKey)
		assert.Equal(t, containers.SshBastionPassword, tm.app.DecryptSecret(t, storedTunnel.SSHPassword))
	})

	t.Run("keeps the tunnel settings in a copy of a bastioned database", func(t *testing.T) {
		tm := newTeam(t)
		createdDatabase := tm.createDatabaseNamed(t, "Bastioned to copy",
			fromLogicalSettings(t, newBastionedLogicalSettings(topology)))

		copiedDatabase := copyDatabase(t, tm.owner, createdDatabase.Id)

		copiedTunnel := getLogicalSettings(t, copiedDatabase).SshTunnel
		assert.True(t, copiedTunnel.IsEnabled)
		assert.Equal(t, new(topology.Bastion.Host), copiedTunnel.Host)
		assert.Equal(t, new(topology.Bastion.Port), copiedTunnel.Port)
		assert.Equal(t, new(containers.SshBastionUsername), copiedTunnel.Username)
		testResponse, err := tm.owner.Client.TestDatabaseWithResponse(t.Context(), copiedDatabase.Id)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, testResponse.StatusCode(), string(testResponse.Body))
		assert.True(t, testResponse.JSON200.IsConnected, string(testResponse.Body))
	})

	t.Run("returns 400 invalid_request naming the private key when private key auth carries only a password",
		func(t *testing.T) {
			tm := newTeam(t)
			logical := newLogicalSettings(server)
			logical.SshTunnel = newExampleSSHTunnel()
			logical.SshTunnel.AuthType = new(apiclient.SshAuthTypePRIVATEKEY)

			response, err := tm.owner.Client.CreateDatabaseWithResponse(t.Context(),
				newCreation(tm, fromLogicalSettings(t, logical)))

			require.NoError(t, err)
			requireInvalidField(t, response.StatusCode(), response.Body, "settings.sshTunnel.privateKey")
			assert.Empty(t, listDatabases(t, tm.owner, tm.workspace.Id))
		})

	// Storing the unused secret would leave a second, invisible way into the bastion behind: the edit
	// form only ever shows the chosen one, so nothing would surface it again.
	t.Run("returns 400 invalid_request when a password tunnel also carries a private key", func(t *testing.T) {
		tm := newTeam(t)
		logical := newLogicalSettings(server)
		logical.SshTunnel = newExampleSSHTunnel()
		logical.SshTunnel.PrivateKey = new("-----BEGIN OPENSSH PRIVATE KEY-----")

		response, err := tm.owner.Client.CreateDatabaseWithResponse(t.Context(),
			newCreation(tm, fromLogicalSettings(t, logical)))

		require.NoError(t, err)
		requireInvalidField(t, response.StatusCode(), response.Body, "settings.sshTunnel.privateKey")
		assert.Empty(t, listDatabases(t, tm.owner, tm.workspace.Id))
	})

	t.Run("returns 400 invalid_request naming the tunnel host when the tunnel has none", func(t *testing.T) {
		tm := newTeam(t)
		logical := newLogicalSettings(server)
		logical.SshTunnel = newExampleSSHTunnel()
		logical.SshTunnel.Host = nil

		response, err := tm.owner.Client.CreateDatabaseWithResponse(t.Context(),
			newCreation(tm, fromLogicalSettings(t, logical)))

		require.NoError(t, err)
		requireInvalidField(t, response.StatusCode(), response.Body, "settings.sshTunnel.host")
		assert.Empty(t, listDatabases(t, tm.owner, tm.workspace.Id))
	})

	t.Run("returns 400 invalid_request when a MongoDB SRV connection goes through a tunnel", func(t *testing.T) {
		tm := newTeam(t)
		mongodb := newMongodbSettings(containers.Endpoint{Host: "cluster0.example.mongodb.net", Port: 27017})
		mongodb.Port = nil
		mongodb.IsSrv = true
		mongodb.SshTunnel = newExampleSSHTunnel()

		response, err := tm.owner.Client.CreateDatabaseWithResponse(t.Context(),
			newCreation(tm, fromMongodbSettings(t, mongodb)))

		require.NoError(t, err)
		requireInvalidField(t, response.StatusCode(), response.Body, "settings.sshTunnel.isEnabled")
		assert.Empty(t, listDatabases(t, tm.owner, tm.workspace.Id))
	})

	// The cluster publishes no port, so reaching it at all proves the replication-protocol probe,
	// which does not go through pgx, went through the bastion.
	t.Run("creates a physical database behind a bastion with its version and cluster identity",
		func(t *testing.T) {
			physicalTopology := containers.StartPhysicalPostgresBehindSshBastion(t, "postgres:17")
			tm := newTeam(t)
			physical := newPhysicalSettings(physicalTopology.Database, apiclient.PhysicalBackupTypeFULL)
			physical.SshTunnel = newBastionTunnel(physicalTopology)

			createdDatabase := tm.createDatabaseNamed(t, "Bastioned cluster", fromPhysicalSettings(t, physical))

			createdPhysical := getPhysicalSettings(t, createdDatabase)
			assert.Equal(t, new(apiclient.PostgresqlVersionN17), createdPhysical.Version)
			assert.NotNil(t, createdPhysical.SystemIdentifier)
			assert.NotNil(t, createdPhysical.WalSegmentSizeBytes)
		})
}

func newBastionTunnel(topology containers.BastionedDatabase) apiclient.SshTunnelSettings {
	return apiclient.SshTunnelSettings{
		IsEnabled:            true,
		AuthType:             new(apiclient.SshAuthTypePASSWORD),
		Host:                 new(topology.Bastion.Host),
		Port:                 new(topology.Bastion.Port),
		Username:             new(containers.SshBastionUsername),
		Password:             new(containers.SshBastionPassword),
		PrivateKey:           nil,
		PrivateKeyPassphrase: nil,
	}
}

func newBastionedLogicalSettings(topology containers.BastionedDatabase) apiclient.PostgresqlLogicalSettings {
	logical := newLogicalSettings(topology.Database)
	logical.SshTunnel = newBastionTunnel(topology)

	return logical
}

// newExampleSSHTunnel is a valid password tunnel to a bastion that does not exist, for requests that
// are refused before any connection.
func newExampleSSHTunnel() apiclient.SshTunnelSettings {
	return apiclient.SshTunnelSettings{
		IsEnabled:            true,
		AuthType:             new(apiclient.SshAuthTypePASSWORD),
		Host:                 new("bastion.example.com"),
		Port:                 new(22),
		Username:             new("tunneluser"),
		Password:             new("tunnelpassword"),
		PrivateKey:           nil,
		PrivateKeyPassphrase: nil,
	}
}

func newCreation(tm team, settings apiclient.DatabaseSettings) apiclient.DatabaseCreate {
	return apiclient.DatabaseCreate{
		WorkspaceId: tm.workspace.Id,
		Name:        "Database " + uuid.NewString()[:8],
		NotifierIds: []uuid.UUID{},
		Settings:    settings,
	}
}

func readBastionTestKey(t *testing.T) string {
	t.Helper()

	privateKey, err := os.ReadFile(filepath.Join(containers.GetSshBastionTestdataDir(t), "test_key"))
	require.NoError(t, err)

	return string(privateKey)
}

type storedSSHTunnel struct {
	SSHHost                 string
	SSHAuthType             string
	SSHPassword             string
	SSHPrivateKey           string
	SSHPrivateKeyPassphrase string
}

// getStoredSSHTunnel reads the tunnel columns with the encrypted secrets, which the API never returns.
func getStoredSSHTunnel(t *testing.T, tm team, settingsTable string, databaseID uuid.UUID) storedSSHTunnel {
	t.Helper()

	var storedTunnels []storedSSHTunnel
	tm.app.QuerySQL(t, &storedTunnels, `SELECT ssh_host, ssh_auth_type, ssh_password, ssh_private_key,
		ssh_private_key_passphrase FROM `+settingsTable+` WHERE database_id = ?`, databaseID)
	require.Len(t, storedTunnels, 1)

	return storedTunnels[0]
}
