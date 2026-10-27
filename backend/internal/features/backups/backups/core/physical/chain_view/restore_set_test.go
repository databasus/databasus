package chain_view_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	"databasus-backend/internal/features/backups/backups/core/physical/chain_view"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	physical_testing "databasus-backend/internal/features/backups/backups/core/physical/testing"
)

// restoreClock hands out increasing timestamps so PITR-by-time selection has
// distinct, ordered points to choose between (the New* helpers otherwise stamp
// everything at time.Now()).
type restoreClock struct {
	base time.Time
}

func newRestoreClock() *restoreClock {
	return &restoreClock{base: time.Now().UTC().Add(-2 * time.Hour)}
}

func (c *restoreClock) at(minutes int) time.Time {
	return c.base.Add(time.Duration(minutes) * time.Minute)
}

func lsnAt(segments int) walmath.LSN {
	return walmath.LSN(segments * segmentBytes)
}

func Test_ResolveRestoreSetForBackup_WhenFull_ReturnsFullOnlyNoWal(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)

	full := physical_testing.CreateTestFullBackup(t, physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1, lsnAt(0), lsnAt(1)))

	// A WAL segment exists but a per-backup restore must NOT ship it.
	physical_testing.CreateTestWalSegment(t, physical_testing.NewTestWalSegment(
		prereqs.database.ID, prereqs.storage.ID, 1, "000000010000000000000001", lsnAt(1), lsnAt(2)))

	set, err := chain_view.GetChainViewService().ResolveRestoreSetForBackup(prereqs.database.ID, full.ID)
	require.NoError(t, err)

	assert.Equal(t, full.ID, set.RootFull.ID)
	assert.Empty(t, set.Incrementals)
	assert.Empty(t, set.WalSegments)
	assert.Equal(t, lsnAt(1), set.LastIncludedStopLSN)
}

func Test_ResolveRestoreSetForBackup_WhenIncremental_ReturnsFullAndAncestorsNoWal(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)

	full := physical_testing.CreateTestFullBackup(t, physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1, lsnAt(0), lsnAt(1)))
	firstIncr := physical_testing.CreateTestIncrementalBackup(t, physical_testing.NewTestCompletedIncrementalBackup(
		prereqs.database.ID, prereqs.storage.ID, full.ID, nil, 1, lsnAt(1), lsnAt(2)))
	secondIncr := physical_testing.CreateTestIncrementalBackup(t, physical_testing.NewTestCompletedIncrementalBackup(
		prereqs.database.ID, prereqs.storage.ID, full.ID, &firstIncr.ID, 1, lsnAt(2), lsnAt(3)))
	thirdIncr := physical_testing.CreateTestIncrementalBackup(t, physical_testing.NewTestCompletedIncrementalBackup(
		prereqs.database.ID, prereqs.storage.ID, full.ID, &secondIncr.ID, 1, lsnAt(3), lsnAt(4)))

	physical_testing.CreateTestWalSegment(t, physical_testing.NewTestWalSegment(
		prereqs.database.ID, prereqs.storage.ID, 1, "000000010000000000000002", lsnAt(2), lsnAt(3)))

	set, err := chain_view.GetChainViewService().ResolveRestoreSetForBackup(prereqs.database.ID, secondIncr.ID)
	require.NoError(t, err)

	assert.Equal(t, full.ID, set.RootFull.ID)

	// The FULL's ancestors up to and including the target — not the later one.
	includedIDs := make([]uuid.UUID, 0, len(set.Incrementals))
	for _, incremental := range set.Incrementals {
		includedIDs = append(includedIDs, incremental.ID)
	}

	assert.Equal(t, []uuid.UUID{firstIncr.ID, secondIncr.ID}, includedIDs)
	assert.NotContains(t, includedIDs, thirdIncr.ID, "incrementals after the target must be excluded")
	assert.Empty(t, set.WalSegments, "per-backup restore ships no WAL")
	assert.Equal(t, lsnAt(3), set.LastIncludedStopLSN)
}

