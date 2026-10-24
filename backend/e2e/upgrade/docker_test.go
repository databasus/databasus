package upgrade

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"strconv"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

// scripts/test.sh exports a run ID, and its exit trap removes the containers, networks and volumes
// that carry it, so an interrupted full run leaves nothing of the upgrade test behind.
const (
	testRunIDVariable = "DATABASUS_TEST_RUN_ID"
	testRunLabel      = "databasus.test-run"
)

const (
	logTailLines       = "80"
	stopTimeoutSeconds = 30
)

// upgradeResources are the unique names of everything one upgrade flow creates in Docker.
type upgradeResources struct {
	networkName           string
	volumeName            string
	sourceContainerName   string
	previousContainerName string
	currentContainerName  string
}

func newUpgradeResources() (upgradeResources, error) {
	suffix := make([]byte, 6)

	_, err := rand.Read(suffix)
	if err != nil {
		return upgradeResources{}, fmt.Errorf("generate resource names: %w", err)
	}

	prefix := "databasus-upgrade-" + hex.EncodeToString(suffix)

	return upgradeResources{
		networkName:           prefix + "-network",
		volumeName:            prefix + "-data",
		sourceContainerName:   prefix + "-source",
		previousContainerName: prefix + "-previous",
		currentContainerName:  prefix + "-current",
	}, nil
}

func (r upgradeResources) getContainerNames() []string {
	return []string{r.sourceContainerName, r.previousContainerName, r.currentContainerName}
}

type dockerHost struct {
	client *client.Client
	labels map[string]string
}

func newDockerHost() (*dockerHost, error) {
	dockerClient, err := client.New(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("connect to Docker: %w", err)
	}

	labels := map[string]string{}

	//nolint:forbidigo // the run ID comes from scripts/test.sh, outside any configuration of ours
	runID, hasRunID := os.LookupEnv(testRunIDVariable)
	if hasRunID && runID != "" {
		labels[testRunLabel] = runID
	}

	return &dockerHost{client: dockerClient, labels: labels}, nil
}

func (d *dockerHost) close() {
	_ = d.client.Close()
}

func (d *dockerHost) createNetwork(ctx context.Context, networkName string) error {
	_, err := d.client.NetworkCreate(ctx, networkName, client.NetworkCreateOptions{Labels: d.labels})
	if err != nil {
		return fmt.Errorf("create network %s: %w", networkName, err)
	}

	return nil
}

func (d *dockerHost) createVolume(ctx context.Context, volumeName string) error {
	_, err := d.client.VolumeCreate(ctx, client.VolumeCreateOptions{Name: volumeName, Labels: d.labels})
	if err != nil {
		return fmt.Errorf("create volume %s: %w", volumeName, err)
	}

	return nil
}

type containerSpec struct {
	name       string
	image      string
	entrypoint []string
	env        []string
	// volumeMount is "<volume>:<path>", or empty for a container without the data volume.
	volumeMount  string
	networkName  string
	networkAlias string
	// publishedPort is the container port, such as "4005/tcp", published on a free loopback port.
	publishedPort string
}

