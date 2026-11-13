package physicaltest

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
)

// RunCNPGPlaintextReplicationRefusal checks the documented access rules: the backup role's
// replication connection without TLS is rejected by the source itself.
func RunCNPGPlaintextReplicationRefusal(t *testing.T, spec CNPGManagerSpec) {
	t.Helper()

	cluster := spec.Pool.StartCluster(t, containers.CNPGClusterSpec{Image: spec.ClusterImage})
	config, err := pgconn.ParseConfig(fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable replication=true connect_timeout=5",
		cluster.Endpoint.Host,
		cluster.Endpoint.Port,
		containers.CNPGBackupRole,
		containers.PostgresPassword,
		containers.CNPGDatabase,
	))
	require.NoError(t, err)

	connection, err := pgconn.ConnectConfig(t.Context(), config)
	if connection != nil {
		_ = connection.Close(context.WithoutCancel(t.Context()))
	}

	var serverError *pgconn.PgError
	require.ErrorAs(t, err, &serverError)
	require.Equal(t, "28000", serverError.Code)
	require.Contains(t, serverError.Message, "no encryption")
}
