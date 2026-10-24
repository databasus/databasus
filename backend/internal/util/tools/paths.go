package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// archAssetsKey returns the assets/tools/<key>/ subdirectory name for the
// current GOOS+GOARCH: x64 for linux/amd64 and arm for linux/arm64. A client
// binary lives at assets/tools/<key>/<engine>/<engine>-<bundle>/bin/<command>,
// except MongoDB's single set under mongodb/bin/. Panics on unsupported
// combinations because the app cannot operate without DB client binaries.
func archAssetsKey() string {
	if runtime.GOOS == "linux" {
		if runtime.GOARCH == "arm64" {
			return "arm"
		}

		return "x64"
	}

	panic(fmt.Sprintf("unsupported OS/arch for DB client tools: %s/%s",
		runtime.GOOS, runtime.GOARCH))
}

// AssetsToolsDir returns the absolute path to assets/tools/<arch-key>/.
// Walks up from cwd looking for the directory. In Docker (cwd=/app) this
// resolves to /app/assets/tools/<arch-key>/.
var AssetsToolsDir = sync.OnceValue(func() string {
	key := archAssetsKey()

	cwd, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("could not get cwd: %v", err))
	}

	candidate := cwd
	for {
		path := filepath.Join(candidate, "assets", "tools", key)
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return path
		}

		parent := filepath.Dir(candidate)
		if parent == candidate {
			break
		}
		candidate = parent
	}

	panic(fmt.Sprintf("could not locate assets/tools/%s starting from %s", key, cwd))
})
