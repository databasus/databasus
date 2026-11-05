package walstreaming

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const testWalSegmentSize = int64(16 * 1024 * 1024)

func walName(timeline uint32, segNo uint64) string {
	const segmentsPerLogID = 256

	return fmt.Sprintf("%08X%08X%08X", timeline, segNo/segmentsPerLogID, segNo%segmentsPerLogID)
}

func writeSegmentOfSize(t *testing.T, dir, name string, sizeBytes int64) string {
	t.Helper()

	path := filepath.Join(dir, name)

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	require.NoError(t, err)

	require.NoError(t, file.Truncate(sizeBytes))
	require.NoError(t, file.Close())

	return path
}

func writeWalFile(t *testing.T, dir, name string) string {
	t.Helper()

	return writeSegmentOfSize(t, dir, name, testWalSegmentSize)
}
