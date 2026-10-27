package chain_view_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	"databasus-backend/internal/features/backups/backups/core/physical/chain_view"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	physical_testing "databasus-backend/internal/features/backups/backups/core/physical/testing"
)

// An incremental's start and stop sit inside one segment, as they do in a real
// chain.
const incrementalLSNSpan = walmath.LSN(0x100)

func Test_GetRestoreCoverage_WhenNoFullBackupCompleted_ReturnsEmptyCoverage(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)
	clock := newRestoreClock()
	seedCoverageSegment(t, prereqs, 1, clock.at(5))

	coverage := getRestoreCoverage(t, prereqs)

	assert.Nil(t, coverage.LatestRestorableAt)
	assert.NotNil(t, coverage.RestorableRanges)
	assert.Empty(t, coverage.RestorableRanges)
	assert.Empty(t, coverage.BlockingGaps)
}

func Test_GetRestoreCoverage_WhenWalIsContiguous_ReturnsOneRangeUpToNewestSegment(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)
	clock := newRestoreClock()

	seedCoverageFull(t, prereqs, 0, clock.at(0))
	seedCoverageSegment(t, prereqs, 1, clock.at(5))
	seedCoverageSegment(t, prereqs, 2, clock.at(10))

	coverage := getRestoreCoverage(t, prereqs)

	assertRestorableRanges(t, coverage, [][2]time.Time{{clock.at(0), clock.at(10)}})
	assert.Empty(t, coverage.BlockingGaps)
}

func Test_GetRestoreCoverage_WhenGapHasNoIncrementalAfterIt_EndsAtGapAndReportsItBlocking(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)
	clock := newRestoreClock()

	seedCoverageFull(t, prereqs, 0, clock.at(0))
	seedCoverageSegment(t, prereqs, 1, clock.at(5))
	seedCoverageSegment(t, prereqs, 3, clock.at(15))

	coverage := getRestoreCoverage(t, prereqs)

	assertRestorableRanges(t, coverage, [][2]time.Time{{clock.at(0), clock.at(5)}})
	assert.Equal(t, []chain_view.LSNRange{{Start: lsnAt(2), End: lsnAt(3)}}, coverage.BlockingGaps)
}

func Test_GetRestoreCoverage_WhenCompletedIncrementalBridgesGap_ReturnsTwoRangesWithoutBlockingGap(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)
	clock := newRestoreClock()

	full := seedCoverageFull(t, prereqs, 0, clock.at(0))
	seedCoverageSegment(t, prereqs, 1, clock.at(5))
	seedCoverageIncremental(t, prereqs, full, lsnAt(3), clock.at(20), physical_enums.PhysicalBackupStatusCompleted)
	seedCoverageSegment(t, prereqs, 3, clock.at(25))
	seedCoverageSegment(t, prereqs, 4, clock.at(30))

	coverage := getRestoreCoverage(t, prereqs)

	assertRestorableRanges(t, coverage, [][2]time.Time{
		{clock.at(0), clock.at(5)},
		{clock.at(20), clock.at(30)},
	})
	assert.Empty(t, coverage.BlockingGaps, "the incremental after the gap bridges it")
}

func Test_GetRestoreCoverage_WhenIncrementalCompletedBeforeGap_KeepsGapBlocking(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)
	clock := newRestoreClock()

	full := seedCoverageFull(t, prereqs, 0, clock.at(0))
	seedCoverageIncremental(t, prereqs, full, lsnAt(1), clock.at(3), physical_enums.PhysicalBackupStatusCompleted)
	seedCoverageSegment(t, prereqs, 1, clock.at(5))
	seedCoverageSegment(t, prereqs, 3, clock.at(15))

	coverage := getRestoreCoverage(t, prereqs)

	assertRestorableRanges(t, coverage, [][2]time.Time{{clock.at(0), clock.at(5)}})
	assert.Equal(t, []chain_view.LSNRange{{Start: lsnAt(2), End: lsnAt(3)}}, coverage.BlockingGaps)
}

