package physicaltest

import (
	"testing"

	"github.com/jackc/pgx/v5"

	containers "databasus-backend/internal/lib/testcontainers"
)

// cnpgCluster drives a CloudNativePG fixture as a FailoverCluster. Its control connection is the
// database owner over verified TLS; CloudNativePG's superuser stays without network access.
type cnpgCluster struct {
	fixture *containers.CNPGCluster
}

func (cluster cnpgCluster) GetEndpoint() containers.Endpoint {
	return cluster.fixture.Endpoint
}

func (cluster cnpgCluster) Connect(t *testing.T) *pgx.Conn {
	t.Helper()

	return cluster.fixture.Connect(t)
}

func (cluster cnpgCluster) GetIdentity(t *testing.T) containers.RouteIdentity {
	t.Helper()

	return cluster.fixture.GetIdentity(t)
}

func (cluster cnpgCluster) Switchover(t *testing.T) containers.RouteIdentity {
	t.Helper()

	return cluster.fixture.Switchover(t)
}

func (cluster cnpgCluster) KillPrimary(t *testing.T) containers.RouteIdentity {
	t.Helper()

	return cluster.fixture.KillPrimary(t)
}

func (cluster cnpgCluster) RequireKilledPrimaryGone(t *testing.T, node string) {
	t.Helper()

	cluster.fixture.RequireKilledPodGone(t, node)
}
