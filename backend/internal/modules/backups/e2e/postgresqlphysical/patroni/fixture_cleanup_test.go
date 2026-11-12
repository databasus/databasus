package patroni_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/config"
	containers "databasus-backend/internal/lib/testcontainers"
)

func runPatroniCleanupScenario(t *testing.T, mode string) {
	t.Helper()

	unrelatedNetwork := containers.StartNetwork(t)
	executable, err := os.Executable()
	require.NoError(t, err)
	cleanup := newPatroniChildCleanup(t)
	manifestPath := filepath.Join(t.TempDir(), "resources.json")
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, executable, "-test.run=^TestPatroniFixtureChild$", "-test.v")
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error {
		err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}

		return err
	}
	command.Env = append(command.Environ(), "TMPDIR="+cleanup.Directory, "DATABASUS_PATRONI_CHILD="+mode,
		"DATABASUS_PATRONI_MANIFEST="+manifestPath, "DATABASUS_PATRONI_ID="+cleanup.FixtureID)
	var output []byte
	var runErr error
	if mode == "parent-interruption" {
		output, runErr = interruptPatroniChild(
			t,
			patroniChildInterruption{Command: command, Cancel: cancel, ReadyPath: manifestPath + ".ready"},
		)
		cleanup.remove(t)
	} else {
		output, runErr = command.CombinedOutput()
	}

	if mode == "success" {
		require.NoError(t, runErr, "%s", output)
	} else {
		require.Error(t, runErr, "%s", output)
		require.Contains(t, string(output), "intentional fixture "+mode)
		require.Contains(t, string(output), "fixture")
	}

	if mode == "parent-interruption" {
		require.ErrorIs(t, ctx.Err(), context.Canceled)
	} else {
		require.NoError(t, ctx.Err(), "%s", output)
	}

	manifest, err := os.ReadFile(manifestPath)
	require.NoError(t, err, "%s", output)

	var resources containers.PatroniResources
	require.NoError(t, json.Unmarshal(manifest, &resources))
	require.NotEmpty(t, resources.ContainerIDs)

	docker := cleanup.Docker
	_, err = docker.NetworkInspect(t.Context(), unrelatedNetwork, client.NetworkInspectOptions{})
	require.NoError(t, err, "cleanup removed an unrelated network")

	for _, containerID := range resources.ContainerIDs {
		_, err := docker.ContainerInspect(t.Context(), containerID, client.ContainerInspectOptions{})
		require.True(t, cerrdefs.IsNotFound(err), "container %s remains: %v", containerID, err)
	}

	_, err = docker.NetworkInspect(t.Context(), resources.NetworkName, client.NetworkInspectOptions{})
	require.True(t, cerrdefs.IsNotFound(err), "network %s remains: %v", resources.NetworkName, err)

	for _, volume := range resources.VolumeNames {
		_, err := docker.VolumeInspect(t.Context(), volume, client.VolumeInspectOptions{})
		require.True(t, cerrdefs.IsNotFound(err), "volume %s remains: %v", volume, err)
	}

	_, err = os.Stat(resources.TemporaryDirectory)
	require.ErrorIs(t, err, os.ErrNotExist)

	t.Logf(
		"removed %d containers, %d volumes, network %s and temporary directory after %s",
		len(resources.ContainerIDs),
		len(resources.VolumeNames),
		resources.NetworkName,
		mode,
	)
}

func TestPatroniFixtureChild(t *testing.T) {
	variables, _, err := config.ReadVariables(config.LoadSpec{})
	require.NoError(t, err)
	mode := variables["DATABASUS_PATRONI_CHILD"]
	if mode == "" {
		return
	}

	spec := containers.PatroniSpec{
		Image: "postgres:17", ID: variables["DATABASUS_PATRONI_ID"],
		ObserveResources: func(resources containers.PatroniResources) {
			manifest, err := json.Marshal(resources)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(variables["DATABASUS_PATRONI_MANIFEST"], manifest, 0o600))
		},
	}
	if mode == "partial-startup" {
		spec.NodeEntrypoint = []string{"sh", "-c", "echo 'intentional fixture partial-startup' >&2; exit 43"}
	}

	fixture := containers.StartPatroni(t, spec)

	if mode == "parent-interruption" {
		t.Log("intentional fixture parent-interruption")
		require.NoError(t, os.WriteFile(variables["DATABASUS_PATRONI_MANIFEST"]+".ready", nil, 0o600))
		<-t.Context().Done()
	}

	if mode == "transfer-failure" {
		runPatroniTransferLifecycle(t, fixture, mode)
	}

	if mode == "cancellation" {
		connection := fixture.Connect(t, fixture.Endpoint)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		_, err := connection.Exec(ctx, "SELECT 1")
		require.ErrorIs(t, err, context.Canceled)
		t.Fatal("intentional fixture cancellation")
	}
}
