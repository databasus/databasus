package physicaltest

import (
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
)

// patroniCluster drives a Patroni fixture as a FailoverCluster. Its control connection uses the
// superuser through HAProxy.
type patroniCluster struct {
	fixture *containers.Patroni
}

func (cluster patroniCluster) GetEndpoint() containers.Endpoint {
	return cluster.fixture.Endpoint
}

func (cluster patroniCluster) Connect(t *testing.T) *pgx.Conn {
	t.Helper()

	return openConnAt(t, cluster.fixture.Endpoint.Host, cluster.fixture.Endpoint.Port)
}

func (cluster patroniCluster) GetIdentity(t *testing.T) containers.RouteIdentity {
	t.Helper()

	return cluster.fixture.GetIdentity(t)
}

func (cluster patroniCluster) Switchover(t *testing.T) containers.RouteIdentity {
	t.Helper()

	return cluster.fixture.Switchover(t)
}

func (cluster patroniCluster) KillPrimary(t *testing.T) containers.RouteIdentity {
	t.Helper()

	return cluster.fixture.KillPrimary(t)
}

func (cluster patroniCluster) RequireKilledPrimaryGone(t *testing.T, node string) {
	t.Helper()

	state, err := cluster.fixture.GetNode(node).Container.State(t.Context())
	require.NoError(t, err)
	require.False(t, state.Running)
}
