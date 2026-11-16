package restoring_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
)

// promotionForkLSN lies inside segment 3, so timeline 1 leaves at a fork point in the middle of a
// segment and both timelines can have a copy of segment 3.
const promotionForkLSN = fixtureLSN(3*segmentBytes + 0x148)

type promotedCatalog struct {
	*catalogFixture
	Timeline *restoreTimeline
	Full     *apptest.PhysicalFullBackupSeed
}

// newPromotedCatalog seeds a timeline 1 FULL and WAL from the source that was promoted to timeline
// 2 at promotionForkLSN. Timeline 1 segment 3 is the former primary's copy of the fork segment,
// with a commit made after the fork point at minute 9. Timeline 1 segment 4 holds only the former
// primary's later writes. Timeline 2 carries the promoted node's writes at minutes 4 and 6.
func newPromotedCatalog(t *testing.T) *promotedCatalog {
	t.Helper()

	catalog := newSwitchedCatalog(t)
	catalog.seedRestoreWal(t, restoreWalSpec{TimelineID: 1, Segment: 3, ReceivedAt: catalog.Timeline.at(9)})
	catalog.seedRestoreWal(t, restoreWalSpec{TimelineID: 1, Segment: 4, ReceivedAt: catalog.Timeline.at(20)})

	return catalog
}

// newSwitchedCatalog seeds the same promotion after a planned switch: the former primary wrote
// nothing after the fork point, and its partial copy of the fork segment was never archived.
func newSwitchedCatalog(t *testing.T) *promotedCatalog {
	t.Helper()

	catalog := newCatalog(t)
	timeline := newRestoreTimeline()
	timeline.base = timeline.base.Truncate(time.Microsecond)
	full := catalog.seedRestoreFull(
		t,
		restoreFullSpec{StartLSN: lsnAt(1), StopLSN: lsnAt(1) + 0x100, CompletedAt: timeline.at(0)},
	)
	catalog.seedRestoreWal(t, restoreWalSpec{TimelineID: 1, Segment: 1, ReceivedAt: timeline.at(1)})
	catalog.seedRestoreWal(t, restoreWalSpec{TimelineID: 1, Segment: 2, ReceivedAt: timeline.at(2)})
	catalog.seedRestoreWal(t, restoreWalSpec{TimelineID: 2, Segment: 3, ReceivedAt: timeline.at(4)})
	catalog.seedRestoreWal(t, restoreWalSpec{TimelineID: 2, Segment: 4, ReceivedAt: timeline.at(6)})

	return &promotedCatalog{catalogFixture: catalog, Timeline: timeline, Full: full}
}

type forkHistorySpec struct {
	StorageID     uuid.UUID
	IsLegacy      bool
	IsConflicting bool
	// TimelineID defaults to 2, leaving timeline 1 at promotionForkLSN.
	TimelineID int
	ForkLSN    fixtureLSN
	Body       string
}

const promotedHistoryBody = "1\t0/3000148\tno recovery target specified\n"

func (catalog *catalogFixture) publishForkHistory(t *testing.T, spec forkHistorySpec) {
	t.Helper()

	storageID := spec.StorageID
	if storageID == uuid.Nil {
		storageID = catalog.StorageID
	}

	timelineID, forkLSN, body := 2, promotionForkLSN, promotedHistoryBody
	if spec.TimelineID != 0 {
		timelineID, forkLSN, body = spec.TimelineID, spec.ForkLSN, spec.Body
	}

	seed := apptest.PhysicalWalHistoryFileSeed{
		DatabaseID: catalog.DatabaseID, StorageID: storageID, TimelineID: timelineID,
		FileName: uuid.NewString() + ".history.zst", HistoryFilename: fmt.Sprintf("%08X.history", timelineID),
		IsConflicting: spec.IsConflicting,
	}
	if !spec.IsLegacy {
		seed.ParentTimelineID = new(timelineID - 1)
		seed.ForkLSN = new(forkLSN.String())
	}

	history := catalog.App.InsertPhysicalWalHistoryFile(t, seed)
	if storageID == catalog.StorageID {
		catalog.publish(t, history.FileName, zstdBytes(t, []byte(body)))
		catalog.publish(t, history.FileName+".metadata", []byte(`{"encryption":"NONE"}`))
	}
}

func (catalog *promotedCatalog) publishPromotedHistory(t *testing.T, spec forkHistorySpec) {
	t.Helper()

	catalog.publishForkHistory(t, spec)
}

