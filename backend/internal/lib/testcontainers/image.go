package containers

import (
	"errors"
	"runtime"
	"testing"

	"databasus-backend/internal/lib/testimages"
)

func GetPinnedImage(t *testing.T, image string) string {
	t.Helper()

	reference, err := testimages.GetReference(testimages.Selection{
		Image:        image,
		Architecture: runtime.GOARCH,
	})
	if errors.Is(err, testimages.ErrUnavailablePlatform) {
		skipUnavailableImage(t, image)
	}

	if err != nil {
		t.Fatalf("select immutable test image: %v", err)
	}

	return reference
}
