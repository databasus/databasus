package containers

import (
	_ "embed"
	"encoding/json"
	"runtime"
	"testing"
)

//go:embed unavailable_images.json
var unavailableImagesJSON []byte

func getUnavailableImageReason(architecture, image string) string {
	switch architecture {
	case "aarch64":
		architecture = "arm64"
	case "x86_64", "x64":
		architecture = "amd64"
	}

	var policy map[string]map[string]string
	if err := json.Unmarshal(unavailableImagesJSON, &policy); err != nil {
		panic(err)
	}

	return policy[architecture][image]
}

func skipUnavailableImage(t *testing.T, image string) {
	t.Helper()

	if reason := getUnavailableImageReason(runtime.GOARCH, image); reason != "" {
		t.Skipf("%s on %s: %s", image, runtime.GOARCH, reason)
	}
}