func getShippedWal(entries map[string]string) []string {
	walNames := []string{}
	for name := range entries {
		if len(name) > 4 && name[:4] == "wal/" {
			walNames = append(walNames, name)
		}
	}

	return walNames
}

func shippedSegment(timelineID, segment int) string {
	return fmt.Sprintf("wal/%s.zst", walFilename(timelineID, segment))
}

func TestPhysicalRestoreAcrossPromotion(t *testing.T) {
	t.Parallel()

	t.Run(
		"ships the former timeline up to the fork segment and the promoted timeline after it for latest",
		func(t *testing.T) {
			t.Parallel()
			catalog := newPromotedCatalog(t)
			catalog.publishPromotedHistory(t, forkHistorySpec{})

			entries := catalog.downloadRestore(t, restoreRequest{})

			assert.Equal(t, catalog.Full.ID.String(), entries["full/backup_manifest"])
			assert.ElementsMatch(t, []string{
				shippedSegment(1, 1), shippedSegment(1, 2), shippedSegment(1, 3),
				shippedSegment(2, 3), shippedSegment(2, 4), "wal/00000002.history",
			}, getShippedWal(entries))
			assert.Equal(t, promotedHistoryBody, entries["wal/00000002.history"])
			assert.Equal(t, "2\n", entries["RECOVERY_TARGET_TIMELINE"])
		},
	)

	t.Run(
		"crosses the fork point when the former timeline's copy of the fork segment was never archived",
		func(t *testing.T) {
			t.Parallel()
			catalog := newSwitchedCatalog(t)
			catalog.publishPromotedHistory(t, forkHistorySpec{})

			entries := catalog.downloadRestore(t, restoreRequest{TargetTime: new(catalog.Timeline.at(5))})

			assert.ElementsMatch(t, []string{
				shippedSegment(1, 1), shippedSegment(1, 2), shippedSegment(2, 3), shippedSegment(2, 4),
				"wal/00000002.history",
			}, getShippedWal(entries))
			assert.Equal(t, "2\n", entries["RECOVERY_TARGET_TIMELINE"])
		},
	)

	t.Run("restores a timeline 1 chain on its own wal while timeline 2 history is pending", func(t *testing.T) {
		t.Parallel()
		catalog := newPromotedCatalog(t)

		entries := catalog.downloadRestore(t, restoreRequest{})

		assert.ElementsMatch(t, []string{
			shippedSegment(1, 1), shippedSegment(1, 2), shippedSegment(1, 3), shippedSegment(1, 4),
		}, getShippedWal(entries))
		assert.Equal(t, "1\n", entries["RECOVERY_TARGET_TIMELINE"])
	})

	t.Run("restores a target one minute after the promotion before any timeline 2 backup", func(t *testing.T) {
		t.Parallel()
		catalog := newPromotedCatalog(t)
		catalog.publishPromotedHistory(t, forkHistorySpec{})

		entries := catalog.downloadRestore(t, restoreRequest{TargetTime: new(catalog.Timeline.at(5))})

		assert.Equal(t, catalog.Full.ID.String(), entries["full/backup_manifest"])
		assert.Contains(t, entries, shippedSegment(2, 4))
		assert.NotContains(t, entries, shippedSegment(1, 4))
		assert.Equal(t, "2\n", entries["RECOVERY_TARGET_TIMELINE"])
	})

	t.Run("restores a target before a completed replacement full from the earlier chain", func(t *testing.T) {
		t.Parallel()
		catalog := newPromotedCatalog(t)
		catalog.publishPromotedHistory(t, forkHistorySpec{})
		replacement := catalog.seedRestoreFull(t, restoreFullSpec{
			TimelineID: 2, StartLSN: lsnAt(5), StopLSN: lsnAt(5) + 0x100, CompletedAt: catalog.Timeline.at(10),
		})

		entries := catalog.downloadRestore(t, restoreRequest{TargetTime: new(catalog.Timeline.at(5))})

		assert.Equal(t, catalog.Full.ID.String(), entries["full/backup_manifest"])
		assert.NotEqual(t, replacement.ID.String(), entries["full/backup_manifest"])
		assert.Contains(t, entries, shippedSegment(2, 4))
	})

	t.Run("refuses a target only the former primary's wal covers", func(t *testing.T) {
		t.Parallel()
		catalog := newPromotedCatalog(t)
		catalog.publishPromotedHistory(t, forkHistorySpec{})

		for _, minutes := range []int{8, 15} {
			response := catalog.call(t, catalogCall{
				Request:       apptest.PhysicalPitrTokenRequest(catalog.DatabaseID, new(catalog.Timeline.at(minutes))),
				Authorization: "Bearer " + catalog.Owner.Token, StatusCode: http.StatusConflict,
			})

			require.Contains(t, string(response.Body), `"code":"target_time_unreachable"`)
		}
	})

	for _, scenario := range []struct {
		name             string
		history          *forkHistorySpec
		isInOtherStorage bool
	}{
		{name: "refuses a target after the promotion when timeline 2 history is unpublished"},
		{name: "refuses a target after the promotion when timeline 2 history conflicts", history: &forkHistorySpec{IsConflicting: true}},
		{name: "refuses a target after the promotion when timeline 2 history has no fork point", history: &forkHistorySpec{IsLegacy: true}},
		{
			name:             "refuses a target after the promotion when timeline 2 history lies in another storage",
			history:          &forkHistorySpec{},
			isInOtherStorage: true,
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			catalog := newSwitchedCatalog(t)
			if scenario.history != nil {
				history := *scenario.history
				if scenario.isInOtherStorage {
					history.StorageID = catalog.App.CreateLocalStorage(t, catalog.Owner, catalog.WorkspaceID)
				}
				catalog.publishPromotedHistory(t, history)
			}

			response := catalog.call(t, catalogCall{
				Request:       apptest.PhysicalPitrTokenRequest(catalog.DatabaseID, new(catalog.Timeline.at(5))),
				Authorization: "Bearer " + catalog.Owner.Token, StatusCode: http.StatusConflict,
			})

			require.Contains(t, string(response.Body), `"code":"target_time_unreachable"`)
		})
	}

	t.Run("ships no timeline 1 wal after an incremental on timeline 2", func(t *testing.T) {
		t.Parallel()
		catalog := newPromotedCatalog(t)
		catalog.publishPromotedHistory(t, forkHistorySpec{})
		incremental := catalog.seedRestoreIncremental(t, restoreIncrementalSpec{
			TimelineID: 2, FullID: catalog.Full.ID, Segment: 4, CompletedAt: catalog.Timeline.at(7),
		})
		catalog.seedRestoreWal(t, restoreWalSpec{TimelineID: 2, Segment: 5, ReceivedAt: catalog.Timeline.at(8)})

		entries := catalog.downloadRestore(t, restoreRequest{})

		assert.Equal(t, incremental.ID.String(), entries["incr-1/backup_manifest"])
		assert.ElementsMatch(t, []string{shippedSegment(2, 5), "wal/00000002.history"}, getShippedWal(entries))
		assert.Equal(t, "2\n", entries["RECOVERY_TARGET_TIMELINE"])
	})

	t.Run("names no target timeline when a backup is restored by its identifier", func(t *testing.T) {
		t.Parallel()
		catalog := newPromotedCatalog(t)
		catalog.publishPromotedHistory(t, forkHistorySpec{})

		entries := catalog.downloadRestore(t, restoreRequest{BackupID: &catalog.Full.ID})

		assert.Empty(t, getShippedWal(entries))
		assert.NotContains(t, entries, "RECOVERY_TARGET_TIMELINE")
	})
}

