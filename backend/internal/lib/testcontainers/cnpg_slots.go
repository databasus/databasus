package containers

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

// CNPGSlotMode maps a slot scenario to CloudNativePG's synchronization of user slots to standbys.
type CNPGSlotMode string

const (
	// CNPGSlotSynchronized copies the slot to the standby, which the protected and the kept-on-every-node
	// scenarios both need: the promoted node must already hold the slot.
	CNPGSlotSynchronized CNPGSlotMode = "synchronized"
	// CNPGSlotExcluded keeps the slot off the standby through an exact exclusion pattern.
	CNPGSlotExcluded CNPGSlotMode = "excluded"
)

const (
	cnpgManagementProbeSlot = "cnpg_management_probe"
	cnpgManagementCycles    = 3
)

// ApplySlotMode configures synchronization for Databasus's slot through the manifest.
func (cluster *CNPGCluster) ApplySlotMode(t *testing.T, mode CNPGSlotMode, slot string) {
	t.Helper()

	synchronization := map[string]any{"enabled": true, "excludePatterns": nil}
	switch mode {
	case CNPGSlotSynchronized:
	case CNPGSlotExcluded:
		synchronization["excludePatterns"] = []string{"^" + regexp.QuoteMeta(slot) + "$"}
	default:
		t.Fatalf("unknown slot mode %s", mode)
	}

	cluster.patchSlotSynchronization(t, synchronization)
}

// StopSlotSynchronization turns synchronization off and waits until CloudNativePG removed the
// standby's copy of the slot, as it does for every copy it created.
func (cluster *CNPGCluster) StopSlotSynchronization(t *testing.T, slot string) {
	t.Helper()

	cluster.patchSlotSynchronization(t, map[string]any{"enabled": false})
	standby := cluster.GetReplica(cluster.GetIdentity(t).Node)
	testwait.Until(t, "CloudNativePG removes the standby's copy of the slot", func(ctx context.Context) error {
		slotState, err := cluster.getSlotState(ctx, SlotOnNode{Node: standby, Slot: slot})
		if err != nil {
			return err
		}

		if slotState.HasSlot {
			return fmt.Errorf("%s still holds a copy of %s", standby, slot)
		}

		return nil
	})
}

func (cluster *CNPGCluster) patchSlotSynchronization(t *testing.T, synchronization map[string]any) {
	t.Helper()

	patch, err := json.Marshal(map[string]any{"spec": map[string]any{
		"replicationSlots": map[string]any{"synchronizeReplicas": synchronization},
	}})
	require.NoError(t, err)
	cluster.kubectl(t, "patch", "cluster", cnpgClusterName, "--type=merge", "-p", string(patch))
}

// WaitForSlotManagement waits for completed synchronization cycles instead of time: each cycle is
// observed as the standby's copy of a probe slot reaching the position the probe was advanced to on
// the primary. The probe is dropped afterwards so it retains no WAL.
func (cluster *CNPGCluster) WaitForSlotManagement(t *testing.T) {
	t.Helper()

	primary := cluster.GetIdentity(t).Node
	standby := cluster.GetReplica(primary)
	cluster.QueryNode(t, NodeQuery{
		Node: primary,
		SQL:  fmt.Sprintf("SELECT pg_create_physical_replication_slot('%s', true)", cnpgManagementProbeSlot),
	})
	defer cluster.QueryNode(t, NodeQuery{
		Node: primary,
		SQL:  fmt.Sprintf("SELECT pg_drop_replication_slot('%s')", cnpgManagementProbeSlot),
	})

	for range cnpgManagementCycles {
		position := cluster.QueryNode(t, NodeQuery{
			Node: primary,
			SQL: fmt.Sprintf(
				"SELECT pg_switch_wal(); SELECT end_lsn FROM pg_replication_slot_advance('%s', pg_current_wal_insert_lsn())",
				cnpgManagementProbeSlot,
			),
		})
		testwait.Until(t, "CloudNativePG completes a slot synchronization cycle", func(ctx context.Context) error {
			slotState, err := cluster.getSlotState(ctx, SlotOnNode{Node: standby, Slot: cnpgManagementProbeSlot})
			if err != nil {
				return err
			}

			isCaughtUp, err := cluster.queryNode(ctx, NodeQuery{
				Node: standby,
				SQL:  fmt.Sprintf("SELECT '%s'::pg_lsn <= '%s'::pg_lsn", getLastLine(position), slotState.RestartLSN),
			})
			if err != nil || !slotState.HasSlot || slotState.RestartLSN == "" {
				return fmt.Errorf("probe copy %+v has not reached %s: %w", slotState, position, err)
			}

			if isCaughtUp != "t" {
				return fmt.Errorf("probe copy at %s has not reached %s", slotState.RestartLSN, position)
			}

			return nil
		})
	}
}

type SlotState struct {
	HasSlot    bool
	RestartLSN string
	IsActive   bool
}

type SlotOnNode struct {
	Node string
	Slot string
}

// GetSlotState reads a slot on one PostgreSQL node; a missing slot has HasSlot false.
func (cluster *CNPGCluster) GetSlotState(t *testing.T, spec SlotOnNode) SlotState {
	t.Helper()

	slotState, err := cluster.getSlotState(t.Context(), spec)
	require.NoError(t, err)

	return slotState
}

func (cluster *CNPGCluster) getSlotState(ctx context.Context, spec SlotOnNode) (SlotState, error) {
	output, err := cluster.queryNode(ctx, NodeQuery{
		Node: spec.Node,
		SQL: fmt.Sprintf(
			"SELECT COALESCE(restart_lsn::text, '') || '|' || active::text FROM pg_replication_slots WHERE slot_name='%s'",
			spec.Slot,
		),
	})
	if err != nil {
		return SlotState{}, err
	}

	if output == "" {
		return SlotState{}, nil
	}

	restartLSN, activity, _ := strings.Cut(output, "|")
	slotState := SlotState{HasSlot: true, RestartLSN: restartLSN, IsActive: activity == "true"}

	return slotState, nil
}

func getLastLine(output string) string {
	lines := strings.Split(output, "\n")

	return lines[len(lines)-1]
}