func (d *dockerHost) startContainer(ctx context.Context, spec containerSpec) error {
	err := d.ensureImage(ctx, spec.image)
	if err != nil {
		return err
	}

	port, err := network.ParsePort(spec.publishedPort)
	if err != nil {
		return fmt.Errorf("parse port of container %s: %w", spec.name, err)
	}

	hostConfig := &container.HostConfig{
		NetworkMode: container.NetworkMode(spec.networkName),
		PortBindings: network.PortMap{
			port: {{HostIP: netip.MustParseAddr("127.0.0.1"), HostPort: ""}},
		},
	}
	if spec.volumeMount != "" {
		hostConfig.Binds = []string{spec.volumeMount}
	}

	_, err = d.client.ContainerCreate(ctx, client.ContainerCreateOptions{
		Name: spec.name,
		Config: &container.Config{
			Image:        spec.image,
			Entrypoint:   spec.entrypoint,
			Env:          spec.env,
			Labels:       d.labels,
			ExposedPorts: network.PortSet{port: {}},
		},
		HostConfig: hostConfig,
		NetworkingConfig: &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				spec.networkName: {Aliases: []string{spec.networkAlias}},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("create container %s from %s: %w", spec.name, spec.image, err)
	}

	_, err = d.client.ContainerStart(ctx, spec.name, client.ContainerStartOptions{})
	if err != nil {
		return fmt.Errorf("start container %s: %w", spec.name, err)
	}

	return nil
}

func (d *dockerHost) ensureImage(ctx context.Context, image string) error {
	_, err := d.client.ImageInspect(ctx, image)
	if err == nil {
		return nil
	}

	if !cerrdefs.IsNotFound(err) {
		return fmt.Errorf("inspect image %s: %w", image, err)
	}

	pullResponse, err := d.client.ImagePull(ctx, image, client.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("pull image %s: %w", image, err)
	}

	for message, messageErr := range pullResponse.JSONMessages(ctx) {
		if messageErr != nil {
			return fmt.Errorf("pull image %s: %w", image, messageErr)
		}

		if message.Error != nil {
			return fmt.Errorf("pull image %s: %s", image, message.Error.Message)
		}
	}

	return nil
}

// containerState is what a wait needs to know about a container: whether it still runs, and the
// loopback port its published port is reachable on while it does.
type containerState struct {
	isRunning bool
	hostPort  int
}

func (d *dockerHost) getContainerState(
	ctx context.Context,
	containerName, publishedPort string,
) (containerState, error) {
	inspection, err := d.client.ContainerInspect(ctx, containerName, client.ContainerInspectOptions{})
	if err != nil {
		return containerState{}, fmt.Errorf("inspect container %s: %w", containerName, err)
	}

	if inspection.Container.State == nil || !inspection.Container.State.Running {
		return containerState{isRunning: false, hostPort: 0}, nil
	}

	port, err := network.ParsePort(publishedPort)
	if err != nil {
		return containerState{}, fmt.Errorf("parse port of container %s: %w", containerName, err)
	}

	if inspection.Container.NetworkSettings == nil {
		return containerState{}, fmt.Errorf("container %s has no network settings yet", containerName)
	}

	bindings := inspection.Container.NetworkSettings.Ports[port]
	if len(bindings) == 0 {
		return containerState{}, fmt.Errorf("container %s has not published %s yet", containerName, publishedPort)
	}

	hostPort, err := strconv.Atoi(bindings[0].HostPort)
	if err != nil {
		return containerState{}, fmt.Errorf("read the published port of container %s: %w", containerName, err)
	}

	return containerState{isRunning: true, hostPort: hostPort}, nil
}

func (d *dockerHost) stopContainer(ctx context.Context, containerName string) error {
	timeoutSeconds := stopTimeoutSeconds

	_, err := d.client.ContainerStop(ctx, containerName, client.ContainerStopOptions{Timeout: &timeoutSeconds})
	if err != nil {
		return fmt.Errorf("stop container %s: %w", containerName, err)
	}

	return nil
}

// withLogTail adds the end of a container's log to err, because the log of the instance that failed
// is what explains an upgrade failure.
func (d *dockerHost) withLogTail(ctx context.Context, containerName string, err error) error {
	logs, logsErr := d.client.ContainerLogs(ctx, containerName, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       logTailLines,
	})
	if logsErr != nil {
		return fmt.Errorf("%w\n(the log of %s is unavailable: %w)", err, containerName, logsErr)
	}
	defer func() { _ = logs.Close() }()

	var logTail bytes.Buffer

	_, copyErr := stdcopy.StdCopy(&logTail, &logTail, logs)
	if copyErr != nil {
		return fmt.Errorf("%w\n(the log of %s is unreadable: %w)", err, containerName, copyErr)
	}

	return fmt.Errorf("%w\nend of the log of %s:\n%s", err, containerName, logTail.String())
}

// removeResources removes everything the flow may have created, skipping what does not exist.
func (d *dockerHost) removeResources(ctx context.Context, resources upgradeResources) error {
	var removeErrors []error

	for _, containerName := range resources.getContainerNames() {
		_, err := d.client.ContainerRemove(ctx, containerName, client.ContainerRemoveOptions{Force: true})
		if err != nil && !cerrdefs.IsNotFound(err) {
			removeErrors = append(removeErrors, fmt.Errorf("remove container %s: %w", containerName, err))
		}
	}

	_, err := d.client.NetworkRemove(ctx, resources.networkName, client.NetworkRemoveOptions{})
	if err != nil && !cerrdefs.IsNotFound(err) {
		removeErrors = append(removeErrors, fmt.Errorf("remove network %s: %w", resources.networkName, err))
	}

	_, err = d.client.VolumeRemove(ctx, resources.volumeName, client.VolumeRemoveOptions{Force: true})
	if err != nil && !cerrdefs.IsNotFound(err) {
		removeErrors = append(removeErrors, fmt.Errorf("remove volume %s: %w", resources.volumeName, err))
	}

	return errors.Join(removeErrors...)
}

// getRemainingResources names every resource of the flow that Docker still has.
func (d *dockerHost) getRemainingResources(ctx context.Context, resources upgradeResources) ([]string, error) {
	var remainingNames []string

	for _, containerName := range resources.getContainerNames() {
		_, err := d.client.ContainerInspect(ctx, containerName, client.ContainerInspectOptions{})
		if err == nil {
			remainingNames = append(remainingNames, containerName)
		} else if !cerrdefs.IsNotFound(err) {
			return nil, fmt.Errorf("inspect container %s: %w", containerName, err)
		}
	}

	_, err := d.client.NetworkInspect(ctx, resources.networkName, client.NetworkInspectOptions{})
	if err == nil {
		remainingNames = append(remainingNames, resources.networkName)
	} else if !cerrdefs.IsNotFound(err) {
		return nil, fmt.Errorf("inspect network %s: %w", resources.networkName, err)
	}

	_, err = d.client.VolumeInspect(ctx, resources.volumeName, client.VolumeInspectOptions{})
	if err == nil {
		remainingNames = append(remainingNames, resources.volumeName)
	} else if !cerrdefs.IsNotFound(err) {
		return nil, fmt.Errorf("inspect volume %s: %w", resources.volumeName, err)
	}

	return remainingNames, nil
}
