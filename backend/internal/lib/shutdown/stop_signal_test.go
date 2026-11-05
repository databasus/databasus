package shutdown

import (
	"context"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The test sends real signals to its own process, so it does not run in parallel with others.
func TestListenForStopSignal(t *testing.T) {
	t.Run("ends the context on the first signal and ignores the second", func(t *testing.T) {
		stopSignalCtx, stopListening := ListenForStopSignal(t.Context())
		defer stopListening()

		require.NoError(t, syscall.Kill(syscall.Getpid(), syscall.SIGTERM))

		select {
		case <-stopSignalCtx.Done():
		case <-time.After(3 * time.Second):
			t.Fatal("the context did not end after the stop signal")
		}

		require.NoError(t, syscall.Kill(syscall.Getpid(), syscall.SIGTERM))

		assert.ErrorIs(t, stopSignalCtx.Err(), context.Canceled)
	})
}
