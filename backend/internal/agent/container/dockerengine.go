// Package container owns the ephemeral Postgres container lifecycle. Every
// Docker-SDK call is confined to this file; the Manager builds a hardened
// SpawnSpec (unit-tested) that this file translates 1:1 to Docker.
package container

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/netip"
	"slices"
	"strconv"
	"strings"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/system"
	"github.com/moby/moby/client"

	"databasus-backend/internal/agent/restore"
	"databasus-backend/internal/lib/background"
)

const (
	// LabelVerificationID / LabelAgentID tag every container and network so the
	// startup purge can remove this agent's leftovers by agent ID, and a retried
	// attempt can remove those of its verification by both labels
	LabelVerificationID = "databasus.verification.verification_id"
	LabelAgentID        = "databasus.verification.agent_id"

	containerPidsLimit = 512

	// pgInternalPort is the in-container Postgres port; it is published to an
	// ephemeral 127.0.0.1 host port for the verifier.
	pgInternalPort = 5432
)

func getPgPublishedPort() network.Port { return mustParsePort(strconv.Itoa(pgInternalPort) + "/tcp") }

func mustParsePort(s string) network.Port {
	p, err := network.ParsePort(s)
	if err != nil {
		panic(fmt.Sprintf("parse port %q: %v", s, err))
	}

	return p
}

type DockerEngine struct {
	cli *client.Client
	log *slog.Logger
}

func NewDockerEngine(log *slog.Logger) (*DockerEngine, error) {
	cli, err := client.New(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("create docker client: %w", err)
	}

	return &DockerEngine{cli: cli, log: log}, nil
}

func (e *DockerEngine) Ping(ctx context.Context) error {
	_, err := e.cli.Ping(ctx, client.PingOptions{})
	return err
}

func (e *DockerEngine) GetHostCapabilities(ctx context.Context) (HostCapabilities, error) {
	result, err := e.cli.Info(ctx, client.InfoOptions{})
	if err != nil {
		return HostCapabilities{}, err
	}

	return toHostCapabilities(result.Info), nil
}

// toHostCapabilities requires both CFS flags because that is the daemon's own
// condition for accepting NanoCPUs; older daemons track them separately.
func toHostCapabilities(info system.Info) HostCapabilities {
	isUserNSRemapEnabled := slices.ContainsFunc(info.SecurityOptions, func(opt string) bool {
		return strings.Contains(opt, "name=userns")
	})

	return HostCapabilities{
		IsUserNSRemapEnabled: isUserNSRemapEnabled,
		IsCPULimitSupported:  info.CPUCfsPeriod && info.CPUCfsQuota,
	}
}

func (e *DockerEngine) EnsureImage(ctx context.Context, ref string) error {
	if _, err := e.cli.ImageInspect(ctx, ref); err == nil {
		return nil
	}

	reader, err := e.cli.ImagePull(ctx, ref, client.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("image pull: %w", err)
	}
	defer func() { _ = reader.Close() }()

	if _, err := io.Copy(io.Discard, reader); err != nil {
		return fmt.Errorf("drain image pull: %w", err)
	}

	return nil
}

func (e *DockerEngine) CreateNetwork(
	ctx context.Context, name string, labels map[string]string,
) (string, error) {
	// Each job gets its own user-defined bridge — that isolates jobs from each
	// other. We deliberately do NOT set Internal:true: Docker silently disables
	// port publishing on internal networks, and the agent-process verifier can
	// only reach the restored DB via the published host port (verifier package
	// doc). Outbound from PG to the host network is therefore not blocked at
	// the docker layer; rely on the host firewall + the other hardening
	// controls (unprivileged user, no capabilities, pids/memory, ephemeral,
	// userns).
	resp, err := e.cli.NetworkCreate(ctx, name, client.NetworkCreateOptions{
		Driver: "bridge",
		Labels: labels,
	})
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

func (e *DockerEngine) RemoveNetwork(ctx context.Context, networkID string) error {
	_, err := e.cli.NetworkRemove(ctx, networkID, client.NetworkRemoveOptions{})
	return err
}

func (e *DockerEngine) CreateContainer(ctx context.Context, spec SpawnSpec) (string, error) {
	config, hostConfig := toContainerConfigs(spec)

	resp, err := e.cli.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config:     config,
		HostConfig: hostConfig,
		Name:       spec.Name,
	})
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

// toContainerConfigs is the 1:1 translation of a SpawnSpec into the engine's
// create request; a unit test pins the hardening it carries.
func toContainerConfigs(spec SpawnSpec) (*container.Config, *container.HostConfig) {
	capDrop := []string{}
	if spec.CapDropAll {
		capDrop = append(capDrop, "ALL")
	}

	var pidsLimit *int64
	if spec.PidsLimit > 0 {
		limit := spec.PidsLimit
		pidsLimit = &limit
	}

	hostConfig := &container.HostConfig{
		CapDrop:        capDrop,
		ReadonlyRootfs: spec.ReadonlyRootfs,
		NetworkMode:    container.NetworkMode(spec.NetworkID),
		PortBindings: network.PortMap{
			getPgPublishedPort(): []network.PortBinding{
				{HostIP: netip.MustParseAddr("127.0.0.1"), HostPort: "0"},
			},
		},
		Resources: container.Resources{
			NanoCPUs:  spec.NanoCPUs,
			Memory:    spec.MemoryBytes,
			PidsLimit: pidsLimit,
		},
	}

	config := &container.Config{
		Image:        spec.Image,
		User:         spec.User,
		Cmd:          spec.Cmd,
		Env:          spec.Env,
		Labels:       spec.Labels,
		ExposedPorts: network.PortSet{getPgPublishedPort(): struct{}{}},
	}

	return config, hostConfig
}

