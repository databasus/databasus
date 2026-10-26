package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mysql_engine "databasus-backend/internal/engines/mysql"
	"databasus-backend/internal/features/sshtunnel"
	"databasus-backend/internal/util/testing/containers"
)

const discoveredVersion = mysql_engine.Version84

const unreachableBastionTimeout = 2 * time.Second

func enabledTunnelDatabase() *MysqlDatabase {
	databaseName := "testdb"

	return &MysqlDatabase{
		Host:     "db.internal",
		Port:     3306,
		Username: "testuser",
		Password: "testpassword",
		Database: &databaseName,
		SshTunnel: sshtunnel.Config{
			IsEnabled: true,
			Host:      "bastion.example.com",
			Port:      22,
			Username:  "tunneluser",
			AuthType:  sshtunnel.AuthTypePassword,
			Password:  "tunnelpassword",
		},
	}
}

func bastionedTunnelDatabase(t *testing.T) *MysqlDatabase {
	t.Helper()

	bastion := containers.StartSshBastion(t)

	database := enabledTunnelDatabase()
	database.SshTunnel.Host = bastion.Host
	database.SshTunnel.Port = bastion.Port
	database.SshTunnel.Username = containers.SshBastionUsername
	database.SshTunnel.Password = containers.SshBastionPassword

	return database
}

func Test_OpenTunnel_WhenTheTunnelIsDisabled_HandsBackTheSameDatabase(t *testing.T) {
	database := enabledTunnelDatabase()
	database.SshTunnel.IsEnabled = false

	tunneledDatabase, err := OpenTunnel(t.Context(), OpenTunnelSpec{Database: database})
	assert.NoError(t, err)

	defer tunneledDatabase.Close()

	assert.Same(t, database, tunneledDatabase.GetDatabaseThroughTunnel())
}

func Test_OpenTunnel_WhenTheBastionIsUnreachable_ReturnsAnError(t *testing.T) {
	database := enabledTunnelDatabase()
	// RFC 5737 TEST-NET-1: routable nowhere, so this fails to connect rather than resolving.
	database.SshTunnel.Host = "192.0.2.1"

	// A blackholing network would otherwise burn the forwarder's full 30s dial timeout here.
	ctx, cancel := context.WithTimeout(t.Context(), unreachableBastionTimeout)
	defer cancel()

	_, err := OpenTunnel(ctx, OpenTunnelSpec{Database: database})

	assert.Error(t, err)
}

func Test_OpenTunnel_WhenTunneled_DisablesTheTunnelFlagOnTheCopy(t *testing.T) {
	database := bastionedTunnelDatabase(t)

	tunneledDatabase, err := OpenTunnel(t.Context(), OpenTunnelSpec{Database: database})
	require.NoError(t, err)

	defer tunneledDatabase.Close()

	databaseThroughTunnel := tunneledDatabase.GetDatabaseThroughTunnel()

	assert.False(t, databaseThroughTunnel.SshTunnel.IsEnabled,
		"a re-read of the copy would otherwise open a second tunnel into the first one")
	assert.True(t, database.SshTunnel.IsEnabled, "the original must keep the stored configuration")
	assert.Equal(t, "db.internal", database.Host, "the original address must not be rewritten")
	assert.Equal(t, 3306, database.Port)
	assert.Equal(t, "127.0.0.1", databaseThroughTunnel.Host)
	assert.NotEqual(t, database.Port, databaseThroughTunnel.Port)
}

// Discovery runs against the copy, so anything it writes there is lost unless it is carried back:
// a bastioned database would otherwise be saved with the privileges of a connection never made.
func Test_CopyDiscoveredMetadataToOriginal_CarriesTheVersionAndPrivileges(t *testing.T) {
	database := bastionedTunnelDatabase(t)

	tunneledDatabase, err := OpenTunnel(t.Context(), OpenTunnelSpec{Database: database})
	require.NoError(t, err)

	defer tunneledDatabase.Close()

	databaseThroughTunnel := tunneledDatabase.GetDatabaseThroughTunnel()
	databaseThroughTunnel.Version = discoveredVersion
	databaseThroughTunnel.Privileges = "SELECT,LOCK TABLES"

	tunneledDatabase.CopyDiscoveredMetadataToOriginal()

	assert.Equal(t, discoveredVersion, database.Version)
	assert.Equal(t, "SELECT,LOCK TABLES", database.Privileges)
}
