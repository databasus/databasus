package containers

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

func (fixture *Patroni) RequireReplay(t *testing.T, candidate PatroniNode) {
	t.Helper()

	primary := fixture.Connect(t, fixture.Endpoint)
	_, err := primary.Exec(t.Context(), "CREATE TABLE IF NOT EXISTS patroni_control (id text PRIMARY KEY)")
	require.NoError(t, err)

	var marker string
	require.NoError(
		t,
		primary.QueryRow(t.Context(), "INSERT INTO patroni_control VALUES (clock_timestamp()::text) RETURNING id").
			Scan(&marker),
	)

	var position string
	require.NoError(t, primary.QueryRow(t.Context(), "SELECT pg_current_wal_flush_lsn()::text").Scan(&position))

	replica := fixture.Connect(t, candidate.SQL)
	testwait.Until(t, "candidate replays the committed control transaction", func(ctx context.Context) error {
		var hasReplayed bool
		err := replica.QueryRow(ctx, "SELECT pg_is_in_recovery() AND pg_last_wal_replay_lsn() >= $1::pg_lsn AND EXISTS(SELECT 1 FROM patroni_control WHERE id=$2)", position, marker).
			Scan(&hasReplayed)
		if err != nil {
			return err
		}

		if !hasReplayed {
			return fmt.Errorf("candidate %s has not replayed %s", candidate.Name, position)
		}

		return nil
	})
}
