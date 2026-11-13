package containers

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// NodeQuery runs as the local postgres user over a PostgreSQL node's Unix socket, which is how the
// harness observes and changes a node without network superuser access.
type NodeQuery struct {
	Node     string
	SQL      string
	Database string
}

// QueryNode returns the unaligned, tuple-only output of the query: one value per line, columns
// separated by "|".
func (cluster *CNPGCluster) QueryNode(t *testing.T, query NodeQuery) string {
	t.Helper()

	output, err := cluster.queryNode(t.Context(), query)
	require.NoError(t, err)

	return output
}

func (cluster *CNPGCluster) queryNode(ctx context.Context, query NodeQuery) (string, error) {
	database := query.Database
	if database == "" {
		database = CNPGDatabase
	}

	output, err := cluster.ExecPostgresNodeCommand(ctx, NodeCommand{
		Node:    query.Node,
		Command: []string{"psql", "-X", "-q", "-A", "-t", "-v", "ON_ERROR_STOP=1", "-d", database, "-c", query.SQL},
	})
	if err != nil {
		return "", fmt.Errorf("query %s: %w", query.Node, err)
	}

	return strings.TrimSpace(output), nil
}
