package upgrade

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"databasus-backend/e2e/upgrade/upgradecheck"
	"databasus-backend/internal/lib/testwait"
)

const (
	instancePublishedPort = "4005/tcp"
	instanceDataPath      = "/databasus-data"
	cleanupTimeout        = 2 * time.Minute
)

var errInstanceExited = errors.New("the instance exited during start-up")

type upgradeSpec struct {
	previousImage string
	currentImage  string
	// currentEntrypoint replaces the entrypoint of the current image when set, so a test can make
	// the current instance fail at start-up.
	currentEntrypoint []string
}

// runUpgrade upgrades an instance of the previous image with data to the current image and checks
// the data and a restore of its backup. It returns an error instead of failing the test, so a test
// can drive it into a failure on purpose. It removes everything it created before it returns, and
// it returns the names it generated, so a caller can check that nothing remains.
func runUpgrade(t *testing.T, spec upgradeSpec) (resources upgradeResources, err error) {
	t.Helper()

	resources, err = newUpgradeResources()
	if err != nil {
		return resources, err
	}

	docker, err := newDockerHost()
	if err != nil {
		return resources, err
	}
	defer docker.close()

	defer func() {
		cleanupCtx, cancelCleanup := context.WithTimeout(context.WithoutCancel(t.Context()), cleanupTimeout)
		defer cancelCleanup()

		err = errors.Join(err, docker.removeResources(cleanupCtx, resources))
	}()

	err = runUpgradeSteps(t, docker, spec, resources)

	return resources, err
}

func runUpgradeSteps(t *testing.T, docker *dockerHost, spec upgradeSpec, resources upgradeResources) error {
	t.Helper()

	ctx := t.Context()

	err := docker.createNetwork(ctx, resources.networkName)
	if err != nil {
		return err
	}

	err = docker.createVolume(ctx, resources.volumeName)
	if err != nil {
		return err
	}

	sourceHostPort, err := prepareSourceServer(t, docker, resources)
	if err != nil {
		return err
	}

	writtenData, err := writeThroughPreviousRelease(t, docker, spec, resources)
	if err != nil {
		return err
	}

	err = addRowAfterBackup(ctx, sourceHostPort)
	if err != nil {
		return err
	}

	err = docker.stopContainer(ctx, resources.previousContainerName)
	if err != nil {
		return err
	}

	return checkThroughCurrentCode(t, docker, spec, resources, writtenData, sourceHostPort)
}

func prepareSourceServer(t *testing.T, docker *dockerHost, resources upgradeResources) (int, error) {
	t.Helper()

	err := startSourceServer(t.Context(), docker, resources)
	if err != nil {
		return 0, err
	}

	var sourceHostPort int

	err = testwait.Poll(t, "the source server to accept connections", func(ctx context.Context) error {
		state, stateErr := docker.getContainerState(ctx, resources.sourceContainerName, sourcePublishedPort)
		if stateErr != nil {
			return stateErr
		}

		if !state.isRunning {
			return fmt.Errorf("container %s is not running", resources.sourceContainerName)
		}

		connection, connectErr := connectToSourceDatabase(ctx, state.hostPort, sourceDatabaseName)
		if connectErr != nil {
			return connectErr
		}

		sourceHostPort = state.hostPort

		return connection.Close(ctx)
	})
	if err != nil {
		return 0, docker.withLogTail(t.Context(), resources.sourceContainerName, err)
	}

	err = fillSourceDatabase(t.Context(), sourceHostPort)
	if err != nil {
		return 0, err
	}

	return sourceHostPort, nil
}

func writeThroughPreviousRelease(
	t *testing.T,
	docker *dockerHost,
	spec upgradeSpec,
	resources upgradeResources,
) (writtenUpgradeData, error) {
	t.Helper()

	previousAPI, err := startInstance(t, docker, containerSpec{
		name:          resources.previousContainerName,
		image:         spec.previousImage,
		entrypoint:    nil,
		env:           nil,
		volumeMount:   resources.volumeName + ":" + instanceDataPath,
		networkName:   resources.networkName,
		networkAlias:  resources.previousContainerName,
		publishedPort: instancePublishedPort,
	})
	if err != nil {
		return writtenUpgradeData{}, err
	}

	writtenData, err := writeUpgradeData(t, previousAPI)
	if err != nil {
		return writtenUpgradeData{}, docker.withLogTail(t.Context(), resources.previousContainerName,
			fmt.Errorf("write through the previous release: %w", err))
	}

	return writtenData, nil
}

func checkThroughCurrentCode(
	t *testing.T,
	docker *dockerHost,
	spec upgradeSpec,
	resources upgradeResources,
	writtenData writtenUpgradeData,
	sourceHostPort int,
) error {
	t.Helper()

	ctx := t.Context()

	currentAPI, err := startInstance(t, docker, containerSpec{
		name:          resources.currentContainerName,
		image:         spec.currentImage,
		entrypoint:    spec.currentEntrypoint,
		env:           nil,
		volumeMount:   resources.volumeName + ":" + instanceDataPath,
		networkName:   resources.networkName,
		networkAlias:  resources.currentContainerName,
		publishedPort: instancePublishedPort,
	})
	if err != nil {
		return err
	}

	listedItems, err := readUpgradeData(ctx, currentAPI)
	if err != nil {
		return docker.withLogTail(ctx, resources.currentContainerName, err)
	}

	err = upgradecheck.Compare(writtenData.items, listedItems)
	if err != nil {
		return err
	}

	err = checkStoredCredentials(ctx, currentAPI, writtenData.databaseID)
	if err != nil {
		return docker.withLogTail(ctx, resources.currentContainerName, err)
	}

	err = createRestoredDatabase(ctx, sourceHostPort)
	if err != nil {
		return err
	}

	err = restoreBackup(t, currentAPI, writtenData.backupID)
	if err != nil {
		return docker.withLogTail(ctx, resources.currentContainerName, err)
	}

	return checkRestoredRows(ctx, sourceHostPort)
}

// startInstance starts a Databasus container and waits until it answers its health check. When it
// stops during start-up or never answers, the error carries the end of its log.
func startInstance(t *testing.T, docker *dockerHost, spec containerSpec) (*apiClient, error) {
	t.Helper()

	err := docker.startContainer(t.Context(), spec)
	if err != nil {
		return nil, err
	}

	var (
		hasExited bool
		hostPort  int
	)

	err = testwait.Poll(t, "instance "+spec.name+" to answer its health check", func(ctx context.Context) error {
		state, stateErr := docker.getContainerState(ctx, spec.name, instancePublishedPort)
		if stateErr != nil {
			return stateErr
		}

		if !state.isRunning {
			hasExited = true

			return nil
		}

		hostPort = state.hostPort

		return newAPIClient(hostPort).get(ctx, "/system/health", nil)
	})
	if err != nil {
		return nil, docker.withLogTail(t.Context(), spec.name, err)
	}

	if hasExited {
		return nil, docker.withLogTail(t.Context(), spec.name,
			fmt.Errorf("%w: %s from %s", errInstanceExited, spec.name, spec.image))
	}

	return newAPIClient(hostPort), nil
}
