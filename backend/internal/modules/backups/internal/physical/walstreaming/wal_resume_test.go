package walstreaming

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

func Test_MovePendingUploadsOutOfResumePath_MovesOnlySegmentsBelowFloor(t *testing.T) {
	watchDir := t.TempDir()

	for _, segmentNo := range []uint64{40, 41, 42, 43} {
		writeSegmentOfSize(t, watchDir, walName(1, segmentNo), testWalSegmentSize)
	}

	movedCount, err := movePendingUploadsOutOfResumePath(watchDir, walmath.WalSegmentNo(42), testWalSegmentSize)
	require.NoError(t, err)
	require.Equal(t, 2, movedCount)

	for _, movedSegment := range []uint64{40, 41} {
		require.NoFileExists(t, filepath.Join(watchDir, walName(1, movedSegment)))
		require.FileExists(t, filepath.Join(watchDir, pendingUploadDirName, walName(1, movedSegment)))
	}

	for _, keptSegment := range []uint64{42, 43} {
		require.FileExists(t, filepath.Join(watchDir, walName(1, keptSegment)))
	}
}

func Test_MovePendingUploadsOutOfResumePath_LeavesHistoryFilesAndSubdirectories(t *testing.T) {
	watchDir := t.TempDir()

	writeSegmentOfSize(t, watchDir, "00000002.history", 64)
	require.NoError(t, os.MkdirAll(filepath.Join(watchDir, "archive_status"), 0o700))

	movedCount, err := movePendingUploadsOutOfResumePath(watchDir, walmath.WalSegmentNo(99), testWalSegmentSize)
	require.NoError(t, err)
	require.Zero(t, movedCount)

	require.FileExists(t, filepath.Join(watchDir, "00000002.history"))
	require.DirExists(t, filepath.Join(watchDir, "archive_status"))
}

func Test_MovePendingUploadsOutOfResumePath_WhenSegmentIsShort_LeavesItForRestreaming(t *testing.T) {
	watchDir := t.TempDir()

	writeSegmentOfSize(t, watchDir, walName(1, 40), testWalSegmentSize)
	writeSegmentOfSize(t, watchDir, walName(1, 41), testWalSegmentSize/2)

	movedCount, err := movePendingUploadsOutOfResumePath(watchDir, walmath.WalSegmentNo(42), testWalSegmentSize)
	require.NoError(t, err)
	require.Equal(t, 1, movedCount)

	require.FileExists(t, filepath.Join(watchDir, pendingUploadDirName, walName(1, 40)))
	require.FileExists(
		t,
		filepath.Join(watchDir, walName(1, 41)),
		"a torn segment is re-streamed, and it cannot anchor the resume point, so staging it would only risk uploading it",
	)
	require.NoFileExists(t, filepath.Join(watchDir, pendingUploadDirName, walName(1, 41)))
}
