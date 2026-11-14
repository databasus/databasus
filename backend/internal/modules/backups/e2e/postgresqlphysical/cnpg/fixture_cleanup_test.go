package cnpg_test

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"

	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

func TestCNPGFixtureCleanup(t *testing.T) {
	t.Parallel()

	for _, mode := range []string{childSuccess, childCancellation, childLiveCancellation} {
		t.Run("removes owned resources after "+mode, func(t *testing.T) {
			t.Parallel()

			unrelatedNetwork := containers.StartNetwork(t)
			child := startCNPGChild(t, cnpgChildSpec{Mode: mode})

			output, err := child.wait(t)

			if mode == childSuccess {
				require.NoError(t, err, "%s", output)
			} else {
				require.Error(t, err, "%s", output)
				require.Contains(t, output, "intentional fixture "+mode)
			}

			child.requireRemoved(t)
			requireNetworkPresent(t, unrelatedNetwork)
		})
	}
}

func TestCNPGFixtureStartupCleanup(t *testing.T) {
	t.Parallel()

	t.Run("removes owned resources after a partial startup fails", func(t *testing.T) {
		t.Parallel()

		child := startCNPGChild(t, cnpgChildSpec{Mode: childUnreachableAddress})

		output, err := child.wait(t)

		require.Error(t, err, "%s", output)
		require.Contains(t, output, "intentional fixture "+childUnreachableAddress)
		resources := child.getResources(t)
		require.NotEmpty(t, resources.ContainerIDs, "the startup failed after it created the k3s container")
		child.requireRemoved(t)
	})

	t.Run("removes owned resources after cancellation with a stalled Kubernetes API", func(t *testing.T) {
		t.Parallel()

		unrelatedNetwork := containers.StartNetwork(t)
		child := startCNPGChild(t, cnpgChildSpec{Mode: childStalledDiagnostics})

		testwait.Within(childLimit).Until(t, "the child stalls its owned Kubernetes API", func(context.Context) error {
			_, err := os.Stat(child.ManifestPath + ".stalled")
			if err == nil {
				return nil
			}

			select {
			case <-child.done:
				t.Fatalf("child exited before stalling the API: %v\n%s", child.waitErr, child.output.String())
			default:
			}

			return err
		})

		testwait.Within(time.Minute).
			Until(t, "startup diagnostics return despite the stalled API", func(context.Context) error {
				select {
				case <-child.done:
					return nil
				default:
					return fmt.Errorf("startup diagnostics are still running")
				}
			})

		output, err := child.wait(t)

		require.Error(t, err, "%s", output)
		require.Contains(t, output, "stopped the owned k3s API process")
		require.Contains(t, output, "intentional fixture "+childStalledDiagnostics)
		require.Contains(t, output, "context deadline exceeded")
		child.requireRemoved(t)
		requireNetworkPresent(t, unrelatedNetwork)
	})
}

func TestCNPGFixtureKilledCleanup(t *testing.T) {
	t.Parallel()

	t.Run("removes owned resources by fixture label after the child is killed", func(t *testing.T) {
		t.Parallel()

		unrelatedNetwork := containers.StartNetwork(t)
		child := startCNPGChild(t, cnpgChildSpec{Mode: childHeld})
		child.waitUntilReady(t)

		child.kill(t)
		child.removeByLabel(t)

		child.requireRemoved(t)
		requireNetworkPresent(t, unrelatedNetwork)
	})

	t.Run("stops the fixture child after its parent is killed", func(t *testing.T) {
		t.Parallel()

		unrelatedNetwork := containers.StartNetwork(t)
		parent := startCNPGChild(t, cnpgChildSpec{Mode: childSupervisor})
		parent.waitUntilReady(t)
		pidText, err := os.ReadFile(parent.ManifestPath + ".pid")
		require.NoError(t, err)
		childPID, err := strconv.Atoi(string(pidText))
		require.NoError(t, err)
		t.Cleanup(func() { _ = syscall.Kill(-childPID, syscall.SIGKILL) })

		parent.kill(t)

		testwait.Until(t, "the orphaned fixture process exits", func(context.Context) error {
			state, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", childPID))
			if os.IsNotExist(err) {
				return nil
			}

			if err != nil {
				return err
			}

			if strings.Contains(string(state), ") Z ") {
				return nil
			}

			return fmt.Errorf("fixture child %d is still alive", childPID)
		})
		parent.removeByLabel(t)
		parent.requireRemoved(t)
		requireNetworkPresent(t, unrelatedNetwork)
	})
}

func TestCNPGFixtureRunLabels(t *testing.T) {
	t.Parallel()

	t.Run("labels every resource with the run ID while the child holds them", func(t *testing.T) {
		t.Parallel()

		runID := "cnpg-lifecycle-" + uuid.NewString()
		child := startCNPGChild(t, cnpgChildSpec{Mode: childHeld, RunID: runID})
		child.waitUntilReady(t)

		listed := child.listLabelled(t)
		resources := child.getResources(t)

		require.Len(t, listed.ContainerLabels, len(resources.ContainerIDs))
		require.Len(t, listed.NetworkLabels, 1)
		require.Len(t, listed.VolumeLabels, len(resources.VolumeNames))
		require.Contains(t, resources.VolumeNames, "databasus-k3s-"+child.FixtureID)
		child.requireRunLabels(t, runID)

		child.kill(t)
		child.removeByLabel(t)
		child.requireRemoved(t)
	})

	t.Run("keeps each run's resources through the other run's cleanup when two runs overlap", func(t *testing.T) {
		t.Parallel()

		heldRunID := "cnpg-held-" + uuid.NewString()
		finishedRunID := "cnpg-finished-" + uuid.NewString()
		held := startCNPGChild(
			t,
			cnpgChildSpec{Mode: childSuccess, RunID: heldRunID, ShouldWaitForRelease: true},
		)
		finished := startCNPGChild(
			t,
			cnpgChildSpec{Mode: childSuccess, RunID: finishedRunID, ShouldWaitForRelease: true},
		)
		held.waitUntilReady(t)
		finished.waitUntilReady(t)
		held.requireRunLabels(t, heldRunID)
		finished.requireRunLabels(t, finishedRunID)

		finished.release(t)
		finishedOutput, err := finished.wait(t)
		require.NoError(t, err, "%s", finishedOutput)
		finished.requireRemoved(t)
		listed := held.listLabelled(t)
		require.NotEmpty(t, listed.ContainerLabels, "the other run's cleanup removed this run's containers")
		require.Len(t, listed.NetworkLabels, 1)
		require.NotEmpty(t, listed.VolumeLabels)

		held.release(t)
		heldOutput, err := held.wait(t)
		require.NoError(t, err, "%s", heldOutput)
		held.requireRemoved(t)
	})
}

func requireNetworkPresent(t *testing.T, name string) {
	t.Helper()

	docker, err := testcontainers.NewDockerClientWithOpts(t.Context())
	require.NoError(t, err)
	defer func() { _ = docker.Close() }()

	_, err = docker.NetworkInspect(t.Context(), name, client.NetworkInspectOptions{})
	require.NoError(t, err, "cleanup removed an unrelated network")
}
