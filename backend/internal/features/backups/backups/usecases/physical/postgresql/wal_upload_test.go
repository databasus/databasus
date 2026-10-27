package usecases_physical_postgresql

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	databases_bridge "databasus-backend/internal/features/databases"
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/util/logger"
)

// newTestUploader wires a WalUploader against the real catalog repo and a mock
// storage, using the fixture's database/storage IDs for FK integrity. Tests write
// the segments they upload into its watch directory, which the gap probe reads.
func newTestUploader(
	t *testing.T,
	fixture *PhysicalDBFixture,
	store *mockWalStorage,
	onGap func(walmath.LSN, walmath.LSN),
) *WalUploader {
	t.Helper()

	return newTestUploaderWithStore(t, fixture, newMockWalStoreFor(store), onGap)
}

func newTestUploaderWithStore(
	t *testing.T,
	fixture *PhysicalDBFixture,
	fileStore FileStore,
	onGap func(walmath.LSN, walmath.LSN),
) *WalUploader {
	t.Helper()

	return NewWalUploader(WalUploadDeps{
		DatabaseID:          fixture.DB.ID,
		BackupStatus:        databases_bridge.GetModule(),
		StorageID:           fixture.Storage.ID,
		FileStore:           fileStore,
		Encryption:          backups_core_enums.BackupEncryptionNone,
		WalSegmentRepo:      physical_repositories.GetWalSegmentRepository(),
		WalSegmentSizeBytes: testWalSegmentSize,
		Logger:              logger.GetLogger(),
		WatchDir:            t.TempDir(),
		OnGapDetected:       onGap,
	})
}

func findWalSegment(
	t *testing.T,
	databaseID uuid.UUID,
	timelineID int,
	startLSN walmath.LSN,
) *physical_models.PhysicalWalSegment {
	t.Helper()

	row, err := physical_repositories.GetWalSegmentRepository().FindByChainKey(databaseID, timelineID, startLSN)
	require.NoError(t, err)

	return row
}

func Test_WalUpload_ClaimSucceeds_SaveFileLands_FileNameUpdatedFromNullToNonNull(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	store := newMockWalStorage()
	uploader := newTestUploader(t, fixture, store, nil)

	dir := uploader.deps.WatchDir
	name := walName(1, 5)
	localPath := writeWalFile(t, dir, name)

	require.NoError(t, uploader.ProcessSegment(context.Background(), localPath, name))

	startLSN := walmath.LSN(5 * uint64(testWalSegmentSize))
	row := findWalSegment(t, fixture.DB.ID, 1, startLSN)
	require.NotNil(t, row)
	require.NotNil(t, row.FileName, "file_name must be flipped from NULL to the object key")

	// The name carries a per-attempt UUID, so the row is the only place that knows
	// it, which is where restore reads it from too.
	objectName := *row.FileName
	require.True(t, store.hasObject(objectName), "artifact must be in storage under the name the row carries")
	require.True(t, store.hasObject(objectName+metadataSuffix), "sidecar must be uploaded")

	require.NoFileExists(t, localPath, "local segment must be removed after a committed upload")
}

func Test_WalUpload_SaveFileFails_ClaimRowDeletedLocalRetained_NextTickSucceeds(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	store := newMockWalStorage()
	store.failSaveTimes = 1
	uploader := newTestUploader(t, fixture, store, nil)

	dir := uploader.deps.WatchDir
	name := walName(1, 9)
	localPath := writeWalFile(t, dir, name)

	startLSN := walmath.LSN(9 * uint64(testWalSegmentSize))

	// First attempt: SaveFile fails -> claim row deleted, local retained.
	require.Error(t, uploader.ProcessSegment(context.Background(), localPath, name))
	require.Nil(t, findWalSegment(t, fixture.DB.ID, 1, startLSN), "failed claim row must be deleted")
	require.FileExists(t, localPath, "local segment must be retained for retry")

	// Second attempt: succeeds end-to-end.
	require.NoError(t, uploader.ProcessSegment(context.Background(), localPath, name))

	row := findWalSegment(t, fixture.DB.ID, 1, startLSN)
	require.NotNil(t, row)
	require.NotNil(t, row.FileName)
	require.NoFileExists(t, localPath)
}

