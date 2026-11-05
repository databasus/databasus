package logical

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runGetTimescaleDBVersionScenarios(t *testing.T, server testServer) {
	t.Helper()

	t.Run("returns empty version without the extension", func(t *testing.T) {
		t.Parallel()
		database := server.createDatabase(t)

		timescaleDBVersion, err := database.newAdminSource().GetTimescaleDBVersion(t.Context(), newTestLogger(t))

		require.NoError(t, err)
		assert.Empty(t, timescaleDBVersion)
	})
}