func Test_GetRestoreCoverage_WhenIncrementalAfterGapIsNotCompleted_KeepsGapBlocking(t *testing.T) {
	t.Parallel()

	for _, status := range []physical_enums.PhysicalBackupStatus{
		physical_enums.PhysicalBackupStatusError,
		physical_enums.PhysicalBackupStatusChainBroken,
		physical_enums.PhysicalBackupStatusInProgress,
	} {
		t.Run(string(status), func(t *testing.T) {
			t.Parallel()

			prereqs := createChainViewTestPrereqs(t)
			clock := newRestoreClock()

			full := seedCoverageFull(t, prereqs, 0, clock.at(0))
			seedCoverageSegment(t, prereqs, 1, clock.at(5))
			seedCoverageIncremental(t, prereqs, full, lsnAt(3), clock.at(20), status)
			seedCoverageSegment(t, prereqs, 3, clock.at(25))

			coverage := getRestoreCoverage(t, prereqs)

			assertRestorableRanges(t, coverage, [][2]time.Time{{clock.at(0), clock.at(5)}})
			assert.Equal(t, []chain_view.LSNRange{{Start: lsnAt(2), End: lsnAt(3)}}, coverage.BlockingGaps)
		})
	}
}

func Test_GetRestoreCoverage_WhenIncrementalBridgesOnlyFirstOfTwoGaps_ReportsSecondGapBlocking(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)
	clock := newRestoreClock()

	full := seedCoverageFull(t, prereqs, 0, clock.at(0))
	seedCoverageSegment(t, prereqs, 1, clock.at(5))
	seedCoverageIncremental(t, prereqs, full, lsnAt(3), clock.at(20), physical_enums.PhysicalBackupStatusCompleted)
	seedCoverageSegment(t, prereqs, 3, clock.at(25))
	seedCoverageSegment(t, prereqs, 5, clock.at(40))

	coverage := getRestoreCoverage(t, prereqs)

	assertRestorableRanges(t, coverage, [][2]time.Time{
		{clock.at(0), clock.at(5)},
		{clock.at(20), clock.at(25)},
	})
	assert.Equal(t, []chain_view.LSNRange{{Start: lsnAt(4), End: lsnAt(5)}}, coverage.BlockingGaps)
}

func Test_GetRestoreCoverage_WhenIncrementalStopsInsideMissingSegment_ReturnsPointRangeAndBlockingGap(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)
	clock := newRestoreClock()

	full := seedCoverageFull(t, prereqs, 0, clock.at(0))
	seedCoverageSegment(t, prereqs, 1, clock.at(5))
	seedCoverageIncremental(t, prereqs, full, lsnAt(2)+physical_testing.FirstRecordOffset, clock.at(10),
		physical_enums.PhysicalBackupStatusCompleted)
	seedCoverageSegment(t, prereqs, 3, clock.at(15))

	coverage := getRestoreCoverage(t, prereqs)

	assertRestorableRanges(t, coverage, [][2]time.Time{
		{clock.at(0), clock.at(5)},
		{clock.at(10), clock.at(10)},
	})
	assert.Equal(t, []chain_view.LSNRange{{Start: lsnAt(2), End: lsnAt(3)}}, coverage.BlockingGaps,
		"the WAL right after the incremental is missing, so the restore cannot move past it")
}

func Test_GetRestoreCoverage_WhenSegmentIsStillUploading_EndsRangeBeforeIt(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)
	clock := newRestoreClock()

	seedCoverageFull(t, prereqs, 0, clock.at(0))
	seedCoverageSegment(t, prereqs, 1, clock.at(5))
	uploadingSegment := physical_testing.NewTestWalSegment(prereqs.database.ID, prereqs.storage.ID, 1,
		walmath.WalSegmentNo(2).GetFilename(1, walmath.DefaultSegmentSizeBytes), lsnAt(2), lsnAt(3))
	uploadingSegment.FileName = nil
	uploadingSegment.ReceivedAt = clock.at(10)
	physical_testing.CreateTestWalSegment(t, uploadingSegment)
	seedCoverageSegment(t, prereqs, 3, clock.at(15))

	coverage := getRestoreCoverage(t, prereqs)

	assertRestorableRanges(t, coverage, [][2]time.Time{{clock.at(0), clock.at(5)}})
	assert.Empty(t, coverage.BlockingGaps,
		"an uploading segment is no gap; the verdict waits for the archive progress point instead")
}

