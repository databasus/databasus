package physicaltest

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

type cnpgRetainedSlotCluster struct {
	FailoverCluster
	source cnpgSource
}

// Blocking all replication here would abort the held BASE_BACKUP before the switch, so the
// retention check observes the live receiver's required position instead.
func (cluster cnpgRetainedSlotCluster) Switchover(t *testing.T) containers.RouteIdentity {
	t.Helper()

	fixture := cluster.source.Fixture
	primary := fixture.GetIdentity(t).Node
	candidate := fixture.GetReplica(primary)
	fixture.WaitForSlotManagement(t)
	connection := cluster.Connect(t)

	var primarySlot containers.SlotState
	var candidateSlot containers.SlotState
	var unarchivedSegment string
	testwait.Until(t, "the candidate slot retains the WAL still required by Databasus", func(context.Context) error {
		unarchivedSegment = getCurrentSegment(t, connection)
		primarySlot = fixture.GetSlotState(t, containers.SlotOnNode{Node: primary, Slot: cluster.source.SlotName})
		candidateSlot = fixture.GetSlotState(t, containers.SlotOnNode{Node: candidate, Slot: cluster.source.SlotName})
		if !primarySlot.HasSlot || primarySlot.RestartLSN == "" ||
			!candidateSlot.HasSlot || candidateSlot.RestartLSN == "" {
			return fmt.Errorf("primary slot %+v, candidate slot %+v", primarySlot, candidateSlot)
		}

		if parseLSN(t, candidateSlot.RestartLSN) > parseLSN(t, primarySlot.RestartLSN) {
			return fmt.Errorf(
				"candidate retains from %s, Databasus still requires %s",
				candidateSlot.RestartLSN,
				primarySlot.RestartLSN,
			)
		}

		if cluster.source.Team.hasArchivedSegment(t, unarchivedSegment) {
			return fmt.Errorf("segment %s completed archival during observation", unarchivedSegment)
		}

		return nil
	})

	require.True(t, candidateSlot.HasSlot)
	require.LessOrEqual(t, parseLSN(t, candidateSlot.RestartLSN), parseLSN(t, primarySlot.RestartLSN))
	t.Logf("candidate %s retains slot %s from %s, covering primary position %s with unarchived segment %s",
		candidate, cluster.source.SlotName, candidateSlot.RestartLSN, primarySlot.RestartLSN, unarchivedSegment)

	promoted := cluster.FailoverCluster.Switchover(t)

	require.Equal(t, candidate, promoted.Node)

	return promoted
}
