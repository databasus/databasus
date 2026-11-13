package cnpg_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"syscall"
	"testing"
	"time"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/google/uuid"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"

	"databasus-backend/internal/lib/config"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

const childLimit = 12 * time.Minute

type cnpgChildSpec struct {
	FixtureID            string
	Mode                 string
	RunID                string
	DockerHost           string
	ShouldWaitForRelease bool
}

// cnpgChild is a child test process that owns one k3s fixture. Its parent registers a cleanup by the
// fixture label before the child starts, so a killed child's resources are removed too.
type cnpgChild struct {
	FixtureID    string
	Directory    string
	ManifestPath string
	ReleasePath  string
	docker       *testcontainers.DockerClient
	command      *exec.Cmd
	cancel       context.CancelFunc
	output       bytes.Buffer
	done         chan struct{}
	waitErr      error
}

func startCNPGChild(t *testing.T, spec cnpgChildSpec) *cnpgChild {
	t.Helper()

	docker, err := testcontainers.NewDockerClientWithOpts(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { _ = docker.Close() })

	child := &cnpgChild{
		FixtureID:    uuid.NewString(),
		Directory:    filepath.Join(t.TempDir(), "child"),
		ManifestPath: filepath.Join(t.TempDir(), "resources.json"),
		docker:       docker,
	}
	if spec.FixtureID != "" {
		child.FixtureID = spec.FixtureID
	}
	require.NoError(t, os.Mkdir(child.Directory, 0o700))
	t.Cleanup(func() { child.removeByLabel(t) })

	executable, err := os.Executable()
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), childLimit)
	child.cancel = cancel
	child.command = exec.CommandContext(ctx, executable, "-test.run=^TestCNPGFixtureChild$", "-test.v", "-test.count=1")
	child.command.WaitDelay = 5 * time.Second
	child.command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	child.command.Cancel = func() error {
		err := syscall.Kill(-child.command.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}

		return err
	}

	child.command.Env = append(
		child.command.Environ(),
		"TMPDIR="+child.Directory,
		childModeVariable+"="+spec.Mode,
		childManifestVariable+"="+child.ManifestPath,
		childIDVariable+"="+child.FixtureID,
	)
	variables, _, err := config.ReadVariables(config.LoadSpec{})
	require.NoError(t, err)

	if spec.RunID != "" {
		rootRunID := variables["DATABASUS_TEST_ROOT_RUN_ID"]
		if rootRunID == "" {
			rootRunID = variables["DATABASUS_TEST_RUN_ID"]
		}

		child.command.Env = append(child.command.Env,
			"DATABASUS_TEST_RUN_ID="+spec.RunID,
			"DATABASUS_TEST_ROOT_RUN_ID="+rootRunID,
		)
	}

	if spec.DockerHost != "" {
		child.command.Env = append(child.command.Env, "DOCKER_HOST="+spec.DockerHost)
	}

	if spec.ShouldWaitForRelease {
		child.ReleasePath = filepath.Join(t.TempDir(), "release")
		child.command.Env = append(child.command.Env, childReleaseVariable+"="+child.ReleasePath)
	}

	ownershipReader, ownershipWriter, err := os.Pipe()
	require.NoError(t, err)
	defer func() { _ = ownershipReader.Close() }()
	child.command.ExtraFiles = []*os.File{ownershipReader}
	t.Cleanup(func() { _ = ownershipWriter.Close() })

	child.command.Stdout = &child.output
	child.command.Stderr = &child.output
	require.NoError(t, child.command.Start())
	child.done = make(chan struct{})
	go func() {
		child.waitErr = child.command.Wait()
		close(child.done)
	}()
	t.Cleanup(func() {
		cancel()
		<-child.done
		if t.Failed() {
			t.Logf("child %s output:\n%s", spec.Mode, child.output.String())
		}
	})

	return child
}

func (child *cnpgChild) wait(t *testing.T) (string, error) {
	t.Helper()

	<-child.done

	return child.output.String(), child.waitErr
}

func (child *cnpgChild) waitUntilReady(t *testing.T) {
	t.Helper()

	testwait.Within(childLimit).
		Until(t, "the child holds its k3s and CloudNativePG cluster", func(context.Context) error {
			select {
			case <-child.done:
				t.Fatalf("the child exited before it was ready: %v\n%s", child.waitErr, child.output.String())
			default:
			}

			if _, err := os.Stat(child.ManifestPath + ".ready"); err != nil {
				return fmt.Errorf("child readiness: %w", err)
			}

			return nil
		})
}

func (child *cnpgChild) kill(t *testing.T) {
	t.Helper()

	child.cancel()
	_, err := child.wait(t)
	require.Error(t, err)
}

func (child *cnpgChild) release(t *testing.T) {
	t.Helper()

	require.NoError(t, os.WriteFile(child.ReleasePath, nil, 0o600))
}

func (child *cnpgChild) getResources(t *testing.T) containers.K3sResources {
	t.Helper()

	manifest, err := os.ReadFile(child.ManifestPath)
	require.NoError(t, err)

	var resources containers.K3sResources
	require.NoError(t, json.Unmarshal(manifest, &resources))

	return resources
}