func Test_WalUpload_WhenSegmentFileIsShort_SkipsClaimAndRetainsFile(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	store := newMockWalStorage()
	uploader := newTestUploader(t, fixture, store, nil)

	dir := uploader.deps.WatchDir
	name := walName(1, 21)
	tornSegmentPath := writeSegmentOfSize(t, dir, name, testWalSegmentSize/2)

	require.NoError(t, uploader.ProcessSegment(context.Background(), tornSegmentPath, name))

	require.Nil(t, findWalSegment(t, fixture.DB.ID, 1, walmath.LSN(21*uint64(testWalSegmentSize))),
		"LSN bounds come from the filename, so claiming a short file would promise WAL the object does not hold")
	require.FileExists(t, tornSegmentPath, "the receiver owns the file and will re-stream it")
}

func Test_WalUpload_WhenSegmentFileIsShort_WritesNothingToStorage(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	store := newMockWalStorage()
	uploader := newTestUploader(t, fixture, store, nil)

	dir := uploader.deps.WatchDir
	name := walName(1, 22)

	require.NoError(t, uploader.ProcessSegment(
		context.Background(), writeSegmentOfSize(t, dir, name, testWalSegmentSize-1), name,
	))
	require.NoError(t, uploader.RecoverSegment(
		context.Background(), writeSegmentOfSize(t, t.TempDir(), name, 0), name,
	))

	require.Zero(t, store.saveCount.Load())
}

func Test_WalUpload_DuplicateClaim_CompletedRow_DropsLocalNoStorageWrite(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	store := newMockWalStorage()
	uploader := newTestUploader(t, fixture, store, nil)

	dir := uploader.deps.WatchDir
	name := walName(1, 11)

	// First uploader commits the segment.
	first := writeWalFile(t, dir, name)
	require.NoError(t, uploader.ProcessSegment(context.Background(), first, name))

	savesAfterFirst := store.saveCount.Load()

	// A second local copy of the same segment (another process) must not re-upload.
	secondDir := t.TempDir()
	second := writeWalFile(t, secondDir, name)

	require.NoError(t, uploader.ProcessSegment(context.Background(), second, name))

	require.Equal(t, savesAfterFirst, store.saveCount.Load(), "completed segment must not be re-uploaded")
	require.NoFileExists(t, second, "redundant local copy must be removed")
}

func Test_WalUpload_DuplicateClaim_InFlightClaim_RetainsLocalNoStorageWrite(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	store := newMockWalStorage()
	uploader := newTestUploader(t, fixture, store, nil)

	name := walName(1, 13)
	startLSN := walmath.LSN(13 * uint64(testWalSegmentSize))
	endLSN := startLSN + walmath.LSN(testWalSegmentSize)

	// Pre-seed an in-flight claim (file_name NULL) as if a live owner holds it.
	repo := physical_repositories.GetWalSegmentRepository()
	inserted, err := repo.ClaimInsert(&physical_models.PhysicalWalSegment{
		DatabaseID:  fixture.DB.ID,
		StorageID:   fixture.Storage.ID,
		TimelineID:  1,
		WalFilename: name,
		StartLSN:    startLSN,
		EndLSN:      endLSN,
		Encryption:  backups_core_enums.BackupEncryptionNone,
	})
	require.NoError(t, err)
	require.True(t, inserted)

	dir := uploader.deps.WatchDir
	local := writeWalFile(t, dir, name)

	require.NoError(t, uploader.ProcessSegment(context.Background(), local, name))

	require.Zero(t, store.saveCount.Load(), "must not upload over another owner's in-flight claim")
	require.FileExists(t, local, "local file must be retained while another claim is in flight")

	row := findWalSegment(t, fixture.DB.ID, 1, startLSN)
	require.NotNil(t, row)
	require.Nil(t, row.FileName, "the in-flight claim must be left untouched")
}

