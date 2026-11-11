package containers

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

func (fixture *Patroni) RequireReady(t *testing.T) {
	t.Helper()

	identity := fixture.GetIdentity(t)

	for _, node := range fixture.Nodes {
		connection := fixture.Connect(t, node.SQL)
		testwait.Until(t, "Patroni node has its expected role", func(ctx context.Context) error {
			var isStandby bool
			if err := connection.QueryRow(ctx, "SELECT pg_is_in_recovery()").Scan(&isStandby); err != nil {
				return err
			}

			if isStandby != (node.Name != identity.Node) {
				return fmt.Errorf("%s standby=%v", node.Name, isStandby)
			}

			return nil
		})

		for _, setting := range []string{"fsync", "full_page_writes", "synchronous_commit", "summarize_wal", "data_checksums"} {
			var value string
			require.NoError(t, connection.QueryRow(t.Context(), "SELECT current_setting($1)", setting).Scan(&value))
			require.Equal(t, "on", value, "%s on %s", setting, node.Name)
		}

		var version string
		require.NoError(t, connection.QueryRow(t.Context(), "SELECT version()").Scan(&version))

		t.Logf("%s: %s", node.Name, version)
	}
}
