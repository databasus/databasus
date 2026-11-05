package streamstatistics_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetPhysicalRestoreCoverage(t *testing.T) {
	t.Parallel()

	t.Run("returns no ranges when no full backup completed", func(t *testing.T) {
		t.Parallel()

		catalog := newCoverageCatalog(t)
		clock := newRestoreTimeline()
		seedCoverageSegment(t, catalog, 1, clock.at(5))

		coverage := getRestoreCoverage(t, catalog)

		assert.Nil(t, coverage.LatestRestorableAt)
		assert.NotNil(t, coverage.RestorableRanges)
		assert.Empty(t, coverage.RestorableRanges)
		assert.NotContains(t, getReasonCodes(coverage.Health.Reasons), "WAL_CHAIN_BROKEN")
	})
	t.Run("returns one range through the newest segment when WAL is contiguous", func(t *testing.T) {
		t.Parallel()

		catalog := newCoverageCatalog(t)
		clock := newRestoreTimeline()

		seedCoverageFull(t, catalog, 0, clock.at(0))
		seedCoverageSegment(t, catalog, 1, clock.at(5))
		seedCoverageSegment(t, catalog, 2, clock.at(10))

		coverage := getRestoreCoverage(t, catalog)

		assertRestorableRanges(t, coverage, [][2]time.Time{{clock.at(0), clock.at(10)}})
		assert.NotContains(t, getReasonCodes(coverage.Health.Reasons), "WAL_CHAIN_BROKEN")
	})
	t.Run("ends at the gap and reports a broken chain when no incremental bridges it", func(t *testing.T) {
		t.Parallel()

		catalog := newCoverageCatalog(t)
		clock := newRestoreTimeline()

		seedCoverageFull(t, catalog, 0, clock.at(0))
		seedCoverageSegment(t, catalog, 1, clock.at(5))
		seedCoverageSegment(t, catalog, 3, clock.at(15))

		coverage := getRestoreCoverage(t, catalog)

		assertRestorableRanges(t, coverage, [][2]time.Time{{clock.at(0), clock.at(5)}})
		assert.Contains(t, getReasonCodes(coverage.Health.Reasons), "WAL_CHAIN_BROKEN")
	})
	t.Run(
		"returns two ranges without a broken verdict when a completed incremental bridges the gap",
		func(t *testing.T) {
			t.Parallel()

			catalog := newCoverageCatalog(t)
			clock := newRestoreTimeline()

			full := seedCoverageFull(t, catalog, 0, clock.at(0))
			seedCoverageSegment(t, catalog, 1, clock.at(5))
			seedCoverageIncremental(
				t,
				catalog,
				coverageIncrementalSpec{Full: full, StartLSN: lsnAt(3), CompletedAt: clock.at(20), Status: "COMPLETED"},
			)
			seedCoverageSegment(t, catalog, 3, clock.at(25))
			seedCoverageSegment(t, catalog, 4, clock.at(30))

			coverage := getRestoreCoverage(t, catalog)

			assertRestorableRanges(t, coverage, [][2]time.Time{
				{clock.at(0), clock.at(5)},
				{clock.at(20), clock.at(30)},
			})
			assert.NotContains(t, getReasonCodes(coverage.Health.Reasons), "WAL_CHAIN_BROKEN")
		},
	)
	t.Run("keeps the gap blocking when the incremental completed before it", func(t *testing.T) {
		t.Parallel()

		catalog := newCoverageCatalog(t)
		clock := newRestoreTimeline()

		full := seedCoverageFull(t, catalog, 0, clock.at(0))
		seedCoverageIncremental(
			t,
			catalog,
			coverageIncrementalSpec{Full: full, StartLSN: lsnAt(1), CompletedAt: clock.at(3), Status: "COMPLETED"},
		)
		seedCoverageSegment(t, catalog, 1, clock.at(5))
		seedCoverageSegment(t, catalog, 3, clock.at(15))

		coverage := getRestoreCoverage(t, catalog)

		assertRestorableRanges(t, coverage, [][2]time.Time{{clock.at(0), clock.at(5)}})
		assert.Contains(t, getReasonCodes(coverage.Health.Reasons), "WAL_CHAIN_BROKEN")
	})
	t.Run("keeps the gap blocking when the incremental after it did not complete", func(t *testing.T) {
		t.Parallel()

		for _, status := range []string{
			"ERROR",
			"CHAIN_BROKEN",
			"IN_PROGRESS",
		} {
			t.Run("keeps the gap blocking when the incremental status is "+strings.ToLower(status), func(t *testing.T) {
				catalog := newCoverageCatalog(t)
				clock := newRestoreTimeline()

				full := seedCoverageFull(t, catalog, 0, clock.at(0))
				seedCoverageSegment(t, catalog, 1, clock.at(5))
				seedCoverageIncremental(
					t,
					catalog,
					coverageIncrementalSpec{Full: full, StartLSN: lsnAt(3), CompletedAt: clock.at(20), Status: status},
				)
				seedCoverageSegment(t, catalog, 3, clock.at(25))

				coverage := getRestoreCoverage(t, catalog)

				assertRestorableRanges(t, coverage, [][2]time.Time{{clock.at(0), clock.at(5)}})
				assert.Contains(t, getReasonCodes(coverage.Health.Reasons), "WAL_CHAIN_BROKEN")
			})
		}
	})
	t.Run("keeps the second gap blocking when an incremental bridges only the first gap", func(t *testing.T) {
		t.Parallel()

		catalog := newCoverageCatalog(t)
		clock := newRestoreTimeline()

		full := seedCoverageFull(t, catalog, 0, clock.at(0))
		seedCoverageSegment(t, catalog, 1, clock.at(5))
		seedCoverageIncremental(
			t,
			catalog,
			coverageIncrementalSpec{Full: full, StartLSN: lsnAt(3), CompletedAt: clock.at(20), Status: "COMPLETED"},
		)
		seedCoverageSegment(t, catalog, 3, clock.at(25))
		seedCoverageSegment(t, catalog, 5, clock.at(40))

		coverage := getRestoreCoverage(t, catalog)

		assertRestorableRanges(t, coverage, [][2]time.Time{
			{clock.at(0), clock.at(5)},
			{clock.at(20), clock.at(25)},
		})
		assert.Contains(t, getReasonCodes(coverage.Health.Reasons), "WAL_CHAIN_BROKEN")
	})
	t.Run("returns a single point for an incremental when it stops inside the missing segment", func(t *testing.T) {
		t.Parallel()

		catalog := newCoverageCatalog(t)
		clock := newRestoreTimeline()

		full := seedCoverageFull(t, catalog, 0, clock.at(0))
		seedCoverageSegment(t, catalog, 1, clock.at(5))
		seedCoverageIncremental(
			t,
			catalog,
			coverageIncrementalSpec{
				Full:        full,
				StartLSN:    lsnAt(2) + 0x28,
				CompletedAt: clock.at(10),
				Status:      "COMPLETED",
			},
		)
		seedCoverageSegment(t, catalog, 3, clock.at(15))

		coverage := getRestoreCoverage(t, catalog)

		assertRestorableRanges(t, coverage, [][2]time.Time{
			{clock.at(0), clock.at(5)},
			{clock.at(10), clock.at(10)},
		})
		assert.Contains(t, getReasonCodes(coverage.Health.Reasons), "WAL_CHAIN_BROKEN")
	})
	t.Run("ends before an uploading segment without a broken verdict while its claim is live", func(t *testing.T) {
		t.Parallel()

		catalog := newCoverageCatalog(t)
		clock := newRestoreTimeline()

		seedCoverageFull(t, catalog, 0, clock.at(0))
		seedCoverageSegment(t, catalog, 1, clock.at(5))

		uploadingSegment := newWalSeed(
			walSeedSpec{
				DatabaseID:  catalog.DatabaseID,
				StorageID:   catalog.StorageID,
				TimelineID:  1,
				WalFilename: fixtureSegmentNo(2).GetFilename(1, segmentBytes),
				StartLSN:    lsnAt(2),
				EndLSN:      lsnAt(3),
			},
		)
		uploadingSegment.FileName = nil
		uploadingSegment.ReceivedAt = clock.at(10)
		catalog.insertWal(t, uploadingSegment)

		seedCoverageSegment(t, catalog, 3, clock.at(15))

		coverage := getRestoreCoverage(t, catalog)

		assertRestorableRanges(t, coverage, [][2]time.Time{{clock.at(0), clock.at(5)}})
		assert.NotContains(t, getReasonCodes(coverage.Health.Reasons), "WAL_CHAIN_BROKEN")
	})
	t.Run("starts the ranges at the newer full when it completes", func(t *testing.T) {
		t.Parallel()

		catalog := newCoverageCatalog(t)
		clock := newRestoreTimeline()

		olderFull := seedCoverageFull(t, catalog, 0, clock.at(0))
		seedCoverageSegment(t, catalog, 1, clock.at(5))
		seedCoverageIncremental(
			t,
			catalog,
			coverageIncrementalSpec{
				Full:        olderFull,
				StartLSN:    lsnAt(3),
				CompletedAt: clock.at(20),
				Status:      "COMPLETED",
			},
		)
		seedCoverageSegment(t, catalog, 3, clock.at(25))

		newerFull := newFullSeed(
			fullSeedSpec{
				DatabaseID: catalog.DatabaseID,
				StorageID:  catalog.StorageID,
				TimelineID: 1,
				StartLSN:   lsnAt(5),
				StopLSN:    lsnAt(6),
			},
		)
		newerFull.CompletedAt = new(clock.at(30))
		newerFull.CreatedAt = olderFull.CreatedAt.Add(time.Minute)
		catalog.insertFull(t, newerFull)
		seedCoverageSegment(t, catalog, 6, clock.at(35))

		coverage := getRestoreCoverage(t, catalog)

		assertRestorableRanges(t, coverage, [][2]time.Time{{clock.at(30), clock.at(35)}})
		assert.NotContains(t, getReasonCodes(coverage.Health.Reasons), "WAL_CHAIN_BROKEN")
	})
}
