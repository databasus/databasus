package upgrade

import (
	"bytes"
	"context"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/require"
)

type agentDockerSocket struct {
	path  string
	group string
}

func getAgentDockerSocket(t *testing.T, docker *dockerHost, image string) agentDockerSocket {
	t.Helper()

	engineURL, err := url.Parse(docker.client.DaemonHost())
	require.NoError(t, err)
	require.Equal(t, "unix", engineURL.Scheme, "agent upgrade fixture needs a local Unix Docker socket")

	// A containerized runner can see different socket ownership from the Docker host's bind-mount.
	probe, err := docker.client.ContainerCreate(t.Context(), client.ContainerCreateOptions{
		Config: &container.Config{
			Image: image, User: "65532:65532", Entrypoint: []string{"/bin/sh"},
			Cmd: []string{"-ec", "test -S /engine.sock; stat -c %g /engine.sock"}, Labels: docker.labels,
		},
		HostConfig: &container.HostConfig{
			Binds: []string{engineURL.Path + ":/engine.sock"}, SecurityOpt: []string{"label=disable"},
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), cleanupTimeout)
		defer cancel()

		require.NoError(t, docker.removeAgentContainer(ctx, probe.ID))
	})

	_, err = docker.client.ContainerStart(t.Context(), probe.ID, client.ContainerStartOptions{})
	require.NoError(t, err)

	wait := docker.client.ContainerWait(t.Context(), probe.ID, client.ContainerWaitOptions{
		Condition: container.WaitConditionNotRunning,
	})
	select {
	case response := <-wait.Result:
		require.Zero(t, response.StatusCode, "inspect the bind-mounted Docker socket")
	case err := <-wait.Error:
		require.NoError(t, err)
	case <-t.Context().Done():
		t.Fatal("test ended while inspecting the Docker socket")
	}

	logs, err := docker.client.ContainerLogs(t.Context(), probe.ID, client.ContainerLogsOptions{ShowStdout: true})
	require.NoError(t, err)
	defer func() { require.NoError(t, logs.Close()) }()

	var output bytes.Buffer
	_, err = stdcopy.StdCopy(&output, &output, logs)
	require.NoError(t, err)
	group := strings.TrimSpace(output.String())
	_, err = strconv.ParseUint(group, 10, 32)
	require.NoError(t, err, "Docker socket group must be numeric")

	return agentDockerSocket{path: engineURL.Path, group: group}
}
