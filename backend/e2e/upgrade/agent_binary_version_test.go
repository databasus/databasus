package upgrade

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/process"
)

func getAgentBinaryVersion(t *testing.T, binaryPath string) string {
	t.Helper()

	output, err := process.CombinedOutput(
		t.Context(),
		process.Spec{Path: binaryPath, Args: []string{"version"}, Dir: filepath.Dir(binaryPath)},
	)
	require.NoError(t, err, "probe agent version: %s", output)

	return strings.TrimSpace(string(output))
}
