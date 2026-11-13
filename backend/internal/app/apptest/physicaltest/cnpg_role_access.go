package physicaltest

import (
	"testing"

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
	source.Team.triggerIncremental(t)
	source.Team.waitForReanchoredFull(t, chain[0].ID, promoted.Timeline)
	source.waitForNewFull(t, chain[0].ID)
	connection := source.connect(t)
	source.Team.waitForTimelineHistory(t, promoted.Timeline)
	source.Team.requirePublishedTimelineHistory(t, connection)

	source.Team.requireRestrictedRoleAttributes(t, connection)
	source.Team.requireRestrictedPromotionRole(t)
}
