// Package upgrade checks that an instance of the newest release keeps its data and its backups
// after an upgrade to the current code. It drives both images through Docker, so it runs only
// through `just test-upgrade` in backend/, which names the images.
package upgrade

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	previousImageVariable = "DATABASUS_PREVIOUS_IMAGE"
	currentImageVariable  = "DATABASUS_IMAGE"
	failedStartMessage    = "upgrade probe: refusing to start"
)

func TestUpgrade(t *testing.T) {
	t.Parallel()

	t.Run("keeps every item and restores the backup after an upgrade from the newest release", func(t *testing.T) {
		t.Parallel()
		previousImage, currentImage := getUpgradeImages(t)
		t.Logf("upgrading %s to %s", previousImage, currentImage)

		_, err := runUpgrade(t, upgradeSpec{
			previousImage:     previousImage,
			currentImage:      currentImage,
			currentEntrypoint: nil,
		})

		require.NoError(t, err)
	})

	t.Run(
		"returns the log of the current instance and leaves nothing behind when it fails to start",
		func(t *testing.T) {
			t.Parallel()
			previousImage, _ := getUpgradeImages(t)

			resources, err := runUpgrade(t, upgradeSpec{
				previousImage:     previousImage,
				currentImage:      sourceImage,
				currentEntrypoint: []string{"sh", "-c", "echo '" + failedStartMessage + "' >&2; exit 1"},
			})

			require.ErrorIs(t, err, errInstanceExited)
			assert.Contains(t, err.Error(), failedStartMessage)

			docker, err := newDockerHost()
			require.NoError(t, err)
			defer docker.close()

			remainingNames, err := docker.getRemainingResources(t.Context(), resources)
			require.NoError(t, err)
			assert.Empty(t, remainingNames)
		},
	)
}

func getUpgradeImages(t *testing.T) (string, string) {
	t.Helper()

	//nolint:forbidigo // just test-upgrade names the images, outside any configuration of ours
	previousImage, currentImage := os.Getenv(previousImageVariable), os.Getenv(currentImageVariable)
	if previousImage == "" || currentImage == "" {
		t.Fatalf("%s and %s name the images to upgrade between; run the test with `just test-upgrade` in backend/",
			previousImageVariable, currentImageVariable)
	}

	return previousImage, currentImage
}