func Test_WalUpload_DeleteCascadesIntoNullClaim_UploadCompletes_DeleteFileCleansOrphan(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	name := walName(1, 17)
	startLSN := walmath.LSN(17 * uint64(testWalSegmentSize))

	store := newBlockingMockWalStorage(name)
	walStore := newMockWalStoreFor(store)
	uploader := newTestUploaderWithStore(t, fixture, walStore, nil)

	dir := uploader.deps.WatchDir
	local := writeWalFile(t, dir, name)

	uploadErr := make(chan error, 1)
	go func() { uploadErr <- uploader.ProcessSegment(context.Background(), local, name) }()

	// Wait until SaveFile is mid-flight, then simulate a DeleteFull cascade
	// removing the NULL claim row out from under the uploader.
	<-store.started

	require.NoError(t, physical_repositories.GetWalSegmentRepository().DeleteClaim(
		findWalSegment(t, fixture.DB.ID, 1, startLSN).ID,
	))

	close(store.release)

	require.NoError(t, <-uploadErr)

	require.Nil(t, findWalSegment(t, fixture.DB.ID, 1, startLSN), "cascade-deleted claim must stay gone")

	references := make([]storages.StoredFileReference, 0, 2)
	for _, objectName := range store.objectNamesFor(fixture.DB.ID, 1, name) {
		references = append(references,
			storages.StoredFileReference{StorageID: fixture.Storage.ID, FileName: objectName})
	}

	require.NoError(t, walStore.DrainFileDeletions(t.Context(), references...))

	require.False(t, store.hasObjectFor(fixture.DB.ID, 1, name),
		"orphaned ciphertext must not survive a lost commit")
	require.NoFileExists(t, local)
}

func Test_WalUpload_PostUpdateGapProbe_EmitsSingleNotificationPerOutage(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	var gapCount atomic.Int64

	var gapStart, gapEnd walmath.LSN

	store := newMockWalStorage()
	uploader := newTestUploader(t, fixture, store, func(start, end walmath.LSN) {
		gapCount.Add(1)
		gapStart, gapEnd = start, end
	})

	dir := uploader.deps.WatchDir

	// Commit segment 20, then a discontiguous backlog 22, 23, 24 (segment 21 was
	// lost to a storage outage). Only the first post-gap segment (22) is
	// non-contiguous, so exactly one notification must fire.
	for _, segNo := range []uint64{20, 22, 23, 24} {
		name := walName(1, segNo)
		require.NoError(t, uploader.ProcessSegment(context.Background(), writeWalFile(t, dir, name), name))
	}

	require.Equal(t, int64(1), gapCount.Load(), "a single multi-segment outage must notify exactly once")
	require.Equal(t, walmath.LSN(21*uint64(testWalSegmentSize)), gapStart)
	require.Equal(t, walmath.LSN(22*uint64(testWalSegmentSize)), gapEnd)
}

func Test_WalUpload_ContiguousSegments_NoGapNotification(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	var gapCount atomic.Int64

	store := newMockWalStorage()
	uploader := newTestUploader(t, fixture, store, func(_, _ walmath.LSN) { gapCount.Add(1) })

	dir := uploader.deps.WatchDir

	for _, segNo := range []uint64{30, 31, 32} {
		name := walName(1, segNo)
		require.NoError(t, uploader.ProcessSegment(context.Background(), writeWalFile(t, dir, name), name))
	}

	require.Zero(t, gapCount.Load(), "contiguous segments must not report a gap")
}

func Test_SegmentBounds_AtLogIdBoundary_ComputesContiguousLSNs(t *testing.T) {
	// The last segment in logid 0 (FF) and the first in logid 1 must yield
	// contiguous LSNs across the boundary — the invariant the post-upload gap
	// probe (prev.end_lsn == this.start_lsn) relies on.
	timelineID, start, end, err := segmentBounds("0000000100000000000000FF", testWalSegmentSize)
	require.NoError(t, err)
	require.Equal(t, 1, timelineID)
	require.Equal(t, walmath.LSN(0xFF)*walmath.LSN(testWalSegmentSize), start)
	require.Equal(t, start+walmath.LSN(testWalSegmentSize), end)

	_, nextStart, _, err := segmentBounds("000000010000000100000000", testWalSegmentSize)
	require.NoError(t, err)
	require.Equal(t, end, nextStart, "segment LSNs must be contiguous across the logid boundary")
}

