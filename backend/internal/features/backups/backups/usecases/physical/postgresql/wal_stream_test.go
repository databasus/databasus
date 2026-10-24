package usecases_physical_postgresql

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	"databasus-backend/internal/features/backups/backups/core/physical/chain_view"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
	"databasus-backend/internal/util/walmath"
)

func Test_WalStream_FullIncrementalAndWalStream_StreamerArchivesSegments(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)
	t.Cleanup(func() {
		_ = physical_repositories.GetWalStreamerRepository().DeleteByDatabaseID(fixture.DB.ID)
	})

	store := newMockWalStorage()

	stop := StartWalStreamerForTest(
		t,
		WalStreamerTestSpec{Fixture: fixture, Storage: store, WatchDirRoot: t.TempDir()},
	).Stop
	t.Cleanup(stop)

	adminConn := OpenAdminConn(t, fixture)

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	// Force three segment rotations so pg_receivewal finalizes segments the
	// uploader can archive.
	for range 3 {
		_, err := ForceWalRotation(ctx, adminConn)
		require.NoError(t, err)
	}

	WaitForCommittedWalSegmentCount(t, fixture.DB.ID, 1, 90*time.Second)

	segments, err := physical_repositories.GetWalSegmentRepository().FindByChainSpan(
		fixture.DB.ID, 1, walmath.LSN(0), lsnSpanUpperBoundForTests,
	)
	require.NoError(t, err)

	var committed int
	for _, seg := range segments {
		if seg.FileName == nil {
			continue
		}

		committed++

		require.True(t, store.hasObject(*seg.FileName), "archived segment must exist in storage: %s", *seg.FileName)
		require.True(t, store.hasObject(*seg.FileName+metadataSuffix), "segment sidecar must exist in storage")
	}

	require.GreaterOrEqual(t, committed, 1, "at least one rotated segment must be archived")
}

func assertDbSegmentsArchivedOnlyIn(
	t *testing.T,
	databaseID uuid.UUID,
	ownStore, otherStore *mockWalStorage,
) {
	t.Helper()

	segments, err := physical_repositories.GetWalSegmentRepository().FindByChainSpan(
		databaseID, 1, walmath.LSN(0), lsnSpanUpperBoundForTests,
	)
	require.NoError(t, err)

	committed := 0
	for _, seg := range segments {
		if seg.FileName == nil {
			continue
		}

		committed++

		require.True(t, ownStore.hasObject(*seg.FileName), "own store must hold %s", *seg.FileName)
		require.False(t, otherStore.hasObject(*seg.FileName), "other DB's store must not hold %s", *seg.FileName)
	}

	require.GreaterOrEqual(t, committed, 1, "database %s must archive at least one segment", databaseID)
}

func committedSegmentsInOrder(t *testing.T, databaseID uuid.UUID) []*physical_models.PhysicalWalSegment {
	t.Helper()

	all, err := physical_repositories.GetWalSegmentRepository().FindByChainSpan(
		databaseID, 1, walmath.LSN(0), lsnSpanUpperBoundForTests,
	)
	require.NoError(t, err)

	committed := make([]*physical_models.PhysicalWalSegment, 0, len(all))
	for _, seg := range all {
		if seg.FileName != nil {
			committed = append(committed, seg)
		}
	}

	return committed
}

func Test_WalStream_MultipleDbs_EachArchivesSegmentsIndependently(t *testing.T) {
	fixtureA := SetupPhysicalDBForBackup(t)
	fixtureB := SetupPhysicalDBForBackup(t)
	t.Cleanup(func() {
		_ = physical_repositories.GetWalStreamerRepository().DeleteByDatabaseID(fixtureA.DB.ID)
		_ = physical_repositories.GetWalStreamerRepository().DeleteByDatabaseID(fixtureB.DB.ID)
	})

	storeA := newMockWalStorage()
	storeB := newMockWalStorage()

	t.Cleanup(
		StartWalStreamerForTest(
			t,
			WalStreamerTestSpec{Fixture: fixtureA, Storage: storeA, WatchDirRoot: t.TempDir()},
		).Stop,
	)
	t.Cleanup(
		StartWalStreamerForTest(
			t,
			WalStreamerTestSpec{Fixture: fixtureB, Storage: storeB, WatchDirRoot: t.TempDir()},
		).Stop,
	)

	connA := OpenAdminConn(t, fixtureA)

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	// One shared physical cluster: rotating WAL feeds both DBs' independent slots.
	for range 4 {
		_, err := ForceWalRotation(ctx, connA)
		require.NoError(t, err)
	}

	WaitForCommittedWalSegmentCount(t, fixtureA.DB.ID, 1, 90*time.Second)
	WaitForCommittedWalSegmentCount(t, fixtureB.DB.ID, 1, 90*time.Second)

	assertDbSegmentsArchivedOnlyIn(t, fixtureA.DB.ID, storeA, storeB)
	assertDbSegmentsArchivedOnlyIn(t, fixtureB.DB.ID, storeB, storeA)
}

