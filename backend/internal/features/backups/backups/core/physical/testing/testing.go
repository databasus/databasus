//go:build testhelpers

package physical_testing

import (
	"testing"
	"time"

	"github.com/google/uuid"

	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	"databasus-backend/internal/storage"
	"databasus-backend/internal/util/walmath"
)

// Past the segment's long page header. A FULL's start_lsn carries this offset,
// while a segment's own bounds are file-aligned.
const FirstRecordOffset = 0x28

// Wide enough to keep a test FULL's start_lsn and stop_lsn apart, narrow enough
// that both stay inside the segment holding the start.
const FullLSNSpan = 0x100

func NewTestCompletedFullBackup(
	databaseID, storageID uuid.UUID,
	timelineID int,
	startLSN, stopLSN walmath.LSN,
) *physical_models.PhysicalFullBackup {
	return &physical_models.PhysicalFullBackup{
		DatabaseID:  databaseID,
		StorageID:   storageID,
		TimelineID:  timelineID,
		Status:      physical_enums.PhysicalBackupStatusCompleted,
		FileName:    new("test-full-" + uuid.New().String()),
		StartLSN:    &startLSN,
		StopLSN:     &stopLSN,
		CreatedAt:   time.Now().UTC(),
		CompletedAt: new(time.Now().UTC()),
	}
}

// NewTestInProgressFullBackup builds a FULL that is still running: no
// start/stop LSN, no stored file, status IN_PROGRESS. Pair it with
// CreateTestInFlightClaim to exercise cancel / delete-while-running paths.
func NewTestInProgressFullBackup(
	databaseID, storageID uuid.UUID,
	timelineID int,
) *physical_models.PhysicalFullBackup {
	return &physical_models.PhysicalFullBackup{
		DatabaseID: databaseID,
		StorageID:  storageID,
		TimelineID: timelineID,
		Status:     physical_enums.PhysicalBackupStatusInProgress,
		CreatedAt:  time.Now().UTC(),
	}
}

func NewTestCompletedIncrementalBackup(
	databaseID, storageID, rootFullBackupID uuid.UUID,
	parentIncrementalBackupID *uuid.UUID,
	timelineID int,
	startLSN, stopLSN walmath.LSN,
) *physical_models.PhysicalIncrementalBackup {
	return &physical_models.PhysicalIncrementalBackup{
		DatabaseID:                databaseID,
		StorageID:                 storageID,
		TimelineID:                timelineID,
		Status:                    physical_enums.PhysicalBackupStatusCompleted,
		FileName:                  new("test-incr-" + uuid.New().String()),
		StartLSN:                  &startLSN,
		StopLSN:                   &stopLSN,
		CreatedAt:                 time.Now().UTC(),
		CompletedAt:               new(time.Now().UTC()),
		RootFullBackupID:          rootFullBackupID,
		ParentIncrementalBackupID: parentIncrementalBackupID,
	}
}

func NewTestWalSegment(
	databaseID, storageID uuid.UUID,
	timelineID int,
	walFilename string,
	startLSN, endLSN walmath.LSN,
) *physical_models.PhysicalWalSegment {
	now := time.Now().UTC()

	return &physical_models.PhysicalWalSegment{
		DatabaseID:       databaseID,
		StorageID:        storageID,
		TimelineID:       timelineID,
		FileName:         new("test-wal-" + uuid.New().String() + ".zst"),
		WalFilename:      walFilename,
		StartLSN:         startLSN,
		EndLSN:           endLSN,
		CompressedSizeMb: 16,
		ReceivedAt:       now,
		ClaimedAt:        now,
	}
}

func NewTestWalHistoryFile(
	databaseID, storageID uuid.UUID,
	timelineID int,
) *physical_models.PhysicalWalHistoryFile {
	return &physical_models.PhysicalWalHistoryFile{
		DatabaseID:       databaseID,
		StorageID:        storageID,
		TimelineID:       timelineID,
		FileName:         "test-hist-" + uuid.New().String() + ".history.zst",
		HistoryFilename:  walmath.FormatHistoryFilename(uint32(timelineID)),
		CompressedSizeMb: 0.01,
		CreatedAt:        time.Now().UTC(),
	}
}

func CreateTestFullBackup(
	t *testing.T,
	fullBackup *physical_models.PhysicalFullBackup,
) *physical_models.PhysicalFullBackup {
	t.Helper()

	if err := physical_repositories.GetFullBackupRepository().Save(fullBackup); err != nil {
		t.Fatalf("save test full backup: %v", err)
	}

	return fullBackup
}

func CreateTestIncrementalBackup(
	t *testing.T,
	incrementalBackup *physical_models.PhysicalIncrementalBackup,
) *physical_models.PhysicalIncrementalBackup {
	t.Helper()

	if err := physical_repositories.GetIncrementalBackupRepository().Save(incrementalBackup); err != nil {
		t.Fatalf("save test incremental backup: %v", err)
	}

	return incrementalBackup
}

func CreateTestWalSegment(
	t *testing.T,
	walSegment *physical_models.PhysicalWalSegment,
) *physical_models.PhysicalWalSegment {
	t.Helper()

	if err := physical_repositories.GetWalSegmentRepository().Insert(walSegment); err != nil {
		t.Fatalf("save test wal segment: %v", err)
	}

	return walSegment
}

