package usecases_physical_postgresql

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReceiverRestartLatch(t *testing.T) {
	t.Run("cancels a receiver when a restart was requested before it started", func(t *testing.T) {
		latch := receiverRestartLatch{}
		latch.Request()
		receiverCtx, cancel := context.WithCancel(t.Context())
		defer cancel()

		latch.SetReceiver(cancel)

		assert.ErrorIs(t, receiverCtx.Err(), context.Canceled)
		assert.True(t, latch.IsRequested())
	})
	t.Run("allows the next receiver when the restart request is reset", func(t *testing.T) {
		latch := receiverRestartLatch{}
		latch.Request()
		latch.Reset()
		receiverCtx, cancel := context.WithCancel(t.Context())
		defer cancel()

		latch.SetReceiver(cancel)

		assert.NoError(t, receiverCtx.Err())
		assert.False(t, latch.IsRequested())
		latch.Request()
		assert.ErrorIs(t, receiverCtx.Err(), context.Canceled)
	})
	t.Run("leaves an old receiver alone when it has been cleared", func(t *testing.T) {
		latch := receiverRestartLatch{}
		receiverCtx, cancel := context.WithCancel(t.Context())
		defer cancel()
		latch.SetReceiver(cancel)
		latch.ClearReceiver()

		latch.Request()

		assert.NoError(t, receiverCtx.Err())
		assert.True(t, latch.IsRequested())
	})
}
