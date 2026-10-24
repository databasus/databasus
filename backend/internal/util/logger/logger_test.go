package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Shutdown must not build the logging just to close it: that would create the log file on a
// process that never logged.
func TestShutdown(t *testing.T) {
	t.Run("returns without building the logging when nothing logged", func(t *testing.T) {
		require.Nil(t, currentLogging.Load(), "no test in this package may log before this one")

		require.NoError(t, Shutdown(t.Context()))

		assert.Nil(t, currentLogging.Load())
	})
}
