package streamstatistics_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
)

type restoreTimeline struct{ base time.Time }

func newRestoreTimeline() *restoreTimeline {
	return &restoreTimeline{base: time.Now().UTC().Add(-2 * time.Hour)}
}

func (timeline *restoreTimeline) at(minutes int) time.Time {
	return timeline.base.Add(time.Duration(minutes) * time.Minute)
}
func lsnAt(segments int) fixtureLSN { return fixtureLSN(segments * segmentBytes) }
func newCoverageCatalog(t *testing.T) *catalogFixture {
	t.Helper()
	catalog := newStatisticsCatalog(t)
	catalog.insertStreamer(
		t,
		&apptest.PhysicalWalStreamerSeed{
			DatabaseID:      catalog.DatabaseID,
			StartedAt:       time.Now().UTC().Add(-time.Hour),
			LastHeartbeatAt: time.Now().UTC(),
			Status:          "RUNNING",
		},
	)
	catalog.insertSamples(t, samplesSeedSpec{DatabaseID: catalog.DatabaseID, Count: 4, NewestAt: time.Now().UTC()})
	return catalog
}

func getRestoreCoverage(t *testing.T, catalog *catalogFixture) *statisticsResponse {
	t.Helper()
	return getWalStreamStatistics(t, catalog, "HOUR")
}

func seedCoverageFull(
	t *testing.T,
	catalog *catalogFixture,
	segmentNo int,
	completedAt time.Time,
) *apptest.PhysicalFullBackupSeed {
	t.Helper()

	full := newFullSeed(
		fullSeedSpec{
			DatabaseID: catalog.DatabaseID,
			StorageID:  catalog.StorageID,
			TimelineID: 1,
			StartLSN:   lsnAt(segmentNo),
			StopLSN:    lsnAt(segmentNo + 1),
		},
	)
	full.CompletedAt = &completedAt

	return catalog.insertFull(t, full)
}

func assertRestorableRanges(t *testing.T, coverage *statisticsResponse, restorableRanges [][2]time.Time) {
	t.Helper()

	require.Len(t, coverage.RestorableRanges, len(restorableRanges))

	for index, expectedRange := range restorableRanges {
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

type coverageIncrementalSpec struct {
	Full        *apptest.PhysicalFullBackupSeed
	StartLSN    fixtureLSN
	CompletedAt time.Time
	Status      string
}

func seedCoverageIncremental(
	t *testing.T,
	catalog *catalogFixture,
	spec coverageIncrementalSpec,
) *apptest.PhysicalIncrementalBackupSeed {
	t.Helper()
	incremental := newIncrementalSeed(
		incrementalSeedSpec{
			DatabaseID:       catalog.DatabaseID,
			StorageID:        catalog.StorageID,
			RootFullBackupID: spec.Full.ID,
			TimelineID:       1,
			StartLSN:         spec.StartLSN,
			StopLSN:          spec.StartLSN + 0x100,
		},
	)
	incremental.Status = spec.Status
	incremental.CompletedAt = &spec.CompletedAt
	if spec.Status == "IN_PROGRESS" {
		incremental.StopLSN = nil
		incremental.CompletedAt = nil
	}
	return catalog.insertIncremental(t, incremental)
}

func seedCoverageSegment(
	t *testing.T,
	catalog *catalogFixture,
	segmentNo int,
	receivedAt time.Time,
) *apptest.PhysicalWalSegmentSeed {
	t.Helper()
	return seedTestWalSegmentReceivedAt(t, catalog, uint64(segmentNo), receivedAt)
}
