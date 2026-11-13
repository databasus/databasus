package physicaltest

import (
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
)

// RunCNPGSynchronizedSlot checks that the standby's synchronized copy of Databasus's slot retains
// the WAL Databasus has not received, so reception resumes after the switch without a gap.
func RunCNPGSynchronizedSlot(t *testing.T, spec CNPGManagerSpec) {
	t.Helper()

	source, connection, chain := startCNPGSlotSource(t, spec, containers.CNPGSlotSynchronized)
	retainedPosition := source.blockReception(t, connection)
	insertMarker(
		t,
		connection,
		markerRow{Phase: "unarchived-before-switch", Payload: "retained by the synchronized slot"},
	)
	unarchivedSegment := getCurrentSegment(t, connection)
	forceWalRotation(t, connection)
	source.Fixture.WaitForSlotManagement(t)

	standby := source.Fixture.GetReplica(source.Fixture.GetIdentity(t).Node)
	standbyCopy := source.Fixture.GetSlotState(t, containers.SlotOnNode{Node: standby, Slot: source.SlotName})
	require.True(t, standbyCopy.HasSlot, "%s holds no copy of %s", standby, source.SlotName)
	require.LessOrEqual(
		t,
		parseLSN(t, standbyCopy.RestartLSN),
		parseLSN(t, retainedPosition),
		"the standby's copy must retain the WAL Databasus has not received",
	)
	require.False(t, source.Team.hasArchivedSegment(t, unarchivedSegment))

	promoted := source.Cluster.Switchover(t)
	require.Equal(t, standby, promoted.Node)
	connection = source.connect(t)
	insertMarker(t, connection, markerRow{Phase: "after-synchronized-switch", Payload: "committed on promoted node"})
	source.Proxy.SetReplicationBlocked(false)
	source.Team.waitForArchivedSegment(t, unarchivedSegment)

	source.Team.triggerIncremental(t)
	source.Team.waitForReanchoredFull(t, chain[0].ID, promoted.Timeline)
	replacement := source.waitForNewFull(t, chain[0].ID)
	source.Team.waitForTimelineHistory(t, promoted.Timeline)
	point := source.captureRecoveryPoint(t, []backupRow{replacement}, "after-synchronized-slot-switch")
	source.restoreRecoveryPoint(t, point)
}

// RunCNPGExcludedSlot promotes a standby that never held Databasus's slot.
func RunCNPGExcludedSlot(t *testing.T, spec CNPGManagerSpec) {
	t.Helper()

	source, connection, chain := startCNPGSlotSource(t, spec, containers.CNPGSlotExcluded)
	source.Fixture.WaitForSlotManagement(t)
	source.requirePromotionWithoutSlot(t, connection, chain)
}

// RunCNPGUnsynchronizedSlot promotes a standby whose copy of Databasus's slot CloudNativePG removed
// when synchronization was turned off.
func RunCNPGUnsynchronizedSlot(t *testing.T, spec CNPGManagerSpec) {
	t.Helper()

	source, connection, chain := startCNPGSlotSource(t, spec, containers.CNPGSlotSynchronized)
	source.Fixture.WaitForSlotManagement(t)
	standby := source.Fixture.GetReplica(source.Fixture.GetIdentity(t).Node)
	require.True(t, source.Fixture.GetSlotState(t, containers.SlotOnNode{Node: standby, Slot: source.SlotName}).HasSlot)
	source.Fixture.StopSlotSynchronization(t, source.SlotName)
	source.requirePromotionWithoutSlot(t, connection, chain)
}

// RunCNPGLostSlot lets the promoted node recycle a segment Databasus needed and never received, then
// requires every target inside it to stay refused.
func RunCNPGLostSlot(t *testing.T, spec CNPGManagerSpec) {
	t.Helper()

	source, connection, _ := startCNPGSlotSource(t, spec, containers.CNPGSlotExcluded)
	source.blockReception(t, connection)
	insertMarker(
		t,
		connection,
		markerRow{Phase: "unarchived-after-slot-loss", Payload: "required WAL will be recycled"},
	)
	requiredSegment := getCurrentSegment(t, connection)
	target := targetTimeWithMargin(t, connection)
	insertMarker(t, connection, markerRow{Phase: "after-lost-target", Payload: "beyond the requested target"})
	forceWalRotation(t, connection)

	source.Cluster.Switchover(t)
	source.Fixture.ApplyParameters(t, map[string]string{
		"wal_keep_size": "0",
		"max_wal_size":  "64MB",
		"min_wal_size":  "32MB",
	})
	source.requireRecycledSegmentRefused(
		t,
		source.connect(t),
		recycledSegment{Segment: requiredSegment, Target: target},
	)
}

func startCNPGSlotSource(
	t *testing.T,
	spec CNPGManagerSpec,
	mode containers.CNPGSlotMode,
) (cnpgSource, *pgx.Conn, []backupRow) {
	t.Helper()

	source := newCNPGSource(t, cnpgSourceSpec{Manager: spec, ShouldIntercept: true, SlotMode: mode})
	connection := source.connect(t)
	createMarkerTable(t, connection)
	source.Team.enableBackups(t, true)
	chain := source.Team.waitForChain(t, 0)
	source.applySchemaChanges(t, connection)
	require.Equal(t, source.SlotName, waitForSlotPresent(t, connection))

	return source, connection, chain
}

// Resuming a new slot does not establish continuity of the earlier interval. Databasus must either
// refuse that target or provide a recovery set that reproduces its recorded state.
func (source cnpgSource) requirePromotionWithoutSlot(t *testing.T, connection *pgx.Conn, chain []backupRow) {
	t.Helper()

	source.blockReception(t, connection)
	point := source.markRecoveryPoint(t, connection, "unarchived-before-promotion")
	point.Timeline = chain[0].TimelineID
	pendingSegment := getCurrentSegment(t, connection)
	forceWalRotation(t, connection)
	require.False(t, source.Team.hasArchivedSegment(t, pendingSegment))

	standby := source.Fixture.GetReplica(source.Fixture.GetIdentity(t).Node)
	require.False(
		t,
		source.Fixture.GetSlotState(t, containers.SlotOnNode{Node: standby, Slot: source.SlotName}).HasSlot,
	)
	promoted := source.Cluster.Switchover(t)
	require.Equal(t, standby, promoted.Node)
	source.Proxy.SetReplicationBlocked(false)

	if !source.requireMissingSlotResponse(t) {
		source.requireUnreachableTarget(t, point.Target)

		return
	}

	status, body := source.Team.App.CallPhysicalAPI(t, source.Team.Owner, source.Team.restoreTokenCall(&point.Target))
	switch status {
	case http.StatusCreated:
		source.restoreRecoveryPoint(t, point)
	case http.StatusConflict:
		source.requireUnreachableTarget(t, point.Target)
		t.Log("WAL reception resumed, but the earlier target remains unreachable across the archived WAL gap")
	default:
		t.Fatalf("restore token answered %d: %s", status, body)
	}
}
