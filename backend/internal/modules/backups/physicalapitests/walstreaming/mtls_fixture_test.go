package walstreaming_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func mtlsDirectory(t *testing.T) string {
	t.Helper()
	_, file, _, isResolved := runtime.Caller(0)
	require.True(t, isResolved)
	return filepath.Join(filepath.Dir(file), "../testdata/mtls")
}

func readMtlsFile(t *testing.T, name string) string {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(mtlsDirectory(t), name))
	require.NoError(t, err)
	return string(contents)
}
