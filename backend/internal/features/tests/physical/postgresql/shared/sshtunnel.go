//go:build testhelpers

package physicaltesting

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	postgresql_executor "databasus-backend/internal/features/backups/backups/usecases/physical/postgresql"
	"databasus-backend/internal/features/databases"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	"databasus-backend/internal/features/sshtunnel"
	"databasus-backend/internal/util/logger"
	"databasus-backend/internal/util/testing/bastion"
	"databasus-backend/internal/util/testing/containers"
	"databasus-backend/internal/util/walmath"
)

// The source cluster publishes no port, so every connection the backend makes has to go through the
// bastion. A direct route would keep these tests green after the tunnel stopped being used.
func setupBastionedFixture(
	t *testing.T,
	version string,
	image string,
	backupType postgresql_physical.BackupType,
) (*gin.Engine, *postgresql_executor.PhysicalDBFixture, containers.BastionedDatabase) {
	t.Helper()

	topology := containers.StartPhysicalPostgresBehindSshBastion(t, image)

	router := newPhysicalTestRouter()
	fixture := postgresql_executor.SetupPhysicalDBFixtureWithTunnel(t, databases.PhysicalTestDatabaseSpec{
		Host:       topology.Database.Host,
		Port:       topology.Database.Port,
		VersionTag: version,
		BackupType: backupType,
		SshTunnel:  bastion.GetTunnelConfig(topology),
	})

	return router, fixture, topology
}

// The test has no other route to the bastioned cluster, so it reaches it the same way the backend
// does. The forwarder is the test's own on purpose: stopping the bastion has to break both it and
// the backend's, and reopening this one must not repair the backend's.
//
// It lands on the seeded test database rather than `postgres`, because that is where the marker rows
// the restore is asserted against live.
func openSourceTestDBConnThroughBastion(
	t *testing.T,
	topology containers.BastionedDatabase,
) *pgx.Conn {
	t.Helper()

	localEndpoint := openForwardedEndpoint(t, topology)

	return openConnAt(t, localEndpoint.Host, localEndpoint.Port)
}

func openForwardedEndpoint(t *testing.T, topology containers.BastionedDatabase) sshtunnel.Endpoint {
	t.Helper()

	forwarder, err := sshtunnel.Open(t.Context(), sshtunnel.OpenSpec{
		Config: bastion.GetTunnelConfig(topology),
		Target: sshtunnel.Endpoint{Host: topology.Database.Host, Port: topology.Database.Port},
		Logger: logger.GetLogger(),
	})
	require.NoError(t, err)
	t.Cleanup(forwarder.Close)

	return forwarder.GetLocalEndpoint()
}

func stopBastion(t *testing.T, topology containers.BastionedDatabase) {
	t.Helper()

	stopTimeout := 10 * time.Second
	require.NoError(t, topology.BastionContainer.Stop(context.Background(), &stopTimeout))
}

func startBastion(t *testing.T, topology containers.BastionedDatabase) {
	t.Helper()

	require.NoError(t, topology.BastionContainer.Start(context.Background()))
	waitForBastionAuthenticating(t, topology.Bastion, 90*time.Second)
}

// A completed SSH handshake rather than an accepted TCP connection: after a restart the published
// port and even sshd's own listener can be up while the first handshake is still reset, and the
// test's next step is exactly such a handshake. The address is the one from before the outage on
// purpose — the backend's forwarder has been retrying it all along, and if the restart moved it the
// premise of the test is gone rather than merely slow.
func waitForBastionAuthenticating(t *testing.T, bastion containers.Endpoint, timeout time.Duration) {
	t.Helper()

	address := net.JoinHostPort(bastion.Host, strconv.Itoa(bastion.Port))

	var lastErr error

	deadline := time.Now().UTC().Add(timeout)
	for time.Now().UTC().Before(deadline) {
		lastErr = probeBastionHandshake(address)
		if lastErr == nil {
			return
		}

		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("the bastion at %s never completed an ssh handshake again within %s: %v", address, timeout, lastErr)
}

func probeBastionHandshake(address string) error {
	const probeTimeout = 5 * time.Second

	dialer := net.Dialer{Timeout: probeTimeout}

	conn, err := dialer.DialContext(context.Background(), "tcp", address)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	// The deadline bounds the handshake too, which ssh.NewClientConn would otherwise wait on forever.
	if err := conn.SetDeadline(time.Now().UTC().Add(probeTimeout)); err != nil {
		return err
	}

	clientConn, channels, requests, err := ssh.NewClientConn(conn, address, &ssh.ClientConfig{
		User: containers.SshBastionUsername,
		Auth: []ssh.AuthMethod{ssh.Password(containers.SshBastionPassword)},
		// The throwaway test bastion regenerates its host key per image build; nothing to pin.
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		return err
	}

	return ssh.NewClient(clientConn, channels, requests).Close()
}

// A FAILED row is how the supervisor hands a streamer back for reclaim, so it is exactly what an
// unclassified transport outage would leave behind.
func requireStreamerNotFailed(t *testing.T, fixture *postgresql_executor.PhysicalDBFixture) {
	t.Helper()

	streamer, err := physical_repositories.GetWalStreamerRepository().FindByDatabaseID(fixture.DB.ID)
	require.NoError(t, err)
	require.NotNil(t, streamer, "the supervisor must still own the streamer")
	require.NotEqual(t, physical_enums.PhysicalWalStreamerStatusFailed, streamer.Status)
}

func getCommittedWalSegmentCount(t *testing.T, fixture *postgresql_executor.PhysicalDBFixture) int {
	t.Helper()

	segments, err := physical_repositories.GetWalSegmentRepository().FindByChainSpan(
		fixture.DB.ID, 1, walmath.LSN(0), walmath.LSN(^uint64(0)),
	)
	require.NoError(t, err)

	committedCount := 0
	for _, segment := range segments {
		if segment.FileName != nil {
			committedCount++
		}
	}

	return committedCount
}

func waitForCommittedWalSegmentsAbove(
	t *testing.T,
	fixture *postgresql_executor.PhysicalDBFixture,
	previousCount int,
	timeout time.Duration,
) {
	t.Helper()

	deadline := time.Now().UTC().Add(timeout)
	for time.Now().UTC().Before(deadline) {
		if getCommittedWalSegmentCount(t, fixture) > previousCount {
			return
		}

		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("no WAL segment was archived within %s after the bastion came back", timeout)
}