func Test_WalStream_MissingSegmentInStreamedChain_SurfacesAsGapChainStaysExtendable(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)
	t.Cleanup(func() {
		_ = physical_repositories.GetWalStreamerRepository().DeleteByDatabaseID(fixture.DB.ID)
	})

	// Anchor a COMPLETED FULL at LSN 0 so every streamed segment falls in its span.
	MarkFullCompleted(t, fixture.BackupID, 1, walmath.LSN(0), walmath.LSN(0))

	store := newMockWalStorage()
	adminConn := OpenAdminConn(t, fixture)

	t.Cleanup(
		StartWalStreamerForTest(
			t,
			WalStreamerTestSpec{Fixture: fixture, Storage: store, WatchDirRoot: t.TempDir()},
		).Stop,
	)

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	for range 5 {
		_, err := ForceWalRotation(ctx, adminConn)
		require.NoError(t, err)
	}

	WaitForCommittedWalSegmentCount(t, fixture.DB.ID, 3, 90*time.Second)

	// A real streamed chain is contiguous, so no gap yet.
	gapsBefore, err := chain_view.GetChainViewService().FindWalGapsInChain(fixture.BackupID)
	require.NoError(t, err)
	require.Empty(t, gapsBefore, "a contiguous streamed chain has no gaps")

	// Drop a middle committed segment to model a lost / retention-trimmed segment.
	// The gap is derived from the surviving rows' LSN math — no marker row exists.
	committed := committedSegmentsInOrder(t, fixture.DB.ID)
	require.GreaterOrEqual(t, len(committed), 3)
	removed := committed[1]
	require.NoError(t, physical_repositories.GetWalSegmentRepository().DeleteByID(removed.ID))

	gaps := WaitForWalGap(t, fixture.BackupID, 30*time.Second)
	require.Len(t, gaps, 1, "exactly the removed segment's range must surface as a gap")
	require.Equal(t, removed.StartLSN, gaps[0].Start)
	require.Equal(t, removed.EndLSN, gaps[0].End)

	// The chain remains extendable despite the internal gap (lossy chain).
	chain := WaitForExtendableChain(t, fixture.DB.ID, 10*time.Second)
	require.Equal(t, fixture.BackupID, chain.RootFull.ID)
}

func Test_WalStream_SlotLagGrowsWithoutConsumer_DrainsOnceStreaming(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)
	t.Cleanup(func() {
		_ = physical_repositories.GetWalStreamerRepository().DeleteByDatabaseID(fixture.DB.ID)
	})

	adminConn := OpenAdminConn(t, fixture)
	slotName := fixture.DB.PostgresqlPhysical.ReplicationSlotName

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	// Create the persistent slot with no consumer attached, then burn WAL so the
	// slot's restart_lsn falls behind — the signal the lag monitor reads.
	require.NoError(
		t,
		fixture.DB.PostgresqlPhysical.VerifyWalSlot(ctx, logger.GetLogger(), encryption.GetFieldEncryptor()),
	)

	const lagTarget = 8 * 1024 * 1024
	ForceReplicationLag(t, adminConn, lagTarget)
	WaitUntilSlotLag(t, adminConn, slotName, lagTarget, 30*time.Second)

	// Once our streamer attaches, it consumes the backlog and the lag drains.
	t.Cleanup(StartWalStreamerForTest(t, WalStreamerTestSpec{
		Fixture:      fixture,
		Storage:      newMockWalStorage(),
		WatchDirRoot: t.TempDir(),
	}).Stop)

	deadline := time.Now().UTC().Add(60 * time.Second)
	for time.Now().UTC().Before(deadline) {
		if SlotLagBytes(t, adminConn, slotName) < lagTarget {
			return
		}

		time.Sleep(250 * time.Millisecond)
	}

	t.Fatalf("slot lag did not drain below %d within 60s after streaming started", lagTarget)
}

func Test_WalStream_CustomWalSegmentSize_LsnMathCorrect(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	const customSegSize = int64(64 * 1024 * 1024) // 64 MB segments

	store := newMockWalStorage()
	uploader := NewWalUploader(WalUploadDeps{
		DatabaseID:          fixture.DB.ID,
		StorageID:           fixture.Storage.ID,
		FileStore:           newMockWalStoreFor(store),
		Encryption:          backups_core_enums.BackupEncryptionNone,
		FieldEncryptor:      encryption.GetFieldEncryptor(),
		WalSegmentRepo:      physical_repositories.GetWalSegmentRepository(),
		WalSegmentSizeBytes: customSegSize,
		Logger:              logger.GetLogger(),
	})

	// At 64 MB segments there are 64 segments per 4 GiB logid. Segment with
	// logid=2, segLow=3 starts at (2<<32) + 3*64MB.
	dir := t.TempDir()
	name := "000000010000000200000003"
	segmentPath := writeSegmentOfSize(t, dir, name, customSegSize)
	require.NoError(t, uploader.ProcessSegment(context.Background(), segmentPath, name))

	expectedStartLSN := walmath.LSN((uint64(2) << 32) + 3*uint64(customSegSize))

	row := findWalSegment(t, fixture.DB.ID, 1, expectedStartLSN)
	require.NotNil(t, row, "segment LSN must be derived from the DB's segsize, not the walmath global")
	require.Equal(t, expectedStartLSN, row.StartLSN)
	require.Equal(t, expectedStartLSN+walmath.LSN(customSegSize), row.EndLSN)
}

