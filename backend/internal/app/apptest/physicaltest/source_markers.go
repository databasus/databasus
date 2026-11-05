package physicaltest

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

type markerRow struct {
	Phase   string
	Payload string
}

func createMarkerTable(t *testing.T, conn *pgx.Conn) {
	t.Helper()

	_, err := conn.Exec(t.Context(), `CREATE TABLE restore_marker (phase TEXT PRIMARY KEY, payload TEXT NOT NULL)`)
	require.NoError(t, err)
}

func insertMarker(t *testing.T, conn *pgx.Conn, marker markerRow) {
	t.Helper()

	_, err := conn.Exec(t.Context(), `INSERT INTO restore_marker VALUES ($1, $2)`, marker.Phase, marker.Payload)
	require.NoError(t, err)
}

func waitForMarkerOnStandby(t *testing.T, conn *pgx.Conn, phase string) {
	t.Helper()

	testwait.Until(t, "the marker replicates to the standby", func(ctx context.Context) error {
		var hasMarker bool
		err := conn.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM restore_marker WHERE phase = $1)`, phase).
			Scan(&hasMarker)
		if err != nil {
			return err
		}
		if !hasMarker {
			return fmt.Errorf("marker %q is absent", phase)
		}
		return nil
	})
}
