package containers

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

// cnpgSwitchoverPhase is the phase the pinned `kubectl cnpg promote` plugin writes together with the
// target primary (internal/cmd/plugin/promote/promote.go in CloudNativePG 1.30.1). The fixture
// applies the same status patch instead of pinning the plugin binary for every architecture.
const cnpgSwitchoverPhase = "Switchover in progress"

// Switchover moves the primary role to the other PostgreSQL node after it replayed a control
// transaction, and waits until the route serves that node on a newer timeline.
func (cluster *CNPGCluster) Switchover(t *testing.T) RouteIdentity {
	t.Helper()

	previous := cluster.GetIdentity(t)
	cluster.RequireReady(t)
	candidate := cluster.GetReplica(previous.Node)
	cluster.RequireReplay(t, candidate)

	patch, err := json.Marshal(map[string]any{"status": map[string]string{
		"targetPrimary":          candidate,
		"targetPrimaryTimestamp": time.Now().UTC().Format("2006-01-02T15:04:05.000000Z07:00"),
		"phase":                  cnpgSwitchoverPhase,
		"phaseReason":            "Switching over to " + candidate,
	}})
	require.NoError(t, err)
	cluster.kubectl(t, "patch", "cluster", cnpgClusterName, "--subresource=status", "--type=merge", "-p", string(patch))

	current := cluster.waitForPromotion(t, promotion{Previous: previous, Candidate: candidate})
	cluster.RequireReady(t)

	return current
}

type promotion struct {
	Previous  RouteIdentity
	Candidate string
}

func (cluster *CNPGCluster) waitForPromotion(t *testing.T, spec promotion) RouteIdentity {
	t.Helper()

	var current RouteIdentity
	testwait.Until(
		t,
		"CloudNativePG moves the Kubernetes Service to the promoted node",
		func(ctx context.Context) error {
			var err error
			current, err = cluster.readIdentity(ctx, t)
			if err != nil {
				return err
			}

			if current.Node != spec.Candidate || current.Timeline <= spec.Previous.Timeline {
				return fmt.Errorf("previous=%+v current=%+v", spec.Previous, current)
			}

			return nil
		},
	)

	require.Equal(t, spec.Previous.SystemIdentifier, current.SystemIdentifier)

	return current
}
