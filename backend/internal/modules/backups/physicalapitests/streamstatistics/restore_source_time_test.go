package streamstatistics_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPhysicalRestoreSourceTime(t *testing.T) {
	t.Parallel()

	t.Run("keeps source time when WAL publishes after the source timeline ended", func(t *testing.T) {
		t.Parallel()

		catalog := newCoverageCatalog(t)
		timeline := newRestoreTimeline()
		timeline.base = timeline.base.Truncate(time.Microsecond)
		seedCoverageFull(t, catalog, 0, timeline.at(0))
		segment := seedCoverageSegment(t, catalog, 1, timeline.at(15))
		catalog.App.ExecuteSQL(t, "UPDATE backups_physical_wal_segments SET last_transaction_at=? WHERE id=?",
			timeline.at(5), segment.ID)

		coverage := getRestoreCoverage(t, catalog)

		requireSourceTimeRange(t, coverage, restorableTimeRange{
			From: timeline.at(0), To: timeline.at(5).Add(-time.Microsecond),
		})
	})

	t.Run("keeps the last confirmed time when a following WAL segment has no transaction evidence", func(t *testing.T) {
		t.Parallel()

		catalog := newCoverageCatalog(t)
		timeline := newRestoreTimeline()
		timeline.base = timeline.base.Truncate(time.Microsecond)
		seedCoverageFull(t, catalog, 0, timeline.at(0))
		seedCoverageSegment(t, catalog, 1, timeline.at(5))
		segment := seedCoverageSegment(t, catalog, 2, timeline.at(15))
		catalog.App.ExecuteSQL(
			t,
			"UPDATE backups_physical_wal_segments SET last_transaction_at=NULL WHERE id=?",
			segment.ID,
		)

		coverage := getRestoreCoverage(t, catalog)

		requireSourceTimeRange(t, coverage, restorableTimeRange{From: timeline.at(0), To: timeline.at(5)})
	})
}

func requireSourceTimeRange(t *testing.T, coverage *statisticsResponse, bounds restorableTimeRange) {
	t.Helper()

	require.Len(t, coverage.RestorableRanges, 1)
	require.Equal(t, bounds.From, coverage.RestorableRanges[0].From)
	require.Equal(t, bounds.To, coverage.RestorableRanges[0].To)
	require.NotNil(t, coverage.LatestRestorableAt)
	require.Equal(t, bounds.To, *coverage.LatestRestorableAt)
}
