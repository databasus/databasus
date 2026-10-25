package app

import (
	"testing"

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
