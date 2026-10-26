package usecases_physical_postgresql

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	"databasus-backend/internal/lib/sshtunnel"
)

const (
	tunneledSourceHost = "cluster.internal"
	forwardedLocalPort = 41000
)

// The copy handed out by OpenTunnel, as the CLI paths receive it: the port already points at the
// forwarder while Host still names the cluster.
func tunneledSourceDatabase() *postgresql_physical.PostgresqlPhysicalDatabase {
	return &postgresql_physical.PostgresqlPhysicalDatabase{
		Host:                tunneledSourceHost,
		Port:                forwardedLocalPort,
		Username:            "replicator",
		ReplicationSlotName: "slot",
		LocalTunnelEndpoint: &sshtunnel.Endpoint{Host: "127.0.0.1", Port: forwardedLocalPort},
	}
}

// libpq connects to hostaddr while still matching the certificate and .pgpass against host, so
// rewriting -h would break sslmode=verify-full and the password lookup at once.
func Test_NewPgBasebackupCommand_WhenTunneled_SendsPgHostAddrAndKeepsTheRealHostFlag(t *testing.T) {
	cmd, err := newPgBasebackupCommand(
		t.Context(),
		"sh",
		tunneledSourceDatabase(),
		&pgshared.CredentialTempFiles{PgpassPath: "/tmp/pgpass"},
		"label",
		physical_enums.PhysicalBackupCompressionNone,
		"",
	)
	require.NoError(t, err)

	assert.Contains(t, cmd.Env, "PGHOSTADDR=127.0.0.1")
	assertHostFlagIsTheRealCluster(t, cmd.Args)
}

func Test_NewPgBasebackupCommand_WithoutATunnel_SendsNoPgHostAddr(t *testing.T) {
	sourceDB := tunneledSourceDatabase()
	sourceDB.LocalTunnelEndpoint = nil

	cmd, err := newPgBasebackupCommand(
		t.Context(),
		"sh",
		sourceDB,
		&pgshared.CredentialTempFiles{PgpassPath: "/tmp/pgpass"},
		"label",
		physical_enums.PhysicalBackupCompressionNone,
		"",
	)
	require.NoError(t, err)

	assertNoPgHostAddr(t, cmd.Env)
}

func assertHostFlagIsTheRealCluster(t *testing.T, args []string) {
	t.Helper()

	hostFlagIndex := slices.Index(args, "-h")
	require.NotEqual(t, -1, hostFlagIndex, "the tools still address the cluster by name")
	require.Less(t, hostFlagIndex+1, len(args))
	assert.Equal(t, tunneledSourceHost, args[hostFlagIndex+1])
}

func assertNoPgHostAddr(t *testing.T, env []string) {
	t.Helper()

	for _, variable := range env {
		assert.NotContains(t, variable, "PGHOSTADDR=",
			"a direct connection must not be redirected anywhere")
	}
}
