package containers

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

// ApplyParameters changes PostgreSQL parameters through the Cluster manifest, the only way
// CloudNativePG accepts them, and waits until the current primary runs with every value as written.
func (cluster *CNPGCluster) ApplyParameters(t *testing.T, parameters map[string]string) {
	t.Helper()

	patch, err := json.Marshal(map[string]any{"spec": map[string]any{
		"postgresql": map[string]any{"parameters": parameters},
	}})
	require.NoError(t, err)
	cluster.kubectl(t, "patch", "cluster", cnpgClusterName, "--type=merge", "-p", string(patch))

	primary := cluster.GetIdentity(t).Node
	testwait.Until(t, "the primary runs with the changed parameters", func(ctx context.Context) error {
		for name, value := range parameters {
			current, err := cluster.queryNode(ctx, NodeQuery{Node: primary, SQL: "SHOW " + name})
			if err != nil {
				return err
			}

			if current != value {
				return fmt.Errorf("%s is %s, not %s", name, current, value)
			}
		}

		return nil
	})
}
