package background

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCancellationRegistry(t *testing.T) {
	t.Parallel()

	t.Run("cancels and removes the work when it is registered", func(t *testing.T) {
		t.Parallel()
		registry := NewCancellationRegistry()
		workID := uuid.New()
		workContext, cancelWork := context.WithCancel(t.Context())
		registry.Register(workID, cancelWork)

		isCancelled := registry.Cancel(workID)

		assert.True(t, isCancelled)
		require.ErrorIs(t, workContext.Err(), context.Canceled)
		assert.False(t, registry.Cancel(workID))
	})

	t.Run("cancels nothing after the work unregistered", func(t *testing.T) {
		t.Parallel()
		registry := NewCancellationRegistry()
		workID := uuid.New()
		workContext, cancelWork := context.WithCancel(t.Context())
		t.Cleanup(cancelWork)
		registry.Register(workID, cancelWork)
		registry.Unregister(workID)

		isCancelled := registry.Cancel(workID)

		assert.False(t, isCancelled)
		require.NoError(t, workContext.Err())
	})

	t.Run("returns from wait when the last work leaves", func(t *testing.T) {
		t.Parallel()
		registry := NewCancellationRegistry()
		firstWorkID := uuid.New()
		secondWorkID := uuid.New()
		registry.Register(firstWorkID, func() {})
		registry.Register(secondWorkID, func() {})
		waitResult := make(chan error, 1)

		go func() { waitResult <- registry.WaitUntilEmpty(t.Context()) }()

		registry.Unregister(firstWorkID)
		registry.Cancel(secondWorkID)

		require.NoError(t, <-waitResult)
	})

	t.Run("returns from wait at once when nothing is registered", func(t *testing.T) {
		t.Parallel()
		registry := NewCancellationRegistry()
		workID := uuid.New()
		registry.Register(workID, func() {})
		registry.Unregister(workID)

		require.NoError(t, registry.WaitUntilEmpty(t.Context()))
	})

	t.Run("returns the context error from wait when work stays registered", func(t *testing.T) {
		t.Parallel()
		registry := NewCancellationRegistry()
		registry.Register(uuid.New(), func() {})
		waitContext, cancelWait := context.WithTimeout(t.Context(), 50*time.Millisecond)
		t.Cleanup(cancelWait)

		require.ErrorIs(t, registry.WaitUntilEmpty(waitContext), context.DeadlineExceeded)
	})
}
