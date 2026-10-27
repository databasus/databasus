package usecases_physical_postgresql

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/config"
	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/engines/postgresql/physical/walmath"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/util/logger"
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

func Test_RealignResumePath_WhenSourceUnreachable_LeavesQueueUntouched(t *testing.T) {
	unreachableSourceDB := databases.PostgresqlPhysicalConnectionSettings{
		Engine: pgphysical.Settings{
			Host:     "127.0.0.1",
			Port:     1,
			Username: "streamer",
			Password: "streamer",
		},
		WalStreamSlotName: "databasus_unreachable_source",
	}

	supervisor := NewWalStreamSupervisor(WalStreamSpec{
		ClientTools:  pgshared.ClientTools{Root: config.GetEnv().ClientToolsDir},
		DatabaseID:   uuid.New(),
		SourceDB:     unreachableSourceDB,
		WatchDirRoot: t.TempDir(),
		Logger:       logger.GetLogger(),
	})

	require.NoError(t, os.MkdirAll(supervisor.watchDir, 0o700))

	for _, segmentNo := range []uint64{60, 61} {
		writeSegmentOfSize(t, supervisor.watchDir, walName(1, segmentNo), testWalSegmentSize)
	}

	supervisor.realignResumePath(t.Context(), logger.GetLogger())

	for _, segmentNo := range []uint64{60, 61} {
		require.FileExists(t, filepath.Join(supervisor.watchDir, walName(1, segmentNo)),
			"without the slot's restart_lsn there is no floor to realign against, so the queue must stay put")
	}

	require.NoDirExists(t, filepath.Join(supervisor.watchDir, pendingUploadDirName))
}
