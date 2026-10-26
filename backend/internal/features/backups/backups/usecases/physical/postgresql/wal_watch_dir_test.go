package usecases_physical_postgresql

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	databases_bridge "databasus-backend/internal/features/databases"
	"databasus-backend/internal/util/logger"
	"databasus-backend/internal/util/walmath"
)

const dispatcherConditionWaitLimit = 10 * time.Second

type capturedLog struct {
	mutex  sync.Mutex
	buffer bytes.Buffer
}

func (c *capturedLog) Write(p []byte) (int, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.buffer.Write(p)
}

func (c *capturedLog) String() string {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.buffer.String()
}

func createTestSupervisor(
	t *testing.T,
	fixture *PhysicalDBFixture,
	store *mockWalStorage,
	supervisorLogger *slog.Logger,
) *WalStreamSupervisor {
	t.Helper()

	supervisor := NewWalStreamSupervisor(WalStreamSpec{
		DatabaseID:     fixture.DB.ID,
		BackupStatus:   databases_bridge.GetModule(),
		SourceDB:       fixture.SourceDB,
		StorageID:      fixture.Storage.ID,
		FileStore:      newMockWalStoreFor(store),
		Encryption:     backups_core_enums.BackupEncryptionNone,
		WalSegmentRepo: physical_repositories.GetWalSegmentRepository(),
		HistoryRepo:    physical_repositories.GetWalHistoryRepository(),
		WatchDirRoot:   t.TempDir(),
		Logger:         supervisorLogger,
	})

	require.NoError(t, os.MkdirAll(filepath.Join(supervisor.watchDir, pendingUploadDirName), 0o700))

	return supervisor
}

func getPendingUploadDir(supervisor *WalStreamSupervisor) string {
	return filepath.Join(supervisor.watchDir, pendingUploadDirName)
}

// The gate is opened before the loop is stopped: the mock's writes ignore the context, and the loop
// waits for its uploads on the way out.
func runUploaderLoopInBackground(t *testing.T, supervisor *WalStreamSupervisor, store *mockWalStorage) {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	loopStopped := make(chan struct{})

	go func() {
		supervisor.runUploaderLoop(ctx, supervisor.logger)
		close(loopStopped)
	}()

	t.Cleanup(func() {
		if store.artifactGate != nil {
			store.artifactGate.releaseAll()
		}

		cancel()
		<-loopStopped
	})
}

func isSegmentCommitted(t *testing.T, fixture *PhysicalDBFixture, segmentNo uint64) bool {
	t.Helper()

	row := findWalSegment(t, fixture.DB.ID, 1, walmath.LSN(segmentNo*uint64(testWalSegmentSize)))

	return row != nil && row.FileName != nil
}

func waitForSegmentCommitted(t *testing.T, fixture *PhysicalDBFixture, segmentNo uint64) {
	t.Helper()

	require.Eventually(t, func() bool {
		return isSegmentCommitted(t, fixture, segmentNo)
	}, dispatcherConditionWaitLimit, 20*time.Millisecond, "segment %d was not committed", segmentNo)
}

func Test_UploaderLoop_WhenEightSegmentsAreQueued_UploadsFourAtOnceAndNeverFive(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	store := newMockWalStorage()
	store.artifactGate = newArtifactWriteGate()
	supervisor := createTestSupervisor(t, fixture, store, logger.GetLogger())

	for segmentNo := uint64(100); segmentNo < 108; segmentNo++ {
		writeWalFile(t, supervisor.watchDir, walName(1, segmentNo))
	}

	runUploaderLoopInBackground(t, supervisor, store)

	require.Eventually(t, func() bool {
		return store.artifactGate.getActiveWriteCount() == maxConcurrentSegmentUploads
	}, dispatcherConditionWaitLimit, 10*time.Millisecond, "four segments must upload at once")

	store.artifactGate.releaseOne()

	require.Eventually(t, func() bool {
		return len(store.artifactGate.getStartedObjectNames()) == maxConcurrentSegmentUploads+1
	}, dispatcherConditionWaitLimit, 10*time.Millisecond, "a freed slot must start the next segment")

	store.artifactGate.releaseAll()

	for segmentNo := uint64(100); segmentNo < 108; segmentNo++ {
		waitForSegmentCommitted(t, fixture, segmentNo)
	}

	assert.Equal(t, maxConcurrentSegmentUploads, store.artifactGate.getHighestActiveWriteCount(),
		"no more than four segments of one database may upload at once")
}

