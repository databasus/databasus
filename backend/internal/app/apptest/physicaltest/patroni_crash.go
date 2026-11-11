package physicaltest

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func RunPatroniPrimaryLoss(t *testing.T, image string) {
	t.Helper()

	source := newPatroniSource(t, patroniSourceSpec{Image: image, ShouldIgnoreSlot: true})
	connection := source.connect(t)
	createMarkerTable(t, connection)
	insertMarker(t, connection, markerRow{Phase: "before-primary-loss", Payload: "confirmed replay before SIGKILL"})
	source.Team.enableBackups(t, true)
	chain := source.Team.waitForChain(t, 0)
	source.applySchemaChanges(t, connection)
	previous := source.Fixture.GetIdentity(t)

	promoted := source.Fixture.KillPrimary(t)
	source.Team.triggerIncremental(t)
	source.Team.waitForReanchoredFull(t, chain[0].ID, promoted.Timeline)
	replacement := source.waitForNewFull(t, chain[0].ID)

	source.requireSingleReplacement(t, patroniReplacement{Preceding: chain[0].ID, Replacement: replacement.ID})
	connection = source.connect(t)
	source.Team.waitForTimelineHistory(t, promoted.Timeline)
	source.Team.requirePublishedTimelineHistory(t, connection)
	source.Team.requireRestrictedPromotionRole(t)
	point := source.captureRecoveryPoint(t, []backupRow{replacement}, "after-crash-replacement-full")
	source.restoreRecoveryPoint(t, point)

	state, err := source.Fixture.GetNode(previous.Node).Container.State(t.Context())
	require.NoError(t, err)
	require.False(t, state.Running)
}
