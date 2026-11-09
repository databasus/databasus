package providertest

import (
	"context"
	"io"
	"log/slog"
	"net"
	"sync"
	"testing"
	"time"

	"databasus-backend/internal/lib/background"
)

// CountingForwarder relays TCP connections to a target address and counts them, so a test can see
// how many connections a client opened instead of inferring it from how long the calls took.
type CountingForwarder struct {
	Host string
	Port int

	work          *background.Group
	ctx           context.Context
	listener      net.Listener
	targetAddress string

	mutex         sync.Mutex
	acceptedCount int
	closedCount   int
	isClosed      bool
	connections   []net.Conn
	responseGate  *ResponseGate
}

func StartCountingForwarder(t *testing.T, targetAddress string) *CountingForwarder {
	t.Helper()

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for counting forwarder: %v", err)
	}

	listenerAddress, isTCPAddress := listener.Addr().(*net.TCPAddr)
	if !isTCPAddress {
		t.Fatalf("counting forwarder listens on a non-TCP address %s", listener.Addr())
	}

	forwarder := &CountingForwarder{
		Host:          listenerAddress.IP.String(),
		Port:          listenerAddress.Port,
		listener:      listener,
		targetAddress: targetAddress,
		ctx:           t.Context(),
		work:          background.New(background.Deps{Logger: slog.New(slog.DiscardHandler)}),
	}

	forwarder.work.Go(
		t.Context(),
		"accept TCP connections",
		func(context.Context) error { forwarder.acceptConnections(); return nil },
	)
	t.Cleanup(func() {
		forwarder.close()
		if err := forwarder.work.Wait(context.WithoutCancel(t.Context())); err != nil {
			t.Errorf("stop TCP forwarder: %v", err)
		}
	})

	return forwarder
}

func (f *CountingForwarder) GetAcceptedCount() int {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	return f.acceptedCount
}

func (f *CountingForwarder) GetClosedCount() int {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	return f.closedCount
}

func (f *CountingForwarder) acceptConnections() {
	for {
		clientConnection, err := f.listener.Accept()
		if err != nil {
			return
		}

		f.mutex.Lock()
		f.acceptedCount++
		f.mutex.Unlock()

		f.work.Go(
			f.ctx,
			"forward TCP connection",
			func(context.Context) error { f.forward(clientConnection); return nil },
		)
	}
}

func (f *CountingForwarder) forward(clientConnection net.Conn) {
	targetConnection, err := (&net.Dialer{Timeout: 5 * time.Second}).DialContext(f.ctx, "tcp", f.targetAddress)
	if err != nil {
		_ = clientConnection.Close()
		return
	}

	// A connection accepted while the forwarder closes would otherwise outlive the test.
	if !f.track(clientConnection, targetConnection) {
		_ = clientConnection.Close()
		_ = targetConnection.Close()

		return
	}

	f.work.Go(
		f.ctx,
		"copy to TCP target",
		func(context.Context) error {
			pipeAndClose(connectionStream{destination: targetConnection, source: clientConnection})

			f.mutex.Lock()
			f.closedCount++
			f.mutex.Unlock()

			return nil
		},
	)
	f.work.Go(
		f.ctx,
		"copy from TCP target",
		func(context.Context) error {
			pipeAndClose(connectionStream{
				destination: &forwardedResponse{Conn: clientConnection, forwarder: f},
				source:      targetConnection,
			})
			return nil
		},
	)
}

func (f *CountingForwarder) track(connections ...net.Conn) bool {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if f.isClosed {
		return false
	}

	f.connections = append(f.connections, connections...)

	return true
}

// Clients such as the storage session pools keep idle connections open past the test, so the
// forwarder closes them itself rather than leave its relay goroutines running.
func (f *CountingForwarder) close() {
	_ = f.listener.Close()

	f.mutex.Lock()
	f.isClosed = true
	connections := f.connections
	f.connections = nil
	f.mutex.Unlock()

	for _, connection := range connections {
		_ = connection.Close()
	}
}

type connectionStream struct {
	destination io.WriteCloser
	source      net.Conn
}

func pipeAndClose(stream connectionStream) {
	_, _ = io.Copy(stream.destination, stream.source)
	_ = stream.destination.Close()
	_ = stream.source.Close()
}