func Test_GetRestoreCoverage_WhenNewerFullCompleted_StartsRangesAtNewerFull(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)
	clock := newRestoreClock()

	olderFull := seedCoverageFull(t, prereqs, 0, clock.at(0))
	seedCoverageSegment(t, prereqs, 1, clock.at(5))
	seedCoverageIncremental(t, prereqs, olderFull, lsnAt(3), clock.at(20), physical_enums.PhysicalBackupStatusCompleted)
	seedCoverageSegment(t, prereqs, 3, clock.at(25))

	newerFull := physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1, lsnAt(5), lsnAt(6))
	newerFull.CompletedAt = new(clock.at(30))
	newerFull.CreatedAt = olderFull.CreatedAt.Add(time.Second)
	physical_testing.CreateTestFullBackup(t, newerFull)
	seedCoverageSegment(t, prereqs, 6, clock.at(35))

	coverage := getRestoreCoverage(t, prereqs)

	assertRestorableRanges(t, coverage, [][2]time.Time{{clock.at(30), clock.at(35)}})
	assert.Empty(t, coverage.BlockingGaps, "the older chain's gap is not the newest chain's")
}

func getRestoreCoverage(t *testing.T, prereqs *chainViewTestPrereqs) *chain_view.RestoreCoverage {
	t.Helper()

	coverage, err := chain_view.GetChainViewService().GetRestoreCoverage(prereqs.database.ID)
	require.NoError(t, err)

	return coverage
}

// Checks the ranges and that the last one ends where a latest restore reaches.
func assertRestorableRanges(t *testing.T, coverage *chain_view.RestoreCoverage, expected [][2]time.Time) {
	t.Helper()

	require.Len(t, coverage.RestorableRanges, len(expected))

	for index, expectedRange := range expected {
		assert.WithinDuration(t, expectedRange[0], coverage.RestorableRanges[index].From, time.Millisecond,
			"range %d starts", index)
		assert.WithinDuration(t, expectedRange[1], coverage.RestorableRanges[index].To, time.Millisecond,
			"range %d ends", index)
	}

	require.NotNil(t, coverage.LatestRestorableAt)
	assert.WithinDuration(t, *coverage.LatestRestorableAt,
		coverage.RestorableRanges[len(coverage.RestorableRanges)-1].To, time.Millisecond,
		"the last range ends at the latest restorable moment")
}

func seedCoverageFull(
	t *testing.T,
	prereqs *chainViewTestPrereqs,
	segmentNo int,
	completedAt time.Time,
) *physical_models.PhysicalFullBackup {
	t.Helper()

	full := physical_testing.NewTestCompletedFullBackup(
		prereqs.database.ID, prereqs.storage.ID, 1, lsnAt(segmentNo), lsnAt(segmentNo+1))
	full.CompletedAt = &completedAt

	return physical_testing.CreateTestFullBackup(t, full)
}

func seedCoverageIncremental(
	t *testing.T,
	prereqs *chainViewTestPrereqs,
	full *physical_models.PhysicalFullBackup,
	startLSN walmath.LSN,
	completedAt time.Time,
	status physical_enums.PhysicalBackupStatus,
) *physical_models.PhysicalIncrementalBackup {
	t.Helper()

	incremental := physical_testing.NewTestCompletedIncrementalBackup(
		prereqs.database.ID, prereqs.storage.ID, full.ID, nil, 1, startLSN, startLSN+incrementalLSNSpan)
	incremental.Status = status
	incremental.CompletedAt = &completedAt

	if status == physical_enums.PhysicalBackupStatusInProgress {
		incremental.StopLSN = nil
		incremental.CompletedAt = nil
	}

	return physical_testing.CreateTestIncrementalBackup(t, incremental)
}

func seedCoverageSegment(
	t *testing.T,
	prereqs *chainViewTestPrereqs,
	segmentNo int,
	receivedAt time.Time,
) *physical_models.PhysicalWalSegment {
	t.Helper()

	return seedWalSegment(t, prereqs, walmath.WalSegmentNo(segmentNo).GetFilename(1, walmath.DefaultSegmentSizeBytes),
		lsnAt(segmentNo), lsnAt(segmentNo+1), receivedAt)
}
