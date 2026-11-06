// Package containers spins up throwaway database containers for integration tests via
// testcontainers-go. Each Start* helper boots one container, waits until it is ready and registers
// a t.Cleanup that terminates it, so a container lives only for the duration of the test that
// created it. Each engine gets its own file; this file holds the shared plumbing.
package containers

import (
	"context"
	"io"
	"maps"
	"strconv"
	"strings"
	"testing"

	dockernetwork "github.com/moby/moby/api/types/network"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"

	"databasus-backend/internal/lib/config"
)

// dataDirTmpfsOptions mounts a container's data directory on tmpfs (RAM) instead of the overlay
// filesystem, so the fsync-heavy cold init of the SQL engines is RAM-fast. The size is pinned
// because Docker's tmpfs default is half the host RAM, which is unsafe to reserve per container
// under go test -p=N; 512m dwarfs every test fixture and tmpfs only consumes the bytes written.
const dataDirTmpfsOptions = "rw,size=512m"

const loopbackIP = "127.0.0.1"

// One retry covers a transient boot failure, a second covers a back-to-back repeat under the same
// load spike. More would only delay the report of a genuinely broken image.
const containerBootAttempts = 3

const failedContainerOutputTailBytes = 8 * 1024

// scripts/test.sh exports a run ID, and its exit trap removes only the containers and networks that
// carry it, so an interrupted full run never removes another run's containers.
const (
	testRunIDVariable = "DATABASUS_TEST_RUN_ID"
	testRunLabel      = "databasus.test-run"
)

// Endpoint is the reachable address of a started container's primary port.
type Endpoint struct {
	Host string
	Port int
}

// ContainerHandle is an Endpoint plus the live container, for tests that must Exec or copy files
// into the container (e.g. the physical restore target reconstructs a cluster in place) rather than
// only dial its port.
type ContainerHandle struct {
	Endpoint
	Container testcontainers.Container
}

// For tests that need one container reachable only from another rather than from the host.
func StartNetwork(t *testing.T) string {
	t.Helper()

	testNetwork, err := newLabelledNetwork(t)
	if err != nil {
		t.Fatalf("failed to create test network: %v", err)
	}

	t.Cleanup(func() {
		if err := testNetwork.Remove(context.WithoutCancel(t.Context())); err != nil {
			t.Logf("failed to remove test network: %v", err)
		}
	})

	return testNetwork.Name
}

// Publishing no ports leaves the server unreachable from the host, dialable only by another
// container on the network. The Endpoint each Start*OnNetwork returns is that in-network
// address, which is what an SSH bastion on the same network resolves.
type OnNetworkSpec struct {
	Image     string
	Placement NetworkPlacement
}

type NetworkPlacement struct {
	NetworkName string
	Alias       string
}

// The engine port consts are testcontainers-shaped ("5432/tcp"); an unpublished container is dialed
// by that raw number rather than by a mapped one.
func getPortNumber(exposedPort string) int {
	number, _, _ := strings.Cut(exposedPort, "/")

	port, err := strconv.Atoi(number)
	if err != nil {
		panic("malformed exposed port " + exposedPort)
	}

	return port
}

func startUnpublished(
	t *testing.T,
	req testcontainers.ContainerRequest,
	placement NetworkPlacement,
) testcontainers.Container {
	t.Helper()

	req.ExposedPorts = nil
	req.Networks = []string{placement.NetworkName}
	req.NetworkAliases = map[string][]string{placement.NetworkName: {placement.Alias}}

	return bootContainer(t, req)
}

func start(t *testing.T, req testcontainers.ContainerRequest, mappedPort string) Endpoint {
	t.Helper()

	return startContainer(t, req, mappedPort).Endpoint
}

func startContainer(t *testing.T, req testcontainers.ContainerRequest, mappedPort string) ContainerHandle {
	t.Helper()

	container := bootContainer(t, req)

	return ContainerHandle{Endpoint: endpointOf(t, container, mappedPort), Container: container}
}

