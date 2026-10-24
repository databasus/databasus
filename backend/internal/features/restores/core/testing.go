//go:build testhelpers

package restores_core

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"
)

// WriteToolStub writes a shell script that stands in for a restore tool. Error output is lost only
// when the tool writes to a real OS pipe created by os/exec, so there is no HTTP-level way to reach
// it, hence a stub binary instead of a controller test.
func WriteToolStub(t *testing.T, toolName, stubScript string) string {
	t.Helper()

	if runtime.GOOS == "windows" {
		t.Skip("stub restore tool relies on a POSIX shell")
	}

	stubPath := filepath.Join(t.TempDir(), toolName)
	require.NoError(t, os.WriteFile(stubPath, []byte(stubScript), 0o700))

	return stubPath
}

// WriteLateFailingToolStub writes a restore tool stub that reads the whole backup and exits with
// status 1, while errorLine reaches its error output only after the stub has exited. That is how a
// fast-exiting tool looks when the reader of its output is scheduled late, as on a loaded CI
// runner, and a capture that stops reading when the process exits loses the line every time.
// errorLine is quoted with single quotes in the script, so it must not contain one.
func WriteLateFailingToolStub(t *testing.T, toolName, errorLine string) string {
	t.Helper()

	return WriteToolStub(t, toolName, "#!/bin/sh\ncat > /dev/null\n"+
		"(sleep 0.2; echo '"+errorLine+"' >&2) &\n"+
		"exit 1\n")
}

func CompressToZstd(t *testing.T, payload []byte) []byte {
	t.Helper()

	var compressed bytes.Buffer

	zstdWriter, err := zstd.NewWriter(&compressed)
	require.NoError(t, err)

	_, err = zstdWriter.Write(payload)
	require.NoError(t, err)
	require.NoError(t, zstdWriter.Close())

	return compressed.Bytes()
}
