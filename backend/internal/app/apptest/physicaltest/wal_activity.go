package physicaltest

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

func generateWalActivity(t *testing.T, conn *pgx.Conn, minBytes uint64) {
	t.Helper()

	_, err := conn.Exec(t.Context(), `CREATE TABLE IF NOT EXISTS wal_activity (payload TEXT)`)
	require.NoError(t, err)
	startLSN := currentLSN(t, conn)
	for currentLSN(t, conn)-startLSN < minBytes {
		_, err := conn.Exec(
			t.Context(),
			`INSERT INTO wal_activity SELECT repeat('x',1024) FROM generate_series(1,1000)`,
		)
		require.NoError(t, err)
	}
	_, err = conn.Exec(t.Context(), `TRUNCATE wal_activity`)
	require.NoError(t, err)
}

func forceWalRotation(t *testing.T, conn *pgx.Conn) {
	t.Helper()

	generateWalActivity(t, conn, 1)
	_, err := conn.Exec(t.Context(), "SELECT pg_switch_wal()")
	require.NoError(t, err)
}

func waitForWalSummaries(t *testing.T, conn *pgx.Conn, throughLSN uint64) {
	t.Helper()

	position := fmt.Sprintf("%X/%X", throughLSN>>32, throughLSN&0xffffffff)
	testwait.Until(t, "WAL summaries cover the parent backup", func(ctx context.Context) error {
		var hasSummary bool
		// The SQL is constant and the formatted WAL position is a bound parameter.
		// nosemgrep: go.lang.security.audit.sqli.pgx-sqli.pgx-sqli
		err := conn.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_available_wal_summaries() WHERE start_lsn <= $1::pg_lsn AND end_lsn >= $1::pg_lsn)`, position).
			Scan(&hasSummary)
		if err != nil {
			return err
		}
		if !hasSummary {
			return fmt.Errorf("no WAL summary covers %s", position)
		}
		return nil
	})
}
