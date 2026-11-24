package container

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeImageHost answers EnsureImage like Docker: a local image is used as it
// is, any other reference is pulled unless the registry lacks it.
type fakeImageHost struct {
	localImages       []string
	unavailableImages []string
	ensuredImages     []string
	pulledImages      []string
}

func (f *fakeImageHost) EnsureImage(_ context.Context, imageReference string) error {
	f.ensuredImages = append(f.ensuredImages, imageReference)

	if slices.Contains(f.unavailableImages, imageReference) {
		return errors.New("manifest unknown")
	}

	if !slices.Contains(f.localImages, imageReference) {
		f.pulledImages = append(f.pulledImages, imageReference)
	}

	return nil
}

func TestEnsureJobImage(t *testing.T) {
	t.Parallel()

	ensure := func(t *testing.T, host *fakeImageHost, repository string, spawn SpawnRequest) (string, error) {
		t.Helper()

		return ensureJobImage(t.Context(), host, jobImageRequest{
			spawn:           spawn,
			engineImageRepo: repository,
			pinnedVersions:  newTestPinnedVersions(t),
		}, slog.New(slog.DiscardHandler))
	}

	t.Run(
		"requests the pinned version when the host holds images of the major and an older version",
		func(t *testing.T) {
			t.Parallel()
			host := &fakeImageHost{localImages: []string{
				"databasus/verification-postgres:18",
				"databasus/verification-postgres:18.6",
			}}

			restoreImage, err := ensure(t, host, "databasus/verification-postgres", SpawnRequest{PgMajor: "18"})

			require.NoError(t, err)
			assert.Equal(t, "databasus/verification-postgres:18.7", restoreImage)
			assert.Equal(t, []string{"databasus/verification-postgres:18.7"}, host.ensuredImages)
			assert.Equal(t, []string{"databasus/verification-postgres:18.7"}, host.pulledImages)
		},
	)

	t.Run("requests the pinned version from a configured repository", func(t *testing.T) {
		t.Parallel()
		host := &fakeImageHost{}

		restoreImage, err := ensure(t, host, "example/pg", SpawnRequest{PgMajor: "17"})

		require.NoError(t, err)
		assert.Equal(t, "example/pg:17.12", restoreImage)
	})

	t.Run(
		"falls back to the official image of the same version when the bundled image is unavailable",
		func(t *testing.T) {
			t.Parallel()
			host := &fakeImageHost{unavailableImages: []string{"databasus/verification-postgres:18.7"}}

			restoreImage, err := ensure(t, host, "databasus/verification-postgres", SpawnRequest{PgMajor: "18"})

			require.NoError(t, err)
			assert.Equal(t, "postgres:18.7", restoreImage)
			assert.Equal(t, []string{"databasus/verification-postgres:18.7", "postgres:18.7"}, host.ensuredImages)
		},
	)

	t.Run("falls back to the official image when a configured repository lacks the version tag", func(t *testing.T) {
		t.Parallel()
		host := &fakeImageHost{
			localImages:       []string{"example/pg:18"},
			unavailableImages: []string{"example/pg:18.7"},
		}

		restoreImage, err := ensure(t, host, "example/pg", SpawnRequest{PgMajor: "18"})

		require.NoError(t, err)
		assert.Equal(t, "postgres:18.7", restoreImage)
	})

	t.Run("fails without another image when the configured official repository lacks the version", func(t *testing.T) {
		t.Parallel()

		for _, officialRepository := range []string{"postgres", "library/postgres", "docker.io/library/postgres"} {
			host := &fakeImageHost{unavailableImages: []string{officialRepository + ":18.7"}}

			_, err := ensure(t, host, officialRepository, SpawnRequest{PgMajor: "18"})

			require.Error(t, err, officialRepository)
			assert.Equal(t, []string{officialRepository + ":18.7"}, host.ensuredImages, officialRepository)
		}
	})

	t.Run("uses the extension version image without fallback when the job uses TimescaleDB", func(t *testing.T) {
		t.Parallel()
		host := &fakeImageHost{unavailableImages: []string{"timescale/timescaledb:2.17.0-pg17"}}

		_, err := ensure(t, host, "databasus/verification-postgres", SpawnRequest{
			PgMajor:            "17",
			TimescaledbVersion: "2.17.0",
		})

		require.Error(t, err)
		assert.Equal(t, []string{"timescale/timescaledb:2.17.0-pg17"}, host.ensuredImages)
	})

	t.Run("fails before pulling when the major has no pinned version", func(t *testing.T) {
		t.Parallel()
		host := &fakeImageHost{}

		_, err := ensure(t, host, "databasus/verification-postgres", SpawnRequest{PgMajor: "16"})

		require.Error(t, err)
		assert.Empty(t, host.ensuredImages)
	})
}
