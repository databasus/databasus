package testimages_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testimages"
)

func TestGetReference(t *testing.T) {
	t.Parallel()

	t.Run("selects different immutable artifacts for each architecture", func(t *testing.T) {
		t.Parallel()

		amd64Image, err := testimages.GetReference(testimages.Selection{
			Image: "postgres:17", Architecture: "amd64",
		})
		require.NoError(t, err)

		arm64Image, err := testimages.GetReference(testimages.Selection{
			Image: "postgres:17", Architecture: "arm64",
		})
		require.NoError(t, err)

		assert.Contains(t, amd64Image, "postgres:17@sha256:")
		assert.Contains(t, arm64Image, "postgres:17@sha256:")
		assert.NotEqual(t, amd64Image, arm64Image)
	})

	t.Run("rejects a historical image on its unsupported architecture", func(t *testing.T) {
		t.Parallel()

		_, err := testimages.GetReference(testimages.Selection{
			Image: "mysql:5.7", Architecture: "arm64",
		})

		require.ErrorIs(t, err, testimages.ErrUnavailablePlatform)
	})

	t.Run("rejects an image without a recorded reference", func(t *testing.T) {
		t.Parallel()

		_, err := testimages.GetReference(testimages.Selection{
			Image: "unrecorded:latest", Architecture: "amd64",
		})

		require.ErrorIs(t, err, testimages.ErrUnknownImage)
	})
}
