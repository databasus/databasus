package usecases_physical_postgresql

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/config"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	databases_bridge "databasus-backend/internal/features/databases"
	"databasus-backend/internal/util/logger"
)

func Test_WalStream_BackpressureWatermarks_ScaleWithWalSegmentSize(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)
	customSegSize := int64(512 * 1024 * 1024)
	fixture.SourceDB.WalSegmentSizeBytes = &customSegSize

	supervisor := NewWalStreamSupervisor(WalStreamSpec{
		ClientTools:  pgshared.ClientTools{Root: config.GetEnv().ClientToolsDir},
		DatabaseID:   fixture.DB.ID,
		BackupStatus: databases_bridge.GetModule(),
		SourceDB:     fixture.SourceDB,
		WatchDirRoot: t.TempDir(),
		Logger:       logger.GetLogger(),
	})

	require.Equal(t, 8*customSegSize, supervisor.highWatermarkBytes.Load())
	require.Equal(t, 8*customSegSize/5, supervisor.lowWatermarkBytes.Load())
}

func Test_BacklogBytes_WhenSegmentsStagedOutOfResumePath_CountsThemTowardTheWatermark(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	supervisor := NewWalStreamSupervisor(WalStreamSpec{
		ClientTools:  pgshared.ClientTools{Root: config.GetEnv().ClientToolsDir},
		DatabaseID:   fixture.DB.ID,
		BackupStatus: databases_bridge.GetModule(),
		SourceDB:     fixture.SourceDB,
		WatchDirRoot: t.TempDir(),
		Logger:       logger.GetLogger(),
	})

	pendingUploadDir := filepath.Join(supervisor.watchDir, pendingUploadDirName)
	require.NoError(t, os.MkdirAll(pendingUploadDir, 0o700))

	for _, segmentNo := range []uint64{50, 51} {
		writeSegmentOfSize(t, supervisor.watchDir, walName(1, segmentNo), testWalSegmentSize)
	}

	for _, stagedSegmentNo := range []uint64{47, 48, 49} {
		writeSegmentOfSize(t, pendingUploadDir, walName(1, stagedSegmentNo), testWalSegmentSize)
	}

	require.Equal(t, 5*testWalSegmentSize, supervisor.backlogBytes(),
		"segments awaiting upload still occupy the local queue's disk, wherever they were staged")
}