func Test_Cleaner_AbandonedNullClaim_OlderThanGrace_DeletedYoungerSurvives(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	repo := physical_repositories.GetWalSegmentRepository()

	oldClaim := &physical_models.PhysicalWalSegment{
		DatabaseID:  fixture.DB.ID,
		StorageID:   fixture.Storage.ID,
		TimelineID:  1,
		WalFilename: walName(1, 50),
		StartLSN:    walmath.LSN(50 * uint64(testWalSegmentSize)),
		EndLSN:      walmath.LSN(51 * uint64(testWalSegmentSize)),
		Encryption:  backups_core_enums.BackupEncryptionNone,
		ClaimedAt:   time.Now().UTC().Add(-2 * time.Hour),
	}
	inserted, err := repo.ClaimInsert(oldClaim)
	require.NoError(t, err)
	require.True(t, inserted)

	youngClaim := &physical_models.PhysicalWalSegment{
		DatabaseID:  fixture.DB.ID,
		StorageID:   fixture.Storage.ID,
		TimelineID:  1,
		WalFilename: walName(1, 51),
		StartLSN:    walmath.LSN(51 * uint64(testWalSegmentSize)),
		EndLSN:      walmath.LSN(52 * uint64(testWalSegmentSize)),
		Encryption:  backups_core_enums.BackupEncryptionNone,
		ClaimedAt:   time.Now().UTC().Add(-30 * time.Minute),
	}
	inserted, err = repo.ClaimInsert(youngClaim)
	require.NoError(t, err)
	require.True(t, inserted)

	deleted, err := repo.DeleteAbandonedClaims(fixture.DB.ID, time.Now().UTC().Add(-1*time.Hour))
	require.NoError(t, err)
	require.Equal(t, int64(1), deleted, "only the over-grace NULL claim must be reaped")

	require.Nil(t, findWalSegment(t, fixture.DB.ID, 1, oldClaim.StartLSN), "aged claim must be gone")
	require.NotNil(t, findWalSegment(t, fixture.DB.ID, 1, youngClaim.StartLSN), "within-grace claim must survive")
}

func Test_WalStream_ResumePointBelowSlotRestartLsn_RealignsAndKeepsStreaming(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)
	t.Cleanup(func() {
		_ = physical_repositories.GetWalStreamerRepository().DeleteByDatabaseID(fixture.DB.ID)
	})

	adminConn := OpenAdminConn(t, fixture)
	slotName := fixture.DB.PostgresqlPhysical.ReplicationSlotName
	watchDirRoot := t.TempDir()

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()

	// A storage that refuses every save leaves finalized segments sitting in the
	// watch dir — the local queue this issue is about. It has to keep failing
	// across the restart, or startup recovery would drain the queue and there
	// would be nothing left to drag the resume point down.
	store := newMockWalStorage()
	store.startFailingSaves()

	firstRun := StartWalStreamerForTest(t, WalStreamerTestSpec{
		Fixture:      fixture,
		FileStore:    newMockWalStoreFor(store),
		WatchDirRoot: watchDirRoot,
	})

	for range 3 {
		_, err := ForceWalRotation(ctx, adminConn)
		require.NoError(t, err)
	}

	waitForQueuedSegments(t, firstRun.WatchDir, 1, 60*time.Second)
	firstRun.Stop()

	queuedBeforeRebuild := queuedSegmentNames(t, firstRun.WatchDir)
	require.NotEmpty(t, queuedBeforeRebuild)

	// Model the incident: the slot is rebuilt while the queue still holds
	// pre-rebuild segments, so the recreated slot reserves from a position far
	// above them and pg_receivewal would otherwise resume below it.
	DropReplicationSlotExternally(t, adminConn, slotName)

	segmentSizeBytes := int64(walmath.WalSegmentSize)

	resumeSegmentNo, isResumeSegmentFound := GetResumeSegmentNo(firstRun.WatchDir, segmentSizeBytes)
	require.True(t, isResumeSegmentFound, "the local queue must hold at least one complete segment")

	burnWalPastSegment(t, ctx, adminConn, resumeSegmentNo+1)

	// pg_create_physical_replication_slot(..., true) reserves from the last
	// checkpoint's redo point, not from the current insert position, so without
	// this the recreated slot can still land below the queue.
	_, err := adminConn.Exec(ctx, "CHECKPOINT")
	require.NoError(t, err)

	require.NoError(t, fixture.DB.PostgresqlPhysical.VerifyWalSlot(
		ctx, logger.GetLogger(), encryption.GetFieldEncryptor(),
	))

	requireQueueBelowSlot(t, ctx, adminConn, slotName, resumeSegmentNo)

	secondRun := StartWalStreamerForTest(t, WalStreamerTestSpec{
		Fixture:      fixture,
		FileStore:    newMockWalStoreFor(store),
		WatchDirRoot: watchDirRoot,
	})
	t.Cleanup(secondRun.Stop)

	pendingUploadDir := filepath.Join(secondRun.WatchDir, pendingUploadDirName)

	for _, staleSegment := range queuedBeforeRebuild {
		require.Eventually(t, func() bool {
			_, err := os.Stat(filepath.Join(pendingUploadDir, staleSegment))

			return err == nil
		}, 60*time.Second, 250*time.Millisecond,
			"segment below the new restart_lsn must leave pg_receivewal's resume path: %s", staleSegment)

		require.NoFileExists(t, filepath.Join(secondRun.WatchDir, staleSegment))
	}

	// Storage recovers: the staged segments are valid WAL of the older chain, so
	// they must still reach storage, and the receiver must keep streaming the new
	// chain rather than crash-looping on the recycled WAL it used to ask for.
	store.stopFailingSaves()

	for range 3 {
		_, err := ForceWalRotation(ctx, adminConn)
		require.NoError(t, err)
	}

	for _, staleSegment := range queuedBeforeRebuild {
		require.Eventually(t, func() bool {
			return store.hasObjectFor(fixture.DB.ID, 1, staleSegment)
		}, 60*time.Second, 250*time.Millisecond,
			"a segment moved out of the resume path must still reach storage: %s", staleSegment)
	}

	WaitForCommittedWalSegmentCount(t, fixture.DB.ID, 1, 90*time.Second)
}