func Test_UploaderLoop_WhenSlotFreesUp_StartsPendingUploadSegmentInWalOrder(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	store := newMockWalStorage()
	store.artifactGate = newArtifactWriteGate()
	supervisor := createTestSupervisor(t, fixture, store, logger.GetLogger())

	for _, segmentNo := range []uint64{110, 111, 112, 113, 115} {
		writeWalFile(t, supervisor.watchDir, walName(1, segmentNo))
	}

	writeWalFile(t, getPendingUploadDir(supervisor), walName(1, 114))

	runUploaderLoopInBackground(t, supervisor, store)

	require.Eventually(t, func() bool {
		return store.artifactGate.getActiveWriteCount() == maxConcurrentSegmentUploads
	}, dispatcherConditionWaitLimit, 10*time.Millisecond)

	store.artifactGate.releaseOne()

	require.Eventually(t, func() bool {
		return len(store.artifactGate.getStartedObjectNames()) == maxConcurrentSegmentUploads+1
	}, dispatcherConditionWaitLimit, 10*time.Millisecond)

	fifthStarted := store.artifactGate.getStartedObjectNames()[maxConcurrentSegmentUploads]
	assert.Contains(t, fifthStarted, walName(1, 114), "segment 14 must start before segment 15")
}

func Test_UploaderLoop_WhenSegmentWriteStaysBlocked_DoesNotStartItAgainOrWarn(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	log := &capturedLog{}
	blockedSegmentName := walName(1, 120)

	store := newMockWalStorage()
	store.blockOn = blockedSegmentName
	store.started = make(chan struct{})
	store.release = make(chan struct{})
	supervisor := createTestSupervisor(t, fixture, store, slog.New(slog.NewTextHandler(log, nil)))

	writeWalFile(t, supervisor.watchDir, blockedSegmentName)
	runUploaderLoopInBackground(t, supervisor, store)
	t.Cleanup(func() { close(store.release) })

	<-store.started

	// A later segment committing proves that passes kept running while the first write was blocked.
	writeWalFile(t, supervisor.watchDir, walName(1, 121))
	waitForSegmentCommitted(t, fixture, 121)

	writeWalFile(t, supervisor.watchDir, walName(1, 122))
	waitForSegmentCommitted(t, fixture, 122)

	assert.Equal(t, int64(5), store.saveCount.Load(),
		"the blocked artifact plus two artifacts and two sidecars; a second start would add more")
	assert.NotContains(t, log.String(), "held by another in-flight claim")
}

func Test_RecoverLocalSegmentsOnStartup_WhenSegmentsHaveOwnNullClaims_CommitsAllConcurrently(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	store := newMockWalStorage()
	store.artifactGate = newArtifactWriteGateOpeningAt(2)
	supervisor := createTestSupervisor(t, fixture, store, logger.GetLogger())

	recoveredSegmentNos := []uint64{130, 131, 132, 133, 134, 135}

	for _, segmentNo := range recoveredSegmentNos {
		seedNullClaim(t, fixture, segmentNo)
		writeWalFile(t, supervisor.watchDir, walName(1, segmentNo))
	}

	supervisor.recoverLocalSegmentsOnStartup(t.Context(), supervisor.logger)

	for _, segmentNo := range recoveredSegmentNos {
		assert.True(t, isSegmentCommitted(t, fixture, segmentNo), "segment %d must be committed", segmentNo)
		assert.NoFileExists(t, filepath.Join(supervisor.watchDir, walName(1, segmentNo)))
	}

	assert.Greater(t, store.artifactGate.getHighestActiveWriteCount(), 1,
		"recovery must use the same concurrent slots as the live loop")
}

