package physical

import (
	"testing"

	"github.com/stretchr/testify/require"
)

//nolint:tparallel // Each version's server stops before the next version starts, which bounds the memory the servers of one test hold.
func TestWalPosition(t *testing.T) {
	t.Parallel()

	runOnEverySupportedVersion(t, func(t *testing.T, server testServer) {
		t.Helper()

		t.Run("returns a permission sentinel when wal switch is refused", func(t *testing.T) {
			t.Parallel()
			role := server.createLoginRole(t, "")
			inspection := server.openInspection(t, role)
			_, err := inspection.SwitchWal(t.Context())
			require.ErrorIs(t, err, ErrWalSwitchDenied)
		})
		t.Run("keeps a lost connection retryable when switching wal", func(t *testing.T) {
			t.Parallel()
			inspection := server.openInspection(t, getAdminRole())
			require.NoError(t, inspection.Close(t.Context()))
			_, err := inspection.SwitchWal(t.Context())
			require.Error(t, err)
			require.NotErrorIs(t, err, ErrWalSwitchDenied)
		})
		t.Run("returns the new current position after switching wal", func(t *testing.T) {
			t.Parallel()
			inspection := server.openInspection(t, getAdminRole())
			switchedLSN, err := inspection.SwitchWal(t.Context())
			require.NoError(t, err)
			currentLSN, err := inspection.GetCurrentWalLSN(t.Context())
			require.NoError(t, err)
			require.GreaterOrEqual(t, currentLSN, switchedLSN)
		})
	})
}