func Test_WalUpload_RecoverSegment_TakesOverOwnNullClaim_CommitsExistingRow(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	store := newMockWalStorage()
	uploader := newTestUploader(t, fixture, store, nil)

	name := walName(1, 23)
	startLSN := walmath.LSN(23 * uint64(testWalSegmentSize))
	endLSN := startLSN + walmath.LSN(testWalSegmentSize)

	// Seed a pre-crash NULL-file_name claim row, as the restart sweep would find.
	repo := physical_repositories.GetWalSegmentRepository()
	inserted, err := repo.ClaimInsert(&physical_models.PhysicalWalSegment{
		DatabaseID:  fixture.DB.ID,
		StorageID:   fixture.Storage.ID,
		TimelineID:  1,
		WalFilename: name,
		StartLSN:    startLSN,
		EndLSN:      endLSN,
		Encryption:  backups_core_enums.BackupEncryptionNone,
	})
	require.NoError(t, err)
	require.True(t, inserted)

	claim := findWalSegment(t, fixture.DB.ID, 1, startLSN)
	require.NotNil(t, claim)
	require.Nil(t, claim.FileName)

	dir := uploader.deps.WatchDir
	local := writeWalFile(t, dir, name)

	require.NoError(t, uploader.RecoverSegment(context.Background(), local, name))

	committed := findWalSegment(t, fixture.DB.ID, 1, startLSN)
	require.NotNil(t, committed)
	require.Equal(t, claim.ID, committed.ID, "takeover must reuse the pre-crash row, not insert a duplicate")
	require.NotNil(t, committed.FileName, "own pre-crash NULL claim must be taken over and committed")
	require.True(t, store.hasObject(*committed.FileName))
	require.True(t, store.hasObject(*committed.FileName+metadataSuffix))
	require.NoFileExists(t, local)
}

func Test_WalUpload_RecoverSegment_WhenAlreadyCommitted_DropsLocalNoReupload(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	store := newMockWalStorage()
	uploader := newTestUploader(t, fixture, store, nil)

	name := walName(1, 29)

	// A normal upload commits the segment (the crash landed after the commit).
	dir1 := uploader.deps.WatchDir
	require.NoError(t, uploader.ProcessSegment(context.Background(), writeWalFile(t, dir1, name), name))

	savesAfterCommit := store.saveCount.Load()

	// The restart sweep finds a leftover local copy whose row is already committed.
	dir2 := t.TempDir()
	local := writeWalFile(t, dir2, name)

	require.NoError(t, uploader.RecoverSegment(context.Background(), local, name))

	require.Equal(t, savesAfterCommit, store.saveCount.Load(), "already-committed segment must not be re-uploaded")
	require.NoFileExists(t, local, "redundant local copy must be removed on recovery")
}