func Test_ResolveRestoreSetForBackup_WhenBackupMissing_ReturnsNoChainError(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)

	_, err := chain_view.GetChainViewService().ResolveRestoreSetForBackup(prereqs.database.ID, uuid.New())
	assert.ErrorIs(t, err, chain_view.ErrNoChainForRestore)
}

func Test_ResolveRestoreSet_WhenNoFullBackupExists_ReturnsNoChainError(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)

	_, err := chain_view.GetChainViewService().ResolveRestoreSet(prereqs.database.ID, nil)
	assert.ErrorIs(t, err, chain_view.ErrNoChainForRestore)
}

func Test_ResolveRestoreSet_WhenLatestFullOnlyNoWal_ReturnsFullWithEmptyWindow(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)
	clock := newRestoreClock()

	full := physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1, lsnAt(0), lsnAt(1))
	full.CompletedAt = new(clock.at(0))
	physical_testing.CreateTestFullBackup(t, full)

	set, err := chain_view.GetChainViewService().ResolveRestoreSet(prereqs.database.ID, nil)
	require.NoError(t, err)

	assert.Equal(t, full.ID, set.RootFull.ID)
	assert.Empty(t, set.Incrementals)
	assert.Empty(t, set.WalSegments)
	assert.Equal(t, lsnAt(1), set.LastIncludedStopLSN)
	assert.Equal(t, lsnAt(1), set.MaxReplayableLSN)
}

func Test_ResolveRestoreSet_WhenLatestWithIncrementalsAndContiguousWal_ShipsWindowFromLastStop(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)
	clock := newRestoreClock()

	full, incr1, incr2 := seedFullTwoIncrementals(t, prereqs, clock)

	// W3/W4 sit past incr2's stop_lsn (3 segments); W2 [2,3) is folded into incr2.
	seedWalSegment(t, prereqs, "000000010000000000000002", lsnAt(2), lsnAt(3), clock.at(12))
	w3 := seedWalSegment(t, prereqs, "000000010000000000000003", lsnAt(3), lsnAt(4), clock.at(25))
	w4 := seedWalSegment(t, prereqs, "000000010000000000000004", lsnAt(4), lsnAt(5), clock.at(30))

	set, err := chain_view.GetChainViewService().ResolveRestoreSet(prereqs.database.ID, nil)
	require.NoError(t, err)

	assert.Equal(t, full.ID, set.RootFull.ID)
	assertIncrementalIDs(t, set, incr1.ID, incr2.ID)
	assertWalFilenames(t, set, w3.WalFilename, w4.WalFilename)
	assert.Equal(t, lsnAt(3), set.LastIncludedStopLSN)
	assert.Equal(t, lsnAt(5), set.MaxReplayableLSN)
}

func Test_ResolveRestoreSet_WhenTargetBetweenIncrementals_StopsAtEarlierIncrementalAndShipsWalToTarget(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)
	clock := newRestoreClock()

	_, incr1, _ := seedFullTwoIncrementals(t, prereqs, clock)

	w2 := seedWalSegment(t, prereqs, "000000010000000000000002", lsnAt(2), lsnAt(3), clock.at(12))
	w3 := seedWalSegment(t, prereqs, "000000010000000000000003", lsnAt(3), lsnAt(4), clock.at(25))
	w4 := seedWalSegment(t, prereqs, "000000010000000000000004", lsnAt(4), lsnAt(5), clock.at(30))

	target := clock.at(15)

	set, err := chain_view.GetChainViewService().ResolveRestoreSet(prereqs.database.ID, &target)
	require.NoError(t, err)

	assertIncrementalIDs(t, set, incr1.ID)
	assert.Equal(t, lsnAt(2), set.LastIncludedStopLSN, "replay must start at incr1's stop_lsn, not chain start")
	assertWalFilenames(t, set, w2.WalFilename, w3.WalFilename, w4.WalFilename)
}

