package walstreaming_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"

	"databasus-backend/internal/lib/testwait"
)

func getActiveWalSlot(t *testing.T, source *pgx.Conn) string {
	t.Helper()

	var slotName string

	testwait.Until(t, "the source reports an active WAL replication slot", func(ctx context.Context) error {
		return source.QueryRow(ctx, "SELECT slot_name FROM pg_replication_slots WHERE slot_name LIKE 'databasus_slot_%' AND active").
			Scan(&slotName)
	})

	return slotName
}
