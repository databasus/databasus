package diskusage

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	t.Parallel()

	t.Run("reads the usage of the filesystem that holds the path", func(t *testing.T) {
		t.Parallel()

		folderUsage, err := Get(t.TempDir())

		require.NoError(t, err)
		assert.Positive(t, folderUsage.TotalBytes)
		assert.LessOrEqual(t, folderUsage.FreeBytes, folderUsage.TotalBytes)
		assert.LessOrEqual(t, folderUsage.UsedBytes, folderUsage.TotalBytes)
	})

	t.Run("returns an error when the path does not exist", func(t *testing.T) {
		t.Parallel()

		_, err := Get(filepath.Join(t.TempDir(), "missing"))

		require.Error(t, err)
	})
}
