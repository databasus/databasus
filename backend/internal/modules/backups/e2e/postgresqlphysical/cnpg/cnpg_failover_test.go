package cnpg_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
)

func TestCNPGFixtureFailover(t *testing.T) {
	t.Parallel()

	for _, image := range []string{containers.CNPGPostgres17Image, containers.CNPGPostgres18Image} {
		t.Run("routes to the other node after primary loss with "+image, func(t *testing.T) {
			t.Parallel()

			cluster := sharedPool.StartCluster(t, containers.CNPGClusterSpec{Image: image})
			initial := cluster.GetIdentity(t)

			promoted := cluster.KillPrimary(t)

			require.NotEqual(t, initial.Node, promoted.Node)
			require.Greater(t, promoted.Timeline, initial.Timeline)
			require.Equal(t, initial.SystemIdentifier, promoted.SystemIdentifier)
		})
	}
}
