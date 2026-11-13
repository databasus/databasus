package containers

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

// RequireReplay commits a control transaction on the primary and waits until the candidate has
// replayed it, so a switch that follows cannot lose it.
func (cluster *CNPGCluster) RequireReplay(t *testing.T, candidate string) {
	t.Helper()

	primary := cluster.Connect(t)
	_, err := primary.Exec(t.Context(), "CREATE TABLE IF NOT EXISTS cnpg_control (id text PRIMARY KEY)")
	require.NoError(t, err)

	var marker string
	require.NoError(
		t,
		primary.QueryRow(t.Context(), "INSERT INTO cnpg_control VALUES (clock_timestamp()::text) RETURNING id").
			Scan(&marker),
	)

	var position string
	require.NoError(t, primary.QueryRow(t.Context(), "SELECT pg_current_wal_flush_lsn()::text").Scan(&position))

	testwait.Until(t, "candidate replays the committed control transaction", func(ctx context.Context) error {
		hasReplayed, err := cluster.queryNode(ctx, NodeQuery{
			Node: candidate,
			SQL: fmt.Sprintf(
				"SELECT pg_is_in_recovery() AND pg_last_wal_replay_lsn() >= '%s'::pg_lsn AND EXISTS(SELECT 1 FROM cnpg_control WHERE id='%s')",
				position,
				marker,
			),
		})
		if err != nil {
			return err
		}

		if hasReplayed != "t" {
			return fmt.Errorf("candidate %s has not replayed %s", candidate, position)
		}

		return nil
	})
}

func (cluster *CNPGCluster) GetReplica(primary string) string {
	for _, node := range cluster.Nodes {
		if node != primary {
			return node
		}
	}

	panic("CloudNativePG replica unavailable")
}
