package physical

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	containers "databasus-backend/internal/lib/testcontainers"
)

// WAL level "minimal" has no scenario: PostgreSQL accepts it only with max_wal_senders=0, which
// refuses the replication connection before the check reads wal_level.
//
//nolint:tparallel // Each version's server stops before the next version starts, which bounds the memory the servers of one test hold.
func TestTestReplicationConnection(t *testing.T) {
	t.Parallel()

	runOnEverySupportedVersion(t, func(t *testing.T, server testServer) {
		t.Helper()

		t.Run("with summarize_wal off", func(t *testing.T) {
			summarizerOffServer := startTestServer(t, server.image, containers.WithoutSummarizer())

			t.Run("returns wal summary disabled code for incremental backups", func(t *testing.T) {
				t.Parallel()

				err := summarizerOffServer.newSource(getAdminRole()).
					TestReplicationConnection(t.Context(), newTestLogger(t), getIncrementalCheck())

				assert.Equal(t, pgshared.ConnErrWalSummaryDisabled, getConnectionTestErrorCode(t, err))
			})

			t.Run("returns wal summary disabled code for wal streaming", func(t *testing.T) {
				t.Parallel()

				err := summarizerOffServer.newSource(getAdminRole()).
					TestReplicationConnection(t.Context(), newTestLogger(t), getWalStreamCheck())

				assert.Equal(t, pgshared.ConnErrWalSummaryDisabled, getConnectionTestErrorCode(t, err))
			})

			t.Run("succeeds for full backups only", func(t *testing.T) {
				t.Parallel()

				err := summarizerOffServer.newSource(getAdminRole()).
					TestReplicationConnection(t.Context(), newTestLogger(t), getFullOnlyCheck())

				assert.NoError(t, err)
			})
		})

		t.Run("returns custom tablespaces code when cluster has a custom tablespace", func(t *testing.T) {
			tablespaceServer := startTestServer(t, server.image, containers.WithTablespace())

			err := tablespaceServer.newSource(getAdminRole()).
				TestReplicationConnection(t.Context(), newTestLogger(t), getFullOnlyCheck())

			assert.Equal(t, pgshared.ConnErrCustomTablespaces, getConnectionTestErrorCode(t, err))
		})

		// A cluster with only "host all all all" in pg_hba accepts an ordinary connection, yet
		// pg_basebackup and pg_receivewal use physical replication, which pg_hba matches only through
		// a "host replication" rule.
		t.Run("returns pg_hba no entry code without a replication pg_hba entry", func(t *testing.T) {
			noReplicationHbaServer := startTestServer(t, server.image, containers.WithoutReplicationHbaEntry())

			err := noReplicationHbaServer.newSource(getAdminRole()).
				TestReplicationConnection(t.Context(), newTestLogger(t), getFullOnlyCheck())

			assert.Equal(t, pgshared.ConnErrPgHbaNoEntry, getConnectionTestErrorCode(t, err))
			assert.ErrorContains(t, err, fmt.Sprintf("host replication %s all scram-sha-256", getAdminRole().Name))
		})

		t.Run("succeeds for every backup type when cluster is replication ready", func(t *testing.T) {
			t.Parallel()

			for _, check := range []ReplicationCheck{getFullOnlyCheck(), getIncrementalCheck(), getWalStreamCheck()} {
				err := server.newSource(getAdminRole()).TestReplicationConnection(t.Context(), newTestLogger(t), check)

				assert.NoError(t, err, "check %+v", check)
			}
		})

		t.Run("returns bad credentials code with wrong password", func(t *testing.T) {
			t.Parallel()

			wrongPasswordRole := testRole{Name: getAdminRole().Name, Password: getAdminRole().Password + "-wrong"}
			err := server.newSource(wrongPasswordRole).
				TestReplicationConnection(t.Context(), newTestLogger(t), getFullOnlyCheck())

			assert.Equal(t, pgshared.ConnErrBadCredentials, getConnectionTestErrorCode(t, err))
		})

		t.Run("returns no replication privilege code when role lacks replication", func(t *testing.T) {
			t.Parallel()
			ordinaryRole := server.createLoginRole(t, "")

			err := server.newSource(ordinaryRole).
				TestReplicationConnection(t.Context(), newTestLogger(t), getFullOnlyCheck())

			assert.Equal(t, pgshared.ConnErrNoReplicationPrivilege, getConnectionTestErrorCode(t, err))
		})

		t.Run("returns no wal switch privilege code for wal streaming when role cannot switch wal", func(t *testing.T) {
			t.Parallel()
			replicationRole := server.createLoginRole(t, "REPLICATION")

			err := server.newSource(replicationRole).
				TestReplicationConnection(t.Context(), newTestLogger(t), getWalStreamCheck())

			assert.Equal(t, pgshared.ConnErrNoWalSwitchPrivilege, getConnectionTestErrorCode(t, err))
			assert.ErrorContains(t, err, "GRANT EXECUTE ON FUNCTION pg_switch_wal()")
		})

		t.Run("succeeds for wal streaming after execute on pg_switch_wal is granted", func(t *testing.T) {
			t.Parallel()
			replicationRole := server.createLoginRole(t, "REPLICATION")
			grantWalSwitch(t, server, replicationRole.Name)

			err := server.newSource(replicationRole).
				TestReplicationConnection(t.Context(), newTestLogger(t), getWalStreamCheck())

			assert.NoError(t, err)
		})

		// FULL and incremental restores replay no archived WAL, so demanding the privilege for them
		// would refuse sources that back up correctly.
		t.Run("succeeds for backups without wal replay when role cannot switch wal", func(t *testing.T) {
			t.Parallel()
			replicationRole := server.createLoginRole(t, "REPLICATION")

			for _, check := range []ReplicationCheck{getFullOnlyCheck(), getIncrementalCheck()} {
				err := server.newSource(replicationRole).
					TestReplicationConnection(t.Context(), newTestLogger(t), check)

				assert.NoError(t, err, "check %+v", check)
			}
		})

		t.Run("succeeds when cluster has the stored system identifier", func(t *testing.T) {
			t.Parallel()
			check := getFullOnlyCheck()
			check.ExpectedSystemIdentifier = server.readSystemIdentifier(t)

			err := server.newSource(getAdminRole()).TestReplicationConnection(t.Context(), newTestLogger(t), check)

			assert.NoError(t, err)
		})

		t.Run("returns system identifier mismatch code when cluster differs from the stored one", func(t *testing.T) {
			t.Parallel()
			check := getFullOnlyCheck()
			check.ExpectedSystemIdentifier = "1"

			err := server.newSource(getAdminRole()).TestReplicationConnection(t.Context(), newTestLogger(t), check)

			assert.Equal(t, pgshared.ConnErrSystemIdentifierMismatch, getConnectionTestErrorCode(t, err))
		})
	})
}

//nolint:tparallel // Each version's server stops before the next version starts, which bounds the memory the servers of one test hold.
func TestReadReplicationSettings(t *testing.T) {
	t.Parallel()

	t.Run("defaults summarize_wal to off when server predates it", func(t *testing.T) {
		server := testServer{endpoint: containers.StartPostgres(t, "postgres:16"), image: "postgres:16", version: ""}

		settings, err := readReplicationSettings(t.Context(), server.connectAs(t, getAdminRole()))

		require.NoError(t, err)
		assert.Equal(t, "off", settings.SummarizeWal)
	})
}
