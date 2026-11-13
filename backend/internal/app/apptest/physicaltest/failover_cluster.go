package physicaltest

import (
	"log/slog"
	"testing"

	"github.com/jackc/pgx/v5"

	containers "databasus-backend/internal/lib/testcontainers"
)

// FailoverCluster is the role a highly available cluster plays in the failover scenarios: a stable
// address whose manager moves the primary role between two PostgreSQL nodes. A planned switch and an
// abrupt primary loss each confirm first that the future primary replayed a control transaction.
type FailoverCluster interface {
	GetEndpoint() containers.Endpoint
	// Connect opens a connection to the routed primary for the harness's control rows and schema
	// changes; Databasus never uses it.
	Connect(t *testing.T) *pgx.Conn
	GetIdentity(t *testing.T) containers.RouteIdentity
	Switchover(t *testing.T) containers.RouteIdentity
	KillPrimary(t *testing.T) containers.RouteIdentity
	// RequireKilledPrimaryGone checks that the node KillPrimary removed is not running as it was.
	RequireKilledPrimaryGone(t *testing.T, node string)
}

// FailoverManager starts sources whose primary a cluster manager selects. Each suite passes its own.
type FailoverManager struct {
	startSource func(t *testing.T, spec failoverSourceSpec) failoverSource
}

type failoverSourceSpec struct {
	ShouldIntercept           bool
	ShouldKeepSlotOnEveryNode bool
	LogSink                   slog.Handler
}