func Test_ResolveRestoreSet_WhenTargetAfterNewerFullStopBeforeCompletion_UsesAllOlderChainWal(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)
	clock := newRestoreClock()

	olderFull := physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID,
		prereqs.storage.ID,
		1,
		lsnAt(0)+physical_testing.FirstRecordOffset,
		lsnAt(1)+physical_testing.FirstRecordOffset,
	)
	olderFull.CreatedAt = clock.at(-5)
	olderFull.CompletedAt = new(clock.at(0))
	physical_testing.CreateTestFullBackup(t, olderFull)

	newerFull := physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID,
		prereqs.storage.ID,
		1,
		lsnAt(2)+physical_testing.FirstRecordOffset,
		lsnAt(2)+physical_testing.FirstRecordOffset+physical_testing.FullLSNSpan,
	)
	newerFull.CreatedAt = clock.at(5)
	newerFull.CompletedAt = new(clock.at(10))
	physical_testing.CreateTestFullBackup(t, newerFull)

	walBeforeNewerFull := seedWalSegment(
		t,
		prereqs,
		"000000010000000000000001",
		lsnAt(1),
		lsnAt(2),
		clock.at(4),
	)
	walAtNewerFullStart := seedWalSegment(
		t,
		prereqs,
		"000000010000000000000002",
		lsnAt(2),
		lsnAt(3),
		clock.at(6),
	)
	walCoveringTarget := seedWalSegment(
		t,
		prereqs,
		"000000010000000000000003",
		lsnAt(3),
		lsnAt(4),
		clock.at(11),
	)
	walAfterTarget := seedWalSegment(
		t,
		prereqs,
		"000000010000000000000004",
		lsnAt(4),
		lsnAt(5),
		clock.at(12),
	)

	target := clock.at(7)

	set, err := chain_view.GetChainViewService().ResolveRestoreSet(prereqs.database.ID, &target)
	require.NoError(t, err)

	assert.Equal(t, olderFull.ID, set.RootFull.ID)
	assertWalFilenames(
		t,
		set,
		walBeforeNewerFull.WalFilename,
		walAtNewerFullStart.WalFilename,
		walCoveringTarget.WalFilename,
		walAfterTarget.WalFilename,
	)
}

func Test_ResolveRestoreSet_WhenWalGapBeforeTarget_ReturnsGapError(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)
	clock := newRestoreClock()

	seedFullTwoIncrementals(t, prereqs, clock)

	// W3 is missing → the run stops at the end of W2 (3 segments). A target past
	// that point is unreachable.
	seedWalSegment(t, prereqs, "000000010000000000000002", lsnAt(2), lsnAt(3), clock.at(12))
	seedWalSegment(t, prereqs, "000000010000000000000004", lsnAt(4), lsnAt(5), clock.at(30))

	target := clock.at(25)

	_, err := chain_view.GetChainViewService().ResolveRestoreSet(prereqs.database.ID, &target)

	var gapErr chain_view.WalGapBeforeTargetError
	require.True(t, errors.As(err, &gapErr), "expected WalGapBeforeTargetError, got %v", err)
	assert.Equal(t, lsnAt(3), gapErr.LatestRestorableLSN)
}

func Test_ResolveRestoreSet_WhenLatestWithGap_ResolvesToPreGapCeilingWithoutError(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)
	clock := newRestoreClock()

	seedFullTwoIncrementals(t, prereqs, clock)

	w3 := seedWalSegment(t, prereqs, "000000010000000000000003", lsnAt(3), lsnAt(4), clock.at(25))
	// W4 missing; W5 is past the gap and must NOT be shipped.
	seedWalSegment(t, prereqs, "000000010000000000000005", lsnAt(5), lsnAt(6), clock.at(40))

	set, err := chain_view.GetChainViewService().ResolveRestoreSet(prereqs.database.ID, nil)
	require.NoError(t, err)

	assertWalFilenames(t, set, w3.WalFilename)
	assert.Equal(t, lsnAt(4), set.MaxReplayableLSN, "latest must clamp to the pre-gap ceiling")
}

