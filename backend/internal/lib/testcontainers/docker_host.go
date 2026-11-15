package containers

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"path"
	"strconv"
	"time"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

const (
	logTailLines       = "80"
	stopTimeoutSeconds = 30
)

// DockerHost drives Docker directly for the suites that run Databasus images as an operator does:
// the upgrade test and the recovery cycle of Databasus backups. Every network, volume and container
// it creates carries the run labels, so scripts/test.sh removes what an interrupted run leaves.
type DockerHost struct {
	client *client.Client
	labels map[string]string
}

func NewDockerHost() (*DockerHost, error) {
	dockerClient, err := client.New(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("connect to Docker: %w", err)
	}

	labels, err := addTestRunLabel(map[string]string{})
	if err != nil {
		_ = dockerClient.Close()

		return nil, err
	}

	return &DockerHost{client: dockerClient, labels: labels}, nil
}

func (d *DockerHost) Close() {
	_ = d.client.Close()
}

// GetClient is the Docker client, for a suite that needs an operation this type does not wrap.
func (d *DockerHost) GetClient() *client.Client {
	return d.client
}

// GetLabels are the labels a suite gives a container it creates through GetClient.
func (d *DockerHost) GetLabels() map[string]string {
	return d.labels
}

// CreateNetwork takes the network's subnet from the test network range, so the default address pools
// stay free for the verification agents, which cannot choose theirs.
func (d *DockerHost) CreateNetwork(ctx context.Context, networkName string) error {
	err := WithFreeSubnet(func(ipam *network.IPAM) error {
		_, createErr := d.client.NetworkCreate(
			ctx,
			networkName,
			client.NetworkCreateOptions{Labels: d.labels, IPAM: ipam},
		)

		return createErr
	})
	if err != nil {
		return fmt.Errorf("create network %s: %w", networkName, err)
	}

	inspected, err := d.client.NetworkInspect(ctx, networkName, client.NetworkInspectOptions{})
	if err != nil {
		return fmt.Errorf("inspect network %s: %w", networkName, err)
	}

	for _, config := range inspected.Network.IPAM.Config {
		if !IsTestNetworkSubnet(config.Subnet) {
			return fmt.Errorf("network %s has subnet %s outside the test network range", networkName, config.Subnet)
		}
	}

	return nil
}

func (d *DockerHost) CreateVolume(ctx context.Context, volumeName string) error {
	_, err := d.client.VolumeCreate(ctx, client.VolumeCreateOptions{Name: volumeName, Labels: d.labels})
	if err != nil {
		return fmt.Errorf("create volume %s: %w", volumeName, err)
	}

	return nil
}

// RunSpec is a container DockerHost creates on one of its networks.
type RunSpec struct {
	Name       string
	Image      string
	Entrypoint []string
	Env        []string
	// Mount is "<volume or host directory>:<path>", or empty for a container without data.
	Mount        string
	NetworkName  string
	NetworkAlias string
	// PublishedPort is the container port, such as "4005/tcp", published on a free loopback port, or
	// empty for a container no test reaches from the host.
	PublishedPort string
}

