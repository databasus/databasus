package sshtunnel

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
)

// The bastion's sshd also listens on 2222, which is never published to the host, so forwarding
// there proves bytes reach the far side without a second container. Port 22 would be a bad target:
// on a machine that runs its own sshd, a forwarder that dialled the host loopback instead of going
// through the bastion would still read a banner and pass.
const (
	bastionInternalPort = 2222
	sshBannerPrefix     = "SSH-2.0-"
)

// Negative control for every other test in this file: it pins that the target port really is
// unreachable without the tunnel. Without it, a forwarder that stopped tunnelling and dialled
// the target directly would keep all the happy-path tests green.
func TestBastionInternalPort(t *testing.T) {
	t.Parallel()

	t.Run("is unreachable from the host without the tunnel", func(t *testing.T) {
		t.Parallel()
		bastion := containers.StartSSHBastion(t)

		dialer := net.Dialer{Timeout: 5 * time.Second}
		conn, err := dialer.DialContext(
			t.Context(),
			"tcp",
			Endpoint{Host: bastion.Host, Port: bastionInternalPort}.String(),
		)
		if err == nil {
			_ = conn.Close()
		}

		require.Error(t, err, "the bastion's internal port must not be published, or the tunnel tests prove nothing")
	})
}

func TestOpen(t *testing.T) {
	t.Parallel()

	t.Run("forwards bytes to the target with password auth", func(t *testing.T) {
		t.Parallel()
		bastion := containers.StartSSHBastion(t)

		forwarder := openTestForwarder(t, bastionSettings(bastion))

		require.NoError(t, readBannerThroughTunnel(t.Context(), forwarder))
	})

	t.Run("forwards bytes to the target with private key auth", func(t *testing.T) {
		t.Parallel()
		bastion := containers.StartSSHBastion(t)
		settings := bastionSettings(bastion)
		settings.AuthType = AuthTypePrivateKey
		settings.Password = ""
		settings.PrivateKey = readTestKey(t, "test_key")

		forwarder := openTestForwarder(t, settings)

		require.NoError(t, readBannerThroughTunnel(t.Context(), forwarder))
	})

	t.Run("forwards bytes to the target with a passphrase protected private key", func(t *testing.T) {
		t.Parallel()
		bastion := containers.StartSSHBastion(t)
		settings := bastionSettings(bastion)
		settings.AuthType = AuthTypePrivateKey
		settings.Password = ""
		settings.PrivateKey = readTestKey(t, "test_key_passphrase")
		settings.PrivateKeyPassphrase = "testpassphrase"

		forwarder := openTestForwarder(t, settings)

		require.NoError(t, readBannerThroughTunnel(t.Context(), forwarder))
	})

	t.Run("returns an error without the secret when the password is wrong", func(t *testing.T) {
		t.Parallel()
		bastion := containers.StartSSHBastion(t)
		settings := bastionSettings(bastion)
		settings.Password = "definitely-not-the-password"

		forwarder, err := Open(t.Context(), testOpenSpec(settings))

		require.Error(t, err)
		assert.Nil(t, forwarder)
		assert.NotContains(t, err.Error(), settings.Password)
	})

	t.Run("returns an error when the passphrase is wrong", func(t *testing.T) {
		t.Parallel()
		bastion := containers.StartSSHBastion(t)
		settings := bastionSettings(bastion)
		settings.AuthType = AuthTypePrivateKey
		settings.Password = ""
		settings.PrivateKey = readTestKey(t, "test_key_passphrase")
		settings.PrivateKeyPassphrase = "wrong-passphrase"

		forwarder, err := Open(t.Context(), testOpenSpec(settings))

		require.Error(t, err)
		assert.Nil(t, forwarder)
		assert.NotContains(t, err.Error(), settings.PrivateKeyPassphrase)
	})

	// A bastion address that completes the TCP connect and then says nothing is what a blackholing
	// middlebox or a wrong port looks like. ssh.ClientConfig.Timeout does not cover the handshake, so
	// without an explicit deadline this blocks forever while holding clientMutex, taking Close with it.
	t.Run("fails instead of hanging when the bastion accepts but never handshakes", func(t *testing.T) {
		t.Parallel()
		openSpec := testOpenSpec(newSilentBastionSettings(t))
		openSpec.HandshakeTimeout = 2 * time.Second

		openErrors := make(chan error, 1)

		go func() {
			forwarder, err := Open(t.Context(), openSpec)
			if forwarder != nil {
				forwarder.Close()
			}

			openErrors <- err
		}()

		select {
		case openErr := <-openErrors:
			require.Error(t, openErr)
		case <-time.After(15 * time.Second):
			t.Fatal("Open blocked on the SSH handshake instead of timing out")
		}
	})

	t.Run("ends the first handshake when the context ends", func(t *testing.T) {
		t.Parallel()

		openSpec := testOpenSpec(newSilentBastionSettings(t))
		openSpec.HandshakeTimeout = 10 * time.Minute

		openCtx, cancelOpen := context.WithCancel(t.Context())
		defer cancelOpen()

		openErrors := make(chan error, 1)

		go func() {
			forwarder, err := Open(openCtx, openSpec)
			if forwarder != nil {
				forwarder.Close()
			}

			openErrors <- err
		}()

		cancelOpen()

		select {
		case openErr := <-openErrors:
			require.ErrorIs(t, openErr, context.Canceled)
		case <-time.After(15 * time.Second):
			t.Fatal("Open kept waiting for the SSH handshake after its context ended")
		}
	})
}

