package postgresql_physical

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/features/sshtunnel"
	"databasus-backend/internal/util/testing/containers"
)

const unreachableBastionTimeout = 2 * time.Second

func enabledTunnelDatabase() *PostgresqlPhysicalDatabase {
	parentDatabaseID := uuid.New()

	return &PostgresqlPhysicalDatabase{
		ID:                  uuid.New(),
		DatabaseID:          &parentDatabaseID,
		Version:             pgshared.Version17,
		BackupType:          BackupTypeFullIncrementalAndWalStream,
		Host:                "cluster.internal",
		Port:                5432,
		Username:            "testuser",
		Password:            "testpassword",
		ReplicationSlotName: "databasus_slot_deadbeef",
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

func bastionedTunnelDatabase(t *testing.T) *PostgresqlPhysicalDatabase {
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

func Test_OpenTunnel_WhenTheStoredTargetIsEmbedded_ReturnsError(t *testing.T) {
	database := enabledTunnelDatabase()
	database.Host = "/tmp"
	database.Port = 5437
	database.SshTunnel.IsEnabled = false

	_, err := OpenTunnel(t.Context(), OpenTunnelSpec{Database: database})

	require.ErrorContains(t, err, "backing up Databasus internal PostgreSQL cluster is not allowed")
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

// PostgreSQL is the one engine whose copy keeps the real host: sslmode=verify-full matches the
// certificate against it, and libpq looks up .pgpass by host and port.
func Test_OpenTunnel_WhenTunneled_KeepsTheRealHostAndCarriesTheForwardedPort(t *testing.T) {
	database := bastionedTunnelDatabase(t)

	tunneledDatabase, err := OpenTunnel(t.Context(), OpenTunnelSpec{Database: database})
	require.NoError(t, err)

	defer tunneledDatabase.Close()

	databaseThroughTunnel := tunneledDatabase.GetDatabaseThroughTunnel()

	assert.Equal(t, "cluster.internal", databaseThroughTunnel.Host)
	require.NotNil(t, databaseThroughTunnel.LocalTunnelEndpoint)
	assert.Equal(t, "127.0.0.1", databaseThroughTunnel.LocalTunnelEndpoint.Host)
	assert.Equal(t, databaseThroughTunnel.LocalTunnelEndpoint.Port, databaseThroughTunnel.Port)
	assert.NotEqual(t, database.Port, databaseThroughTunnel.Port)
	assert.Equal(t, "127.0.0.1", databaseThroughTunnel.GetHostAddr())
}

func Test_OpenTunnel_WhenTunneled_DisablesTheTunnelFlagOnTheCopy(t *testing.T) {
	database := bastionedTunnelDatabase(t)

	tunneledDatabase, err := OpenTunnel(t.Context(), OpenTunnelSpec{Database: database})
	require.NoError(t, err)

	defer tunneledDatabase.Close()

	assert.False(t, tunneledDatabase.GetDatabaseThroughTunnel().SshTunnel.IsEnabled,
		"a re-read of the copy would otherwise open a second tunnel into the first one")
	assert.True(t, database.SshTunnel.IsEnabled, "the original must keep the stored configuration")
	assert.Equal(t, 5432, database.Port, "the original address must not be rewritten")
	assert.Nil(t, database.LocalTunnelEndpoint)
}

// Both slot families are named from these: the per-backup slot from ID, the streamer slot from the
// stored name, and the receiver's application_name from DatabaseID. A copy that blanked them would
// mint a fresh slot on the source cluster every time the tunnel is reopened.
func Test_OpenTunnel_WhenTunneled_KeepsTheSlotIdentityOnTheCopy(t *testing.T) {
	database := bastionedTunnelDatabase(t)

	tunneledDatabase, err := OpenTunnel(t.Context(), OpenTunnelSpec{Database: database})
	require.NoError(t, err)

	defer tunneledDatabase.Close()

	databaseThroughTunnel := tunneledDatabase.GetDatabaseThroughTunnel()

	assert.Equal(t, database.ID, databaseThroughTunnel.ID)
	require.NotNil(t, databaseThroughTunnel.DatabaseID)
	assert.Equal(t, *database.DatabaseID, *databaseThroughTunnel.DatabaseID)
	assert.Equal(t, database.ReplicationSlotName, databaseThroughTunnel.ReplicationSlotName)
	assert.Equal(t, database.ParentDatabaseID(), databaseThroughTunnel.ParentDatabaseID())
}

// PopulateDbData writes all three into the model it was handed, which is the copy. Carrying back
// only Version would persist a NULL system_identifier and put 0 into every backup manifest.
func Test_CopyDiscoveredMetadataToOriginal_AfterPopulateDbData_CarriesVersionAndClusterIdentity(t *testing.T) {
	database := bastionedTunnelDatabase(t)
	database.Version = ""

	tunneledDatabase, err := OpenTunnel(t.Context(), OpenTunnelSpec{Database: database})
	require.NoError(t, err)

	defer tunneledDatabase.Close()

	discoveredSystemIdentifier := "7150000000000000000"
	discoveredWalSegmentSizeBytes := int64(16 * 1024 * 1024)

	databaseThroughTunnel := tunneledDatabase.GetDatabaseThroughTunnel()
	databaseThroughTunnel.Version = pgshared.Version18
	databaseThroughTunnel.SystemIdentifier = &discoveredSystemIdentifier
	databaseThroughTunnel.WalSegmentSizeBytes = &discoveredWalSegmentSizeBytes

	tunneledDatabase.CopyDiscoveredMetadataToOriginal()

	assert.Equal(t, pgshared.Version18, database.Version)
	require.NotNil(t, database.SystemIdentifier)
	assert.Equal(t, discoveredSystemIdentifier, *database.SystemIdentifier)
	require.NotNil(t, database.WalSegmentSizeBytes)
	assert.Equal(t, discoveredWalSegmentSizeBytes, *database.WalSegmentSizeBytes)
}

func Test_IsBastionReachable_WhenTheTunnelIsDisabled_ReportsReachable(t *testing.T) {
	database := enabledTunnelDatabase()
	database.SshTunnel.IsEnabled = false

	tunneledDatabase, err := OpenTunnel(t.Context(), OpenTunnelSpec{Database: database})
	require.NoError(t, err)

	defer tunneledDatabase.Close()

	assert.True(t, tunneledDatabase.IsBastionReachable(t.Context()),
		"with no bastion in the path a receiver failure is the source's own")
}
