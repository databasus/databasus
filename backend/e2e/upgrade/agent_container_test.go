package upgrade

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

type upgradeAgent struct {
	docker      *dockerHost
	containerID string
	binaryPath  string
	workingPath string
	agentID     string
	token       string
	image       string
	gatewayURL  string
	isReadOnly  bool
	socket      agentDockerSocket
}

type upgradeAgentSpec struct {
	api            *apiClient
	docker         *dockerHost
	image          string
	releasedBinary string
	gatewayURL     string
	isReadOnly     bool
}

func newUpgradeAgent(t *testing.T, spec upgradeAgentSpec) *upgradeAgent {
	t.Helper()

	var credentials struct {
		Agent struct {
			ID string `json:"id"`
		} `json:"agent"`
		Token string `json:"token"`
	}

	require.NoError(
		t,
		spec.api.post(
			t.Context(),
			"/verification-agents",
			map[string]string{"name": "released agent upgrade"},
			&credentials,
		),
	)

	directory := t.TempDir()
	binaryDirectory := filepath.Join(directory, "bin")
	workingDirectory := filepath.Join(directory, "work")
	require.NoError(t, os.Mkdir(binaryDirectory, 0o755))
	require.NoError(t, os.Mkdir(workingDirectory, 0o777))
	require.NoError(t, os.Chmod(workingDirectory, 0o777))

	if os.Getuid() == 0 {
		require.NoError(t, os.Chown(binaryDirectory, 65532, -1))
	}

	input, err := os.Open(spec.releasedBinary)
	require.NoError(t, err)
	defer func() { require.NoError(t, input.Close()) }()

	binaryPath := filepath.Join(binaryDirectory, "verification-agent")
	require.NoError(t, installAgentBinary(binaryPath, input))

	agent := &upgradeAgent{
		docker: spec.docker, binaryPath: binaryPath, workingPath: workingDirectory,
		agentID: credentials.Agent.ID, token: credentials.Token,
		image: spec.image, gatewayURL: spec.gatewayURL, isReadOnly: spec.isReadOnly,
		socket: getAgentDockerSocket(t, spec.docker, spec.image),
	}

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), cleanupTimeout)
		defer cancel()

		if t.Failed() && agent.containerID != "" {
			logs, logErr := agent.getLogs(ctx)
			t.Logf("agent diagnostics: %s (%v)", logs, logErr)
		}

		require.NoError(t, agent.remove(ctx))
		require.NoError(t, agent.removeRestoreResources(ctx))
	})

	return agent
}

func (a *upgradeAgent) startWithInitialConfig(ctx context.Context) error {
	return a.startContainer(ctx, []string{
		"run", "--skip-update",
		"--databasus-host", a.gatewayURL,
		"--agent-id", a.agentID, "--token", a.token, "--allow-insecure-http",
		"--max-cpu", "1", "--max-ram-mb", "2048", "--max-disk-gb", "10",
		"--max-concurrent-jobs", "1", "--verification-pg-image-repo", "postgres",
	})
}

func (a *upgradeAgent) startFromSavedConfig(ctx context.Context) error {
	_, err := os.Stat(filepath.Join(a.workingPath, "databasus-verification.json"))
	if err != nil {
		return fmt.Errorf("inspect saved agent configuration: %w", err)
	}

	return a.startContainer(ctx, []string{"run", "--skip-update"})
}

func (a *upgradeAgent) startContainer(ctx context.Context, arguments []string) error {
	uid := os.Getuid()
	if uid == 0 {
		uid = 65532
	}

	binaryMount := filepath.Dir(a.binaryPath) + ":/agent/bin"
	if a.isReadOnly {
		binaryMount += ":ro"
	}

	created, err := a.docker.client.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config: &container.Config{
			Image: a.image, User: fmt.Sprintf("%d:%d", uid, os.Getgid()), WorkingDir: "/work",
			Entrypoint: []string{"/agent/bin/verification-agent"},
			Cmd:        arguments,
			Env:        []string{"DOCKER_HOST=unix:///var/run/docker.sock"}, Labels: a.docker.labels,
		},
		HostConfig: &container.HostConfig{
			NetworkMode: "host", SecurityOpt: []string{"label=disable"},
			GroupAdd: []string{a.socket.group},
			Binds:    []string{binaryMount, a.workingPath + ":/work", a.socket.path + ":/var/run/docker.sock"},
		},
	})
	if err != nil {
		return err
	}

	a.containerID = created.ID
	_, err = a.docker.client.ContainerStart(ctx, a.containerID, client.ContainerStartOptions{})

	return err
}

