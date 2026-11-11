package containers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

func (fixture *Patroni) IgnoreSlot(t *testing.T, name string) {
	t.Helper()

	body, err := json.Marshal(map[string]any{"ignore_slots": []map[string]string{{"name": name, "type": "physical"}}})
	require.NoError(t, err)

	fixture.request(
		t,
		patroniRequest{
			Endpoint: fixture.GetNode(fixture.GetIdentity(t).Node).API,
			Method:   http.MethodPatch,
			Path:     "/config",
			Body:     body,
		},
	)
}

func (fixture *Patroni) ReconcileSlots(t *testing.T) {
	t.Helper()

	primary := fixture.GetNode(fixture.GetIdentity(t).Node)

	fixture.request(
		t,
		patroniRequest{
			Endpoint: primary.API,
			Method:   http.MethodPatch,
			Path:     "/config",
			Body:     []byte(`{"pause":true}`),
		},
	)

	fixture.RequireCycles(t, primary, 1)

	fixture.request(
		t,
		patroniRequest{
			Endpoint: primary.API,
			Method:   http.MethodPatch,
			Path:     "/config",
			Body:     []byte(`{"pause":false}`),
		},
	)

	fixture.RequireCycles(t, primary, 3)
}

func (fixture *Patroni) KeepSlotOnAllNodes(t *testing.T, name string) {
	t.Helper()

	body, err := json.Marshal(map[string]any{"slots": map[string]any{name: map[string]string{"type": "physical"}}})
	require.NoError(t, err)

	fixture.request(
		t,
		patroniRequest{
			Endpoint: fixture.GetNode(fixture.GetIdentity(t).Node).API,
			Method:   http.MethodPatch,
			Path:     "/config",
			Body:     body,
		},
	)

	for _, node := range fixture.Nodes {
		connection := fixture.Connect(t, node.SQL)
		testwait.Until(
			t,
			"Patroni creates the configured permanent slot on each node",
			func(ctx context.Context) error {
				var hasSlot bool
				if err := connection.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_replication_slots WHERE slot_name=$1 AND slot_type='physical')", name).
					Scan(&hasSlot); err != nil {
					return err
				}

				if !hasSlot {
					return fmt.Errorf("%s has no permanent slot %s", node.Name, name)
				}

				return nil
			},
		)
	}
}
