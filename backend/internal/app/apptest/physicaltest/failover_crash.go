package physicaltest

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func RunFailoverPrimaryLoss(t *testing.T, manager FailoverManager) {
	t.Helper()

	source := manager.startSource(t, failoverSourceSpec{})
	connection := source.connect(t)
	createMarkerTable(t, connection)
	insertMarker(t, connection, markerRow{Phase: "before-primary-loss", Payload: "confirmed replay before SIGKILL"})
	source.Team.enableBackups(t, true)
	chain := source.Team.waitForChain(t, 0)
	source.applySchemaChanges(t, connection)
	previous := source.Cluster.GetIdentity(t)

	promoted := source.Cluster.KillPrimary(t)
	connection = source.connect(t)
	source.Team.waitForTimelineHistory(t, promoted.Timeline)
	source.Team.requirePublishedTimelineHistory(t, connection)
	source.Team.requireRestrictedPromotionRole(t)
	point := source.markReplayablePoint(t, connection, "after-primary-loss")
	require.Equal(
		t,
		[]backupRow{chain[0]},
		source.Team.listBackups(t, "FULL"),
		"the chain crosses the failover without a replacement FULL",
	)
	if source.stopSource != nil {
		source.stopSource(t)
	}

	source.restoreRecoveryPoint(t, point)

	source.Cluster.RequireKilledPrimaryGone(t, previous.Node)
}
