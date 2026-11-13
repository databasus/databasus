package physicaltest

import (
	"testing"

	"github.com/jackc/pgx/v5"

	containers "databasus-backend/internal/lib/testcontainers"
)

// failoverSource is one Databasus database whose source is a failover cluster, with the schema and
// sequence changes the scenarios restore.
type failoverSource struct {
	Team      Team
	Cluster   FailoverCluster
	Image     string
	Schema    SchemaRecovery
	Sequences SequenceRecovery
	Proxy     *containers.PostgresProxy
	SlotName  string
	// prepareRestoredCluster readies a reconstructed cluster to boot outside the source's manager, for
	// sources whose configuration points outside the data directory.
	prepareRestoredCluster func(t *testing.T, target containers.RestoreTarget, output []byte)
	// stopSource takes the whole source out of service before a recovery boots, for managers that can
	// stop every node declaratively, so the recovery proves it needs nothing from the source.
	stopSource func(t *testing.T)
}

func (source failoverSource) connect(t *testing.T) *pgx.Conn {
	t.Helper()

	return source.Cluster.Connect(t)
}

func (source failoverSource) applySchemaChanges(t *testing.T, connection *pgx.Conn) {
	t.Helper()

	source.Schema.CreatePostFullObjects(t, connection)
	source.Sequences.PrepareCalls(t, connection)
	source.Schema.ApplyBetweenIncrementals(t, connection)
	source.Schema.BeginWalChanges(t, connection)
	source.Sequences.ApplyWalChanges(t, connection)
	source.Schema.execute(t, connection, "COMMIT")
	source.Schema.RequireStage(t, connection, SchemaAfterWal)
	source.Sequences.RequireDefinitions(t, connection, SchemaAfterWal)
}

func (source failoverSource) requireRestoredSchema(t *testing.T, connection *pgx.Conn) {
	t.Helper()

	source.Schema.RequireStage(t, connection, SchemaAfterWal)
	source.Sequences.RequireDefinitions(t, connection, SchemaAfterWal)
	source.Sequences.RequireAllocations(t, connection, SchemaAfterWal)
	source.Schema.RequireWrites(t, connection)
	requireMatchingCollationVersions(t, connection)
}

// reconstructForBoot reconstructs a downloaded recovery set and readies it to boot.
func (source failoverSource) reconstructForBoot(
	t *testing.T,
	target containers.RestoreTarget,
	spec restoreSpec,
) []byte {
	t.Helper()

	output := reconstructCluster(t, target, spec)
	if source.prepareRestoredCluster != nil {
		source.prepareRestoredCluster(t, target, output)
	}

	return output
}
