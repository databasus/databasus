package databases_test

import (
	"encoding/pem"
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
	"golang.org/x/crypto/ssh"

	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

// Without this the tunnel tests would stay green if the tunnel silently stopped being used, because
// every other assertion only proves that some route to the database exists.
func TestBastionedPostgres(t *testing.T) {
	t.Parallel()

	t.Run("refuses a connection from the host without the tunnel", func(t *testing.T) {
		t.Parallel()

		topology := containers.StartPostgresBehindSSHBastion(t, "postgres:16")
		address := net.JoinHostPort(topology.Database.Host, strconv.Itoa(topology.Database.Port))

		dialer := net.Dialer{Timeout: 5 * time.Second}
		conn, err := dialer.DialContext(t.Context(), "tcp", address)

		if err == nil {
			_ = conn.Close()
			t.Fatalf("the bastioned database answered on %s without a tunnel", address)
		}
	})
}

//nolint:tparallel // The subtests share one source server.
func TestDatabaseSSHTunnel(t *testing.T) {
	t.Parallel()

	topology := containers.StartPostgresBehindSSHBastion(t, "postgres:16")
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
			assert.NotContains(t, string(readResponse.Body), containers.SSHBastionPassword)
			readTunnel := getLogicalSettings(t, *readResponse.JSON200).SshTunnel
			assert.Nil(t, readTunnel.Password)
			assert.Nil(t, readTunnel.PrivateKey)
			assert.Nil(t, readTunnel.PrivateKeyPassphrase)
			assert.Equal(t, new(containers.SSHBastionUsername), readTunnel.Username)

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
		switchedLogical.SshTunnel.Password = new(containers.SSHBastionPassword)
		update := toUpdate(createdDatabase)
		update.Settings = fromLogicalSettings(t, switchedLogical)

		response := updateDatabase(t, tm.owner, createdDatabase.Id, update)

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		storedTunnel := getStoredSSHTunnel(t, tm, "databases_postgresql_logical", createdDatabase.Id)
		assert.Empty(t, storedTunnel.SSHPrivateKey)
		assert.Equal(t, containers.SSHBastionPassword, tm.app.DecryptSecret(t, storedTunnel.SSHPassword))
	})

	t.Run("returns 400 invalid_request and connects nowhere when the bastion moves without the secrets",
		func(t *testing.T) {
			tm := newTeam(t)
			recorder := startConnectionRecorder(t)
			createdDatabase := tm.createDatabaseNamed(t, "Bastioned",
				fromLogicalSettings(t, newBastionedLogicalSettings(topology)))
			movedLogical := newBastionedLogicalSettings(topology)
			movedLogical.SshTunnel.Host = new(recorder.host)
			movedLogical.SshTunnel.Port = new(recorder.port)
			movedLogical.SshTunnel.Password = nil
			movedLogical.Password = new(containers.PostgresPassword)

			testResponse, err := tm.owner.Client.TestDatabaseSettingsWithResponse(t.Context(), newSettingsTarget(
				tm.workspace.Id, &createdDatabase.Id, fromLogicalSettings(t, movedLogical),
			))

			require.NoError(t, err)
			requireInvalidField(t, testResponse.StatusCode(), testResponse.Body, "settings.sshTunnel.password")

			movedLogical.Password = nil
			update := toUpdate(createdDatabase)
			update.Settings = fromLogicalSettings(t, movedLogical)

			updateResponse := updateDatabase(t, tm.owner, createdDatabase.Id, update)

			requireInvalidField(t, updateResponse.StatusCode(), updateResponse.Body, "settings.password")
			assert.Zero(t, recorder.getAcceptedCount())
			assert.Equal(t, new(topology.Bastion.Host),
				getLogicalSettings(t, getDatabase(t, tm.owner, createdDatabase.Id)).SshTunnel.Host)
		})

	// The second server publishes no port, so connecting to it proves the backend opened the tunnel
	// with the stored key.
	t.Run(
		"opens the tunnel with the stored key and reaches the new host when the database moves behind the same bastion",
		func(t *testing.T) {
			tm := newTeam(t)
			member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
			bastionPrivateKey := readBastionTestKey(t)
			secondServer := startSecondPostgresBehindBastion(t, topology)
			keyTunnelLogical := newBastionedLogicalSettings(topology)
			keyTunnelLogical.SshTunnel.AuthType = new(apiclient.SshAuthTypePRIVATEKEY)
			keyTunnelLogical.SshTunnel.Password = nil
			keyTunnelLogical.SshTunnel.PrivateKey = new(bastionPrivateKey)
			createdDatabase := tm.createDatabaseNamed(t, "Bastioned", fromLogicalSettings(t, keyTunnelLogical))
			movedLogical := keyTunnelLogical
			movedLogical.Host = secondServer.Host
			movedLogical.Port = secondServer.Port
			movedLogical.Password = new(containers.PostgresPassword)
			movedLogical.SshTunnel.PrivateKey = nil

			testResult := testUnsavedSettings(t, member, newSettingsTarget(
				tm.workspace.Id, &createdDatabase.Id, fromLogicalSettings(t, movedLogical),
			))

			assert.True(t, testResult.IsConnected, "%v", testResult.Error)

			update := toUpdate(createdDatabase)
			update.Settings = fromLogicalSettings(t, movedLogical)

			response := updateDatabase(t, member, createdDatabase.Id, update)

			require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
			assert.Equal(t, secondServer.Host, getLogicalSettings(t, getDatabase(t, member, createdDatabase.Id)).Host)
			storedTunnel := getStoredSSHTunnel(t, tm, "databases_postgresql_logical", createdDatabase.Id)
			assert.Equal(t, bastionPrivateKey, tm.app.DecryptSecret(t, storedTunnel.SSHPrivateKey))
		},
	)

	t.Run("removes the stored passphrase when a moved tunnel gets a new key without one", func(t *testing.T) {
		tm := newTeam(t)
		member := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
		bastionPrivateKey := readBastionTestKey(t)
		keyTunnelLogical := newBastionedLogicalSettings(topology)
		keyTunnelLogical.SshTunnel.AuthType = new(apiclient.SshAuthTypePRIVATEKEY)
		keyTunnelLogical.SshTunnel.Password = nil
		keyTunnelLogical.SshTunnel.PrivateKey = new(encryptPrivateKey(t, bastionPrivateKey, "stored-passphrase"))
		keyTunnelLogical.SshTunnel.PrivateKeyPassphrase = new("stored-passphrase")
		createdDatabase := tm.createDatabaseNamed(t, "Bastioned by an encrypted key",
			fromLogicalSettings(t, keyTunnelLogical))
		movedLogical := keyTunnelLogical
		movedLogical.Password = new(containers.PostgresPassword)
		movedLogical.SshTunnel.Host = new(getAnotherNameOfLocalHost(t, topology.Bastion.Host))
		movedLogical.SshTunnel.PrivateKey = new(bastionPrivateKey)
		movedLogical.SshTunnel.PrivateKeyPassphrase = nil
		update := toUpdate(createdDatabase)
		update.Settings = fromLogicalSettings(t, movedLogical)

		response := updateDatabase(t, member, createdDatabase.Id, update)

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		storedTunnel := getStoredSSHTunnel(t, tm, "databases_postgresql_logical", createdDatabase.Id)
		assert.Empty(t, storedTunnel.SSHPrivateKeyPassphrase)
		assert.Equal(t, bastionPrivateKey, tm.app.DecryptSecret(t, storedTunnel.SSHPrivateKey))
	})

	t.Run("returns 400 invalid_request when a disabled tunnel with a moved host is turned on without the secrets",
		func(t *testing.T) {
			tm := newTeam(t)
			logical := newLogicalSettings(server)
			logical.SshTunnel = newStoredSSHTunnel()
			createdDatabase := tm.createDatabaseNamed(t, "Direct", fromLogicalSettings(t, logical))
			movedLogical := getLogicalSettings(t, createdDatabase)
			movedLogical.SshTunnel = newSubmittedSSHTunnelWithoutSecrets()
			update := toUpdate(createdDatabase)
			update.Settings = fromLogicalSettings(t, movedLogical)

			moveResponse := updateDatabase(t, tm.owner, createdDatabase.Id, update)
			require.Equal(t, http.StatusOK, moveResponse.StatusCode(), string(moveResponse.Body))

			enabledLogical := getLogicalSettings(t, getDatabase(t, tm.owner, createdDatabase.Id))
			enabledLogical.Password = new(containers.PostgresPassword)
			enabledLogical.SshTunnel.IsEnabled = true
			update.Settings = fromLogicalSettings(t, enabledLogical)

			enableResponse := updateDatabase(t, tm.owner, createdDatabase.Id, update)

			requireInvalidField(t, enableResponse.StatusCode(), enableResponse.Body, "settings.sshTunnel.privateKey")
			storedTunnel := getStoredSSHTunnel(t, tm, "databases_postgresql_logical", createdDatabase.Id)
			assert.Equal(t, "tunnel-private-key-value", tm.app.DecryptSecret(t, storedTunnel.SSHPrivateKey))
		})

	t.Run("returns 400 invalid_request and keeps the secrets when the tunnel is turned off without the password",
		func(t *testing.T) {
			tm := newTeam(t)
			createdDatabase := tm.createDatabaseNamed(t, "Bastioned",
				fromLogicalSettings(t, newBastionedLogicalSettings(topology)))
			directLogical := getLogicalSettings(t, createdDatabase)
			directLogical.SshTunnel.IsEnabled = false
			update := toUpdate(createdDatabase)
			update.Settings = fromLogicalSettings(t, directLogical)

			response := updateDatabase(t, tm.owner, createdDatabase.Id, update)

			requireInvalidField(t, response.StatusCode(), response.Body, "settings.password")
			assert.True(t, getLogicalSettings(t, getDatabase(t, tm.owner, createdDatabase.Id)).SshTunnel.IsEnabled)
			storedTunnel := getStoredSSHTunnel(t, tm, "databases_postgresql_logical", createdDatabase.Id)
			assert.Equal(t, containers.SSHBastionPassword, tm.app.DecryptSecret(t, storedTunnel.SSHPassword))
		})

	t.Run("removes the ssh secrets when the tunnel is turned off with the password", func(t *testing.T) {
		tm := newTeam(t)
		logical := newLogicalSettings(server)
		logical.SshTunnel = newStoredSSHTunnel()
		createdDatabase := tm.createDatabaseNamed(t, "Direct", fromLogicalSettings(t, logical))
		// A database that answers both through a tunnel and directly needs a bastion on the host's
		// network, and only the stored flag matters to this change.
		tm.app.ExecuteSQL(t, "UPDATE databases_postgresql_logical SET ssh_is_enabled = TRUE WHERE database_id = ?",
			createdDatabase.Id)
		directLogical := newLogicalSettings(server)
		directLogical.SshTunnel = newDisabledSSHTunnel()
		update := toUpdate(createdDatabase)
		update.Settings = fromLogicalSettings(t, directLogical)

		response := updateDatabase(t, tm.owner, createdDatabase.Id, update)

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		storedTunnel := getStoredSSHTunnel(t, tm, "databases_postgresql_logical", createdDatabase.Id)
		assert.Empty(t, storedTunnel.SSHPrivateKey)
		assert.Empty(t, storedTunnel.SSHPrivateKeyPassphrase)
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
		assert.Equal(t, new(containers.SSHBastionUsername), copiedTunnel.Username)

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
			physicalTopology := containers.StartPhysicalPostgresBehindSSHBastion(t, "postgres:17")
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

func startSecondPostgresBehindBastion(t *testing.T, topology containers.BastionedDatabase) containers.Endpoint {
	t.Helper()

	bastionNetworks, err := topology.BastionContainer.Networks(t.Context())
	require.NoError(t, err)
	require.Len(t, bastionNetworks, 1)

	return containers.StartPostgresOnNetwork(t, containers.OnNetworkSpec{
		Image: "postgres:16",
		Placement: containers.NetworkPlacement{
			NetworkName: bastionNetworks[0],
			Alias:       "second-postgres-behind-bastion",
		},
	})
}

func encryptPrivateKey(t *testing.T, privateKey, passphrase string) string {
	t.Helper()

	rawKey, err := ssh.ParseRawPrivateKey([]byte(privateKey))
	require.NoError(t, err)
	encryptedBlock, err := ssh.MarshalPrivateKeyWithPassphrase(rawKey, "", []byte(passphrase))
	require.NoError(t, err)

	return string(pem.EncodeToMemory(encryptedBlock))
}

// getAnotherNameOfLocalHost gives a published container a second address, which the resource address
// comparison sees as a move while the connection still reaches the same container.
func getAnotherNameOfLocalHost(t *testing.T, host string) string {
	t.Helper()

	require.Contains(t, []string{"localhost", "127.0.0.1"}, host)

	if host == "localhost" {
		return "127.0.0.1"
	}

	return "localhost"
}

func newBastionTunnel(topology containers.BastionedDatabase) apiclient.SshTunnelSettings {
	return apiclient.SshTunnelSettings{
		IsEnabled:            true,
		AuthType:             new(apiclient.SshAuthTypePASSWORD),
		Host:                 new(topology.Bastion.Host),
		Port:                 new(topology.Bastion.Port),
		Username:             new(containers.SSHBastionUsername),
		Password:             new(containers.SSHBastionPassword),
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

	privateKey, err := os.ReadFile(filepath.Join(containers.GetSSHBastionTestdataDir(t), "test_key"))
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
