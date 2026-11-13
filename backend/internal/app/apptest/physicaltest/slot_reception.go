package physicaltest

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"

	"databasus-backend/internal/lib/testwait"
)

// blockReception stops Databasus's WAL reception at the proxy until the test ends or unblocks it,
// waits until the slot on the connected primary has no consumer and returns the position from which
// the slot retains WAL.
func (source failoverSource) blockReception(t *testing.T, connection *pgx.Conn) string {
	t.Helper()

	source.Proxy.SetReplicationBlocked(true)
	t.Cleanup(func() { source.Proxy.SetReplicationBlocked(false) })
	var retainedPosition string
	testwait.Until(t, "the test disconnects the external slot consumer", func(ctx context.Context) error {
		var isActive bool
		if err := connection.QueryRow(ctx, "SELECT active, restart_lsn::text FROM pg_replication_slots WHERE slot_name=$1", source.SlotName).
			Scan(&isActive, &retainedPosition); err != nil {
			return err
		}

		if isActive {
			return fmt.Errorf("slot %s still has an active consumer", source.SlotName)
		}

		return nil
	})

	return retainedPosition
}
