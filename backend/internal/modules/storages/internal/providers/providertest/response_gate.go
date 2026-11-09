package providertest

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"

	"databasus-backend/internal/lib/testwait"
)

type ResponseGate struct {
	blocked     chan struct{}
	released    chan struct{}
	blockOnce   sync.Once
	releaseOnce sync.Once
}

func (f *CountingForwarder) PauseResponses(t *testing.T) *ResponseGate {
	t.Helper()

	gate := &ResponseGate{blocked: make(chan struct{}), released: make(chan struct{})}
	f.mutex.Lock()
	f.responseGate = gate
	f.mutex.Unlock()
	t.Cleanup(gate.Release)

	return gate
}

func (g *ResponseGate) WaitUntilBlocked(t *testing.T) {
	t.Helper()

	testwait.Until(t, "a storage response to reach the paused gateway", func(context.Context) error {
		select {
		case <-g.blocked:
			return nil
		default:
			return errors.New("no storage response has reached the gateway")
		}
	})
}

func (g *ResponseGate) Release() {
	g.releaseOnce.Do(func() { close(g.released) })
}

type forwardedResponse struct {
	net.Conn
	forwarder *CountingForwarder
}

func (r *forwardedResponse) Write(content []byte) (int, error) {
	r.forwarder.mutex.Lock()
	gate := r.forwarder.responseGate
	r.forwarder.mutex.Unlock()

	if gate != nil {
		gate.blockOnce.Do(func() { close(gate.blocked) })
		select {
		case <-gate.released:
		case <-r.forwarder.ctx.Done():
			return 0, r.forwarder.ctx.Err()
		}
	}

	return r.Conn.Write(content)
}
