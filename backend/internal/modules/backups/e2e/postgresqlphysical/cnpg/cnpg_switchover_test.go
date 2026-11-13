package cnpg_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
)

func TestCNPGFixtureSwitchover(t *testing.T) {
	t.Parallel()

	for _, image := range []string{containers.CNPGPostgres17Image, containers.CNPGPostgres18Image} {
		t.Run("routes to both nodes after two switchovers with "+image, func(t *testing.T) {
			t.Parallel()

			cluster := sharedPool.StartCluster(t, containers.CNPGClusterSpec{Image: image})
			initial := cluster.GetIdentity(t)

			second := cluster.Switchover(t)
			third := cluster.Switchover(t)

			require.NotEqual(t, initial.Node, second.Node)
			require.Equal(t, initial.Node, third.Node)
			require.Greater(t, second.Timeline, initial.Timeline)
			require.Greater(t, third.Timeline, second.Timeline)
			require.Equal(t, initial.SystemIdentifier, third.SystemIdentifier)
		})
	}
}