func TestClose(t *testing.T) {
	t.Parallel()

	t.Run("releases the local port", func(t *testing.T) {
		t.Parallel()
		bastion := containers.StartSSHBastion(t)
		forwarder := openTestForwarder(t, bastionSettings(bastion))
		localEndpoint := forwarder.GetLocalEndpoint()
		require.NoError(t, readBannerThroughTunnel(t.Context(), forwarder))

		forwarder.Close()

		dialer := net.Dialer{Timeout: 5 * time.Second}
		conn, err := dialer.DialContext(t.Context(), "tcp", localEndpoint.String())
		if err == nil {
			_ = conn.Close()
		}

		assert.Error(t, err)
	})

	t.Run("does not panic when called twice", func(t *testing.T) {
		t.Parallel()
		bastion := containers.StartSSHBastion(t)
		forwarder := openTestForwarder(t, bastionSettings(bastion))

		forwarder.Close()
		forwarder.Close()
	})

	// Close cancels the redial context before taking clientMutex. Without that, a redial parked on a
	// TCP connect to an unreachable bastion holds the mutex for the full dial timeout, and Close
	// waits behind it. Exercised through the dial seam because no real address stalls a connect on
	// every host.
	t.Run("cancels a redial stuck on connect instead of waiting out the timeout", func(t *testing.T) {
		t.Parallel()
		bastion := containers.StartSSHBastion(t)
		forwarder := openTestForwarder(t, bastionSettings(bastion))
		dialStarted := make(chan struct{})

		var dialStartedOnce sync.Once

		forwarder.dialBastionConn = func(ctx context.Context, _ string) (net.Conn, error) {
			dialStartedOnce.Do(func() { close(dialStarted) })

			<-ctx.Done()

			return nil, ctx.Err()
		}

		discardCachedClient(forwarder)

		go func() { _, _ = forwarder.getOrDialBastion() }()

		<-dialStarted

		closed := make(chan struct{})

		go func() {
			forwarder.Close()
			close(closed)
		}()

		select {
		case <-closed:
		case <-time.After(bastionDialTimeout / 3):
			t.Fatal("Close waited on the stuck redial instead of cancelling it")
		}
	})
}

func TestGetOrDialBastion(t *testing.T) {
	t.Parallel()

	// A connection that got past Accept before Close must not be able to open a fresh bastion
	// session: nothing would be left to close it, and Close would sit in its Wait until the far end
	// hung up. Driving the guard directly rather than racing Close against live traffic, because the
	// race window is far too narrow to hit on purpose.
	t.Run("refuses to open a new session after close", func(t *testing.T) {
		t.Parallel()
		bastion := containers.StartSSHBastion(t)
		forwarder := openTestForwarder(t, bastionSettings(bastion))
		require.NoError(t, readBannerThroughTunnel(t.Context(), forwarder))

		forwarder.Close()
		client, err := forwarder.getOrDialBastion()

		require.ErrorIs(t, err, ErrForwarderClosed)
		assert.Nil(t, client)

		forwarder.clientMutex.Lock()
		defer forwarder.clientMutex.Unlock()

		assert.Nil(t, forwarder.client, "a session opened after Close would never be closed")
	})
}

