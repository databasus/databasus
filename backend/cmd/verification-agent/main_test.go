package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersion(t *testing.T) {
	t.Run("prints the version without creating files", func(t *testing.T) {
		binaryPath := filepath.Join(t.TempDir(), "verification-agent")
		build := exec.CommandContext(t.Context(), "go", "build", "-o", binaryPath, ".")
		output, err := build.CombinedOutput()
		require.NoError(t, err, "%s", output)

		workingDir := t.TempDir()
		command := exec.CommandContext(t.Context(), binaryPath, "version")
		command.Dir = workingDir
		output, err = command.CombinedOutput()
		require.NoError(t, err, "%s", output)
		require.Equal(t, "dev\n", string(output))

		files, err := os.ReadDir(workingDir)
		require.NoError(t, err)
		require.Empty(t, files)
	})
}
