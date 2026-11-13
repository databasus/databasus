package physicaltest

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

type PatroniSlotSpec struct {
	Image string
	Mode  string
}

func RunPatroniSlot(t *testing.T, spec PatroniSlotSpec) {
	t.Helper()

	isProtected := spec.Mode == "protected" || spec.Mode == "absent"
	source := newPatroniSource(
		t,
		patroniSourceSpec{Image: spec.Image, ShouldIntercept: true, ShouldIgnoreSlot: isProtected},
	)
	connection := source.connect(t)
	createMarkerTable(t, connection)
	source.Team.enableBackups(t, true)
	chain := source.Team.waitForChain(t, 0)
	source.applySchemaChanges(t, connection)
	slot := waitForSlotPresent(t, connection)
	require.Equal(t, source.SlotName, slot)

	if spec.Mode == "absent" {
		candidate := source.Fixture.GetReplica(source.Fixture.GetIdentity(t).Node)
		replica := source.Fixture.Connect(t, candidate.SQL)
		requireSlotAbsent(t, replica, slot)
		source.Fixture.Switchover(t)
		source.requireMissingSlotResponse(t)

		return
	}

	retainedPosition := source.blockReception(t, connection)

	source.Fixture.ReconcileSlots(t)

	if spec.Mode == "protected" {
		var position string
		require.NoError(
			t,
			connection.QueryRow(t.Context(), "SELECT restart_lsn::text FROM pg_replication_slots WHERE slot_name=$1 AND NOT active", slot).
				Scan(&position),
		)
		require.Equal(t, retainedPosition, position)

		generateWalActivity(t, connection, 16*1024*1024)
		source.Proxy.SetReplicationBlocked(false)
		testwait.Until(t, "the protected external slot resumes and advances", func(ctx context.Context) error {
			var hasAdvanced bool
			if err := connection.QueryRow(ctx, "SELECT active AND restart_lsn > $2::pg_lsn FROM pg_replication_slots WHERE slot_name=$1", slot, retainedPosition).
				Scan(&hasAdvanced); err != nil {
				return err
			}

			if !hasAdvanced {
				forceWalRotation(t, connection)
				return fmt.Errorf("slot %s has not advanced past %s", slot, retainedPosition)
			}

			return nil
		})

		point := source.captureRecoveryPoint(t, chain, "after-protected-slot-reconnect")
		source.restoreRecoveryPoint(t, point)

		return
	}

	testwait.Until(t, "Patroni deletes the inactive unprotected external slot", func(ctx context.Context) error {
		var hasSlot bool
		if err := connection.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_replication_slots WHERE slot_name=$1)", slot).
			Scan(&hasSlot); err != nil {
			return err
		}

		if hasSlot {
			return fmt.Errorf("slot %s remains", slot)
		}

		return nil
	})

	if spec.Mode == "lost" {
		source.requireRecycledWalRefused(t, connection)
	}

	source.Proxy.SetReplicationBlocked(false)
	source.requireMissingSlotResponse(t)
}

func requireSlotAbsent(t *testing.T, connection *pgx.Conn, slot string) {
	t.Helper()

	var hasSlot bool
	require.NoError(
		t,
		connection.QueryRow(t.Context(), "SELECT EXISTS(SELECT 1 FROM pg_replication_slots WHERE slot_name=$1)", slot).
			Scan(&hasSlot),
	)
	require.False(t, hasSlot)
}

func (source patroniSource) requireRecycledWalRefused(t *testing.T, connection *pgx.Conn) {
	t.Helper()

	for _, statement := range []string{"ALTER SYSTEM SET wal_keep_size = '0'", "ALTER SYSTEM SET max_wal_size = '64MB'", "ALTER SYSTEM SET min_wal_size = '32MB'", "SELECT pg_reload_conf()"} {
		_, err := connection.Exec(t.Context(), statement)
		require.NoError(t, err)
	}

	insertMarker(
		t,
		connection,
		markerRow{Phase: "unarchived-after-slot-loss", Payload: "required WAL will be recycled"},
	)
	requiredSegment := getCurrentSegment(t, connection)
	target := targetTimeWithMargin(t, connection)
	insertMarker(t, connection, markerRow{Phase: "after-lost-target", Payload: "beyond the requested target"})
	source.requireRecycledSegmentRefused(t, connection, recycledSegment{Segment: requiredSegment, Target: target})
}
