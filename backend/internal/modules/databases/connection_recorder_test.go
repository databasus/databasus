package databases_test

import (
	"net"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
)

// connectionRecorder is a TCP listener that counts the connections it accepts, so a test can show
// that a refused request never reached the host it named.
type connectionRecorder struct {
	host          string
	port          int
	acceptedCount atomic.Int64
}

func startConnectionRecorder(t *testing.T) *connectionRecorder {
	t.Helper()

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	t.Cleanup(func() { _ = listener.Close() })

	listenerAddress, isTCPAddress := listener.Addr().(*net.TCPAddr)
	require.True(t, isTCPAddress)

	recorder := &connectionRecorder{host: listenerAddress.IP.String(), port: listenerAddress.Port}

	go func() {
		for {
			acceptedConn, err := listener.Accept()
			if err != nil {
				return
			}

			recorder.acceptedCount.Add(1)
			_ = acceptedConn.Close()
		}
	}()

	return recorder
}

func (r *connectionRecorder) getAcceptedCount() int64 {
	return r.acceptedCount.Load()
}

func (r *connectionRecorder) getPostgresqlTarget() postgresqlTarget {
	return postgresqlTarget{
		host:     r.host,
		port:     r.port,
		database: containers.PostgresDatabase,
		username: containers.PostgresUsername,
		password: nil,
	}
}
