package clienttools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindBundleRoot(t *testing.T) {
	t.Parallel()

	t.Run("finds the root from a nested directory", func(t *testing.T) {
		t.Parallel()
		architectureKey, err := getArchitectureKey()
		require.NoError(t, err)
		projectDir := t.TempDir()
		bundleRoot := filepath.Join(projectDir, "assets", "tools", architectureKey)
		nestedDir := filepath.Join(projectDir, "backend", "internal", "app")
		require.NoError(t, os.MkdirAll(bundleRoot, 0o755))
		require.NoError(t, os.MkdirAll(nestedDir, 0o755))

		foundRoot, err := FindBundleRoot(nestedDir)

		require.NoError(t, err)
		assert.Equal(t, bundleRoot, foundRoot)
	})

	t.Run("returns an error when no directory above holds the bundles", func(t *testing.T) {
		t.Parallel()

		_, err := FindBundleRoot(t.TempDir())

		require.ErrorIs(t, err, ErrBundleRootNotFound)
	})
}
