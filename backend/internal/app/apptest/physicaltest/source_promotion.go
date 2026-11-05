package physicaltest

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

func promoteStandby(t *testing.T, conn *pgx.Conn) {
	t.Helper()

	var isPromoted bool
	require.NoError(
		t,
		conn.QueryRow(t.Context(), `SELECT pg_promote(wait => true, wait_seconds => 60)`).Scan(&isPromoted),
	)
	require.True(t, isPromoted)
	testwait.Until(t, "the standby leaves recovery", func(ctx context.Context) error {
		var isInRecovery bool
		if err := conn.QueryRow(ctx, `SELECT pg_is_in_recovery()`).Scan(&isInRecovery); err != nil {
			return err
		}
		if isInRecovery {
			return fmt.Errorf("standby is still in recovery")
		}
		return nil
	})
}
