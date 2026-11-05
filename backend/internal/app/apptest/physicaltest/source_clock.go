package physicaltest

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

func sourceTime(t *testing.T, conn *pgx.Conn) time.Time {
	t.Helper()

	var timestamp time.Time
	require.NoError(t, conn.QueryRow(t.Context(), "SELECT clock_timestamp()").Scan(&timestamp))
	return timestamp.UTC()
}

func waitForSourceTime(t *testing.T, conn *pgx.Conn, timestamp time.Time) {
	t.Helper()

	testwait.Until(t, "the source clock passes the PITR margin", func(context.Context) error {
		if sourceTime(t, conn).Before(timestamp) {
			return fmt.Errorf("source clock has not reached %s", timestamp)
		}
		return nil
	})
}

func targetTimeWithMargin(t *testing.T, conn *pgx.Conn) time.Time {
	t.Helper()

	waitForSourceTime(t, conn, sourceTime(t, conn).Add(2*time.Second))
	targetTime := sourceTime(t, conn)
	waitForSourceTime(t, conn, targetTime.Add(2*time.Second))
	return targetTime
}
