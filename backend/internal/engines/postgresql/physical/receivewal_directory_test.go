package physical

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

func TestGetResumeSegmentNo(t *testing.T) {
	t.Parallel()
	t.Run("resumes after the highest complete segment with partial and torn files", func(t *testing.T) {
		t.Parallel()

		watchDir := t.TempDir()

		writeSegmentOfSize(
			t,
			filepath.Join(watchDir, walmath.WalSegmentNo(40).GetFilename(1, walmath.DefaultSegmentSizeBytes)),
			int64(walmath.DefaultSegmentSizeBytes),
		)
		writeSegmentOfSize(
			t,
			filepath.Join(watchDir, walmath.WalSegmentNo(41).GetFilename(1, walmath.DefaultSegmentSizeBytes)),
			int64(walmath.DefaultSegmentSizeBytes),
		)
		writeSegmentOfSize(
			t,
			filepath.Join(watchDir, walmath.WalSegmentNo(42).GetFilename(1, walmath.DefaultSegmentSizeBytes)),
			int64(walmath.DefaultSegmentSizeBytes)/2,
		)
		writeSegmentOfSize(
			t,
			filepath.Join(
				watchDir,
				walmath.WalSegmentNo(43).GetFilename(1, walmath.DefaultSegmentSizeBytes)+".partial",
			),
			int64(walmath.DefaultSegmentSizeBytes),
		)
		writeSegmentOfSize(t, filepath.Join(watchDir, "00000002.history"), 64)

		resumeSegmentNo, isResumeSegmentFound := GetResumeSegmentNo(watchDir, int64(walmath.DefaultSegmentSizeBytes))

		require.True(t, isResumeSegmentFound)
		require.Equal(t, walmath.WalSegmentNo(42), resumeSegmentNo,
			"a torn segment is re-streamed, so 42 is not a resume point — 41 is the highest complete one")
	})
}

func TestGetResumeSegmentNoEmpty(t *testing.T) {
	t.Parallel()
	t.Run("reports no resume point without top level segments", func(t *testing.T) {
		t.Parallel()

		watchDir := t.TempDir()

		writeSegmentOfSize(t, filepath.Join(watchDir, "00000002.history"), 64)
		require.NoError(t, os.MkdirAll(filepath.Join(watchDir, "pending-upload"), 0o700))
		writeSegmentOfSize(
			t,
			filepath.Join(
				filepath.Join(watchDir, "pending-upload"),
				walmath.WalSegmentNo(40).GetFilename(1, walmath.DefaultSegmentSizeBytes),
			),
			int64(walmath.DefaultSegmentSizeBytes),
		)

		_, isResumeSegmentFound := GetResumeSegmentNo(watchDir, int64(walmath.DefaultSegmentSizeBytes))

		require.False(t, isResumeSegmentFound,
			"pg_receivewal reads only the top level, so a staged segment must not anchor the resume point")
	})
}

func writeSegmentOfSize(t *testing.T, localPath string, size int64) {
	t.Helper()
	file, err := os.Create(localPath)
	require.NoError(t, err)
	require.NoError(t, file.Truncate(size))
	require.NoError(t, file.Close())
}

func TestGetWalSegmentFiles(t *testing.T) {
	t.Parallel()
	t.Run("reports bounds and completeness with custom size segments and unrelated files", func(t *testing.T) {
		t.Parallel()
		directory := t.TempDir()
		const segmentSize = uint64(1024 * 1024)
		completeName := walmath.WalSegmentNo(257).GetFilename(2, segmentSize)
		tornName := walmath.WalSegmentNo(258).GetFilename(2, segmentSize)

		writeSegmentOfSize(t, filepath.Join(directory, completeName), int64(segmentSize))
		writeSegmentOfSize(t, filepath.Join(directory, tornName), int64(segmentSize/2))
		writeSegmentOfSize(t, filepath.Join(directory, completeName+".partial"), int64(segmentSize))
		writeSegmentOfSize(t, filepath.Join(directory, "00000002.history"), 10)
		require.NoError(t, os.Mkdir(filepath.Join(directory, "archive_status"), 0o700))

		segments := GetWalSegmentFiles(directory, int64(segmentSize))

		require.Len(t, segments, 2)
		require.Equal(t, completeName, segments[0].Filename)
		require.Equal(t, filepath.Join(directory, completeName), segments[0].LocalPath)
		require.Equal(t, 2, segments[0].TimelineID)
		require.Equal(t, walmath.LSN(257*segmentSize), segments[0].StartLSN)
		require.Equal(t, walmath.LSN(258*segmentSize), segments[0].EndLSN)
		require.True(t, segments[0].IsComplete)

		require.Equal(t, tornName, segments[1].Filename)
		require.False(t, segments[1].IsComplete)

		require.True(t, IsPartialWalFile(completeName+".partial"))
		require.False(t, IsPartialWalFile(completeName))
		require.True(t, IsTimelineHistoryFile("00000002.history"))
		require.False(t, IsTimelineHistoryFile(completeName))
		require.Nil(t, GetWalSegmentFiles(filepath.Join(directory, "absent"), int64(segmentSize)))
	})
}
