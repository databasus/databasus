package sshtunnel

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
)

func TestIsBastionReachable(t *testing.T) {
	t.Parallel()

	// The caller is a supervision loop between two receiver spawns, so an answer that took the
	// transport's own 30s dial budget would be useless — it has to come back inside the deadline it
	// was given.
	t.Run("returns false within the caller's deadline when the bastion is gone", func(t *testing.T) {
		t.Parallel()
		bastion := containers.StartSSHBastion(t)
		forwarder := openTestForwarder(t, bastionSettings(bastion))
		require.NoError(t, readBannerThroughTunnel(t.Context(), forwarder))
		discardCachedClient(forwarder)

		forwarder.dialBastionConn = func(ctx context.Context, _ string) (net.Conn, error) {
			<-ctx.Done()

			return nil, ctx.Err()
		}

		ctx, cancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
		defer cancel()

		probeStartedAt := time.Now()
		isReachable := forwarder.IsBastionReachable(ctx)

		assert.False(t, isReachable)
		assert.Less(t, time.Since(probeStartedAt), bastionDialTimeout/2,
			"the probe outlived the caller's deadline")
	})

	// Asking a question must not authenticate: getOrDialBastion would have installed a live session
	// as a side effect, so a probe built on it would silently reconnect on every call.
	t.Run("does not cache the dialed client when no client is cached", func(t *testing.T) {
		t.Parallel()
		bastion := containers.StartSSHBastion(t)
		forwarder := openTestForwarder(t, bastionSettings(bastion))
		discardCachedClient(forwarder)

		assert.True(t, forwarder.IsBastionReachable(t.Context()))

		forwarder.clientMutex.Lock()
		defer forwarder.clientMutex.Unlock()

		assert.Nil(t, forwarder.client, "the probe installed a session nobody asked for")
	})

	// The cached-session branch cannot dial, so this is the only place the caller's deadline reaches
	// it. With the keepalive loop's own 15s budget hard-coded here instead, a supervision loop asking
	// for one second would block for fifteen between two receiver spawns.
	t.Run("returns false within the caller's deadline when the cached session hangs", func(t *testing.T) {
		t.Parallel()
		bastion := containers.StartSSHBastion(t)
		proxy, isStalled := startStallableBastionProxy(t, bastion)
		forwarder := openTestForwarder(t, bastionSettings(proxy))

		forwarder.clientMutex.Lock()
		hasCachedClient := forwarder.client != nil
		forwarder.clientMutex.Unlock()
		require.True(t, hasCachedClient, "the probe must take the cached-session branch")

		isStalled.Store(true)

		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()

		probeStartedAt := time.Now()

		assert.False(t, forwarder.IsBastionReachable(ctx))
		assert.Less(t, time.Since(probeStartedAt), keepaliveTimeout/2,
			"the probe waited out the keepalive loop's budget instead of the caller's")
	})
}

func TestBudgetUntil(t *testing.T) {
	t.Parallel()

	t.Run("reports the remaining time rather than the dial timeout with a caller deadline", func(t *testing.T) {
		t.Parallel()
		callerBudget := 3 * time.Second

		ctx, cancel := context.WithTimeout(t.Context(), callerBudget)
		defer cancel()

		assert.InDelta(t, callerBudget, budgetUntil(ctx), float64(200*time.Millisecond))
	})

	t.Run("falls back to the dial timeout without a caller deadline", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, bastionDialTimeout, budgetUntil(context.WithoutCancel(t.Context())))
	})

	t.Run("reports zero rather than a negative budget with an expired deadline", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(-time.Minute))
		defer cancel()

		assert.Equal(t, time.Duration(0), budgetUntil(ctx))
	})
}

// startStallableBastionProxy fronts the real bastion with a pipe the test can stop feeding. Halting
// it mid-session is the failure keepalive exists for: the TCP connection stays open with no FIN, so
// a request written into it is never answered and never fails either.
func startStallableBastionProxy(t *testing.T, bastion containers.Endpoint) (containers.Endpoint, *atomic.Bool) {
	t.Helper()

	var listenConfig net.ListenConfig

	proxy, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = proxy.Close() })

	isStalled := &atomic.Bool{}

	go func() {
		for {
			clientConn, acceptErr := proxy.Accept()
			if acceptErr != nil {
				return
			}

			go pipeUntilStalled(t.Context(), clientConn, bastion, isStalled)
		}
	}()

	proxyAddr, isTCPAddr := proxy.Addr().(*net.TCPAddr)
	require.True(t, isTCPAddr)

	return containers.Endpoint{Host: "127.0.0.1", Port: proxyAddr.Port}, isStalled
}

func pipeUntilStalled(
	ctx context.Context,
	clientConn net.Conn,
	bastion containers.Endpoint,
	isStalled *atomic.Bool,
) {
	defer func() { _ = clientConn.Close() }()

	dialer := net.Dialer{Timeout: 10 * time.Second}

	bastionConn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", bastion.Host, bastion.Port))
	if err != nil {
		return
	}
	defer func() { _ = bastionConn.Close() }()

	copyUntilStalled := func(dst, src net.Conn) {
		buffer := make([]byte, 32*1024)
		for {
			readCount, readErr := src.Read(buffer)
			if isStalled.Load() {
				// Neither answering nor hanging up is the whole point; unwinding at test end keeps
				// that from outliving the test.
				<-ctx.Done()

				return
			}

			if readCount > 0 {
				if _, writeErr := dst.Write(buffer[:readCount]); writeErr != nil {
					return
				}
			}

			if readErr != nil {
				return
			}
		}
	}

	go copyUntilStalled(bastionConn, clientConn)
	copyUntilStalled(clientConn, bastionConn)
}