func Test_RecoverLocalSegmentsOnStartup_WhenStorageFails_ReturnsAfterOneAttemptPerSegment(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	store := newMockWalStorage()
	store.startFailingSaves()
	supervisor := createTestSupervisor(t, fixture, store, logger.GetLogger())

	recoveredSegmentNos := []uint64{140, 141, 142, 143, 144}

	for _, segmentNo := range recoveredSegmentNos {
		writeWalFile(t, supervisor.watchDir, walName(1, segmentNo))
	}

	supervisor.recoverLocalSegmentsOnStartup(t.Context(), supervisor.logger)

	assert.Equal(t, int64(len(recoveredSegmentNos)), store.saveCount.Load(),
		"recovery is one pass, so a storage outage at startup cannot keep the receiver from spawning")

	for _, segmentNo := range recoveredSegmentNos {
		assert.FileExists(t, filepath.Join(supervisor.watchDir, walName(1, segmentNo)))
	}
}

func Test_UploaderLoop_WhenPendingUploadSucceeds_RemovesSegmentFromStagingDir(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	store := newMockWalStorage()
	supervisor := createTestSupervisor(t, fixture, store, logger.GetLogger())
	stagedSegmentPath := writeWalFile(t, getPendingUploadDir(supervisor), walName(1, 70))

	runUploaderLoopInBackground(t, supervisor, store)
	waitForSegmentCommitted(t, fixture, 70)

	require.True(t, store.hasObjectFor(fixture.DB.ID, 1, walName(1, 70)),
		"a segment staged out of the resume path is valid WAL of the older chain and must still reach storage")
	require.Eventually(t, func() bool {
		_, err := os.Stat(stagedSegmentPath)
		return os.IsNotExist(err)
	}, dispatcherConditionWaitLimit, 20*time.Millisecond, "an uploaded segment must not keep occupying the local queue")
}

func Test_UploaderLoop_WhenPendingUploadFails_RetainsSegmentForNextPass(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	store := newMockWalStorage()
	store.startFailingSaves()
	supervisor := createTestSupervisor(t, fixture, store, logger.GetLogger())
	stagedSegmentPath := writeWalFile(t, getPendingUploadDir(supervisor), walName(1, 71))

	runUploaderLoopInBackground(t, supervisor, store)

	require.Eventually(t, func() bool {
		return store.saveCount.Load() >= 1
	}, dispatcherConditionWaitLimit, 20*time.Millisecond)
	require.FileExists(t, stagedSegmentPath, "staged WAL must survive a storage outage")

	store.stopFailingSaves()
	waitForSegmentCommitted(t, fixture, 71)

	require.True(t, store.hasObjectFor(fixture.DB.ID, 1, walName(1, 71)))
}

func Test_UploaderLoop_WhenSegmentMovesToPendingUploadMidUpload_WritesItOnce(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	movedSegmentName := walName(1, 150)

	store := newMockWalStorage()
	store.blockOn = movedSegmentName
	store.started = make(chan struct{})
	store.release = make(chan struct{})
	supervisor := createTestSupervisor(t, fixture, store, logger.GetLogger())

	watchDirPath := writeWalFile(t, supervisor.watchDir, movedSegmentName)
	runUploaderLoopInBackground(t, supervisor, store)

	releaseBlockedWrite := sync.OnceFunc(func() { close(store.release) })
	t.Cleanup(releaseBlockedWrite)

	<-store.started

	movedPath := filepath.Join(getPendingUploadDir(supervisor), movedSegmentName)
	require.NoError(t, os.Rename(watchDirPath, movedPath))

	// A later segment committing proves that passes saw the moved copy while its upload was running.
	writeWalFile(t, supervisor.watchDir, walName(1, 151))
	waitForSegmentCommitted(t, fixture, 151)

	releaseBlockedWrite()

	waitForSegmentCommitted(t, fixture, 150)
	require.Eventually(t, func() bool {
		_, err := os.Stat(movedPath)
		return os.IsNotExist(err)
	}, dispatcherConditionWaitLimit, 20*time.Millisecond, "the moved copy of a committed segment must be dropped")

	assert.Len(t, store.objectNamesFor(fixture.DB.ID, 1, movedSegmentName), 2,
		"one artifact and one sidecar: the moved copy must not be uploaded again")
}

