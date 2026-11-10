package streamstatistics_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/app/apptest"
)

func TestPhysicalHistoryRestoreCoverage(t *testing.T) {
	t.Parallel()

	t.Run(
		"extends coverage only after the selected timeline history publishes without reporting a WAL gap",
		func(t *testing.T) {
			t.Parallel()

			catalog := newCoverageCatalog(t)
			clock := newRestoreTimeline()
			full := seedCoverageFull(t, catalog, 0, clock.at(0))
			wal := seedCoverageSegment(t, catalog, 1, clock.at(5))
			incremental := seedCoverageIncremental(
				t,
				catalog,
				coverageIncrementalSpec{Full: full, StartLSN: lsnAt(1), CompletedAt: clock.at(3), Status: "COMPLETED"},
			)
			catalog.App.ExecuteSQL(t, "UPDATE backups_physical SET timeline_id=2 WHERE id=?", incremental.ID)
			catalog.App.ExecuteSQL(t, "UPDATE backups_physical SET timeline_id=2 WHERE id=?", full.ID)
			catalog.App.ExecuteSQL(t, "UPDATE backups_physical_wal_segments SET timeline_id=2 WHERE id=?", wal.ID)

			coverage := getRestoreCoverage(t, catalog)
			assertRestorableRanges(t, coverage, [][2]time.Time{{clock.at(0), clock.at(0)}, {clock.at(3), clock.at(3)}})
			assert.NotContains(t, getReasonCodes(coverage.Health.Reasons), "WAL_CHAIN_BROKEN")

			history := catalog.App.InsertPhysicalWalHistoryFile(t, apptest.PhysicalWalHistoryFileSeed{
				DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID, TimelineID: 2,
				FileName: "selected.history.zst", HistoryFilename: "00000002.history",
			})
			coverage = getRestoreCoverage(t, catalog)
			assertRestorableRanges(t, coverage, [][2]time.Time{{clock.at(0), clock.at(5)}})
			assert.NotContains(t, getReasonCodes(coverage.Health.Reasons), "WAL_CHAIN_BROKEN")

			otherStorage := catalog.App.CreateLocalStorage(t, catalog.Owner, catalog.WorkspaceID)
			catalog.App.ExecuteSQL(
				t,
				"UPDATE backups_physical_wal_history_files SET storage_id=? WHERE id=?",
				otherStorage,
				history.ID,
			)
			coverage = getRestoreCoverage(t, catalog)
			assertRestorableRanges(t, coverage, [][2]time.Time{{clock.at(0), clock.at(0)}, {clock.at(3), clock.at(3)}})
			assert.NotContains(t, getReasonCodes(coverage.Health.Reasons), "WAL_CHAIN_BROKEN")

			catalog.App.ExecuteSQL(t, "DELETE FROM backups_physical_wal_history_files WHERE id=?", history.ID)
			coverage = getRestoreCoverage(t, catalog)
			assertRestorableRanges(t, coverage, [][2]time.Time{{clock.at(0), clock.at(0)}, {clock.at(3), clock.at(3)}})
			assert.NotContains(t, getReasonCodes(coverage.Health.Reasons), "WAL_CHAIN_BROKEN")
		},
	)
}
