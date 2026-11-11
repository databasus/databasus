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

func (fixture *Patroni) Switchover(t *testing.T) PatroniIdentity {
	t.Helper()

	previous := fixture.GetIdentity(t)
	candidate := fixture.GetReplica(previous.Node)
	fixture.RequireReplay(t, candidate)

	request, err := json.Marshal(map[string]string{"leader": previous.Node, "candidate": candidate.Name})
	require.NoError(t, err)

	leader := fixture.GetNode(previous.Node)
	fixture.request(
		t,
		patroniRequest{Endpoint: leader.API, Method: http.MethodPost, Path: "/switchover", Body: request},
	)

	var current PatroniIdentity
	testwait.Until(t, "Patroni switches the stable endpoint", func(context.Context) error {
		current = fixture.GetIdentity(t)
		if current.Node != candidate.Name || current.Timeline <= previous.Timeline {
			return fmt.Errorf("previous=%+v current=%+v", previous, current)
		}

		return nil
	})

	require.Equal(t, previous.SystemIdentifier, current.SystemIdentifier)
	fixture.RequireReady(t)

	return current
}