func Test_UploaderLoop_WhenOlderSegmentWriteIsBlocked_RecordsItAsLowestQueuedSegment(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	blockedSegmentName := walName(1, 50)

	store := newMockWalStorage()
	store.blockOn = blockedSegmentName
	store.started = make(chan struct{})
	store.release = make(chan struct{})
	supervisor := createTestSupervisor(t, fixture, store, logger.GetLogger())

	for segmentNo := uint64(50); segmentNo <= 53; segmentNo++ {
		writeWalFile(t, supervisor.watchDir, walName(1, segmentNo))
	}

	runUploaderLoopInBackground(t, supervisor, store)
	t.Cleanup(func() { close(store.release) })

	for segmentNo := uint64(51); segmentNo <= 53; segmentNo++ {
		waitForSegmentCommitted(t, fixture, segmentNo)
	}

	// The listing after the last commit no longer holds 53, so the value read below was stored after
	// the newer segments finished.
	require.Eventually(t, func() bool {
		return !slices.Contains(listQueuedFilenames(supervisor), walName(1, 53))
	}, dispatcherConditionWaitLimit, 20*time.Millisecond)

	assert.Equal(t, segmentStart(1, 50), supervisor.getLowestQueuedSegment(),
		"newer segments finishing must not move the lowest queued segment past one still uploading")
}

func Test_UploaderLoop_WhenShortSegmentFileStaysQueued_RecordsItAsLowestQueuedSegment(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	store := newMockWalStorage()
	supervisor := createTestSupervisor(t, fixture, store, logger.GetLogger())

	writeSegmentOfSize(t, supervisor.watchDir, walName(1, 60), testWalSegmentSize/2)
	writeWalFile(t, supervisor.watchDir, walName(1, 61))

	runUploaderLoopInBackground(t, supervisor, store)
	waitForSegmentCommitted(t, fixture, 61)

	require.Eventually(t, func() bool {
		lowestQueuedSegment := supervisor.getLowestQueuedSegment()
		return lowestQueuedSegment != nil && *lowestQueuedSegment == *segmentStart(1, 60) &&
			len(listQueuedFilenames(supervisor)) == 1
	}, dispatcherConditionWaitLimit, 20*time.Millisecond,
		"a short file holds the archive progress point, because restore cannot pass it either")
}

func listQueuedFilenames(supervisor *WalStreamSupervisor) []string {
	queuedSegments := listQueuedWalSegments(supervisor.watchDir, testWalSegmentSize)
	filenames := make([]string, 0, len(queuedSegments))

	for _, queued := range queuedSegments {
		filenames = append(filenames, queued.walFilename)
	}

	return filenames
}

func seedNullClaim(t *testing.T, fixture *PhysicalDBFixture, segmentNo uint64) {
	t.Helper()

	startLSN := walmath.LSN(segmentNo * uint64(testWalSegmentSize))

	inserted, err := physical_repositories.GetWalSegmentRepository().ClaimInsert(&physical_models.PhysicalWalSegment{
		DatabaseID:  fixture.DB.ID,
		StorageID:   fixture.Storage.ID,
		TimelineID:  1,
		WalFilename: walName(1, segmentNo),
		StartLSN:    startLSN,
		EndLSN:      startLSN + walmath.LSN(testWalSegmentSize),
		Encryption:  backups_core_enums.BackupEncryptionNone,
	})
	require.NoError(t, err)
	require.True(t, inserted)
}