func queuedSegmentNames(t *testing.T, watchDir string) []string {
	t.Helper()

	entries, err := os.ReadDir(watchDir)
	require.NoError(t, err)

	var queuedSegments []string

	for _, entry := range entries {
		if !entry.IsDir() && walmath.IsWalFilename(entry.Name()) {
			queuedSegments = append(queuedSegments, entry.Name())
		}
	}

	return queuedSegments
}

func waitForQueuedSegments(t *testing.T, watchDir string, minCount int, timeout time.Duration) {
	t.Helper()

	require.Eventually(t, func() bool {
		return len(queuedSegmentNames(t, watchDir)) >= minCount
	}, timeout, 250*time.Millisecond, "watch dir never accumulated %d finalized segments", minCount)
}

// pg_switch_wal is a no-op on an already-empty segment, so the cluster only
// moves past the queue if real WAL is written between switches.
func burnWalPastSegment(
	t *testing.T,
	ctx context.Context,
	adminConn *pgx.Conn,
	targetSegmentNo walmath.WalSegmentNo,
) {
	t.Helper()

	segmentSizeBytes := int64(walmath.WalSegmentSize)

	require.Eventually(t, func() bool {
		if _, err := GenerateWalActivity(ctx, adminConn, segmentSizeBytes); err != nil {
			return false
		}

		if _, err := ForceWalRotation(ctx, adminConn); err != nil {
			return false
		}

		var currentLSN walmath.LSN
		if err := adminConn.QueryRow(ctx, "SELECT pg_current_wal_lsn()::text").Scan(&currentLSN); err != nil {
			return false
		}

		return segmentNoAtLSN(currentLSN, segmentSizeBytes) > targetSegmentNo
	}, 2*time.Minute, 100*time.Millisecond, "cluster never wrote past segment %d", uint64(targetSegmentNo))
}

// The realign only has work to do when the recreated slot reserves above the
// queue pg_receivewal would otherwise resume from, so assert that rather than
// let the test pass without exercising anything.
func requireQueueBelowSlot(
	t *testing.T,
	ctx context.Context,
	adminConn *pgx.Conn,
	slotName string,
	resumeSegmentNo walmath.WalSegmentNo,
) {
	t.Helper()

	segmentSizeBytes := int64(walmath.WalSegmentSize)

	slotState, err := InspectSlot(ctx, adminConn, slotName)
	require.NoError(t, err)
	require.NotNil(t, slotState)

	require.Less(t, uint64(resumeSegmentNo), uint64(segmentNoAtLSN(slotState.RestartLSN, segmentSizeBytes)),
		"the queue must sit below the recreated slot for this test to mean anything")
}

// The supervisor invokes these callbacks from its own loops, so a plain slice
// would race with the assertions (and with require.Eventually's polling
// goroutine).
type chainEventRecorder struct {
	mu      sync.Mutex
	reasons []string
}

func (r *chainEventRecorder) recordReason(reason string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.reasons = append(r.reasons, reason)
}

func (r *chainEventRecorder) getReasons() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return slices.Clone(r.reasons)
}

func (r *chainEventRecorder) getReasonCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return len(r.reasons)
}

func Test_RecordMismatchAndDecideEscalation_WhenFirstMismatch_RetriesAfterRealign(t *testing.T) {
	var mismatchEscalator resumeMismatchEscalator

	require.Equal(t, resumeMismatchActionRetry, mismatchEscalator.recordMismatchAndDecideEscalation(),
		"a realign clears the resume path, so the next spawn deserves a chance before the chain is broken")
}

func Test_RecordMismatchAndDecideEscalation_WhenSecondMismatch_RequestsSlotRebuild(t *testing.T) {
	var mismatchEscalator resumeMismatchEscalator

	require.Equal(t, resumeMismatchActionRetry, mismatchEscalator.recordMismatchAndDecideEscalation())
	require.Equal(t, resumeMismatchActionRebuildSlot, mismatchEscalator.recordMismatchAndDecideEscalation(),
		"a realigned queue that still asks for recycled WAL means the slot itself no longer covers the source")
}

