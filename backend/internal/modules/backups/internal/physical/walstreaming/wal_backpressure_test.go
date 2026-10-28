package walstreaming

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/modules/databases"
)

func TestLocalQueuePauseBytes(t *testing.T) {
	t.Run("scales the watermark when the WAL segment size exceeds the default", func(t *testing.T) {
		customSegSize := int64(512 * 1024 * 1024)
		sourceDB := databases.PostgresqlPhysicalConnectionSettings{WalSegmentSizeBytes: &customSegSize}

		require.Equal(t, 8*customSegSize, getDefaultLocalQueuePauseBytes(sourceDB))
		require.Equal(t, 8*customSegSize/5, getDefaultLocalQueuePauseBytes(sourceDB)/5)
	})
}

func TestWalBacklogBytes(t *testing.T) {
	t.Run("counts every staged segment when it leaves the receivewal resume directory", func(t *testing.T) {
		watchDir := t.TempDir()

		pendingUploadDir := filepath.Join(watchDir, pendingUploadDirName)
		require.NoError(t, os.MkdirAll(pendingUploadDir, 0o700))

		for _, segmentNo := range []uint64{50, 51} {
			writeSegmentOfSize(t, watchDir, walName(1, segmentNo), testWalSegmentSize)
		}

		for _, stagedSegmentNo := range []uint64{47, 48, 49} {
			writeSegmentOfSize(t, pendingUploadDir, walName(1, stagedSegmentNo), testWalSegmentSize)
		}

		require.Equal(t, 5*testWalSegmentSize, getWalBacklogBytes(watchDir),
			"segments awaiting upload still occupy the local queue's disk, wherever they were staged")
	})
}