// Test_ResolveRestoreSet_WhenFullBoundariesAreMidSegment_ShipsTheBoundarySegment
// guards against the chain-span filter dropping the WAL segment that physically
// holds the FULL's stop_lsn. A FULL's start/stop are normally mid-segment, so the
// covering segment's file-boundary start_lsn is BELOW the FULL's start_lsn;
// excluding it would manufacture a gap right at the replay window's start and make
// every WAL restore unreachable.
func Test_ResolveRestoreSet_WhenFullBoundariesAreMidSegment_ShipsTheBoundarySegment(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)
	clock := newRestoreClock()

	// FULL starts and stops inside segment 0 ([lsnAt(0), lsnAt(1))).
	full := physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1, lsnAt(0)+1000, lsnAt(0)+2000)
	full.CompletedAt = new(clock.at(0))
	physical_testing.CreateTestFullBackup(t, full)

	boundary := seedWalSegment(t, prereqs, "000000010000000000000000", lsnAt(0), lsnAt(1), clock.at(5))
	next := seedWalSegment(t, prereqs, "000000010000000000000001", lsnAt(1), lsnAt(2), clock.at(10))

	set, err := chain_view.GetChainViewService().ResolveRestoreSet(prereqs.database.ID, nil)
	require.NoError(t, err)

	assertWalFilenames(t, set, boundary.WalFilename, next.WalFilename)
	assert.Equal(t, lsnAt(2), set.MaxReplayableLSN,
		"replay must bridge the FULL's stop through the boundary segment, not stall at it")
}

func Test_ResolveRestoreSet_WhenTargetBeforeEarliestFull_ReturnsTargetBeforeEarliestError(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)
	clock := newRestoreClock()

	full := physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1, lsnAt(0), lsnAt(1))
	full.CompletedAt = new(clock.at(0))
	physical_testing.CreateTestFullBackup(t, full)

	target := clock.at(-60)

	_, err := chain_view.GetChainViewService().ResolveRestoreSet(prereqs.database.ID, &target)
	assert.ErrorIs(t, err, chain_view.ErrTargetBeforeEarliest)
}

// seedFullTwoIncrementals lays down FULL [0,1) → incr1 [1,2) → incr2 [2,3) with
// ordered completion times, returning all three.
func seedFullTwoIncrementals(
	t *testing.T,
	prereqs *chainViewTestPrereqs,
	clock *restoreClock,
) (*physical_models.PhysicalFullBackup, *physical_models.PhysicalIncrementalBackup, *physical_models.PhysicalIncrementalBackup) {
	t.Helper()

	full := physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1, lsnAt(0), lsnAt(1))
	full.CompletedAt = new(clock.at(0))
	physical_testing.CreateTestFullBackup(t, full)

	incr1 := physical_testing.NewTestCompletedIncrementalBackup(
		prereqs.database.ID, prereqs.storage.ID, full.ID, nil, 1, lsnAt(1), lsnAt(2))
	incr1.CompletedAt = new(clock.at(10))
	physical_testing.CreateTestIncrementalBackup(t, incr1)

	incr2 := physical_testing.NewTestCompletedIncrementalBackup(
		prereqs.database.ID, prereqs.storage.ID, full.ID, &incr1.ID, 1, lsnAt(2), lsnAt(3))
	incr2.CompletedAt = new(clock.at(20))
	physical_testing.CreateTestIncrementalBackup(t, incr2)

	return full, incr1, incr2
}

func seedWalSegment(
	t *testing.T,
	prereqs *chainViewTestPrereqs,
	walFilename string,
	startLSN, endLSN walmath.LSN,
	receivedAt time.Time,
) *physical_models.PhysicalWalSegment {
	t.Helper()

	segment := physical_testing.NewTestWalSegment(
		prereqs.database.ID, prereqs.storage.ID, 1, walFilename, startLSN, endLSN)
	segment.ReceivedAt = receivedAt
	physical_testing.CreateTestWalSegment(t, segment)

	return segment
}

func assertIncrementalIDs(t *testing.T, set *chain_view.RestoreSet, expected ...uuid.UUID) {
	t.Helper()

	actual := make([]uuid.UUID, 0, len(set.Incrementals))
	for _, incremental := range set.Incrementals {
		actual = append(actual, incremental.ID)
	}

	assert.Equal(t, expected, actual)
}

func assertWalFilenames(t *testing.T, set *chain_view.RestoreSet, expected ...string) {
	t.Helper()

	actual := make([]string, 0, len(set.WalSegments))
	for _, segment := range set.WalSegments {
		actual = append(actual, segment.WalFilename)
	}

	assert.Equal(t, expected, actual)
}