func Test_RecordMismatchAndDecideEscalation_WhenHealthyRunBetweenMismatches_ResetsCounter(t *testing.T) {
	var mismatchEscalator resumeMismatchEscalator

	require.Equal(t, resumeMismatchActionRetry, mismatchEscalator.recordMismatchAndDecideEscalation())

	mismatchEscalator.reset()

	require.Equal(t, resumeMismatchActionRetry, mismatchEscalator.recordMismatchAndDecideEscalation(),
		"mismatches must be back-to-back to escalate; a healthy run in between clears the incident")
}

func Test_RecordMismatchAndDecideEscalation_AfterRebuild_StartsCountingFromScratch(t *testing.T) {
	var mismatchEscalator resumeMismatchEscalator

	mismatchEscalator.recordMismatchAndDecideEscalation()
	require.Equal(t, resumeMismatchActionRebuildSlot, mismatchEscalator.recordMismatchAndDecideEscalation())

	require.Equal(t, resumeMismatchActionRetry, mismatchEscalator.recordMismatchAndDecideEscalation(),
		"a rebuild is expensive, so the next one needs its own pair of mismatches")
}

const instantReceiverExit = 100 * time.Millisecond

func fatalReceiverRun() receiverRunResult {
	return receiverRunResult{
		Exit:     receiverFatal,
		RanFor:   instantReceiverExit,
		FatalErr: errors.New("pg_receivewal fatal error: could not write 8192 bytes to WAL file"),
	}
}

// Escalating on a verdict read off a receiver whose transport was already gone hands the streamer
// back as FAILED and fires a chain-broken alert for a blip the forwarder heals by itself.
func Test_DemoteFatalExitWhenBastionUnreachable_WhenTheBastionIsUnreachable_TurnsTheExitRetryable(
	t *testing.T,
) {
	demotedRun := demoteFatalExitWhenBastionUnreachable(logger.GetLogger(), fatalReceiverRun(), false)

	assert.Equal(t, receiverRetryable, demotedRun.Exit)
}

// The mirror: a fatal cause a respawn can never fix must still hand the streamer back for reclaim.
func Test_DemoteFatalExitWhenBastionUnreachable_WhenTheBastionIsReachable_LeavesTheExitFatal(
	t *testing.T,
) {
	fatalRun := fatalReceiverRun()
	judgedRun := demoteFatalExitWhenBastionUnreachable(logger.GetLogger(), fatalRun, true)

	assert.Equal(t, receiverFatal, judgedRun.Exit)
	assert.Equal(t, fatalRun.FatalErr, judgedRun.FatalErr)
}

func Test_DemoteFatalExitWhenBastionUnreachable_WhenTheExitIsNotFatal_LeavesTheExitUnchanged(
	t *testing.T,
) {
	for _, exit := range []receiverExit{receiverCtxCancelled, receiverRetryable, receiverInternalRestart} {
		judgedRun := demoteFatalExitWhenBastionUnreachable(
			logger.GetLogger(), receiverRunResult{Exit: exit}, false)

		assert.Equal(t, exit, judgedRun.Exit)
	}
}

// A restarted sshd refuses connections instantly, so every spawn dies well under the healthy-uptime
// bar. Counting those would escalate to streamer-FAILED in about half a minute and fire a
// chain-broken alert for a transport blip the forwarder heals on its own.
func Test_RecordExitAndDecideRetry_WhenTheBastionIsUnreachable_DoesNotEscalate(t *testing.T) {
	var failureEscalator rapidFailureEscalator

	for range receivewalMaxRapidFailures * 2 {
		require.False(t,
			failureEscalator.recordExitAndDecideRetry(instantReceiverExit, false).isEscalationRequired)
	}
}

// The mirror of the case above: without it the reachability check would silently swallow a real
// crash loop and the streamer would never be handed back for reclaim.
func Test_RecordExitAndDecideRetry_WhenTheBastionIsReachable_EscalatesAfterTheRapidFailureLimit(
	t *testing.T,
) {
	var failureEscalator rapidFailureEscalator

	for range receivewalMaxRapidFailures - 1 {
		require.False(t,
			failureEscalator.recordExitAndDecideRetry(instantReceiverExit, true).isEscalationRequired)
	}

	require.True(t,
		failureEscalator.recordExitAndDecideRetry(instantReceiverExit, true).isEscalationRequired)
}

func Test_RecordExitAndDecideRetry_WhenAHealthyRunFollowsRapidFailures_ClearsTheCount(t *testing.T) {
	var failureEscalator rapidFailureEscalator

	for range receivewalMaxRapidFailures - 1 {
		failureEscalator.recordExitAndDecideRetry(instantReceiverExit, true)
	}

	require.False(t,
		failureEscalator.recordExitAndDecideRetry(receivewalMinHealthyUptime, true).isEscalationRequired)
	require.False(t,
		failureEscalator.recordExitAndDecideRetry(instantReceiverExit, true).isEscalationRequired)
}

