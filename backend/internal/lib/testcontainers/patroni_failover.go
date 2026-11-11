package containers

import (
	"context"
	"fmt"
	"testing"

	"github.com/moby/moby/client"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"

	"databasus-backend/internal/lib/testwait"
)

func (fixture *Patroni) KillPrimary(t *testing.T) PatroniIdentity {
	t.Helper()

	previous := fixture.GetIdentity(t)
	fixture.RequireReady(t)
	fixture.RequireReplay(t, fixture.GetReplica(previous.Node))

	fixture.KillNode(t, previous.Node)

	var current PatroniIdentity
	testwait.Until(t, "Patroni elects a primary after abrupt whole-node loss", func(context.Context) error {
		current = fixture.GetIdentity(t)
		if current.Node == previous.Node || current.Timeline <= previous.Timeline {
			return fmt.Errorf("previous=%+v current=%+v", previous, current)
		}

		return nil
	})

	require.Equal(t, previous.SystemIdentifier, current.SystemIdentifier)

	return current
}

func (fixture *Patroni) KillNode(t *testing.T, name string) {
	t.Helper()

	docker, err := testcontainers.NewDockerClientWithOpts(t.Context())
	require.NoError(t, err)
	defer func() { _ = docker.Close() }()

	_, err = docker.ContainerKill(
		t.Context(),
		fixture.GetNode(name).Container.GetContainerID(),
		client.ContainerKillOptions{Signal: "SIGKILL"},
	)
	require.NoError(t, err)

	state, err := fixture.GetNode(name).Container.State(t.Context())
	require.NoError(t, err)
	require.False(t, state.Running)
}