// A boot can fail for reasons outside the test under CPU contention: the mongo entrypoint, for one,
// starts mongod twice and the second start sometimes cannot bind its port and exits with code 48.
// Each failed attempt is logged, so an image that keeps needing a retry still shows up in the output.
func bootContainer(t *testing.T, req testcontainers.ContainerRequest) testcontainers.Container {
	t.Helper()
	skipUnavailableImage(t, req.Image)
	imageName := req.Image
	if imageName == "" {
		imageName = req.Repo + ":" + req.Tag
	}

	var lastErr error

	req.Labels = withTestRunLabel(t, req.Labels)

	for attempt := 1; attempt <= containerBootAttempts; attempt++ {
		container, err := testcontainers.GenericContainer(t.Context(), testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		if err == nil {
			t.Cleanup(func() {
				// The container's data is thrown away with it, so a graceful stop only costs time.
				if err := container.Terminate(
					context.WithoutCancel(t.Context()),
					testcontainers.StopTimeout(0),
				); err != nil {
					t.Logf("failed to terminate %s container: %v", imageName, err)
				}
			})

			return container
		}

		lastErr = err
		t.Logf("attempt %d/%d to start %s container failed: %v", attempt, containerBootAttempts, imageName, err)
		logContainerOutputTail(t, container, imageName)

		// GenericContainer hands back the half-started container alongside the error.
		if err := testcontainers.TerminateContainer(container); err != nil {
			t.Logf("failed to terminate %s container after a failed start: %v", imageName, err)
		}
	}

	t.Fatalf("failed to start %s container after %d attempts: %v", imageName, containerBootAttempts, lastErr)

	return nil
}

func newLabelledNetwork(t *testing.T) (*testcontainers.DockerNetwork, error) {
	t.Helper()
	var createdNetwork *testcontainers.DockerNetwork

	err := WithFreeSubnet(func(ipam *dockernetwork.IPAM) error {
		var err error
		createdNetwork, err = network.New(
			t.Context(),
			network.WithLabels(withTestRunLabel(t, nil)),
			network.WithIPAM(ipam),
		)

		return err
	})

	return createdNetwork, err
}

func withTestRunLabel(t *testing.T, labels map[string]string) map[string]string {
	t.Helper()
	variables, _, err := config.ReadVariables(config.LoadSpec{})
	if err != nil {
		t.Fatalf("read test run labels: %v", err)
	}
	testRunID := variables[testRunIDVariable]
	if testRunID == "" {
		return labels
	}

	labelled := maps.Clone(labels)
	if labelled == nil {
		labelled = map[string]string{}
	}

	labelled[testRunLabel] = testRunID

	return labelled
}

// A container that exits during boot surfaces only as "exited with code N"; the reason sits in its
// own output, which is gone once the container is terminated.
func logContainerOutputTail(t *testing.T, container testcontainers.Container, image string) {
	t.Helper()

	if container == nil {
		return
	}

	logs, err := container.Logs(context.WithoutCancel(t.Context()))
	if err != nil {
		t.Logf("failed to read %s container output: %v", image, err)
		return
	}
	defer func() { _ = logs.Close() }()

	output, err := io.ReadAll(logs)
	if err != nil {
		t.Logf("failed to read %s container output: %v", image, err)
		return
	}

	if len(output) > failedContainerOutputTailBytes {
		output = output[len(output)-failedContainerOutputTailBytes:]
	}

	t.Logf("%s container output tail:\n%s", image, output)
}

// The MySQL client resolves the name "localhost" to a unix socket rather than TCP, and the server
// lives in a container where no such socket exists.
func endpointOf(t *testing.T, container testcontainers.Container, mappedPort string) Endpoint {
	t.Helper()

	ctx := context.WithoutCancel(t.Context())

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get container host: %v", err)
	}

	if host == "localhost" {
		host = loopbackIP
	}

	port, err := container.MappedPort(ctx, mappedPort)
	if err != nil {
		t.Fatalf("failed to get container mapped port: %v", err)
	}

	return Endpoint{Host: host, Port: int(port.Num())}
}
