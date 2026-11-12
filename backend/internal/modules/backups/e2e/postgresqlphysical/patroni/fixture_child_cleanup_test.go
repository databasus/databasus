package patroni_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"

	containers "databasus-backend/internal/lib/testcontainers"
)

type patroniChildCleanup struct {
	Docker    *testcontainers.DockerClient
	FixtureID string
	Directory string
}

func newPatroniChildCleanup(t *testing.T) patroniChildCleanup {
	t.Helper()

	docker, err := testcontainers.NewDockerClientWithOpts(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { _ = docker.Close() })
	cleanup := patroniChildCleanup{
		Docker:    docker,
		FixtureID: uuid.NewString(),
		Directory: filepath.Join(t.TempDir(), "child"),
	}
	require.NoError(t, os.Mkdir(cleanup.Directory, 0o700))
	t.Cleanup(func() { cleanup.remove(t) })

	return cleanup
}

func (cleanup patroniChildCleanup) remove(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), time.Minute)
	defer cancel()
	filters := make(client.Filters)
	filters.Add("label", containers.LabelPatroniFixture+"="+cleanup.FixtureID)
	instances, err := cleanup.Docker.ContainerList(ctx, client.ContainerListOptions{All: true, Filters: filters})
	if assert.NoError(t, err) {
		for _, instance := range instances.Items {
			_, err := cleanup.Docker.ContainerRemove(
				ctx,
				instance.ID,
				client.ContainerRemoveOptions{Force: true, RemoveVolumes: true},
			)
			assert.NoError(t, err)
		}
	}

	networks, err := cleanup.Docker.NetworkList(ctx, client.NetworkListOptions{Filters: filters})
	if assert.NoError(t, err) {
		for _, network := range networks.Items {
			_, err := cleanup.Docker.NetworkRemove(ctx, network.ID, client.NetworkRemoveOptions{})
			assert.NoError(t, err)
		}
	}

	assert.NoError(t, os.RemoveAll(cleanup.Directory))
}
