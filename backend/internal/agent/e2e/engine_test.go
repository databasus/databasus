package e2e_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/require"

	agent_container "databasus-backend/internal/agent/container"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

func newEngine(t *testing.T, agentID uuid.UUID) *client.Client {
	t.Helper()
	options := []client.Opt{client.FromEnv}
	if host := getEngineHost(); host != "" {
		options = append(options, client.WithHost(host))
	}

	engine, err := client.New(options...)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, engine.Close()) })
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), time.Minute)
		defer cancel()

		containers, err := engine.ContainerList(
			ctx,
			client.ContainerListOptions{All: true, Filters: agentFilter(agentID)},
		)
		require.NoError(t, err)
		for _, instance := range containers.Items {
			_, err := engine.ContainerRemove(
				ctx,
				instance.ID,
				client.ContainerRemoveOptions{Force: true, RemoveVolumes: true},
			)
			require.NoError(t, err)
		}

		networks, err := engine.NetworkList(ctx, client.NetworkListOptions{Filters: agentFilter(agentID)})
		require.NoError(t, err)
		for _, network := range networks.Items {
			_, err := engine.NetworkRemove(ctx, network.ID, client.NetworkRemoveOptions{})
			require.NoError(t, err)
		}
	})

	return engine
}

func agentFilter(agentID uuid.UUID) client.Filters {
	filters := make(client.Filters)
	filters.Add("label", agent_container.LabelAgentID+"="+agentID.String())
	return filters
}

func requireNoLeftovers(t *testing.T, engine *client.Client, agentID uuid.UUID) {
	t.Helper()
	testwait.Until(t, "the agent removes its containers and networks", func(ctx context.Context) error {
		containers, err := engine.ContainerList(
			ctx,
			client.ContainerListOptions{All: true, Filters: agentFilter(agentID)},
		)
		if err != nil {
			return err
		}
		networks, err := engine.NetworkList(ctx, client.NetworkListOptions{Filters: agentFilter(agentID)})
		if err != nil {
			return err
		}
		if len(containers.Items)+len(networks.Items) > 0 {
			return fmt.Errorf("%d containers and %d networks remain", len(containers.Items), len(networks.Items))
		}
		return nil
	})
}

func createNetwork(t *testing.T, engine *client.Client, agentID, verificationID uuid.UUID) string {
	t.Helper()
	var networkID string

	err := containers.WithFreeSubnet(func(ipam *network.IPAM) error {
		created, err := engine.NetworkCreate(
			t.Context(),
			"databasus-verif-"+verificationID.String(),
			client.NetworkCreateOptions{
				Driver: "bridge",
				IPAM:   ipam,
				Labels: map[string]string{
					agent_container.LabelAgentID:        agentID.String(),
					agent_container.LabelVerificationID: verificationID.String(),
				},
			},
		)
		networkID = created.ID

		return err
	})
	require.NoError(t, err)

	return networkID
}

func readContainerStatus(t *testing.T, engine *client.Client, containerID string) string {
	t.Helper()
	execution, err := engine.ExecCreate(
		t.Context(),
		containerID,
		client.ExecCreateOptions{Cmd: []string{"cat", "/proc/1/status"}, AttachStdout: true, AttachStderr: true},
	)
	require.NoError(t, err)

	attached, err := engine.ExecAttach(t.Context(), execution.ID, client.ExecAttachOptions{})
	require.NoError(t, err)
	defer attached.Close()

	var contents bytes.Buffer
	_, err = stdcopy.StdCopy(&contents, &contents, attached.Reader)
	require.NoError(t, err)

	inspection, err := engine.ExecInspect(t.Context(), execution.ID, client.ExecInspectOptions{})
	require.NoError(t, err)
	require.Zero(t, inspection.ExitCode)

	return contents.String()
}

func createStrayContainer(t *testing.T, engine *client.Client, agentID uuid.UUID) string {
	t.Helper()
	if _, err := engine.ImageInspect(t.Context(), getRestoreImages()[6]); err != nil {
		download, err := engine.ImagePull(t.Context(), getRestoreImages()[6], client.ImagePullOptions{})
		require.NoError(t, err)
		_, err = io.Copy(io.Discard, download)
		closeErr := download.Close()
		require.NoError(t, err)
		require.NoError(t, closeErr)
	}

	instance, err := engine.ContainerCreate(
		t.Context(),
		client.ContainerCreateOptions{
			Config: &container.Config{
				Image:  getRestoreImages()[6],
				Cmd:    []string{"sleep", "600"},
				Labels: map[string]string{agent_container.LabelAgentID: agentID.String()},
			},
			Name: "databasus-stray-" + uuid.NewString(),
		},
	)
	require.NoError(t, err)
	_, err = engine.ContainerStart(t.Context(), instance.ID, client.ContainerStartOptions{})
	require.NoError(t, err)
	return instance.ID
}

func getEngineHost() string {
	//nolint:forbidigo // The test task selects the agent engine independently of the application's configuration.
	return os.Getenv("DATABASUS_AGENT_E2E_ENGINE_HOST")
}
