package container

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

// fakeImageRegistry keeps local tags per repository, as Docker lists them for
// a reference filter, and records which references were removed.
type fakeImageRegistry struct {
	tagsByRepository map[string][]string
	listErr          error
	removeErrs       map[string]error
	listedRepos      []string
	removedImages    []string
}

func (f *fakeImageRegistry) ListImageTags(_ context.Context, repository string) ([]ImageTag, error) {
	f.listedRepos = append(f.listedRepos, repository)
	if f.listErr != nil {
		return nil, f.listErr
	}

	imageTags := make([]ImageTag, 0, len(f.tagsByRepository[repository]))
	for _, tag := range f.tagsByRepository[repository] {
		imageTags = append(imageTags, ImageTag{Reference: repository + ":" + tag, Tag: tag})
	}

	return imageTags, nil
}

func (f *fakeImageRegistry) RemoveImageTag(_ context.Context, imageReference string) error {
	if err := f.removeErrs[imageReference]; err != nil {
		return err
	}

	f.removedImages = append(f.removedImages, imageReference)

	return nil
}

func TestRemoveSupersededVerificationImages(t *testing.T) {
	t.Parallel()

	removeSuperseded := func(t *testing.T, registry *fakeImageRegistry, repository string) string {
		t.Helper()

		var logOutput bytes.Buffer
		removeSupersededVerificationImages(
			t.Context(),
			registry,
			repository,
			newTestPinnedVersions(t),
			slog.New(slog.NewTextHandler(&logOutput, nil)),
		)

		return logOutput.String()
	}

	t.Run("removes the major and older version tags and keeps the pinned versions", func(t *testing.T) {
		t.Parallel()
		registry := &fakeImageRegistry{tagsByRepository: map[string][]string{
			"databasus/verification-postgres": {"18", "18.6", "18.7", "17.12", "18-v3.70.1"},
		}}

		removeSuperseded(t, registry, "databasus/verification-postgres")

		assert.ElementsMatch(t, []string{
			"databasus/verification-postgres:18",
			"databasus/verification-postgres:18.6",
			"databasus/verification-postgres:18-v3.70.1",
		}, registry.removedImages)
	})

	t.Run("cleans only the configured repository when another one is configured", func(t *testing.T) {
		t.Parallel()
		registry := &fakeImageRegistry{tagsByRepository: map[string][]string{
			"example/pg":                      {"18", "18.7"},
			"databasus/verification-postgres": {"18"},
		}}

		removeSuperseded(t, registry, "example/pg")

		assert.Equal(t, []string{"example/pg"}, registry.listedRepos)
		assert.Equal(t, []string{"example/pg:18"}, registry.removedImages)
	})

	t.Run("removes nothing when the official repository is configured", func(t *testing.T) {
		t.Parallel()

		for _, officialRepository := range []string{"postgres", "docker.io/library/postgres"} {
			registry := &fakeImageRegistry{tagsByRepository: map[string][]string{
				officialRepository: {"18.6"},
			}}

			removeSuperseded(t, registry, officialRepository)

			assert.Empty(t, registry.listedRepos, officialRepository)
			assert.Empty(t, registry.removedImages, officialRepository)
		}
	})

	t.Run("keeps an image and removes the others when a container uses it", func(t *testing.T) {
		t.Parallel()
		registry := &fakeImageRegistry{
			tagsByRepository: map[string][]string{
				"databasus/verification-postgres": {"18", "18.6"},
			},
			removeErrs: map[string]error{
				"databasus/verification-postgres:18": errors.New("image is being used by running container"),
			},
		}

		logOutput := removeSuperseded(t, registry, "databasus/verification-postgres")

		assert.Equal(t, []string{"databasus/verification-postgres:18.6"}, registry.removedImages)
		assert.Contains(t, logOutput, "level=WARN")
		assert.Contains(t, logOutput, "could not remove superseded verification image")
	})

	t.Run("removes nothing and warns when the images cannot be listed", func(t *testing.T) {
		t.Parallel()
		registry := &fakeImageRegistry{listErr: errors.New("unsupported filter")}

		logOutput := removeSuperseded(t, registry, "databasus/verification-postgres")

		assert.Empty(t, registry.removedImages)
		assert.True(t, slices.Contains(registry.listedRepos, "databasus/verification-postgres"))
		assert.Contains(t, logOutput, "could not list verification images")
	})
}
