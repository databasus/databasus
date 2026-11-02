package databases_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

const movedSSHTunnelHost = "bastion-moved.example.com"

// secretLifecycleCase saves a database of one type with a password and a disabled tunnel that keeps
// its secrets, then changes it with every secret left out.
type secretLifecycleCase struct {
	name           string
	settingsTable  string
	storedPassword string
	startServer    func(t *testing.T) containers.Endpoint
	newSettings    func(t *testing.T, server containers.Endpoint, tunnel apiclient.SshTunnelSettings) apiclient.DatabaseSettings
	withoutSecrets func(t *testing.T, database apiclient.Database, tunnel apiclient.SshTunnelSettings) apiclient.DatabaseSettings
}

func TestDatabaseSecrets(t *testing.T) {
	t.Parallel()

	secretLifecycleCases := []secretLifecycleCase{
		{
			name:           "PostgreSQL logical",
			settingsTable:  "databases_postgresql_logical",
			storedPassword: containers.PostgresPassword,
			startServer: func(t *testing.T) containers.Endpoint {
				t.Helper()

				return containers.StartPostgres(t, "postgres:16")
			},
			newSettings: func(
				t *testing.T, server containers.Endpoint, tunnel apiclient.SshTunnelSettings,
			) apiclient.DatabaseSettings {
				t.Helper()

				logical := newLogicalSettings(server)
				logical.SshTunnel = tunnel

				return fromLogicalSettings(t, logical)
			},
			withoutSecrets: func(
				t *testing.T, database apiclient.Database, tunnel apiclient.SshTunnelSettings,
			) apiclient.DatabaseSettings {
				t.Helper()

				logical := getLogicalSettings(t, database)
				logical.Password = nil
				logical.SshTunnel = tunnel

				return fromLogicalSettings(t, logical)
			},
		},
		{
			name:           "PostgreSQL physical",
			settingsTable:  "databases_postgresql_physical",
			storedPassword: containers.PostgresPassword,
			startServer: func(t *testing.T) containers.Endpoint {
				t.Helper()

				return containers.StartPhysicalPostgres(t, "postgres:17")
			},
			newSettings: func(
				t *testing.T, server containers.Endpoint, tunnel apiclient.SshTunnelSettings,
			) apiclient.DatabaseSettings {
				t.Helper()

				physical := newPhysicalSettings(server, apiclient.PhysicalBackupTypeFULL)
				physical.SshTunnel = tunnel

				return fromPhysicalSettings(t, physical)
			},
			withoutSecrets: func(
				t *testing.T, database apiclient.Database, tunnel apiclient.SshTunnelSettings,
			) apiclient.DatabaseSettings {
				t.Helper()

				physical := getPhysicalSettings(t, database)
				physical.Password = nil
				physical.SshTunnel = tunnel

				return fromPhysicalSettings(t, physical)
			},
		},
		{
			name:           "MySQL",
			settingsTable:  "databases_mysql",
			storedPassword: containers.MysqlPassword,
			startServer: func(t *testing.T) containers.Endpoint {
				t.Helper()

				return containers.StartMysql(t, "mysql:8.0")
			},
			newSettings: func(
				t *testing.T, server containers.Endpoint, tunnel apiclient.SshTunnelSettings,
			) apiclient.DatabaseSettings {
				t.Helper()

				mysql := newMysqlSettings(server)
				mysql.SshTunnel = tunnel

				return fromMysqlSettings(t, mysql)
			},
			withoutSecrets: func(
				t *testing.T, database apiclient.Database, tunnel apiclient.SshTunnelSettings,
			) apiclient.DatabaseSettings {
				t.Helper()

				mysql := getMysqlSettings(t, database)
				mysql.Password = nil
				mysql.SshTunnel = tunnel

				return fromMysqlSettings(t, mysql)
			},
		},
		{
			name:           "MariaDB",
			settingsTable:  "databases_mariadb",
			storedPassword: containers.MariadbPassword,
			startServer: func(t *testing.T) containers.Endpoint {
				t.Helper()

				return containers.StartMariadb(t, "mariadb:10.11")
			},
			newSettings: func(
				t *testing.T, server containers.Endpoint, tunnel apiclient.SshTunnelSettings,
			) apiclient.DatabaseSettings {
				t.Helper()

				mariadb := newMariadbSettings(server)
				mariadb.SshTunnel = tunnel

				return fromMariadbSettings(t, mariadb)
			},
			withoutSecrets: func(
				t *testing.T, database apiclient.Database, tunnel apiclient.SshTunnelSettings,
			) apiclient.DatabaseSettings {
				t.Helper()

				mariadb := getMariadbSettings(t, database)
				mariadb.Password = nil
				mariadb.SshTunnel = tunnel

				return fromMariadbSettings(t, mariadb)
			},
		},
		{
			name:           "MongoDB",
			settingsTable:  "databases_mongodb",
			storedPassword: containers.MongodbPassword,
			startServer: func(t *testing.T) containers.Endpoint {
				t.Helper()

				return containers.StartMongodb(t, "mongo:7.0")
			},
			newSettings: func(
				t *testing.T, server containers.Endpoint, tunnel apiclient.SshTunnelSettings,
			) apiclient.DatabaseSettings {
				t.Helper()

				mongodb := newMongodbSettings(server)
				mongodb.SshTunnel = tunnel

				return fromMongodbSettings(t, mongodb)
			},
			withoutSecrets: func(
				t *testing.T, database apiclient.Database, tunnel apiclient.SshTunnelSettings,
			) apiclient.DatabaseSettings {
				t.Helper()

				mongodb := getMongodbSettings(t, database)
				mongodb.Password = nil
				mongodb.SshTunnel = tunnel

				return fromMongodbSettings(t, mongodb)
			},
		},
	}

	for _, lifecycleCase := range secretLifecycleCases {
		t.Run("keeps the stored "+lifecycleCase.name+" secrets encrypted and hidden when a change leaves them out",
			func(t *testing.T) {
				t.Parallel()

				server := lifecycleCase.startServer(t)
				tm := newTeam(t)
				createdDatabase := tm.createDatabaseNamed(t, "With secrets",
					lifecycleCase.newSettings(t, server, newStoredSSHTunnel()))
				update := toUpdate(getDatabase(t, tm.owner, createdDatabase.Id))
				update.Name = "With secrets renamed"
				update.Settings = lifecycleCase.withoutSecrets(
					t,
					createdDatabase,
					newSubmittedSSHTunnelWithoutSecrets(),
				)

				response := updateDatabase(t, tm.owner, createdDatabase.Id, update)

				require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
				storedPassword := getStoredPassword(t, tm, lifecycleCase.settingsTable, createdDatabase.Id)
				assert.True(t, strings.HasPrefix(storedPassword, "enc:"))
				assert.Equal(t, lifecycleCase.storedPassword, tm.app.DecryptSecret(t, storedPassword))

				storedTunnel := getStoredSSHTunnel(t, tm, lifecycleCase.settingsTable, createdDatabase.Id)
				assert.Equal(t, movedSSHTunnelHost, storedTunnel.SSHHost)
				assert.Equal(t, string(apiclient.SshAuthTypePRIVATEKEY), storedTunnel.SSHAuthType)
				assert.True(t, strings.HasPrefix(storedTunnel.SSHPrivateKey, "enc:"))
				assert.Equal(t, "tunnel-private-key-value", tm.app.DecryptSecret(t, storedTunnel.SSHPrivateKey))
				assert.True(t, strings.HasPrefix(storedTunnel.SSHPrivateKeyPassphrase, "enc:"))
				assert.Equal(t, "tunnel-passphrase-value",
					tm.app.DecryptSecret(t, storedTunnel.SSHPrivateKeyPassphrase))
				assert.Equal(t, "With secrets renamed", getDatabase(t, tm.owner, createdDatabase.Id).Name)

				for _, responseBody := range [][]byte{
					response.Body,
					getRawDatabase(t, tm, createdDatabase),
					getRawDatabaseList(t, tm),
				} {
					assert.NotContains(t, string(responseBody), lifecycleCase.storedPassword)
					assert.NotContains(t, string(responseBody), "tunnel-private-key-value")
					assert.NotContains(t, string(responseBody), "tunnel-passphrase-value")
					assert.NotContains(t, string(responseBody), "enc:")
					assert.Contains(t, string(responseBody), "tunneluser")
				}
			})
	}
}

