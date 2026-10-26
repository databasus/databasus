package clienttools

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// FindBundleRoot walks up from startDir to the first assets/tools/<architecture> directory. In the
// image the working directory is /app, so the root is /app/assets/tools/<architecture>. A client
// binary lives at <root>/<engine>/<engine>-<bundle>/bin/<command>, except MongoDB's single set
// under <root>/mongodb/bin/.
func FindBundleRoot(startDir string) (string, error) {
	architectureKey, err := getArchitectureKey()
	if err != nil {
		return "", err
	}

	candidate := startDir

	for {
		path := filepath.Join(candidate, "assets", "tools", architectureKey)
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return path, nil
		}

		parent := filepath.Dir(candidate)
		if parent == candidate {
			return "", fmt.Errorf("%w: assets/tools/%s above %s", ErrBundleRootNotFound, architectureKey, startDir)
		}

		candidate = parent
	}
}

func getArchitectureKey() (string, error) {
	if runtime.GOOS != "linux" {
		return "", fmt.Errorf("%w: %s/%s", ErrUnsupportedPlatform, runtime.GOOS, runtime.GOARCH)
	}

	if runtime.GOARCH == "arm64" {
		return "arm", nil
	}

	return "x64", nil
}