// A run that only looked healthy because it spent its whole life waiting on a dead bastion must not
// reset the backoff, or the loop hammers the bastion while it is down.
func Test_RecordExitAndDecideRetry_WhenTheBastionIsUnreachable_DoesNotResetTheBackoff(t *testing.T) {
	var failureEscalator rapidFailureEscalator

	require.False(t,
		failureEscalator.recordExitAndDecideRetry(receivewalMinHealthyUptime, false).isBackoffResettable)
	require.True(t,
		failureEscalator.recordExitAndDecideRetry(receivewalMinHealthyUptime, true).isBackoffResettable)
	require.False(t,
		failureEscalator.recordExitAndDecideRetry(instantReceiverExit, true).isBackoffResettable)
}

// Growing without a low ceiling would leave half an hour of silence on a transport that healed
// minutes ago: WAL piling up on the source and a staleness alert fired after the fact.
func Test_RecordExitAndDecideRetry_WhenTheBastionIsUnreachable_CapsTheRespawnBackoff(t *testing.T) {
	var failureEscalator rapidFailureEscalator

	require.Equal(t, receivewalBastionDownMaxBackoff,
		failureEscalator.recordExitAndDecideRetry(instantReceiverExit, false).maxRespawnBackoff)
	require.Less(t, receivewalBastionDownMaxBackoff, receivewalRespawnMaxBackoff,
		"a cap that is not lower than the ordinary ceiling caps nothing")
}

func Test_RecordExitAndDecideRetry_WhenTheBastionIsReachable_KeepsTheOrdinaryBackoffCeiling(t *testing.T) {
	var failureEscalator rapidFailureEscalator

	require.Equal(t, receivewalRespawnMaxBackoff,
		failureEscalator.recordExitAndDecideRetry(instantReceiverExit, true).maxRespawnBackoff)
	require.Equal(t, receivewalRespawnMaxBackoff,
		failureEscalator.recordExitAndDecideRetry(receivewalMinHealthyUptime, true).maxRespawnBackoff)
}

func Test_WalStream_WhenSourceIsIdle_KeepsReceiverAliveAndDoesNotRebuildSlot(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)
	t.Cleanup(func() {
		_ = physical_repositories.GetWalStreamerRepository().DeleteByDatabaseID(fixture.DB.ID)
	})

	adminConn := OpenAdminConn(t, fixture)
	slotName := fixture.DB.PostgresqlPhysical.ReplicationSlotName

	var slotRebuilds, chainRisks chainEventRecorder

	streamer := StartWalStreamerForTest(t, WalStreamerTestSpec{
		Fixture:      fixture,
		Storage:      newMockWalStorage(),
		WatchDirRoot: t.TempDir(),
		OnSlotRebuilt: func(_ context.Context, reason string) error {
			slotRebuilds.recordReason(reason)

			return nil
		},
		OnChainAtRisk: func(report ChainRiskReport) {
			chainRisks.recordReason(report.Reason)
		},
	})
	t.Cleanup(streamer.Stop)

	attachedReceiverPID := waitForAttachedReceiverPID(t, adminConn, slotName)

	// The stall window is 60 s and the watcher polls every 10 s, so this outlasts
	// several full detection cycles: restart_lsn parking on an idle source is not a
	// stall, and must not cost the receiver its connection.
	idleDeadline := time.Now().UTC().Add(150 * time.Second)
	for time.Now().UTC().Before(idleDeadline) {
		slotState, err := InspectSlot(t.Context(), adminConn, slotName)
		require.NoError(t, err)
		require.NotNil(t, slotState)
		require.NotNil(t, slotState.ActivePID, "an idle database is not a reason to drop the replication connection")
		require.Equal(t, attachedReceiverPID, *slotState.ActivePID,
			"restart_lsn parks legitimately while nothing is written; restarting pg_receivewal over it is the bug")

		time.Sleep(2 * time.Second)
	}

	streamer.Stop()

	require.Empty(t, slotRebuilds.getReasons(),
		"an idle database must never cost a WAL gap and an out-of-cadence full backup")
	require.Empty(t, chainRisks.getReasons())
	require.Empty(t, streamer.Supervisor.GetSlotRebuildTimestamps())
}

func waitForAttachedReceiverPID(t *testing.T, adminConn *pgx.Conn, slotName string) int {
	t.Helper()

	var attachedReceiverPID int

	require.Eventually(t, func() bool {
		slotState, err := InspectSlot(t.Context(), adminConn, slotName)
		// While the slot is being created, the creating backend holds it, so an active PID alone
		// can belong to that backend rather than to pg_receivewal.
		if err != nil || slotState == nil || !isOwnedReceiverBackend(slotState) {
			return false
		}

		attachedReceiverPID = *slotState.ActivePID

		return true
	}, 60*time.Second, 250*time.Millisecond, "pg_receivewal never attached to the slot")

	return attachedReceiverPID
}

