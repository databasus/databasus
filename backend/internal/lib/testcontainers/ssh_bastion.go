package containers

import (
	"net"
	"net/netip"
	"strconv"
	"testing"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Credentials baked into the test bastion image.
const (
	SSHBastionUsername = "testuser"
	SSHBastionPassword = "testpassword"
)

const sshBastionPort = "22/tcp"

// The image is built from contextDir rather than pulled because the stock openssh images disagree
// on whether AllowTcpForwarding defaults to yes, and the baked-in authorized_keys has to match the
// private key the test signs with.
func StartSSHBastion(t *testing.T) Endpoint {
	t.Helper()

	return start(t, sshBastionRequest(t), sshBastionPort)
}

// Port 22 stays published so the test can dial the bastion; the network membership is what lets the
// bastion in turn reach containers that publish nothing. The container comes back too, so a test can
// stop and restart the bastion under a running tunnel.
func StartSSHBastionContainerOnNetwork(t *testing.T, networkName string) ContainerHandle {
	t.Helper()

	req := sshBastionRequest(t)
	req.Networks = []string{networkName}
	pinBastionHostPort(t, &req)

	return startContainer(t, req, sshBastionPort)
}

// Resolved from this file rather than the working directory: callers live in other packages, and
// testcontainers resolves a relative context against the CWD. Tests that need the key material
// baked into the image read it from here too.
func GetSSHBastionTestdataDir(t *testing.T) string {
	t.Helper()

	return getFixtureDirectory(t, "ssh_bastion")
}

// Database is the in-network address, reachable only through Bastion. BastionContainer is what lets
// a test kill the transport mid-stream and bring it back.
type BastionedDatabase struct {
	Bastion          Endpoint
	BastionContainer testcontainers.Container
	Database         Endpoint
}

// One alias per engine so a run that boots several topologies stays readable in docker inspect.
const (
	bastionedPostgresAlias = "postgres-behind-bastion"
	bastionedMysqlAlias    = "mysql-behind-bastion"
	bastionedMariadbAlias  = "mariadb-behind-bastion"
	bastionedMongodbAlias  = "mongodb-behind-bastion"

	bastionedPhysicalPostgresAlias = "physical-postgres-behind-bastion"
)

// The database publishes no ports and the bastion does, so a tunnel is the only way in. Tests
// asserting that traffic really goes through the tunnel need that: a direct route would keep them
// green after the tunnel stopped being used.
func StartPostgresBehindSSHBastion(t *testing.T, image string) BastionedDatabase {
	t.Helper()

	networkName := StartNetwork(t)

	bastion := StartSSHBastionContainerOnNetwork(t, networkName)

	return BastionedDatabase{
		Bastion:          bastion.Endpoint,
		BastionContainer: bastion.Container,
		Database: StartPostgresOnNetwork(t, OnNetworkSpec{
			Image: image,
			Placement: NetworkPlacement{
				NetworkName: networkName,
				Alias:       bastionedPostgresAlias,
			},
		}),
	}
}

func StartMysqlBehindSSHBastion(t *testing.T, image string) BastionedDatabase {
	t.Helper()

	networkName := StartNetwork(t)

	bastion := StartSSHBastionContainerOnNetwork(t, networkName)

	return BastionedDatabase{
		Bastion:          bastion.Endpoint,
		BastionContainer: bastion.Container,
		Database: StartMysqlOnNetwork(t, OnNetworkSpec{
			Image: image,
			Placement: NetworkPlacement{
				NetworkName: networkName,
				Alias:       bastionedMysqlAlias,
			},
		}),
	}
}

func StartMariadbBehindSSHBastion(t *testing.T, image string) BastionedDatabase {
	t.Helper()

	networkName := StartNetwork(t)

	bastion := StartSSHBastionContainerOnNetwork(t, networkName)

	return BastionedDatabase{
		Bastion:          bastion.Endpoint,
		BastionContainer: bastion.Container,
		Database: StartMariadbOnNetwork(t, OnNetworkSpec{
			Image: image,
			Placement: NetworkPlacement{
				NetworkName: networkName,
				Alias:       bastionedMariadbAlias,
			},
		}),
	}
}

func StartMongodbBehindSSHBastion(t *testing.T, image string) BastionedDatabase {
	t.Helper()

	networkName := StartNetwork(t)

	bastion := StartSSHBastionContainerOnNetwork(t, networkName)
	database := startMongodbOnNetwork(t, OnNetworkSpec{
		Image: image,
		Placement: NetworkPlacement{
			NetworkName: networkName,
			Alias:       bastionedMongodbAlias,
		},
	})
	t.Cleanup(func() {
		if t.Failed() {
			logContainerOutputTail(t, database.Container, image)
			logContainerOutputTail(t, bastion.Container, "SSH bastion")
		}
	})

	return BastionedDatabase{
		Bastion:          bastion.Endpoint,
		BastionContainer: bastion.Container,
		Database:         database.Endpoint,
	}
}

// Docker re-picks an ephemeral host port on every `start`, so a bastion published dynamically moves
// out from under a tunnel the moment a test restarts it. Pinning the binding is what a real bastion
// has anyway, and it is the only way a forwarder can find its way back after an outage. Docker can
// hand a released port to another container before this one binds it, so each boot attempt reserves
// its own port.
func pinBastionHostPort(t *testing.T, req *testcontainers.ContainerRequest) {
	t.Helper()

	bastionPort, err := network.ParsePort(sshBastionPort)
	if err != nil {
		t.Fatalf("failed to parse the ssh bastion port %q: %v", sshBastionPort, err)
	}

	req.HostConfigModifier = func(hostConfig *container.HostConfig) {
		hostConfig.PortBindings = network.PortMap{
			bastionPort: []network.PortBinding{
				{HostIP: netip.MustParseAddr("127.0.0.1"), HostPort: strconv.Itoa(reserveBastionHostPort(t))},
			},
		}
	}
}

func reserveBastionHostPort(t *testing.T) int {
	t.Helper()

	var listenConfig net.ListenConfig

	listener, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to reserve a host port for the ssh bastion: %v", err)
	}

	reservedPort, isTCPAddr := listener.Addr().(*net.TCPAddr)
	if !isTCPAddr {
		t.Fatalf("failed to read the reserved ssh bastion port from %s", listener.Addr())
	}

	if err := listener.Close(); err != nil {
		t.Fatalf("failed to release the reserved ssh bastion port: %v", err)
	}

	return reservedPort.Port
}

func sshBastionRequest(t *testing.T) testcontainers.ContainerRequest {
	t.Helper()
	return testcontainers.ContainerRequest{
		Context:      GetSSHBastionTestdataDir(t),
		Dockerfile:   "Dockerfile",
		Repo:         "databasus-test-ssh-bastion",
		Tag:          "latest",
		KeepImage:    true,
		ExposedPorts: []string{sshBastionPort},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort(sshBastionPort).WithStartupTimeout(120*time.Second),
			forGreeting(sshBastionPort, 120*time.Second, "SSH-2.0-"),
		),
	}
}
