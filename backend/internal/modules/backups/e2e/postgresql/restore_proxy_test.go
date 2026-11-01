package postgresql

import (
	"context"
	"io"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
)

type restoreProxy struct {
	endpoint    containers.Endpoint
	isHolding   atomic.Bool
	heldCount   atomic.Int64
	release     chan struct{}
	releaseOnce sync.Once
}

func startRestoreProxy(t *testing.T, target containers.Endpoint) *restoreProxy {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address, isTCPAddress := listener.Addr().(*net.TCPAddr)
	require.True(t, isTCPAddress)

	proxy := &restoreProxy{
		endpoint:  containers.Endpoint{Host: "127.0.0.1", Port: address.Port},
		isHolding: atomic.Bool{}, heldCount: atomic.Int64{}, release: make(chan struct{}), releaseOnce: sync.Once{},
	}

	var connections sync.WaitGroup
	acceptDone := make(chan struct{})
	go func() {
		defer close(acceptDone)
		for {
			connection, err := listener.Accept()
			if err != nil {
				return
			}
			connections.Go(func() {
				defer func() { _ = connection.Close() }()
				stopClosing := context.AfterFunc(ctx, func() { _ = connection.Close() })
				defer stopClosing()

				if proxy.isHolding.Load() {
					proxy.heldCount.Add(1)
					select {
					case <-proxy.release:
					case <-ctx.Done():
						return
					}
				}

				upstream, err := (&net.Dialer{}).DialContext(
					ctx,
					"tcp",
					net.JoinHostPort(target.Host, strconv.Itoa(target.Port)),
				)
				if err != nil {
					return
				}
				defer func() { _ = upstream.Close() }()
				stopUpstream := context.AfterFunc(ctx, func() { _ = upstream.Close() })
				defer stopUpstream()

				copied := make(chan struct{})
				go func() { _, _ = io.Copy(upstream, connection); _ = upstream.Close(); close(copied) }()
				_, _ = io.Copy(connection, upstream)
				_ = connection.Close()
				<-copied
			})
		}
	}()

	t.Cleanup(func() { cancel(); _ = listener.Close(); <-acceptDone; connections.Wait() })
	return proxy
}

func (p *restoreProxy) Release() {
	p.isHolding.Store(false)
	p.releaseOnce.Do(func() { close(p.release) })
}