func TestPhysicalRestoreAcrossPromotionEdges(t *testing.T) {
	t.Parallel()

	t.Run("splits the wal at a fork point on a segment boundary", func(t *testing.T) {
		t.Parallel()
		catalog := newCatalog(t)
		timeline := newRestoreTimeline()
		catalog.seedRestoreFull(
			t,
			restoreFullSpec{StartLSN: lsnAt(1), StopLSN: lsnAt(1) + 0x100, CompletedAt: timeline.at(0)},
		)
		for segment := 1; segment <= 3; segment++ {
			catalog.seedRestoreWal(t, restoreWalSpec{TimelineID: 1, Segment: segment, ReceivedAt: timeline.at(segment)})
		}
		for segment := 2; segment <= 4; segment++ {
			catalog.seedRestoreWal(
				t,
				restoreWalSpec{TimelineID: 2, Segment: segment, ReceivedAt: timeline.at(segment + 2)},
			)
		}
		catalog.publishForkHistory(t, forkHistorySpec{
			TimelineID: 2, ForkLSN: lsnAt(3), Body: "1\t0/3000000\tno recovery target specified\n",
		})

		entries := catalog.downloadRestore(t, restoreRequest{})

		assert.ElementsMatch(t, []string{
			shippedSegment(1, 1), shippedSegment(1, 2), shippedSegment(2, 3), shippedSegment(2, 4),
			"wal/00000002.history",
		}, getShippedWal(entries), "timeline 1 ends before the boundary and timeline 2 starts at it")
		assert.Equal(t, "2\n", entries["RECOVERY_TARGET_TIMELINE"])
	})

	t.Run("refuses a target whose promoted timeline wal lies only in another storage", func(t *testing.T) {
		t.Parallel()
		catalog := newCatalog(t)
		timeline := newRestoreTimeline()
		timeline.base = timeline.base.Truncate(time.Microsecond)
		otherStorageID := catalog.App.CreateLocalStorage(t, catalog.Owner, catalog.WorkspaceID)
		catalog.seedRestoreFull(
			t,
			restoreFullSpec{StartLSN: lsnAt(1), StopLSN: lsnAt(1) + 0x100, CompletedAt: timeline.at(0)},
		)
		catalog.seedRestoreWal(t, restoreWalSpec{TimelineID: 1, Segment: 1, ReceivedAt: timeline.at(1)})
		catalog.seedRestoreWal(t, restoreWalSpec{TimelineID: 1, Segment: 2, ReceivedAt: timeline.at(2)})
		catalog.seedRestoreWal(
			t,
			restoreWalSpec{TimelineID: 2, Segment: 3, ReceivedAt: timeline.at(4), StorageID: otherStorageID},
		)
		catalog.seedRestoreWal(
			t,
			restoreWalSpec{TimelineID: 2, Segment: 4, ReceivedAt: timeline.at(6), StorageID: otherStorageID},
		)
		catalog.publishForkHistory(t, forkHistorySpec{})

		response := catalog.call(t, catalogCall{
			Request:       apptest.PhysicalPitrTokenRequest(catalog.DatabaseID, new(timeline.at(5))),
			Authorization: "Bearer " + catalog.Owner.Token, StatusCode: http.StatusConflict,
		})

		require.Contains(t, string(response.Body), `"code":"target_time_unreachable"`)
	})

	t.Run("crosses two fork points to a target on timeline 3", func(t *testing.T) {
		t.Parallel()
		catalog := newCatalog(t)
		timeline := newRestoreTimeline()
		timeline.base = timeline.base.Truncate(time.Microsecond)
		full := catalog.seedRestoreFull(
			t,
			restoreFullSpec{StartLSN: lsnAt(1), StopLSN: lsnAt(1) + 0x100, CompletedAt: timeline.at(0)},
		)
		for _, wal := range []restoreWalSpec{
			{TimelineID: 1, Segment: 1, ReceivedAt: timeline.at(1)},
			{TimelineID: 1, Segment: 2, ReceivedAt: timeline.at(2)},
			{TimelineID: 2, Segment: 3, ReceivedAt: timeline.at(3)},
			{TimelineID: 2, Segment: 4, ReceivedAt: timeline.at(4)},
			{TimelineID: 2, Segment: 5, ReceivedAt: timeline.at(5)},
			{TimelineID: 3, Segment: 5, ReceivedAt: timeline.at(6)},
			{TimelineID: 3, Segment: 6, ReceivedAt: timeline.at(8)},
		} {
			catalog.seedRestoreWal(t, wal)
		}
		catalog.publishForkHistory(t, forkHistorySpec{})
		catalog.publishForkHistory(t, forkHistorySpec{
			TimelineID: 3, ForkLSN: lsnAt(5) + 0x60,
			Body: promotedHistoryBody + "2\t0/5000060\tno recovery target specified\n",
		})

		entries := catalog.downloadRestore(t, restoreRequest{TargetTime: new(timeline.at(7))})

		assert.Equal(t, full.ID.String(), entries["full/backup_manifest"])
		assert.ElementsMatch(t, []string{
			shippedSegment(1, 1), shippedSegment(1, 2),
			shippedSegment(2, 3), shippedSegment(2, 4), shippedSegment(2, 5),
			shippedSegment(3, 5), shippedSegment(3, 6),
			"wal/00000002.history", "wal/00000003.history",
		}, getShippedWal(entries))
		assert.Equal(t, "3\n", entries["RECOVERY_TARGET_TIMELINE"])
	})
}
