package physicaltest

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

func queryRestoredMarkerRows(t *testing.T, target containers.RestoreTarget) []string {
	t.Helper()

	return queryMarkerRowsAt(t, target.Host(), target.MappedPort())
}

func queryMarkerRowsAt(t *testing.T, host string, port int) []string {
	t.Helper()

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, restoredPgUser, restoredPgPassword, restoredPgDatabase)

	conn := connectWithRetry(t, dsn)
	defer func() { _ = conn.Close(t.Context()) }()

	rows, err := conn.Query(t.Context(), `SELECT phase FROM restore_marker ORDER BY phase`)
	require.NoError(t, err)
	defer rows.Close()

	var phases []string
	for rows.Next() {
		var phase string
		require.NoError(t, rows.Scan(&phase))
		phases = append(phases, phase)
	}
	require.NoError(t, rows.Err())

	return phases
}

func connectWithRetry(t *testing.T, dsn string) *pgx.Conn {
	t.Helper()

	var conn *pgx.Conn
	testwait.Until(t, "the restored PostgreSQL accepts connections", func(ctx context.Context) error {
		var err error
		conn, err = pgx.Connect(ctx, dsn)
		return err
	})
	return conn
}
