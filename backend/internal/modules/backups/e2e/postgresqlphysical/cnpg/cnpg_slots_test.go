package cnpg_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
)

func TestCNPGFixtureSlots(t *testing.T) {
	t.Parallel()

	for _, image := range []string{containers.CNPGPostgres17Image, containers.CNPGPostgres18Image} {
		t.Run("copies, excludes and removes a standby slot with "+image, func(t *testing.T) {
			t.Parallel()

			cluster := sharedPool.StartCluster(t, containers.CNPGClusterSpec{Image: image})
			primary := cluster.GetIdentity(t).Node
			standby := cluster.GetReplica(primary)
			for _, slot := range []string{"databasus_slot_copied", "databasus_slot_excluded"} {
				cluster.QueryNode(t, containers.NodeQuery{
					Node: primary,
					SQL:  "SELECT pg_create_physical_replication_slot('" + slot + "', true)",
				})
			}

			cluster.ApplySlotMode(t, containers.CNPGSlotExcluded, "databasus_slot_excluded")
			cluster.WaitForSlotManagement(t)

			copied := cluster.GetSlotState(t, containers.SlotOnNode{Node: standby, Slot: "databasus_slot_copied"})
			require.True(t, copied.HasSlot)
			require.NotEmpty(t, copied.RestartLSN)
			require.False(
				t,
				cluster.GetSlotState(t, containers.SlotOnNode{Node: standby, Slot: "databasus_slot_excluded"}).HasSlot,
			)

			cluster.StopSlotSynchronization(t, "databasus_slot_copied")

			require.False(
				t,
				cluster.GetSlotState(t, containers.SlotOnNode{Node: standby, Slot: "databasus_slot_copied"}).HasSlot,
			)
			require.True(
				t,
				cluster.GetSlotState(t, containers.SlotOnNode{Node: primary, Slot: "databasus_slot_copied"}).HasSlot,
			)
		})
	}
}