func TestForwarder(t *testing.T) {
	t.Parallel()

	// The cached client dies between operations far more often than it dies mid-dial - a bastion
	// restart, an idle timeout, a NAT drop. Every waiting connection then discovers the corpse at
	// once, so the redial has to be serialized or the losers leak clients and goroutines.
	t.Run("recovers every concurrent connection when the cached client is dead", func(t *testing.T) {
		t.Parallel()
		bastion := containers.StartSSHBastion(t)
		forwarder := openTestForwarder(t, bastionSettings(bastion))
		require.NoError(t, readBannerThroughTunnel(t.Context(), forwarder))

		var redialCount atomic.Int32

		dialOverTCP := forwarder.dialBastionConn
		forwarder.dialBastionConn = func(ctx context.Context, address string) (net.Conn, error) {
			redialCount.Add(1)

			return dialOverTCP(ctx, address)
		}

		forwarder.clientMutex.Lock()
		require.NotNil(t, forwarder.client)
		require.NoError(t, forwarder.client.Close())
		forwarder.clientMutex.Unlock()

		const concurrentConnections = 8

		// testify's FailNow only works on the goroutine running the test, so each connection reports
		// back and the assertions happen after the group is joined.
		connectionErrors := make([]error, concurrentConnections)

		var connectionGroup sync.WaitGroup

		for connectionIndex := range concurrentConnections {
			connectionGroup.Go(func() {
				connectionErrors[connectionIndex] = readBannerThroughTunnel(t.Context(), forwarder)
			})
		}

		connectionGroup.Wait()

		for connectionIndex, connectionError := range connectionErrors {
			require.NoErrorf(t, connectionError, "connection %d failed to recover", connectionIndex)
		}

		// Not exactly one: dialTarget releases clientMutex between discardClient and its retry, so a
		// second waiter can slip in. Unserialized it would be one per connection.
		assert.Less(t, redialCount.Load(), int32(3), "the redial must be serialized, or the losers leak sessions")
	})
}

func bastionSettings(bastion containers.Endpoint) Settings {
	return Settings{
		Host:     bastion.Host,
		Port:     bastion.Port,
		Username: containers.SSHBastionUsername,
		AuthType: AuthTypePassword,
		Password: containers.SSHBastionPassword,
	}
}

// newSilentBastionSettings points at a bastion that completes the TCP connect and then says nothing.
func newSilentBastionSettings(t *testing.T) Settings {
	t.Helper()

	var listenConfig net.ListenConfig
	silentBastion, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = silentBastion.Close() })

	go func() {
		for {
			conn, err := silentBastion.Accept()
			if err != nil {
				return
			}

			t.Cleanup(func() { _ = conn.Close() })
		}
	}()

	silentAddress, isTCPAddress := silentBastion.Addr().(*net.TCPAddr)
	require.True(t, isTCPAddress)

	return Settings{
		Host:     "127.0.0.1",
		Port:     silentAddress.Port,
		Username: containers.SSHBastionUsername,
		AuthType: AuthTypePassword,
		Password: containers.SSHBastionPassword,
	}
}

func testOpenSpec(settings Settings) OpenSpec {
	return OpenSpec{
		Settings:         settings,
		Target:           Endpoint{Host: "127.0.0.1", Port: bastionInternalPort},
		Logger:           slog.New(slog.DiscardHandler),
		HandshakeTimeout: 0,
	}
}

func openTestForwarder(t *testing.T, settings Settings) *Forwarder {
	t.Helper()

	forwarder, err := Open(t.Context(), testOpenSpec(settings))
	require.NoError(t, err)

	t.Cleanup(forwarder.Close)

	return forwarder
}

func discardCachedClient(forwarder *Forwarder) {
	forwarder.clientMutex.Lock()
	cachedClient := forwarder.client
	forwarder.clientMutex.Unlock()

	forwarder.discardClient(cachedClient)
}

func readTestKey(t *testing.T, name string) string {
	t.Helper()

	key, err := os.ReadFile(filepath.Join(containers.GetSSHBastionTestdataDir(t), name))
	require.NoError(t, err)

	return string(key)
}

func readBannerThroughTunnel(ctx context.Context, forwarder *Forwarder) error {
	dialer := net.Dialer{Timeout: 15 * time.Second}

	conn, err := dialer.DialContext(ctx, "tcp", forwarder.GetLocalEndpoint().String())
	if err != nil {
		return err
	}

	defer func() { _ = conn.Close() }()

	if err := conn.SetReadDeadline(time.Now().Add(15 * time.Second)); err != nil {
		return err
	}

	banner := make([]byte, len(sshBannerPrefix))
	if _, err := io.ReadFull(conn, banner); err != nil {
		return err
	}

	if string(banner) != sshBannerPrefix {
		return fmt.Errorf("expected %q through the tunnel, got %q", sshBannerPrefix, string(banner))
	}

	return nil
}
