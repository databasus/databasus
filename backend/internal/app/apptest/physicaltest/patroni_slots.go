package physicaltest

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
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

	source.Proxy.SetReplicationBlocked(true)
	t.Cleanup(func() { source.Proxy.SetReplicationBlocked(false) })
	var retainedPosition string
	testwait.Until(t, "the test disconnects the external slot consumer", func(ctx context.Context) error {
		var isActive bool
		if err := connection.QueryRow(ctx, "SELECT active, restart_lsn::text FROM pg_replication_slots WHERE slot_name=$1", slot).
			Scan(&isActive, &retainedPosition); err != nil {
			return err
		}

		if isActive {
			return fmt.Errorf("slot %s still has an active consumer", slot)
		}

		return nil
	})

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

func (source patroniSource) requireMissingSlotResponse(t *testing.T) {
	t.Helper()

	connection := source.connect(t)
	hasRecovered := false
	testwait.Until(t, "the missing slot is recovered or diagnosed as broken", func(ctx context.Context) error {
		response, err := source.Team.Owner.Client.GetWalStreamStatisticsWithResponse(
			ctx,
			source.Team.Database.Id,
			&apiclient.GetWalStreamStatisticsParams{Period: apiclient.WalStreamStatisticsPeriodHOUR},
		)
		if err != nil {
			return err
		}

		if response.JSON200 == nil {
			return fmt.Errorf("stream statistics: %d %s", response.StatusCode(), response.Body)
		}

		health := response.JSON200.Health
		for _, reason := range health.Reasons {
			if reason.Code == apiclient.WalStreamReasonCodeSLOTMISSING {
				require.Equal(t, apiclient.WalStreamHealthLevelBROKEN, health.Level)
				t.Logf("missing slot %s is diagnosed as BROKEN/SLOT_MISSING", source.SlotName)
				return nil
			}
		}

		var isActive bool
		if err := connection.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_replication_slots WHERE slot_name=$1 AND active)", source.SlotName).
			Scan(&isActive); err != nil {
			return err
		}

		if isActive {
			hasRecovered = true
			t.Logf("Databasus recovered missing slot %s", source.SlotName)
			return nil
		}

		return fmt.Errorf("missing slot health: %+v", health)
	})

	if hasRecovered {
		var resumedSegment string
		require.NoError(
			t,
			connection.QueryRow(t.Context(), "SELECT pg_walfile_name(pg_current_wal_lsn())").Scan(&resumedSegment),
		)
		generateWalActivity(t, connection, 16*1024*1024)
		forceWalRotation(t, connection)
		testwait.Until(t, "the recovered slot archives new WAL", func(context.Context) error {
			var hasArchived bool
			source.Team.App.QuerySQL(
				t,
				&hasArchived,
				"SELECT EXISTS(SELECT 1 FROM backups_physical_wal_segments WHERE database_id=? AND wal_filename=? AND file_name IS NOT NULL)",
				source.Team.Database.Id,
				resumedSegment,
			)
			if !hasArchived {
				return fmt.Errorf("resumed segment %s is not archived", resumedSegment)
			}

			return nil
		})
	}
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
	var requiredSegment string
	require.NoError(
		t,
		connection.QueryRow(t.Context(), "SELECT pg_walfile_name(pg_current_wal_lsn())").Scan(&requiredSegment),
	)
	target := targetTimeWithMargin(t, connection)
	insertMarker(t, connection, markerRow{Phase: "after-lost-target", Payload: "beyond the requested target"})
	testwait.Until(t, "PostgreSQL recycles the identified required source WAL", func(ctx context.Context) error {
		generateWalActivity(t, connection, 32*1024*1024)
		forceWalRotation(t, connection)
		_, err := connection.Exec(ctx, "CHECKPOINT")
		if err != nil {
			return err
		}

		var hasSegment bool
		if err := connection.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_ls_waldir() WHERE name=$1)", requiredSegment).
			Scan(&hasSegment); err != nil {
			return err
		}

		if hasSegment {
			return fmt.Errorf("required segment %s is still retained", requiredSegment)
		}

		return nil
	})

	var hasArchivedSegment bool
	source.Team.App.QuerySQL(
		t,
		&hasArchivedSegment,
		"SELECT EXISTS(SELECT 1 FROM backups_physical_wal_segments WHERE database_id=? AND wal_filename=? AND file_name IS NOT NULL)",
		source.Team.Database.Id,
		requiredSegment,
	)
	require.False(t, hasArchivedSegment)
	source.requireUnreachableTarget(t, target)
	source.Proxy.SetReplicationBlocked(false)
	source.requireMissingSlotResponse(t)
	source.Team.App.RunJobPass(t, "backups_physical_wal_stream_manager")
	source.requireUnreachableTarget(t, target)
	backups := source.Team.listBackups(t, "FULL")
	require.NotEmpty(t, backups)
	source.Team.call(
		t,
		physicalResponseSpec{
			Call:   apptest.PhysicalStartRequest(source.Team.Database.Id, "FULL"),
			Status: http.StatusAccepted,
		},
	)
	source.waitForNewFull(t, backups[0].ID)
	source.requireUnreachableTarget(t, target)
	t.Logf("required segment %s was recycled without archival; target %s remains refused", requiredSegment, target)
}