func Test_BuildWalSegmentArtifactReader_WhenLargeSegment_StreamsAndCountsArtifact(t *testing.T) {
	dir := t.TempDir()
	localPath := filepath.Join(dir, "000000010000000000000001")
	require.NoError(t, os.WriteFile(localPath, bytes.Repeat([]byte("x"), 2*1024*1024), 0o600))

	artifact, salt, iv, err := buildWalSegmentArtifactReader(
		localPath,
		backups_core_enums.BackupEncryptionNone,
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

func Test_WalUpload_ConcurrentClaimSameSegment_OnlyWinnerInserts(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	repo := physical_repositories.GetWalSegmentRepository()
	startLSN := walmath.LSN(40 * uint64(testWalSegmentSize))
	endLSN := startLSN + walmath.LSN(testWalSegmentSize)

	const racers = 6

	type claimOutcome struct {
		inserted bool
		err      error
	}

	results := make(chan claimOutcome, racers)
	start := make(chan struct{})

	for range racers {
		go func() {
			<-start

			// Don't call require.* off the test goroutine — collect and assert below.
			inserted, err := repo.ClaimInsert(&physical_models.PhysicalWalSegment{
				DatabaseID:  fixture.DB.ID,
				StorageID:   fixture.Storage.ID,
				TimelineID:  1,
				WalFilename: walName(1, 40),
				StartLSN:    startLSN,
				EndLSN:      endLSN,
				Encryption:  backups_core_enums.BackupEncryptionNone,
			})
			results <- claimOutcome{inserted: inserted, err: err}
		}()
	}

	close(start)

	winners := 0
	for range racers {
		outcome := <-results
		require.NoError(t, outcome.err)

		if outcome.inserted {
			winners++
		}
	}

	require.Equal(t, 1, winners, "exactly one concurrent claim may win the (db, tl, start_lsn) slot")
}

func Test_TakeArchivingSnapshot_AfterTwoSegmentsArchived_ReportsCountsTimingsAndConcurrencyOnce(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	store := newMockWalStorage()
	uploader := newTestUploader(t, fixture, store, nil)

	for _, segmentNo := range []uint64{60, 61} {
		name := walName(1, segmentNo)
		require.NoError(t, uploader.ProcessSegment(t.Context(), writeWalFile(t, uploader.deps.WatchDir, name), name))
	}

	uploader.RecordConcurrentUploads(3)
	uploader.RecordConcurrentUploads(2)

	snapshot := uploader.TakeArchivingSnapshot()

	require.Equal(t, int64(2), snapshot.ArchivedSegments)
	require.Positive(t, snapshot.ArchivedBytes)
	require.Positive(t, snapshot.Claim.Longest)
	require.Positive(t, snapshot.SegmentWrite.Longest)
	require.Positive(t, snapshot.SidecarWrite.Longest)
	require.Positive(t, snapshot.Commit.Longest)
	require.Equal(t, 3, snapshot.HighestConcurrentUploads)

	require.Equal(t, ArchivingSnapshot{}, uploader.TakeArchivingSnapshot(),
		"each rollup interval reports only what happened since the previous one")
}

type recordedGap struct {
	start walmath.LSN
	end   walmath.LSN
}

type gapRecorder struct {
	mutex sync.Mutex
	gaps  []recordedGap
}

func (r *gapRecorder) record(start, end walmath.LSN) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.gaps = append(r.gaps, recordedGap{start: start, end: end})
}

func (r *gapRecorder) getGaps() []recordedGap {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return slices.Clone(r.gaps)
}

func newGapProbeUploader(t *testing.T) (*WalUploader, *mockWalStorage, *gapRecorder) {
	t.Helper()

	fixture := SetupPhysicalDBForBackup(t)
	store := newMockWalStorage()
	recorder := &gapRecorder{}

	return newTestUploader(t, fixture, store, recorder.record), store, recorder
}

func archiveSegment(t *testing.T, uploader *WalUploader, segmentNo uint64) {
	t.Helper()

	name := walName(1, segmentNo)
	require.NoError(t, uploader.ProcessSegment(t.Context(), writeWalFile(t, uploader.deps.WatchDir, name), name))
}

func segmentStartLSN(segmentNo uint64) walmath.LSN {
	return walmath.LSN(segmentNo * uint64(testWalSegmentSize))
}

func Test_WalUpload_WhenLaterSegmentFinishesFirst_ReportsNoGap(t *testing.T) {
	uploader, _, recorder := newGapProbeUploader(t)

	archiveSegment(t, uploader, 9)

	earlierPath := writeWalFile(t, uploader.deps.WatchDir, walName(1, 10))
	archiveSegment(t, uploader, 11)
	require.NoError(t, uploader.ProcessSegment(t.Context(), earlierPath, walName(1, 10)))

	require.Empty(t, recorder.getGaps(), "a segment still uploading is not a gap, before or after it finishes")
}

func Test_WalUpload_WhenEarlierSegmentFailedAndWaitsForRetry_ReportsNoGap(t *testing.T) {
	uploader, store, recorder := newGapProbeUploader(t)

	archiveSegment(t, uploader, 9)

	store.failSaveTimes = 1
	failedPath := writeWalFile(t, uploader.deps.WatchDir, walName(1, 10))
	require.Error(t, uploader.ProcessSegment(t.Context(), failedPath, walName(1, 10)))

	archiveSegment(t, uploader, 11)

	require.Empty(t, recorder.getGaps(), "a failed segment retried on a later pass is still coming")
}

func Test_WalUpload_WhenSegmentWasNeverReceived_ReportsGapFromNewestArchived(t *testing.T) {
	uploader, _, recorder := newGapProbeUploader(t)

	archiveSegment(t, uploader, 18)
	archiveSegment(t, uploader, 20)

	require.Equal(t, []recordedGap{{start: segmentStartLSN(19), end: segmentStartLSN(20)}}, recorder.getGaps())
}

func Test_WalUpload_WhenSegmentBeforeHoleIsStillUploading_StartsGapAfterIt(t *testing.T) {
	uploader, _, recorder := newGapProbeUploader(t)

	archiveSegment(t, uploader, 17)
	writeWalFile(t, uploader.deps.WatchDir, walName(1, 18))
	archiveSegment(t, uploader, 20)

	require.Equal(t, []recordedGap{{start: segmentStartLSN(19), end: segmentStartLSN(20)}}, recorder.getGaps(),
		"the hole starts after the segment that is still uploading, not after the newest archived one")
}

func Test_WalUpload_WhenHoleSpansSeveralSegments_ReportsOneGap(t *testing.T) {
	uploader, _, recorder := newGapProbeUploader(t)

	archiveSegment(t, uploader, 27)
	archiveSegment(t, uploader, 30)

	require.Equal(t, []recordedGap{{start: segmentStartLSN(28), end: segmentStartLSN(30)}}, recorder.getGaps())
}

func Test_WalUpload_WhenTwoGapsFinishOutOfOrder_ReportsBoth(t *testing.T) {
	uploader, _, recorder := newGapProbeUploader(t)

	archiveSegment(t, uploader, 28)

	lowerSegmentPath := writeWalFile(t, uploader.deps.WatchDir, walName(1, 30))
	archiveSegment(t, uploader, 40)
	require.NoError(t, uploader.ProcessSegment(t.Context(), lowerSegmentPath, walName(1, 30)))

	require.Equal(t, []recordedGap{
		{start: segmentStartLSN(31), end: segmentStartLSN(40)},
		{start: segmentStartLSN(29), end: segmentStartLSN(30)},
	}, recorder.getGaps(), "a lower gap found after a higher one is a separate gap")
}

func Test_WalUpload_WhenPredecessorWaitsInPendingUpload_ReportsNoGap(t *testing.T) {
	uploader, _, recorder := newGapProbeUploader(t)

	archiveSegment(t, uploader, 9)

	pendingUploadDir := filepath.Join(uploader.deps.WatchDir, pendingUploadDirName)
	require.NoError(t, os.MkdirAll(pendingUploadDir, 0o700))
	writeWalFile(t, pendingUploadDir, walName(1, 10))

	archiveSegment(t, uploader, 11)

	require.Empty(t, recorder.getGaps())
}

func Test_WalUpload_WhenSegmentBeforeGapLeftOnlyShortFile_ReportsGap(t *testing.T) {
	uploader, _, recorder := newGapProbeUploader(t)

	archiveSegment(t, uploader, 19)
	writeSegmentOfSize(t, uploader.deps.WatchDir, walName(1, 20), testWalSegmentSize/2)
	archiveSegment(t, uploader, 21)

	require.Equal(t, []recordedGap{{start: segmentStartLSN(20), end: segmentStartLSN(21)}}, recorder.getGaps(),
		"the receiver never streams a segment below its resume point again, so a short file is a hole")
}

func TestWalUploadRacingDatabaseDeletion(t *testing.T) {
	t.Run("deletes the database and drops the segment when the deletion locks the database before the commit",
		func(t *testing.T) {
			fixture := SetupPhysicalDBForBackup(t)

			name := walName(1, 18)
			startLSN := walmath.LSN(18 * uint64(testWalSegmentSize))

			store := newBlockingMockWalStorage(name)
			uploader := newTestUploader(t, fixture, store, nil)

			local := writeWalFile(t, uploader.deps.WatchDir, name)

			uploadErr := make(chan error, 1)
			go func() { uploadErr <- uploader.ProcessSegment(context.Background(), local, name) }()

			<-store.started

			deletion := databases_bridge.BeginTestDatabaseDeletion(t, fixture.DB.ID)

			close(store.release)
			deletion.WaitForBlockedDatabaseWrite(t)

			require.NoError(t, deletion.Finish())

			require.NoError(t, <-uploadErr)
			require.Nil(t, findWalSegment(t, fixture.DB.ID, 1, startLSN))
			require.NoFileExists(t, local)
		})
}