func (e *DockerEngine) StartContainer(ctx context.Context, containerID string) error {
	_, err := e.cli.ContainerStart(ctx, containerID, client.ContainerStartOptions{})
	return err
}

func (e *DockerEngine) HostPort(
	ctx context.Context, containerID, containerPort string,
) (int, error) {
	insp, err := e.cli.ContainerInspect(ctx, containerID, client.ContainerInspectOptions{})
	if err != nil {
		return 0, err
	}

	requestedPort, err := network.ParsePort(containerPort)
	if err != nil {
		return 0, fmt.Errorf("parse container port %q: %w", containerPort, err)
	}

	bindings, ok := insp.Container.NetworkSettings.Ports[requestedPort]
	if !ok || len(bindings) == 0 {
		return 0, fmt.Errorf("container port %s is not published", containerPort)
	}

	return strconv.Atoi(bindings[0].HostPort)
}

func (e *DockerEngine) Exec(
	ctx context.Context,
	containerID string,
	cmd []string,
	stdin io.Reader,
	env []string,
) (restore.ExecResult, error) {
	created, err := e.cli.ExecCreate(ctx, containerID, toExecOptions(cmd, env, stdin != nil))
	if err != nil {
		return restore.ExecResult{}, fmt.Errorf("exec create: %w", err)
	}

	attached, err := e.cli.ExecAttach(ctx, created.ID, client.ExecAttachOptions{})
	if err != nil {
		return restore.ExecResult{}, fmt.Errorf("exec attach: %w", err)
	}
	defer attached.Close()

	// The hijacked conn's Read/Write are not ctx-aware: a container that wedges
	// while consuming stdin (or producing output) would otherwise hang Exec
	// indefinitely, defeating jobCtx/heartbeat-abort. Closing it on cancellation
	// unblocks the in-flight copy; the deferred Close above is then a harmless
	// second close.
	stopOnCancel := context.AfterFunc(ctx, func() { attached.Close() })
	defer stopOnCancel()

	if stdin != nil {
		copyErr := make(chan error, 1)
		group := background.New(background.Deps{Logger: e.log})

		group.Go(ctx, "container_exec_stdin", func(ctx context.Context) error {
			err := background.RunRecovering(ctx, func(context.Context) error {
				_, err := io.Copy(attached.Conn, stdin)
				return err
			})
			_ = attached.CloseWrite()
			copyErr <- err
			return err
		})

		_ = group.Wait(context.WithoutCancel(ctx))

		if cErr := <-copyErr; cErr != nil {
			return restore.ExecResult{}, fmt.Errorf("exec stdin copy: %w", cErr)
		}
	}

	var stdout, stderr bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdout, &stderr, attached.Reader); err != nil {
		return restore.ExecResult{}, fmt.Errorf("exec stream copy: %w", err)
	}

	inspect, err := e.cli.ExecInspect(ctx, created.ID, client.ExecInspectOptions{})
	if err != nil {
		return restore.ExecResult{}, fmt.Errorf("exec inspect: %w", err)
	}

	return restore.ExecResult{
		ExitCode: inspect.ExitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}, nil
}

// toExecOptions builds the engine's exec request. It names no user, so the
// command runs as the container's user, the same unprivileged user the
// container starts as; a unit test pins that.
func toExecOptions(cmd, env []string, isStdinAttached bool) client.ExecCreateOptions {
	return client.ExecCreateOptions{
		Cmd:          cmd,
		Env:          env,
		AttachStdin:  isStdinAttached,
		AttachStdout: true,
		AttachStderr: true,
	}
}

func (e *DockerEngine) RemoveContainer(ctx context.Context, containerID string) error {
	_, err := e.cli.ContainerRemove(ctx, containerID, client.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
	return err
}

func (e *DockerEngine) ListManagedContainers(
	ctx context.Context, labels map[string]string,
) ([]ManagedResource, error) {
	listed, err := e.cli.ContainerList(ctx, client.ContainerListOptions{
		All:     true,
		Filters: labelFilters(labels),
	})
	if err != nil {
		return nil, err
	}

	managed := make([]ManagedResource, 0, len(listed.Items))
	for _, s := range listed.Items {
		managed = append(managed, ManagedResource{ID: s.ID, VerificationID: s.Labels[LabelVerificationID]})
	}

	return managed, nil
}

func (e *DockerEngine) ListManagedNetworks(
	ctx context.Context, labels map[string]string,
) ([]ManagedResource, error) {
	listed, err := e.cli.NetworkList(ctx, client.NetworkListOptions{Filters: labelFilters(labels)})
	if err != nil {
		return nil, err
	}

	managed := make([]ManagedResource, 0, len(listed.Items))
	for _, n := range listed.Items {
		managed = append(managed, ManagedResource{ID: n.ID, VerificationID: n.Labels[LabelVerificationID]})
	}

	return managed, nil
}

// labelFilters passes every label as its own value of the label filter, which
// Docker combines with AND; one value holding several labels would match none.
func labelFilters(labels map[string]string) client.Filters {
	filters := make(client.Filters)
	for key, value := range labels {
		filters.Add("label", key+"="+value)
	}

	return filters
}