// newStoredSSHTunnel is disabled, so the save connects directly, yet keeps its secrets like a tunnel
// the user turned off for a while.
func newStoredSSHTunnel() apiclient.SshTunnelSettings {
	return apiclient.SshTunnelSettings{
		IsEnabled:            false,
		AuthType:             new(apiclient.SshAuthTypePRIVATEKEY),
		Host:                 new("bastion.example.com"),
		Port:                 new(2222),
		Username:             new("tunneluser"),
		Password:             nil,
		PrivateKey:           new("tunnel-private-key-value"),
		PrivateKeyPassphrase: new("tunnel-passphrase-value"),
	}
}

// newSubmittedSSHTunnelWithoutSecrets moves the host as well, so a change that kept the stored tunnel
// by ignoring the submitted one would not pass. The auth type is left out with the secrets: blank
// means unchanged for it too, and overwriting it would drop the stored key with it.
func newSubmittedSSHTunnelWithoutSecrets() apiclient.SshTunnelSettings {
	sshTunnel := newStoredSSHTunnel()
	sshTunnel.Host = new(movedSSHTunnelHost)
	sshTunnel.AuthType = nil
	sshTunnel.PrivateKey = nil
	sshTunnel.PrivateKeyPassphrase = nil

	return sshTunnel
}

func getRawDatabase(t *testing.T, tm team, database apiclient.Database) []byte {
	t.Helper()

	response, err := tm.owner.Client.GetDatabaseWithResponse(t.Context(), database.Id)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return response.Body
}

func getRawDatabaseList(t *testing.T, tm team) []byte {
	t.Helper()

	response, err := tm.owner.Client.ListDatabasesWithResponse(t.Context(), &apiclient.ListDatabasesParams{
		WorkspaceId: tm.workspace.Id,
		NotifierId:  nil,
		Limit:       nil,
		After:       nil,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return response.Body
}