func Test_WalStream_WhenSourceWritesRarely_UploadsSegmentWithinRotationInterval(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)
	t.Cleanup(func() {
		_ = physical_repositories.GetWalStreamerRepository().DeleteByDatabaseID(fixture.DB.ID)
	})

	adminConn := OpenAdminConn(t, fixture)

	t.Cleanup(StartWalStreamerForTest(t, WalStreamerTestSpec{
		Fixture:                fixture,
		Storage:                newMockWalStorage(),
		WatchDirRoot:           t.TempDir(),
		ForcedRotationInterval: time.Second,
	}).Stop)

	// A handful of rows is nowhere near a full segment, so without a forced
	// rotation this WAL would sit in the local queue for as long as the source
	// stays quiet.
	_, err := GenerateWalActivity(t.Context(), adminConn, 4096)
	require.NoError(t, err)

	WaitForCommittedWalSegmentCount(t, fixture.DB.ID, 1, 90*time.Second)
}

func Test_WalStream_WhenSwitchWalIsRefused_AlertsOnceAndStopsRotating(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)
	t.Cleanup(func() {
		_ = physical_repositories.GetWalStreamerRepository().DeleteByDatabaseID(fixture.DB.ID)
	})

	adminConn := OpenAdminConn(t, fixture)

	fixture.DB.PostgresqlPhysical = createReplicationOnlyRole(t, adminConn, fixture)

	var chainRisks chainEventRecorder

	t.Cleanup(StartWalStreamerForTest(t, WalStreamerTestSpec{
		Fixture:                fixture,
		Storage:                newMockWalStorage(),
		WatchDirRoot:           t.TempDir(),
		ForcedRotationInterval: time.Second,
		OnChainAtRisk: func(report ChainRiskReport) {
			chainRisks.recordReason(report.Reason)
		},
	}).Stop)

	_, err := GenerateWalActivity(t.Context(), adminConn, 4096)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return chainRisks.getReasonCount() > 0
	}, 90*time.Second, 500*time.Millisecond, "a source that refuses pg_switch_wal must be reported, not retried silently")

	require.Equal(t, []string{ChainRiskReasonRotationDenied}, chainRisks.getReasons())

	// The refusal is permanent, so the loop must be gone rather than re-alerting
	// every interval; streaming itself is unaffected.
	require.Never(t, func() bool {
		return chainRisks.getReasonCount() > 1
	}, 5*time.Second, time.Second, "a permanent refusal must be reported once, not once per interval")

	_, err = ForceWalRotation(t.Context(), adminConn)
	require.NoError(t, err)

	WaitForCommittedWalSegmentCount(t, fixture.DB.ID, 1, 90*time.Second)
}

// The physical backup role only needs REPLICATION, and PostgreSQL restricts
// pg_switch_wal to superusers unless EXECUTE is granted — so this is what a real
// deployment looks like, not a contrived setup.
func createReplicationOnlyRole(
	t *testing.T,
	adminConn *pgx.Conn,
	fixture *PhysicalDBFixture,
) *postgresql_physical.PostgresqlPhysicalDatabase {
	t.Helper()

	roleName := "databasus_no_switch_" + strings.ReplaceAll(uuid.New().String(), "-", "")[:12]

	_, err := adminConn.Exec(t.Context(),
		fmt.Sprintf("CREATE ROLE %s LOGIN REPLICATION PASSWORD 'switchless'", pgx.Identifier{roleName}.Sanitize()),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = adminConn.Exec(context.Background(),
			fmt.Sprintf(
				"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE usename = %s",
				pgx.Identifier{roleName}.Sanitize(),
			),
		)
		_, _ = adminConn.Exec(context.Background(),
			fmt.Sprintf("DROP ROLE IF EXISTS %s", pgx.Identifier{roleName}.Sanitize()),
		)
	})

	replicationOnlySourceDB := *fixture.DB.PostgresqlPhysical
	replicationOnlySourceDB.Username = roleName
	replicationOnlySourceDB.Password = "switchless"

	return &replicationOnlySourceDB
}

func Test_WalStream_WhenUploadsKeepFailing_AlertsArchiveStaleOnce(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)
	t.Cleanup(func() {
		_ = physical_repositories.GetWalStreamerRepository().DeleteByDatabaseID(fixture.DB.ID)
	})

	adminConn := OpenAdminConn(t, fixture)

	store := newMockWalStorage()
	store.startFailingSaves()

	var chainRisks chainEventRecorder

	t.Cleanup(StartWalStreamerForTest(t, WalStreamerTestSpec{
		Fixture:                   fixture,
		FileStore:                 newMockWalStoreFor(store),
		WatchDirRoot:              t.TempDir(),
		ArchiveStalenessThreshold: time.Second,
		OnChainAtRisk: func(report ChainRiskReport) {
			chainRisks.recordReason(report.Reason)
		},
	}).Stop)

	// The source has to keep writing: an archive that stands still behind an idle
	// source is not falling behind, it is just waiting for the next segment.
	require.Eventually(t, func() bool {
		_, err := GenerateWalActivity(t.Context(), adminConn, 4096)

		return err == nil && chainRisks.getReasonCount() > 0
	}, 3*time.Minute, 2*time.Second,
		"WAL the source already wrote is not reaching storage — the recovery point has stopped advancing")

	require.Equal(t, []string{ChainRiskReasonArchiveStale}, chainRisks.getReasons(),
		"one alert per incident, not one per lag-monitor tick")

	store.stopFailingSaves()

	WaitForCommittedWalSegmentCount(t, fixture.DB.ID, 1, 90*time.Second)

	require.Len(t, chainRisks.getReasons(), 1)
}

