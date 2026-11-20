package shutdown

import (
	"context"
	"runtime"
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

		sendStopSignalToThisThread(t)

		select {
		case <-stopSignalCtx.Done():
		case <-time.After(3 * time.Second):
			t.Fatal("the context did not end after the stop signal")
		}

		sendStopSignalToThisThread(t)

		assert.ErrorIs(t, stopSignalCtx.Err(), context.Canceled)
	})
}

// A signal sent to the whole process reaches one of its threads later, so a late one could arrive
// after stopListening and kill the test binary. A signal sent to the calling thread is delivered
// before the system call returns.
func sendStopSignalToThisThread(t *testing.T) {
	t.Helper()

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	require.NoError(t, syscall.Tgkill(syscall.Getpid(), syscall.Gettid(), syscall.SIGTERM))
}
