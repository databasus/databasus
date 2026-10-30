package physical

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
)

func startStallableSourceProxy(t *testing.T, source containers.Endpoint) (containers.Endpoint, *atomic.Bool) {
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

			session := sourceProxySession{
				ctx:        t.Context(),
				clientConn: clientConn,
				source:     source,
				isStalled:  isStalled,
			}
			go session.run()
		}
	}()

	proxyAddr, isTCPAddr := proxy.Addr().(*net.TCPAddr)
	require.True(t, isTCPAddr)

	return containers.Endpoint{Host: "127.0.0.1", Port: proxyAddr.Port}, isStalled
}

type sourceProxySession struct {
	ctx        context.Context
	clientConn net.Conn
	source     containers.Endpoint
	isStalled  *atomic.Bool
}

func (s sourceProxySession) run() {
	defer func() { _ = s.clientConn.Close() }()

	dialer := net.Dialer{Timeout: 10 * time.Second}

	sourceConn, err := dialer.DialContext(s.ctx, "tcp", fmt.Sprintf("%s:%d", s.source.Host, s.source.Port))
	if err != nil {
		return
	}
	defer func() { _ = sourceConn.Close() }()

	copyUntilStalled := func(dst, src net.Conn) {
		buffer := make([]byte, 32*1024)
		for {
			readCount, readErr := src.Read(buffer)
			if s.isStalled.Load() {
				// Neither answering nor hanging up is the whole point; unwinding at test end keeps
				// that from outliving the test.
				<-s.ctx.Done()

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

	go copyUntilStalled(sourceConn, s.clientConn)
	copyUntilStalled(s.clientConn, sourceConn)
}
