package walstreaming

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

func Test_RealignQueueToFork_DropsFormerTimelineSegmentsFromForkSegmentAndPlacesResumePoint(t *testing.T) {
	watchDir := t.TempDir()
	pendingDir := filepath.Join(watchDir, pendingUploadDirName)
	require.NoError(t, os.MkdirAll(pendingDir, 0o700))

	writeSegmentOfSize(t, pendingDir, walName(1, 41), testWalSegmentSize)
	writeSegmentOfSize(t, pendingDir, walName(1, 42), testWalSegmentSize)
	writeSegmentOfSize(t, watchDir, walName(1, 43), testWalSegmentSize)
	writeSegmentOfSize(t, watchDir, walName(2, 42), testWalSegmentSize)
	fork := pgphysical.WalReceiverFork{
		LeftTimelineID: 1,
		ForkLSN:        walmath.WalSegmentNo(42).FirstLSN(uint64(testWalSegmentSize)) + 0x148,
	}

	droppedCount, err := realignQueueToFork(watchDir, fork, testWalSegmentSize)
	require.NoError(t, err)

	require.Equal(t, 2, droppedCount)
	require.NoFileExists(t, filepath.Join(pendingDir, walName(1, 42)))
	require.NoFileExists(t, filepath.Join(watchDir, walName(1, 43)))
	require.FileExists(
		t,
		filepath.Join(pendingDir, walName(1, 41)),
		"the segment before the fork segment holds the line",
	)
	require.FileExists(t, filepath.Join(watchDir, walName(2, 42)), "the promoted timeline's copy holds the line")

	placeholder, err := os.Stat(filepath.Join(watchDir, walName(1, 42)+".partial"))
	require.NoError(t, err)
	require.Equal(t, testWalSegmentSize, placeholder.Size())
}

func Test_RealignQueueToFork_WhenQueueIsEmpty_PlacesResumePointAtForkSegment(t *testing.T) {
	watchDir := t.TempDir()
	fork := pgphysical.WalReceiverFork{
		LeftTimelineID: 3,
		ForkLSN:        walmath.WalSegmentNo(7).FirstLSN(uint64(testWalSegmentSize)),
	}

	droppedCount, err := realignQueueToFork(watchDir, fork, testWalSegmentSize)
	require.NoError(t, err)

	require.Zero(t, droppedCount)
	require.FileExists(t, filepath.Join(watchDir, walName(3, 7)+".partial"))
}