func Test_WalStream_WhenQueueFillsAndReceiverExitsOnItsOwn_CountsOnePauseAndOneRestart(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)
	t.Cleanup(func() {
		_ = physical_repositories.GetWalStreamerRepository().DeleteByDatabaseID(fixture.DB.ID)
	})

	adminConn := OpenAdminConn(t, fixture)
	segmentSizeBytes := walSegmentSizeBytes(fixture.DB.PostgresqlPhysical)

	store := newMockWalStorage()
	store.startFailingSaves()

	// Three segments keep the shared source's background WAL from pausing the
	// receiver a second time once uploads succeed again.
	streamer := StartWalStreamerForTest(t, WalStreamerTestSpec{
		Fixture:              fixture,
		FileStore:            newMockWalStoreFor(store),
		WatchDirRoot:         t.TempDir(),
		LocalQueuePauseBytes: 3 * segmentSizeBytes,
	})
	t.Cleanup(streamer.Stop)

	supervisor := streamer.Supervisor

	require.Eventually(t, supervisor.isReceiverRunning, 60*time.Second, 200*time.Millisecond)

	// Written once: WAL written during the pause would refill the queue as soon
	// as the receiver resumes and pause it a second time.
	ForceReplicationLag(t, adminConn, 4*segmentSizeBytes)

	require.Eventually(t, func() bool {
		return !supervisor.isReceiverRunning() && supervisor.backlogBytes() >= supervisor.highWatermarkBytes.Load()
	}, 90*time.Second, 200*time.Millisecond, "a full local queue must pause the receiver")

	store.stopFailingSaves()

	require.Eventually(t, supervisor.isReceiverRunning, 90*time.Second, 200*time.Millisecond,
		"the receiver resumes once the queue drains")

	var restarts, pauses int

	takeCounts := func() {
		takenRestarts, takenPauses := supervisor.receiverRespawns.takeCounts()
		restarts += takenRestarts
		pauses += takenPauses
	}

	require.Eventually(t, func() bool {
		takeCounts()

		return pauses == 1
	}, 10*time.Second, 200*time.Millisecond)

	require.Zero(t, restarts, "the respawn after back pressure is a pause, not a restart")

	// Other tests write WAL to the same source, and a burst of it would fill a
	// three-segment queue again; from here on the production pause size applies,
	// so the only respawn left to count is the one this test causes.
	supervisor.setLocalQueuePauseBytes(getDefaultLocalQueuePauseBytes(fixture.DB.PostgresqlPhysical))

	terminateWalReceiverBackend(t, adminConn, fixture.DB.PostgresqlPhysical.ReplicationSlotName)

	require.Eventually(t, func() bool {
		takeCounts()

		return restarts == 1
	}, 60*time.Second, 200*time.Millisecond, "a receiver that exits on its own is respawned as a restart")

	require.Equal(t, 1, pauses, "back pressure is a pause, not a restart")

	nextRestarts, nextPauses := supervisor.receiverRespawns.takeCounts()
	require.Zero(t, nextRestarts)
	require.Zero(t, nextPauses)
}

func terminateWalReceiverBackend(t *testing.T, adminConn *pgx.Conn, slotName string) {
	t.Helper()

	require.Eventually(t, func() bool {
		var terminatedCount int

		err := adminConn.QueryRow(t.Context(), `
			SELECT COUNT(pg_terminate_backend(active_pid))
			FROM pg_replication_slots
			WHERE slot_name = $1 AND active_pid IS NOT NULL
		`, slotName).Scan(&terminatedCount)

		return err == nil && terminatedCount == 1
	}, 30*time.Second, 200*time.Millisecond, "the receiver must be attached to the slot before it can be cut off")
}

func Test_WalStream_WhileReceiverRunsAndAfterItExits_SlotBreakSampleCarriesItsRunStartTime(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)
	t.Cleanup(func() {
		_ = physical_repositories.GetWalStreamerRepository().DeleteByDatabaseID(fixture.DB.ID)
	})

	streamerStartedAt := time.Now().UTC()

	streamer := StartWalStreamerForTest(t, WalStreamerTestSpec{
		Fixture:      fixture,
		Storage:      newMockWalStorage(),
		WatchDirRoot: t.TempDir(),
	})
	t.Cleanup(streamer.Stop)

	supervisor := streamer.Supervisor

	var runningSample slotBreakSample

	require.Eventually(t, func() bool {
		runningSample = supervisor.buildSlotBreakSample(&SlotState{}, time.Now().UTC())

		return !runningSample.ReceiverRunStartedAt.IsZero()
	}, 60*time.Second, 200*time.Millisecond, "a running receiver gives the sample its start time")

	assert.False(t, runningSample.ReceiverRunStartedAt.Before(streamerStartedAt))
	assert.False(t, runningSample.ReceiverRunStartedAt.After(runningSample.ObservedAt))

	streamer.Stop()

	require.Eventually(t, func() bool {
		return supervisor.buildSlotBreakSample(&SlotState{}, time.Now().UTC()).ReceiverRunStartedAt.IsZero()
	}, 30*time.Second, 200*time.Millisecond, "an exited receiver leaves the sample without a start time")
}