// CreateContainer creates the container without starting it, so files can be copied into its
// mount first.
func (d *DockerHost) CreateContainer(ctx context.Context, spec RunSpec) error {
	err := d.EnsureImage(ctx, spec.Image)
	if err != nil {
		return err
	}

	hostConfig := &container.HostConfig{NetworkMode: container.NetworkMode(spec.NetworkName)}
	config := &container.Config{
		Image:      spec.Image,
		Entrypoint: spec.Entrypoint,
		Env:        spec.Env,
		Labels:     d.labels,
	}

	if spec.PublishedPort != "" {
		port, parseErr := network.ParsePort(spec.PublishedPort)
		if parseErr != nil {
			return fmt.Errorf("parse port of container %s: %w", spec.Name, parseErr)
		}

		hostConfig.PortBindings = network.PortMap{
			port: {{HostIP: netip.MustParseAddr("127.0.0.1"), HostPort: ""}},
		}
		config.ExposedPorts = network.PortSet{port: {}}
	}

	if spec.Mount != "" {
		hostConfig.Binds = []string{spec.Mount}
	}

	_, err = d.client.ContainerCreate(ctx, client.ContainerCreateOptions{
		Name:       spec.Name,
		Config:     config,
		HostConfig: hostConfig,
		NetworkingConfig: &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				spec.NetworkName: {Aliases: []string{spec.NetworkAlias}},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("create container %s from %s: %w", spec.Name, spec.Image, err)
	}

	return nil
}

func (d *DockerHost) StartCreatedContainer(ctx context.Context, containerName string) error {
	_, err := d.client.ContainerStart(ctx, containerName, client.ContainerStartOptions{})
	if err != nil {
		return fmt.Errorf("start container %s: %w", containerName, err)
	}

	return nil
}

func (d *DockerHost) StartContainer(ctx context.Context, spec RunSpec) error {
	err := d.CreateContainer(ctx, spec)
	if err != nil {
		return err
	}

	return d.StartCreatedContainer(ctx, spec.Name)
}

func (d *DockerHost) EnsureImage(ctx context.Context, image string) error {
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

// ContainerState is what a wait needs to know about a container: whether it still runs, the
// loopback port its published port is reachable on while it does, and its exit code once it stopped.
type ContainerState struct {
	IsRunning bool
	HostPort  int
	ExitCode  int
	// StartedAt changes only when Docker starts the container again.
	StartedAt string
}

func (d *DockerHost) GetContainerState(
	ctx context.Context,
	containerName, publishedPort string,
) (ContainerState, error) {
	inspection, err := d.client.ContainerInspect(ctx, containerName, client.ContainerInspectOptions{})
	if err != nil {
		return ContainerState{}, fmt.Errorf("inspect container %s: %w", containerName, err)
	}

	state := inspection.Container.State
	if state == nil || !state.Running {
		exitCode := 0
		if state != nil {
			exitCode = state.ExitCode
		}

		return ContainerState{IsRunning: false, HostPort: 0, ExitCode: exitCode, StartedAt: ""}, nil
	}

	port, err := network.ParsePort(publishedPort)
	if err != nil {
		return ContainerState{}, fmt.Errorf("parse port of container %s: %w", containerName, err)
	}

	if inspection.Container.NetworkSettings == nil {
		return ContainerState{}, fmt.Errorf("container %s has no network settings yet", containerName)
	}

	bindings := inspection.Container.NetworkSettings.Ports[port]
	if len(bindings) == 0 {
		return ContainerState{}, fmt.Errorf("container %s has not published %s yet", containerName, publishedPort)
	}

	hostPort, err := strconv.Atoi(bindings[0].HostPort)
	if err != nil {
		return ContainerState{}, fmt.Errorf("read the published port of container %s: %w", containerName, err)
	}

	return ContainerState{IsRunning: true, HostPort: hostPort, ExitCode: 0, StartedAt: state.StartedAt}, nil
}

func (d *DockerHost) StopContainer(ctx context.Context, containerName string) error {
	timeoutSeconds := stopTimeoutSeconds

	_, err := d.client.ContainerStop(ctx, containerName, client.ContainerStopOptions{Timeout: &timeoutSeconds})
	if err != nil {
		return fmt.Errorf("stop container %s: %w", containerName, err)
	}

	return nil
}

// WaitForExit waits until the container stopped by itself and returns its exit code.
func (d *DockerHost) WaitForExit(ctx context.Context, containerName string, limit time.Duration) (int, error) {
	waitCtx, cancelWait := context.WithTimeout(ctx, limit)
	defer cancelWait()

	waited := d.client.ContainerWait(waitCtx, containerName, client.ContainerWaitOptions{
		Condition: container.WaitConditionNotRunning,
	})

	select {
	case response := <-waited.Result:
		return int(response.StatusCode), nil
	case err := <-waited.Error:
		return 0, fmt.Errorf("wait for container %s to exit: %w", containerName, err)
	}
}

// GetLogs returns everything the container wrote to its standard output and error.
func (d *DockerHost) GetLogs(ctx context.Context, containerName string) (string, error) {
	return d.readLogs(ctx, containerName, "all")
}

// WithLogTail adds the end of a container's log to err, because the log of the instance that failed
// is what explains the failure.
func (d *DockerHost) WithLogTail(ctx context.Context, containerName string, err error) error {
	logTail, logsErr := d.readLogs(ctx, containerName, logTailLines)
	if logsErr != nil {
		return fmt.Errorf("%w\n(the log of %s is unavailable: %w)", err, containerName, logsErr)
	}

	return fmt.Errorf("%w\nend of the log of %s:\n%s", err, containerName, logTail)
}

// Execute runs the command in the running container and returns its standard output. A command
// that exits with another code than zero is an error carrying its standard error.
func (d *DockerHost) Execute(ctx context.Context, containerName string, command []string) (string, error) {
	created, err := d.client.ExecCreate(ctx, containerName, client.ExecCreateOptions{
		Cmd: command, AttachStdout: true, AttachStderr: true,
	})
	if err != nil {
		return "", fmt.Errorf("create exec %q in %s: %w", command, containerName, err)
	}

	attached, err := d.client.ExecAttach(ctx, created.ID, client.ExecAttachOptions{})
	if err != nil {
		return "", fmt.Errorf("attach exec %q in %s: %w", command, containerName, err)
	}
	defer attached.Close()

	var standardOutput, standardError bytes.Buffer

	_, err = stdcopy.StdCopy(&standardOutput, &standardError, attached.Reader)
	if err != nil {
		return "", fmt.Errorf("read exec %q in %s: %w", command, containerName, err)
	}

	inspection, err := d.client.ExecInspect(ctx, created.ID, client.ExecInspectOptions{})
	if err != nil {
		return "", fmt.Errorf("inspect exec %q in %s: %w", command, containerName, err)
	}

	if inspection.ExitCode != 0 {
		return "", fmt.Errorf("exec %q in %s exited with code %d: %s",
			command, containerName, inspection.ExitCode, standardError.String())
	}

	return standardOutput.String(), nil
}

// FileOwner is the numeric owner a file copied into a container gets.
type FileOwner struct {
	UID int
	GID int
}

// CopyFileToContainer writes content to filePath inside the container, which may be stopped,
// creating the file's directory. A path inside a mounted volume lands in that volume.
func (d *DockerHost) CopyFileToContainer(
	ctx context.Context,
	containerName, filePath string,
	content []byte,
	owner FileOwner,
) error {
	var archive bytes.Buffer

	writer := tar.NewWriter(&archive)
	directory := path.Dir(filePath)

	err := errors.Join(
		writer.WriteHeader(&tar.Header{
			Typeflag: tar.TypeDir,
			Name:     path.Base(directory) + "/",
			Mode:     0o700,
			Uid:      owner.UID,
			Gid:      owner.GID,
		}),
		writer.WriteHeader(&tar.Header{
			Typeflag: tar.TypeReg,
			Name:     path.Base(directory) + "/" + path.Base(filePath),
			Mode:     0o600,
			Size:     int64(len(content)),
			Uid:      owner.UID,
			Gid:      owner.GID,
		}),
	)
	if err != nil {
		return fmt.Errorf("pack %s: %w", filePath, err)
	}

	_, err = writer.Write(content)
	if err := errors.Join(err, writer.Close()); err != nil {
		return fmt.Errorf("pack %s: %w", filePath, err)
	}

	_, err = d.client.CopyToContainer(ctx, containerName, client.CopyToContainerOptions{
		DestinationPath: path.Dir(directory),
		Content:         &archive,
	})
	if err != nil {
		return fmt.Errorf("copy %s into container %s: %w", filePath, containerName, err)
	}

	return nil
}

// ReadFileFromContainer returns the content of one regular file of the container, which may be
// stopped.
func (d *DockerHost) ReadFileFromContainer(ctx context.Context, containerName, filePath string) ([]byte, error) {
	copied, err := d.client.CopyFromContainer(ctx, containerName, client.CopyFromContainerOptions{
		SourcePath: filePath,
	})
	if err != nil {
		return nil, fmt.Errorf("copy %s from container %s: %w", filePath, containerName, err)
	}
	defer func() { _ = copied.Content.Close() }()

	archive := tar.NewReader(copied.Content)

	header, err := archive.Next()
	if err != nil {
		return nil, fmt.Errorf("read %s from container %s: %w", filePath, containerName, err)
	}

	if header.Typeflag != tar.TypeReg {
		return nil, fmt.Errorf("%s in container %s is not a regular file", filePath, containerName)
	}

	content, err := io.ReadAll(archive)
	if err != nil {
		return nil, fmt.Errorf("read %s from container %s: %w", filePath, containerName, err)
	}

	return content, nil
}

// RemoveContainer removes the container with its anonymous volumes, and nothing when it is gone.
func (d *DockerHost) RemoveContainer(ctx context.Context, containerName string) error {
	_, err := d.client.ContainerRemove(ctx, containerName, client.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
	if err != nil && !cerrdefs.IsNotFound(err) {
		return fmt.Errorf("remove container %s: %w", containerName, err)
	}

	return nil
}

func (d *DockerHost) RemoveNetwork(ctx context.Context, networkName string) error {
	_, err := d.client.NetworkRemove(ctx, networkName, client.NetworkRemoveOptions{})
	if err != nil && !cerrdefs.IsNotFound(err) {
		return fmt.Errorf("remove network %s: %w", networkName, err)
	}

	return nil
}

func (d *DockerHost) RemoveVolume(ctx context.Context, volumeName string) error {
	_, err := d.client.VolumeRemove(ctx, volumeName, client.VolumeRemoveOptions{Force: true})
	if err != nil && !cerrdefs.IsNotFound(err) {
		return fmt.Errorf("remove volume %s: %w", volumeName, err)
	}

	return nil
}

// DockerResources names containers, networks and volumes of one suite.
type DockerResources struct {
	ContainerNames []string
	NetworkNames   []string
	VolumeNames    []string
}

// RemoveResources removes every named resource, containers first, skipping what does not exist.
func (d *DockerHost) RemoveResources(ctx context.Context, resources DockerResources) error {
	var removeErrors []error

	for _, containerName := range resources.ContainerNames {
		removeErrors = append(removeErrors, d.RemoveContainer(ctx, containerName))
	}

	for _, networkName := range resources.NetworkNames {
		removeErrors = append(removeErrors, d.RemoveNetwork(ctx, networkName))
	}

	for _, volumeName := range resources.VolumeNames {
		removeErrors = append(removeErrors, d.RemoveVolume(ctx, volumeName))
	}

	return errors.Join(removeErrors...)
}

// GetRemainingResources names every resource Docker still has.
func (d *DockerHost) GetRemainingResources(ctx context.Context, resources DockerResources) ([]string, error) {
	var remainingNames []string

	for _, containerName := range resources.ContainerNames {
		_, err := d.client.ContainerInspect(ctx, containerName, client.ContainerInspectOptions{})
		if err == nil {
			remainingNames = append(remainingNames, containerName)
		} else if !cerrdefs.IsNotFound(err) {
			return nil, fmt.Errorf("inspect container %s: %w", containerName, err)
		}
	}

	for _, networkName := range resources.NetworkNames {
		_, err := d.client.NetworkInspect(ctx, networkName, client.NetworkInspectOptions{})
		if err == nil {
			remainingNames = append(remainingNames, networkName)
		} else if !cerrdefs.IsNotFound(err) {
			return nil, fmt.Errorf("inspect network %s: %w", networkName, err)
		}
	}

	for _, volumeName := range resources.VolumeNames {
		_, err := d.client.VolumeInspect(ctx, volumeName, client.VolumeInspectOptions{})
		if err == nil {
			remainingNames = append(remainingNames, volumeName)
		} else if !cerrdefs.IsNotFound(err) {
			return nil, fmt.Errorf("inspect volume %s: %w", volumeName, err)
		}
	}

	return remainingNames, nil
}

func (d *DockerHost) readLogs(ctx context.Context, containerName, tail string) (string, error) {
	logs, err := d.client.ContainerLogs(ctx, containerName, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
	})
	if err != nil {
		return "", fmt.Errorf("read the log of %s: %w", containerName, err)
	}
	defer func() { _ = logs.Close() }()

	var content bytes.Buffer

	_, err = stdcopy.StdCopy(&content, &content, logs)
	if err != nil {
		return "", fmt.Errorf("read the log of %s: %w", containerName, err)
	}

	return content.String(), nil
}