func CreateTestWalHistoryFile(
	t *testing.T,
	walHistoryFile *physical_models.PhysicalWalHistoryFile,
) *physical_models.PhysicalWalHistoryFile {
	t.Helper()

	if err := physical_repositories.GetWalHistoryRepository().Insert(walHistoryFile); err != nil {
		t.Fatalf("save test wal history: %v", err)
	}

	return walHistoryFile
}

// CreateTestInFlightClaim reserves the cross-table single-in-flight slot for a
// database, simulating a running backup so cancel / delete paths have something
// to stop.
func CreateTestInFlightClaim(
	t *testing.T,
	databaseID, backupID uuid.UUID,
	backupType physical_enums.PhysicalBackupType,
) {
	t.Helper()

	claimed, err := physical_repositories.GetInFlightBackupRepository().
		Claim(storage.GetDb(), physical_repositories.ClaimSpec{
			DatabaseID: databaseID,
			BackupType: backupType,
			BackupID:   backupID,
		})
	if err != nil {
		t.Fatalf("claim test in-flight backup: %v", err)
	}
	if !claimed {
		t.Fatalf("claim test in-flight backup: database already has an in-flight claim")
	}
}

// Call from t.Cleanup; deletes in FK-safe order so the cascade can't fight
// the explicit deletes.
func DeleteAllPhysicalCatalogForDatabase(t *testing.T, databaseID uuid.UUID) {
	t.Helper()

	db := storage.GetDb()

	if err := db.
		Exec(`DELETE FROM physical_wal_segments WHERE database_id = ?`, databaseID).Error; err != nil {
		t.Fatalf("cleanup wal segments: %v", err)
	}

	if err := db.
		Exec(`DELETE FROM physical_wal_history_files WHERE database_id = ?`, databaseID).Error; err != nil {
		t.Fatalf("cleanup wal history: %v", err)
	}

	if err := db.
		Exec(`DELETE FROM physical_in_flight_backups WHERE database_id = ?`, databaseID).Error; err != nil {
		t.Fatalf("cleanup in-flight: %v", err)
	}

	if err := db.
		Exec(`DELETE FROM physical_wal_streamers WHERE database_id = ?`, databaseID).Error; err != nil {
		t.Fatalf("cleanup wal streamers: %v", err)
	}

	if err := db.
		Exec(`DELETE FROM physical_incremental_backups WHERE database_id = ?`, databaseID).Error; err != nil {
		t.Fatalf("cleanup incremental backups: %v", err)
	}

	if err := db.
		Exec(`DELETE FROM physical_full_backups WHERE database_id = ?`, databaseID).Error; err != nil {
		t.Fatalf("cleanup full backups: %v", err)
	}
}

// NewTestWalStreamSample builds a healthy reachable sample of a streaming
// database; tests change the fields their scenario is about.
func NewTestWalStreamSample(databaseID uuid.UUID, sampledAt time.Time) *physical_models.PhysicalWalStreamSample {
	const segmentSizeBytes = 16 * 1024 * 1024

	return &physical_models.PhysicalWalStreamSample{
		DatabaseID:               databaseID,
		SampledAt:                sampledAt,
		SessionStartedAt:         sampledAt.Add(-time.Hour),
		SourceState:              physical_enums.PhysicalWalStreamSourceStateReachable,
		SourceCurrentLSN:         new(walmath.LSN(0x50000000)),
		ArchiveProgressLSN:       new(walmath.LSN(0x4F000000)),
		SourceWrittenBytes:       new(int64(segmentSizeBytes)),
		ArchivedSegments:         1,
		ArchivedWalBytes:         segmentSizeBytes,
		ArchivedStoredBytes:      4 * 1024 * 1024,
		WalSegmentSizeBytes:      segmentSizeBytes,
		HighestConcurrentUploads: 1,
		SegmentWriteAvgMillis:    new(int64(100)),
		SegmentWriteMaxMillis:    new(int64(150)),
		SlotWalStatus:            new("reserved"),
		SlotLagBytes:             new(int64(1024 * 1024)),
		SlotSafeWalSizeBytes:     new(int64(10 * 1024 * 1024 * 1024)),
		UnarchivedWalBytes:       new(int64(segmentSizeBytes)),
		LocalQueueBytes:          segmentSizeBytes,
		LocalQueuePauseBytes:     512 * 1024 * 1024,
		IsReceiverRunning:        true,
	}
}

func CreateTestWalStreamSample(
	t *testing.T,
	sample *physical_models.PhysicalWalStreamSample,
) *physical_models.PhysicalWalStreamSample {
	t.Helper()

	if err := physical_repositories.GetWalStreamSampleRepository().Insert(sample); err != nil {
		t.Fatalf("save test wal stream sample: %v", err)
	}

	return sample
}

// CreateTestWalStreamer writes a streamer row with the exact times a scenario
// needs, which Claim cannot do because it stamps every time with NOW().
func CreateTestWalStreamer(
	t *testing.T,
	walStreamer *physical_models.PhysicalWalStreamer,
) *physical_models.PhysicalWalStreamer {
	t.Helper()

	if err := storage.GetDb().Create(walStreamer).Error; err != nil {
		t.Fatalf("save test wal streamer: %v", err)
	}

	t.Cleanup(func() {
		_ = physical_repositories.GetWalStreamerRepository().DeleteByDatabaseID(walStreamer.DatabaseID)
	})

	return walStreamer
}
