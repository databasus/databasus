package containers

import (
	"path/filepath"
	"runtime"
	"testing"
)

func getFixtureDirectory(t *testing.T, fixture string) string {
	t.Helper()

	_, sourceFile, _, isResolved := runtime.Caller(0)
	if !isResolved {
		t.Fatal("failed to resolve the container fixture directory")
	}

	return filepath.Join(filepath.Dir(sourceFile), "testdata", fixture)
}
