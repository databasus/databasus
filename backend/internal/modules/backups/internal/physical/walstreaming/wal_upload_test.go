package walstreaming

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

func Test_BuildWalSegmentArtifactReader_WhenLargeSegment_StreamsAndCountsArtifact(t *testing.T) {
	dir := t.TempDir()
	localPath := filepath.Join(dir, "000000010000000000000001")
	require.NoError(t, os.WriteFile(localPath, bytes.Repeat([]byte("x"), 2*1024*1024), 0o600))

	artifact, salt, iv, err := buildWalSegmentArtifactReader(
		t.Context(), slog.Default(),
		localPath,
		catalog.BackupEncryptionNone,
		nil,
		uuid.New(),
	)
	require.NoError(t, err)
	require.Empty(t, salt)
	require.Empty(t, iv)

	body, err := io.ReadAll(artifact)
	require.NoError(t, err)

	compressedSizeBytes, err := artifact.wait()
	require.NoError(t, err)
	require.Equal(t, int64(len(body)), compressedSizeBytes)
	require.NotEmpty(t, body)
}

func segmentStartLSN(segmentNo uint64) walmath.LSN {
	return walmath.LSN(segmentNo * uint64(testWalSegmentSize))
}
