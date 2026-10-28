package app

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadEnvironment(t *testing.T) {
	t.Run("returns an error naming the variable when a required one is empty", func(t *testing.T) {
		t.Setenv("ENV_MODE", "")
		t.Setenv("DATABASE_DSN", "host=localhost dbname=databasus")

		_, err := LoadEnvironment(t.Context(), newIsolatedLoadSpec(t, true))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "ENV_MODE")
	})
}

func TestWalStreamTimings(t *testing.T) {
	t.Run("uses production timings when the server loads its environment", func(t *testing.T) {
		t.Setenv("ENV_MODE", "development")
		environment, err := LoadEnvironment(t.Context(), newIsolatedLoadSpec(t, false))
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, environment.Logging.Shutdown(context.WithoutCancel(t.Context()))) })
		assert.Equal(t, 5*time.Minute, environment.WalStreamTimings.ForcedRotationInterval)
		assert.Equal(t, 30*time.Minute, environment.WalStreamTimings.ArchiveStalenessThreshold)
		assert.Zero(t, environment.WalStreamTimings.LocalQueuePauseBytes)
	})
}
