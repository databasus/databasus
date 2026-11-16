package physicaltest

import (
	"testing"

	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
)

// RunCNPGRestrictedAccessAcrossSwitch backs up with the declared role before and after a switch and
// checks that the role gains nothing on the promoted node.
func RunCNPGRestrictedAccessAcrossSwitch(t *testing.T, spec CNPGManagerSpec) {
	t.Helper()

	source := newCNPGSource(t, cnpgSourceSpec{Manager: spec, SlotMode: containers.CNPGSlotSynchronized})
	source.Team.enableBackups(t, true)
	chain := source.Team.waitForChain(t, 0)

	promoted := source.Cluster.Switchover(t)
	connection := source.connect(t)
	source.Team.waitForTimelineHistory(t, promoted.Timeline)
	source.Team.requirePublishedTimelineHistory(t, connection)
	chain = source.Team.buildIncremental(
		t,
		connection,
		incrementalSpec{ParentStopLSN: chainTipStopLSN(t, chain), Count: 1},
	)
	require.Equal(t, promoted.Timeline, chain[len(chain)-1].TimelineID)

	source.Team.requireRestrictedRoleAttributes(t, connection)
	source.Team.requireRestrictedPromotionRole(t)
}
