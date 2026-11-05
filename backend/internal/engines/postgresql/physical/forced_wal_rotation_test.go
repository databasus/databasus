package physical

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:tparallel // Each version's server stops before the next version starts, which bounds the memory the servers of one test hold.
func TestCanForceWalRotation(t *testing.T) {
	t.Parallel()

	runOnEverySupportedVersion(t, func(t *testing.T, server testServer) {
		t.Helper()

		t.Run("returns false for a replication-only role", func(t *testing.T) {
			t.Parallel()
			replicationRole := server.createLoginRole(t, "REPLICATION")

			canRotate, err := canForceWalRotation(
				t.Context(),
				server.connectAs(t, getAdminRole()),
				replicationRole.Name,
			)

			require.NoError(t, err)
			assert.False(t, canRotate)
		})

		t.Run("returns true after execute on pg_switch_wal is granted", func(t *testing.T) {
			t.Parallel()
			replicationRole := server.createLoginRole(t, "REPLICATION")
			grantWalSwitch(t, server, replicationRole.Name)

			canRotate, err := canForceWalRotation(
				t.Context(),
				server.connectAs(t, getAdminRole()),
				replicationRole.Name,
			)

			require.NoError(t, err)
			assert.True(t, canRotate)
		})

		// The connected administrator can always execute pg_switch_wal, so a probe that ignored its
		// role argument would report every source as capable.
		t.Run("answers for the probed role when an administrator probes another role", func(t *testing.T) {
			t.Parallel()
			replicationRole := server.createLoginRole(t, "REPLICATION")
			adminConn := server.connectAs(t, getAdminRole())

			canAdminRotate, err := canForceWalRotation(t.Context(), adminConn, getAdminRole().Name)
			require.NoError(t, err)
			require.True(t, canAdminRotate, "the test administrator must be able to rotate")

			canRoleRotate, err := canForceWalRotation(t.Context(), adminConn, replicationRole.Name)

			require.NoError(t, err)
			assert.False(t, canRoleRotate)
		})
	})
}

func grantWalSwitch(t *testing.T, server testServer, roleName string) {
	t.Helper()

	execStatements(t, server.connectAs(t, getAdminRole()),
		fmt.Sprintf(`GRANT EXECUTE ON FUNCTION pg_switch_wal() TO "%s"`, roleName))
}
