package enginetest

import (
	"path/filepath"
	"testing"
)

// GetTestdataDirectory returns the absolute path of a directory under the testdata of the running
// test package.
func GetTestdataDirectory(t *testing.T, pathParts ...string) string {
	t.Helper()

	directory, err := filepath.Abs(filepath.Join(append([]string{"testdata"}, pathParts...)...))
	if err != nil {
		t.Fatalf("resolve the testdata directory: %v", err)
	}

	return directory
}
