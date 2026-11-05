package container

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/agent/dbconn"
)

func TestWaitForReady(t *testing.T) {
	t.Run("returns deadline with last ping error when the port refuses connections", func(t *testing.T) {
		listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
		require.NoError(t, err)

		address, isTCPAddress := listener.Addr().(*net.TCPAddr)
		require.True(t, isTCPAddress)
		closedPort := address.Port
		require.NoError(t, listener.Close())

		const password = "secret-restore-pw-123"

		ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
		defer cancel()

		err = waitForReady(ctx, dbconn.Conn{
			Host: "127.0.0.1", Port: closedPort, User: restoreUser, Password: password, Database: restoreDB,
		})

		require.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Contains(t, err.Error(), "last ping error")
		assert.Contains(t, err.Error(), "connection refused",
			"the error must say why Postgres stayed unreachable, not only that time ran out")
		assert.NotContains(t, err.Error(), password, "the report must not leak the database password")
	})
}
