package containers

import (
	"context"
	// Random passive port selection is not a credential or security boundary.
	// nosemgrep: go.lang.security.audit.crypto.math_random.math-random-used
	"math/rand/v2"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Credentials baked into the test pure-ftpd container.
const (
	FtpUsername = "testuser"
	FtpPassword = "testpassword"
)

const ftpControlPort = "21/tcp"

// pure-ftpd hands out a port from its passive range on EPSV, and jlaffaye/ftp (EPSV by default) then
// dials the control host at that exact port. So the range must be published 1:1 (container port ==
// host port) instead of at testcontainers' usual random host port, or the data connection lands
// nowhere. Several test binaries start FTP containers at once, so each container takes a free range of
// its own. The ranges sit below the Linux ephemeral range (32768 and up), from which Docker publishes
// random host ports and the kernel picks outgoing ones, so neither can take a port between the probe
// and the container start.
const (
	ftpPassivePortCount   = 10
	ftpPassivePortLowest  = 20000
	ftpPassivePortHighest = 29999
)

// passivePortRange is the first and the last port of an FTP container's passive range.
type passivePortRange struct {
	first int
	last  int
}

// StartFtp binds the passive port range 1:1 (see the ftpPassivePortCount constant comment); the
// control port stays dynamic and is read back as the endpoint.
func StartFtp(t *testing.T) Endpoint {
	t.Helper()

	return StartFtpContainer(t).Endpoint
}

func StartFtpContainer(t *testing.T) ContainerHandle {
	t.Helper()

	passivePorts := findFreePassivePortRange(t)

	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:   getFixtureDirectory(t, "ftp"),
			Repo:      "databasus-test-ftp",
			Tag:       "latest",
			KeepImage: true,
		},
		ExposedPorts: ftpExposedPorts(passivePorts),
		Env: map[string]string{
			"FTP_USER_PASS":     FtpPassword,
			"FTP_PASSIVE_PORTS": strconv.Itoa(passivePorts.first) + ":" + strconv.Itoa(passivePorts.last),
		},
		HostConfigModifier: func(hc *container.HostConfig) {
			// Debian's daemon requests these startup capabilities, then drops session privileges.
			hc.CapAdd = append(hc.CapAdd, "SYS_NICE", "DAC_READ_SEARCH")
			hc.PortBindings = withPassivePortRange(hc.PortBindings, passivePorts)
		},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort(ftpControlPort).WithStartupTimeout(120*time.Second),
			forGreeting(ftpControlPort, 120*time.Second, "220"),
		),
	}

	return startContainer(t, req, ftpControlPort)
}

// ftpExposedPorts lists the control port plus every passive port. testcontainers drops any host
// binding whose port is not also exposed (its mergePortBindings filters to the exposed set), so the
// passive range must appear here for the fixed 1:1 bindings in withPassivePortRange to survive.
func ftpExposedPorts(passivePorts passivePortRange) []string {
	ports := []string{ftpControlPort}
	for passivePort := passivePorts.first; passivePort <= passivePorts.last; passivePort++ {
		ports = append(ports, strconv.Itoa(passivePort)+"/tcp")
	}

	return ports
}

// withPassivePortRange adds a fixed host-port binding for each FTP passive port, preserving the
// dynamic control-port binding testcontainers populates from ExposedPorts.
func withPassivePortRange(bindings network.PortMap, passivePorts passivePortRange) network.PortMap {
	if bindings == nil {
		bindings = network.PortMap{}
	}

	for passivePort := passivePorts.first; passivePort <= passivePorts.last; passivePort++ {
		hostPort := strconv.Itoa(passivePort)
		bindings[network.MustParsePort(hostPort+"/tcp")] = []network.PortBinding{{HostPort: hostPort}}
	}

	return bindings
}

// findFreePassivePortRange starts at a random range, so two test binaries rarely probe the same one,
// and takes the first range whose every port can be bound on the host right now.
func findFreePassivePortRange(t *testing.T) passivePortRange {
	t.Helper()

	rangeCount := (ftpPassivePortHighest - ftpPassivePortLowest + 1) / ftpPassivePortCount
	// math/rand only spreads concurrent tests over the ranges, it guards nothing.
	// nosemgrep: go.lang.security.audit.crypto.math_random.math-random-used
	startIndex := rand.IntN(rangeCount)

	for offset := range rangeCount {
		first := ftpPassivePortLowest + ((startIndex+offset)%rangeCount)*ftpPassivePortCount
		candidate := passivePortRange{first: first, last: first + ftpPassivePortCount - 1}

		if isPortRangeFree(t.Context(), candidate) {
			return candidate
		}
	}

	t.Fatalf("no free range of %d FTP passive ports between %d and %d",
		ftpPassivePortCount, ftpPassivePortLowest, ftpPassivePortHighest)

	return passivePortRange{}
}

func isPortRangeFree(ctx context.Context, candidate passivePortRange) bool {
	var listenConfig net.ListenConfig

	for port := candidate.first; port <= candidate.last; port++ {
		listener, err := listenConfig.Listen(ctx, "tcp", net.JoinHostPort("0.0.0.0", strconv.Itoa(port)))
		if err != nil {
			return false
		}

		_ = listener.Close()
	}

	return true
}
