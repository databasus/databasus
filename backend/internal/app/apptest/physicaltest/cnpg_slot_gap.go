package physicaltest

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

func RunCNPGLostSlotGap(t *testing.T, spec CNPGManagerSpec) {
	t.Helper()

	source, connection, chain := startCNPGSlotSource(t, spec, containers.CNPGSlotExcluded)
	beforeGap := source.captureRecoveryPoint(t, chain, "before-slot-gap")
	timeline := source.Cluster.GetIdentity(t).Timeline
	require.Equal(t, chain[0].TimelineID, timeline)
	source.Fixture.ApplyParameters(
		t,
		map[string]string{"wal_keep_size": "0", "max_wal_size": "64MB", "min_wal_size": "32MB"},
	)
	source.blockReception(t, connection)
	primary := source.Fixture.GetIdentity(t).Node
	source.Fixture.QueryNode(
		t,
		containers.NodeQuery{Node: primary, SQL: fmt.Sprintf("SELECT pg_drop_replication_slot('%s')", source.SlotName)},
	)

	requiredSegment := getCurrentSegment(t, connection)
	gap := source.markRecoveryPoint(t, connection, "inside-slot-gap")
	require.Less(t, chainTipStopLSN(t, chain), currentLSN(t, connection))
	forceWalRotation(t, connection)
	source.waitForRecycledSegment(t, connection, requiredSegment)
	require.False(t, source.Team.hasArchivedSegment(t, requiredSegment))
	source.requireUnreachableTarget(t, gap.Target)

	source.Fixture.QueryNode(t, containers.NodeQuery{Node: primary, SQL: fmt.Sprintf(
		"SELECT pg_create_physical_replication_slot('%s',true) WHERE NOT EXISTS (SELECT 1 FROM pg_replication_slots WHERE slot_name='%s')",
		source.SlotName,
		source.SlotName,
	)})
	source.Proxy.SetReplicationBlocked(false)
	source.waitForSlotConsumer(t, connection)
	resumedSegment := getCurrentSegment(t, connection)
	insertMarker(t, connection, markerRow{Phase: "after-slot-gap", Payload: "committed beyond the refused target"})
	generateWalActivity(t, connection, 32*1024*1024)
	forceWalRotation(t, connection)
	source.Team.waitForArchivedSegment(t, resumedSegment)
	testwait.Until(t, "archived transactions extend beyond the target on its timeline", func(context.Context) error {
		var hasLaterArchive bool
		source.Team.App.QuerySQL(
			t,
			&hasLaterArchive,
			"SELECT EXISTS(SELECT 1 FROM backups_physical_wal_segments WHERE database_id=? AND timeline_id=? AND file_name IS NOT NULL AND last_transaction_at>?)",
			source.Team.Database.Id,
			timeline,
			gap.Target,
		)
		if !hasLaterArchive {
			return fmt.Errorf("no archived transaction beyond %s on timeline %d", gap.Target, timeline)
		}

		return nil
	})

	require.Equal(t, timeline, source.Cluster.GetIdentity(t).Timeline)
	require.False(t, source.Team.hasArchivedSegment(t, requiredSegment))
	source.Team.restoreToken(t, &beforeGap.Target)
	source.requireWalGapRefused(t, gap.Target)
	source.Team.call(
		t,
		physicalResponseSpec{
			Call:   apptest.PhysicalStartRequest(source.Team.Database.Id, "FULL"),
			Status: http.StatusAccepted,
		},
	)
	source.waitForNewFull(t, chain[0].ID)
	source.Team.restoreToken(t, &beforeGap.Target)
	source.requireWalGapRefused(t, gap.Target)
}

func (source cnpgSource) waitForSlotConsumer(t *testing.T, connection *pgx.Conn) {
	t.Helper()

	testwait.Until(t, "Databasus consumes the recreated slot", func(ctx context.Context) error {
		source.Team.App.RunJobPass(t, "backups_physical_wal_stream_manager")

		var isActive bool
		if err := connection.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_replication_slots WHERE slot_name=$1 AND active)", source.SlotName).
			Scan(&isActive); err != nil {
			return err
		}

		if !isActive {
			return fmt.Errorf("slot %s has no consumer", source.SlotName)
		}

		return nil
	})
}

func (source cnpgSource) requireWalGapRefused(t *testing.T, target time.Time) {
	t.Helper()

	var problem apiclient.Problem
	source.Team.call(
		t,
		physicalResponseSpec{
			Call:        source.Team.restoreTokenCall(&target),
			Status:      http.StatusConflict,
			Destination: &problem,
		},
	)
	require.Equal(t, apiclient.ProblemCodeTargetTimeUnreachable, problem.Code)
	require.Contains(t, problem.Detail, "WAL gap before target")
}
