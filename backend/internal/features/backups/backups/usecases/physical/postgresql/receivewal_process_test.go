package usecases_physical_postgresql

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/modules/databases"
)

func Test_NewReceivewalCommand_SetsParentDeathSignalAndApplicationName(t *testing.T) {
	databaseID := uuid.New()
	sourceDB := databases.PostgresqlPhysicalConnectionSettings{
		DatabaseID: databaseID,
		Engine: pgphysical.Settings{
			Host:                "localhost",
			Port:                5432,
			Username:            "replicator",
			ReplicationSlotName: "slot",
		},
	}

	cmd, err := newReceivewalCommand(t.Context(), receivewalCommandSpec{
		PgBin:    "sh",
		SourceDB: sourceDB,
		Creds:    &pgshared.CredentialTempFiles{PgpassPath: "/tmp/pgpass"},
		WatchDir: t.TempDir(),
		SlotName: "slot",
	})
	require.NoError(t, err)
	require.NotNil(t, cmd.SysProcAttr)
	require.Equal(t, syscall.SIGTERM, cmd.SysProcAttr.Pdeathsig)

	applicationName := "PGAPPNAME=" + receivewalApplicationNamePrefix + databaseID.String()
	require.True(t, strings.Contains(strings.Join(cmd.Env, "\n"), applicationName))
}

func Test_IsFatalReceivewalError_ClassifiesNonRetryableStderr(t *testing.T) {
	fatal := []string{
		"pg_receivewal: error: could not write 16777216 bytes to WAL file: No space left on device",
		`pg_receivewal: error: connection failed: FATAL:  password authentication failed for user "repl"`,
		"FATAL:  no pg_hba.conf entry for replication connection from host 10.0.0.1",
		"could not create archive status file: Permission denied",
		`ERROR:  replication slot "db_slot" is active for PID 4242`,
	}
	for _, stderr := range fatal {
		require.True(t, isFatalReceivewalError([]byte(stderr)), stderr)
	}

	transientStderrs := []string{
		"pg_receivewal: error: could not receive data from WAL stream: server closed the connection unexpectedly",
		"pg_receivewal: error: connection to server failed: Connection refused",
		"",
	}
	for _, stderr := range transientStderrs {
		require.False(t, isFatalReceivewalError([]byte(stderr)), stderr)
	}
}

func Test_IsResumeMismatchError_WhenSegmentAlreadyRemoved_ReturnsTrue(t *testing.T) {
	stderr := "pg_receivewal: error: could not send replication command \"START_REPLICATION\": " +
		"ERROR:  requested WAL segment 0000000100000000E000002F has already been removed"

	require.True(t, isResumeMismatchError([]byte(stderr)))
	require.False(t, isFatalReceivewalError([]byte(stderr)),
		"a realign fixes this, so it must not escalate the streamer to FAILED")
}

func Test_IsResumeMismatchError_WhenTransientStderr_ReturnsFalse(t *testing.T) {
	transientStderrs := []string{
		"pg_receivewal: error: could not receive data from WAL stream: server closed the connection unexpectedly",
		"pg_receivewal: error: connection to server failed: Connection refused",
		"",
	}

	for _, stderr := range transientStderrs {
		require.False(t, isResumeMismatchError([]byte(stderr)), stderr)
	}
}

// The days-long process is the one the forwarder's fixed local port exists for, so its address has
// to arrive the same way pg_basebackup's does: -h keeps the cluster name, PGHOSTADDR does the work.
func Test_NewReceivewalCommand_WhenTunneled_SendsPgHostAddrAndKeepsTheRealHostFlag(t *testing.T) {
	cmd, err := newReceivewalCommand(t.Context(), receivewalCommandSpec{
		PgBin:    "sh",
		SourceDB: tunneledSourceDatabase(),
		Creds:    &pgshared.CredentialTempFiles{PgpassPath: "/tmp/pgpass"},
		WatchDir: t.TempDir(),
		SlotName: "slot",
	})
	require.NoError(t, err)

	require.Contains(t, cmd.Env, "PGHOSTADDR=127.0.0.1")
	assertHostFlagIsTheRealCluster(t, cmd.Args)
}

func Test_NewReceivewalCommand_WithoutATunnel_SendsNoPgHostAddr(t *testing.T) {
	sourceDB := tunneledSourceDatabase()
	sourceDB.Engine.HostAddr = ""

	cmd, err := newReceivewalCommand(t.Context(), receivewalCommandSpec{
		PgBin:    "sh",
		SourceDB: sourceDB,
		Creds:    &pgshared.CredentialTempFiles{PgpassPath: "/tmp/pgpass"},
		WatchDir: t.TempDir(),
		SlotName: "slot",
	})
	require.NoError(t, err)

	assertNoPgHostAddr(t, cmd.Env)
}

// A rejected password exits pg_receivewal at once. The error reaches its error output only after
// the process has exited, which is how that exit looks when the reader of the output is
// scheduled late, as on a loaded CI runner. A capture that stops reading when the process exits
// loses it and treats the exit as one worth retrying.
func Test_SpawnAndSupervise_WhenReceiverExitsRightAfterAuthenticationFailure_EndsAsFatal(t *testing.T) {
	receiverStub := filepath.Join(t.TempDir(), "pg_receivewal")
	receiverScript := "#!/bin/sh\n" +
		"(sleep 0.2; echo 'pg_receivewal: error: connection to server failed: " +
		"FATAL:  password authentication failed for user \"repl\"' >&2) >/dev/null &\n" +
		"exit 1\n"
	require.NoError(t, os.WriteFile(receiverStub, []byte(receiverScript), 0o700))

	supervisor := &WalStreamSupervisor{
		spec: WalStreamSpec{
			SourceDB: databases.PostgresqlPhysicalConnectionSettings{
				DatabaseID: uuid.New(),
				Engine: pgphysical.Settings{
					Host:     "localhost",
					Port:     5432,
					Username: "repl",
				},
			},
		},
		watchDir:      t.TempDir(),
		slotName:      "slot",
		restartSignal: make(chan struct{}, 1),
	}

	runResult := supervisor.spawnAndSupervise(t.Context(), slog.New(slog.DiscardHandler), receiverStub)

	require.Equal(t, receiverFatal, runResult.Exit)
	require.ErrorContains(t, runResult.FatalErr, "FATAL:  password authentication failed")
}