func (a *upgradeAgent) remove(ctx context.Context) error {
	if a.containerID == "" {
		return nil
	}

	err := a.docker.removeAgentContainer(ctx, a.containerID)
	if err == nil {
		a.containerID = ""
	}

	return err
}

func (a *upgradeAgent) stop(ctx context.Context) error {
	if err := a.docker.stopContainer(ctx, a.containerID); err != nil {
		return err
	}

	return a.remove(ctx)
}

func (a *upgradeAgent) getLogs(ctx context.Context) (string, error) {
	logs, err := a.docker.client.ContainerLogs(ctx, a.containerID, client.ContainerLogsOptions{
		ShowStdout: true, ShowStderr: true,
	})
	if err != nil {
		return "", err
	}

	defer func() { _ = logs.Close() }()

	var output bytes.Buffer
	_, err = stdcopy.StdCopy(&output, &output, logs)

	return strings.ReplaceAll(output.String(), a.token, "[REDACTED]"), err
}

func (a *upgradeAgent) waitForSignal(t *testing.T, description string, signal <-chan struct{}) {
	t.Helper()

	testwait.Until(t, description, func(ctx context.Context) error {
		select {
		case <-signal:
			return nil
		default:
		}

		inspection, err := a.docker.client.ContainerInspect(ctx, a.containerID, client.ContainerInspectOptions{})
		if err != nil {
			return err
		}

		if !inspection.Container.State.Running {
			logs, logErr := a.getLogs(ctx)
			t.Fatalf("agent exited before %s: %s (%v)", description, logs, logErr)
		}

		return fmt.Errorf("waiting for %s", description)
	})
}

func (a *upgradeAgent) removeRestoreResources(ctx context.Context) error {
	filters := make(client.Filters)
	filters.Add("label", "databasus.verification.agent_id="+a.agentID)

	instances, err := a.docker.client.ContainerList(ctx, client.ContainerListOptions{All: true, Filters: filters})
	if err != nil {
		return err
	}

	for _, instance := range instances.Items {
		if _, err := a.docker.client.ContainerRemove(
			ctx,
			instance.ID,
			client.ContainerRemoveOptions{Force: true, RemoveVolumes: true},
		); err != nil {
			return err
		}
	}

	networks, err := a.docker.client.NetworkList(ctx, client.NetworkListOptions{Filters: filters})
	if err != nil {
		return err
	}

	for _, network := range networks.Items {
		if _, err := a.docker.client.NetworkRemove(ctx, network.ID, client.NetworkRemoveOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func (a *upgradeAgent) installCurrentBinary(t *testing.T, api *apiClient) {
	t.Helper()

	request, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		api.baseURL+"/system/verification-agent?arch="+runtime.GOARCH,
		nil,
	)
	require.NoError(t, err)

	response, err := api.httpClient.Do(request)
	require.NoError(t, err)
	defer func() { require.NoError(t, response.Body.Close()) }()

	require.Equal(t, 200, response.StatusCode)

	require.NoError(t, installAgentBinary(a.binaryPath, response.Body))
}

func (a *upgradeAgent) waitForRestoreCleanup(t *testing.T) {
	t.Helper()

	filters := make(client.Filters)
	filters.Add("label", "databasus.verification.agent_id="+a.agentID)
	testwait.Until(t, "the agent to remove restore containers and networks", func(ctx context.Context) error {
		instances, err := a.docker.client.ContainerList(
			ctx,
			client.ContainerListOptions{All: true, Filters: filters},
		)
		if err != nil {
			return err
		}

		networks, err := a.docker.client.NetworkList(ctx, client.NetworkListOptions{Filters: filters})
		if err != nil {
			return err
		}

		if len(instances.Items)+len(networks.Items) > 0 {
			return fmt.Errorf("%d restore containers and %d networks remain", len(instances.Items), len(networks.Items))
		}

		return nil
	})
}
