package cnpg_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
)

func TestCNPGFixturePool(t *testing.T) {
	t.Parallel()

	t.Run("starts another k3s when the first holds its capacity", func(t *testing.T) {
		t.Parallel()

		pool, err := containers.NewCNPGPool(containers.CNPGPoolSpec{Capacity: 1, Logf: t.Logf})
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, pool.Close(context.WithoutCancel(t.Context()))) })

		first := pool.StartCluster(t, containers.CNPGClusterSpec{Image: containers.CNPGPostgres17Image})
		second := pool.StartCluster(t, containers.CNPGClusterSpec{Image: containers.CNPGPostgres17Image})

		require.Equal(t, 2, pool.GetK3sCount())
		require.NotEqual(t, first.K3s.ID, second.K3s.ID)
	})
}
