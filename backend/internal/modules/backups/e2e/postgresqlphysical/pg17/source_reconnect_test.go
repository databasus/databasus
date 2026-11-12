package pg17_test

import (
	"testing"

	"databasus-backend/internal/app/apptest/physicaltest"
)

func TestWalSourceReconnect(t *testing.T) {
	t.Parallel()
	t.Run("restores original queued WAL after a foreign source is refused across restart", func(t *testing.T) {
		t.Parallel()
		physicaltest.RunForeignSourceQueuedRecovery(t, "postgres:17")
	})
	t.Run("restores a new full backup after the stable address switches to a promoted replica", func(t *testing.T) {
		t.Parallel()
		physicaltest.RunStableEndpointPromotion(t, "postgres:17")
	})
}
