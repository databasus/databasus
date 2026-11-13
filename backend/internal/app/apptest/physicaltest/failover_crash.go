package physicaltest

import "testing"

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
	source.Team.triggerIncremental(t)
	source.Team.waitForReanchoredFull(t, chain[0].ID, promoted.Timeline)
	replacement := source.waitForNewFull(t, chain[0].ID)

	source.requireSingleReplacement(t, fullReplacement{Preceding: chain[0].ID, Replacement: replacement.ID})
	connection = source.connect(t)
	source.Team.waitForTimelineHistory(t, promoted.Timeline)
	source.Team.requirePublishedTimelineHistory(t, connection)
	source.Team.requireRestrictedPromotionRole(t)
	point := source.captureRecoveryPoint(t, []backupRow{replacement}, "after-crash-replacement-full")
	if source.stopSource != nil {
		source.stopSource(t)
	}

	source.restoreRecoveryPoint(t, point)

	source.Cluster.RequireKilledPrimaryGone(t, previous.Node)
}
