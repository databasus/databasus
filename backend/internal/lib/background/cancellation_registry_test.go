package background

import (
	"context"
	"testing"

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
}
