package containers

import (
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Any image with iproute2 fits; netshoot is multi-arch and maintained. The tag is pinned for the
// same reason MinioImage is.
const NetemSidecarImage = "nicolaka/netshoot:v0.14"

const netemSidecarTimeout = 60 * time.Second

const shapedProbeDialTimeout = 2 * time.Second

// AddEgressDelay delays every packet the container sends, so each round trip a test makes to it
// grows by delay, FTP passive data connections included. A host where shaping silently does
// nothing fails the test here instead of producing a fast, meaningless run.
func AddEgressDelay(t *testing.T, handle ContainerHandle, delay time.Duration) {
	t.Helper()

	runTrafficControl(t, handle, "qdisc", "add", "dev", "eth0", "root", "netem", "delay", formatNetemDuration(delay))

	handshakeDuration, err := MeasureTcpHandshake(t, handle)
	if err != nil {
		logContainerStateAndOutputTail(t, handle.Container)
		t.Fatalf("failed to dial the container after adding egress delay: %v", err)
	}

	if handshakeDuration < delay {
		t.Fatalf("egress delay of %s is not in effect: the TCP handshake took %s", delay, handshakeDuration)
	}
}

// SetEgressLoss drops everything the container sends, so it goes silent mid-connection without
// closing anything.
func SetEgressLoss(t *testing.T, handle ContainerHandle) {
	t.Helper()

	runTrafficControl(t, handle, "qdisc", "replace", "dev", "eth0", "root", "netem", "loss", "100%")
}

// A plain FIFO replaces whatever shaping is in place, so a test may also call this from a cleanup
// that runs after it already cleared the shaping itself.
func ClearEgressShaping(t *testing.T, handle ContainerHandle) {
	t.Helper()

	runTrafficControl(t, handle, "qdisc", "replace", "dev", "eth0", "root", "pfifo")
}

// The mapped host port is served by Docker's userland proxy, which completes the handshake itself
// before it reaches the container. Only a dial to the container's own address crosses the shaped
// interface during the handshake.
func MeasureTcpHandshake(t *testing.T, handle ContainerHandle) (time.Duration, error) {
	t.Helper()

	address := getContainerAddress(t, handle)
	startedAt := time.Now()

	dialer := net.Dialer{Timeout: shapedProbeDialTimeout}

	connection, err := dialer.DialContext(t.Context(), "tcp", address)
	if err != nil {
		return 0, err
	}

	handshakeDuration := time.Since(startedAt)
	_ = connection.Close()

	return handshakeDuration, nil
}

// A refused dial to a container that passed its readiness check means its server went away, and
// only the container's state and output say whether it crashed or was killed.
func logContainerStateAndOutputTail(t *testing.T, target testcontainers.Container) {
	t.Helper()

	state, err := target.State(context.Background())
	if err != nil {
		t.Logf("failed to read container state: %v", err)
	} else {
		t.Logf(
			"container status %s, exit code %d, OOM killed %t, error %q",
			state.Status, state.ExitCode, state.OOMKilled, state.Error,
		)
	}

	inspect, err := target.Inspect(context.Background())
	if err != nil {
		t.Logf("failed to inspect container: %v", err)
		return
	}

	logContainerOutputTail(t, target, inspect.Config.Image)
}

func getContainerAddress(t *testing.T, handle ContainerHandle) string {
	t.Helper()

	ctx := context.Background()

	inspect, err := handle.Container.Inspect(ctx)
	if err != nil {
		t.Fatalf("failed to inspect container: %v", err)
	}

	containerIP, err := handle.Container.ContainerIP(ctx)
	if err != nil {
		t.Fatalf("failed to get container IP: %v", err)
	}

	mappedPort := strconv.Itoa(handle.Port)

	for containerPort, bindings := range inspect.NetworkSettings.Ports {
		for _, binding := range bindings {
			if binding.HostPort == mappedPort {
				return net.JoinHostPort(containerIP, strconv.Itoa(int(containerPort.Num())))
			}
		}
	}

	t.Fatalf("no container port is published on host port %s", mappedPort)

	return ""
}

func runTrafficControl(t *testing.T, handle ContainerHandle, args ...string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), netemSidecarTimeout)
	defer cancel()

	sidecar, err := testcontainers.Run(
		ctx,
		NetemSidecarImage,
		testcontainers.WithCmd(append([]string{"tc"}, args...)...),
		testcontainers.WithHostConfigModifier(func(hostConfig *container.HostConfig) {
			hostConfig.NetworkMode = container.NetworkMode("container:" + handle.Container.GetContainerID())
			hostConfig.CapAdd = []string{"NET_ADMIN"}
		}),
		testcontainers.WithWaitStrategy(wait.ForExit().WithExitTimeout(netemSidecarTimeout)),
		testcontainers.WithLabels(withTestRunLabel(nil)),
	)
	if sidecar != nil {
		defer func() {
			if err := testcontainers.TerminateContainer(sidecar); err != nil {
				t.Logf("failed to terminate netem sidecar: %v", err)
			}
		}()
	}

	if err != nil {
		t.Fatalf("failed to run tc %v in a netem sidecar: %v", args, err)
	}

	state, err := sidecar.State(ctx)
	if err != nil {
		t.Fatalf("failed to read netem sidecar state: %v", err)
	}

	if state.ExitCode != 0 {
		t.Fatalf("tc %v exited with code %d: %s", args, state.ExitCode, readContainerLog(ctx, sidecar))
	}
}

func readContainerLog(ctx context.Context, target testcontainers.Container) string {
	logReader, err := target.Logs(ctx)
	if err != nil {
		return fmt.Sprintf("<failed to read log: %v>", err)
	}
	defer func() { _ = logReader.Close() }()

	logContent, err := io.ReadAll(logReader)
	if err != nil {
		return fmt.Sprintf("<failed to read log: %v>", err)
	}

	return string(logContent)
}

func formatNetemDuration(delay time.Duration) string {
	return strconv.FormatInt(delay.Microseconds(), 10) + "us"
}
