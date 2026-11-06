package containers

import (
	_ "embed"
	"encoding/json"
	"runtime"
	"testing"
)

//go:embed unavailable_images.json
var unavailableImagesJSON []byte

type imageAvailabilitySpec struct {
	Architecture string
	Image        string
}

func getUnavailableImageReason(spec imageAvailabilitySpec) string {
	switch spec.Architecture {
	case "aarch64":
		spec.Architecture = "arm64"
	case "x86_64", "x64":
		spec.Architecture = "amd64"
	}

	var policy map[string]map[string]string
	if err := json.Unmarshal(unavailableImagesJSON, &policy); err != nil {
		panic(err)
	}

	return policy[spec.Architecture][spec.Image]
}

func skipUnavailableImage(t *testing.T, image string) {
	t.Helper()

	if reason := getUnavailableImageReason(
		imageAvailabilitySpec{Architecture: runtime.GOARCH, Image: image},
	); reason != "" {
		t.Skipf("%s on %s: %s", image, runtime.GOARCH, reason)
	}
}
