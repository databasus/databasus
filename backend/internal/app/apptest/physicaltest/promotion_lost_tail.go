package physicaltest

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest/enginetest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

// RunPromotionAfterLostTail promotes a standby that never received the former primary's last
// commits, which Databasus did archive. Reception moves to the promoted timeline at the fork point
// without a WAL gap, a target after the promotion restores without the lost rows, and a target only
// the lost rows cover is refused.
func RunPromotionAfterLostTail(t *testing.T, image string) {
	t.Helper()

	team := newTeam(t)
	pair := containers.StartPhysicalReplicationPair(t, image)
	proxy := containers.StartPostgresProxy(t, pair.Primary.Endpoint)
	team.createDatabase(t, physicalDatabaseSpec{
		Endpoint: proxy.Endpoint, BackupType: apiclient.PhysicalBackupTypeFULLINCREMENTALWALSTREAM,
		Tunnel: enginetest.NewDisabledSSHTunnelSettings(),
	})
	team.useReplicationOnlyUser(t)
	primaryConn := openConnAt(t, pair.Primary.Host, pair.Primary.Port)
	standbyConn := openConnAt(t, pair.Standby.Host, pair.Standby.Port)
	schema := NewSchemaRecovery(t, primaryConn)
	source := restorableSource{
		Team: team, Image: image, Schema: schema, Sequences: NewSequenceRecovery(t, primaryConn, schema),
	}
	createMarkerTable(t, primaryConn)
	insertMarker(t, primaryConn, markerRow{Phase: "before-full", Payload: "row-in-base-backup"})
	team.enableBackups(t, true)
	chain := team.waitForChain(t, 0)
	source.applySchemaChanges(t, primaryConn)
	insertMarker(t, primaryConn, markerRow{Phase: "before-detach", Payload: "replayed by the standby"})
	waitForMarkerOnStandby(t, standbyConn, "before-detach")
	slotName := team.getSlotName(t)
	createStandbySlot(t, standbyConn, slotName)

	detachStandby(t, standbyConn)
	insertMarker(t, primaryConn, markerRow{Phase: "lost-tail", Payload: "only the former primary has it"})
	lostTarget := targetTimeWithMargin(t, primaryConn)
	throughLSN := currentLSN(t, primaryConn)
	generateWalActivity(t, primaryConn, 32*1024*1024)
	forceWalRotation(t, primaryConn)
	team.waitForReplayableThroughLSN(t, chain, throughLSN)
	team.waitForTargetTimeReplayable(t, lostTarget)
	pair.Primary.StopContainer(t)

	promoteStandby(t, standbyConn)
	proxy.Route(
		containers.PostgresProxyRoute{
			SQLDestination:         pair.Standby.Endpoint,
			ReplicationDestination: pair.Standby.Endpoint,
		},
	)
	proxy.CutConnections()
	team.waitForTimelineHistory(t, 2)
	// Until the promoted node commits, nothing on the line is newer than the fork point, so only the
	// former primary's WAL covers the lost commit.
	source.requireUnreachableTarget(t, lostTarget)

	afterPromotion := source.markRecoveryPoint(t, standbyConn, "after-promotion")
	generateWalActivity(t, standbyConn, 32*1024*1024)
	forceWalRotation(t, standbyConn)
	team.waitForTargetTimeReplayable(t, afterPromotion.Target)
	require.NotContains(t, afterPromotion.Phases, "lost-tail")
	source.restoreRecoveryPoint(t, afterPromotion)
	team.requireNoWalGap(t)
	require.Equal(t, slotName, team.getSlotName(t), "reception resumed without a slot rebuild")
	require.Len(t, team.listBackups(t, "FULL"), 1)
}

// createStandbySlot creates Databasus's slot on the standby, as slot synchronization would, so the
// promoted node keeps the WAL from the fork point for reception.
func createStandbySlot(t *testing.T, standbyConn *pgx.Conn, slotName string) {
	t.Helper()

	_, err := standbyConn.Exec(t.Context(), "SELECT pg_create_physical_replication_slot($1, true)", slotName)
	require.NoError(t, err)
}

// detachStandby stops the standby's replication while it stays in recovery, so the primary's later
// commits never reach it.
func detachStandby(t *testing.T, standbyConn *pgx.Conn) {
	t.Helper()

	for _, statement := range []string{"ALTER SYSTEM SET primary_conninfo = ''", "SELECT pg_reload_conf()"} {
		_, err := standbyConn.Exec(t.Context(), statement)
		require.NoError(t, err)
	}

	testwait.Until(t, "the standby stops receiving WAL", func(ctx context.Context) error {
		var isReceiving bool
		if err := standbyConn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_stat_wal_receiver)").
			Scan(&isReceiving); err != nil {
			return err
		}

		if isReceiving {
			return fmt.Errorf("the standby still receives WAL")
		}

		return nil
	})
}

func (team Team) requireNoWalGap(t *testing.T) {
	t.Helper()

	for _, wal := range team.listBackups(t, "WAL") {
		require.False(t, wal.HasGapBefore, "WAL range %+v reports a gap", wal)
	}
}