// requireRemoved checks that every resource the child reported is gone and that it left no temporary
// files.
func (child *cnpgChild) requireRemoved(t *testing.T) {
	t.Helper()

	resources := child.getResources(t)
	require.NotEmpty(t, resources.NetworkName)
	for _, containerID := range resources.ContainerIDs {
		_, err := child.docker.ContainerInspect(t.Context(), containerID, client.ContainerInspectOptions{})
		require.True(t, cerrdefs.IsNotFound(err), "container %s remains: %v", containerID, err)
	}

	_, err := child.docker.NetworkInspect(t.Context(), resources.NetworkName, client.NetworkInspectOptions{})
	require.True(t, cerrdefs.IsNotFound(err), "network %s remains: %v", resources.NetworkName, err)

	for _, volume := range resources.VolumeNames {
		_, err := child.docker.VolumeInspect(t.Context(), volume, client.VolumeInspectOptions{})
		require.True(t, cerrdefs.IsNotFound(err), "volume %s remains: %v", volume, err)
	}

	entries, err := os.ReadDir(child.Directory)
	if !errors.Is(err, os.ErrNotExist) {
		require.NoError(t, err)
		require.Empty(t, entries, "the child left temporary files")
	}
}

type labelledResources struct {
	ContainerLabels []map[string]string
	NetworkLabels   []map[string]string
	VolumeLabels    []map[string]string
}

func (child *cnpgChild) listLabelled(t *testing.T) labelledResources {
	t.Helper()

	filters := child.getFilters()
	var listed labelledResources
	instances, err := child.docker.ContainerList(t.Context(), client.ContainerListOptions{All: true, Filters: filters})
	require.NoError(t, err)
	for _, instance := range instances.Items {
		listed.ContainerLabels = append(listed.ContainerLabels, instance.Labels)
	}

	networks, err := child.docker.NetworkList(t.Context(), client.NetworkListOptions{Filters: filters})
	require.NoError(t, err)
	for _, network := range networks.Items {
		listed.NetworkLabels = append(listed.NetworkLabels, network.Labels)
	}

	volumes, err := child.docker.VolumeList(t.Context(), client.VolumeListOptions{Filters: filters})
	require.NoError(t, err)
	for _, volume := range volumes.Items {
		listed.VolumeLabels = append(listed.VolumeLabels, volume.Labels)
	}

	return listed
}

func (child *cnpgChild) requireRunLabels(t *testing.T, runID string) {
	t.Helper()

	listed := child.listLabelled(t)
	require.NotEmpty(t, listed.ContainerLabels)
	require.NotEmpty(t, listed.NetworkLabels)
	require.NotEmpty(t, listed.VolumeLabels)

	variables, _, err := config.ReadVariables(config.LoadSpec{})
	require.NoError(t, err)
	rootRunID := variables["DATABASUS_TEST_ROOT_RUN_ID"]
	if rootRunID == "" {
		rootRunID = variables["DATABASUS_TEST_RUN_ID"]
	}

	for _, labels := range slices.Concat(listed.ContainerLabels, listed.NetworkLabels, listed.VolumeLabels) {
		require.Equal(t, runID, labels["databasus.test-run"])

		if rootRunID != "" {
			require.Equal(t, rootRunID, labels["databasus.test-root-run"])
		}
	}
}

func (child *cnpgChild) getFilters() client.Filters {
	return make(client.Filters).Add("label", containers.LabelK3sFixture+"="+child.FixtureID)
}

func (child *cnpgChild) removeByLabel(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 2*time.Minute)
	defer cancel()
	filters := child.getFilters()
	instances, err := child.docker.ContainerList(ctx, client.ContainerListOptions{All: true, Filters: filters})
	if assert.NoError(t, err) {
		for _, instance := range instances.Items {
			_, err := child.docker.ContainerRemove(ctx, instance.ID, client.ContainerRemoveOptions{Force: true})
			assert.NoError(t, err)
		}
	}

	volumes, err := child.docker.VolumeList(ctx, client.VolumeListOptions{Filters: filters})
	if assert.NoError(t, err) {
		for _, volume := range volumes.Items {
			_, err := child.docker.VolumeRemove(ctx, volume.Name, client.VolumeRemoveOptions{Force: true})
			assert.NoError(t, err)
		}
	}

	networks, err := child.docker.NetworkList(ctx, client.NetworkListOptions{Filters: filters})
	if assert.NoError(t, err) {
		for _, network := range networks.Items {
			_, err := child.docker.NetworkRemove(ctx, network.ID, client.NetworkRemoveOptions{})
			assert.NoError(t, err)
		}
	}

	assert.NoError(t, os.RemoveAll(child.Directory))
}

// The inherited pipe closes when the parent dies, including SIGKILL. Each child has its own
// process group, so the watchdog can stop its whole group without signalling the suite.
func watchCNPGParent(t *testing.T) {
	t.Helper()

	pipe := os.NewFile(3, "cnpg-parent")
	require.NotNil(t, pipe)
	go func() {
		_, _ = io.Copy(io.Discard, pipe)
		_ = pipe.Close()
		_ = syscall.Kill(-syscall.Getpgrp(), syscall.SIGKILL)
	}()
}
