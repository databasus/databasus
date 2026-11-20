package catalog_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

// promotionForkLSN lies inside segment 3: timeline 2 leaves timeline 1 there, so segment 3 has a
// copy on both timelines.
const promotionForkLSN = fixtureLSN(3*segmentBytes + 0x148)

type promotionHistorySpec struct {
	IsLegacy      bool
	IsConflicting bool
}

func (catalog *catalogFixture) seedPromotionHistory(
	t *testing.T,
	spec promotionHistorySpec,
) apptest.PhysicalWalHistoryFileSeed {
	t.Helper()

	seed := apptest.PhysicalWalHistoryFileSeed{
		DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID, TimelineID: 2,
		FileName: uuid.NewString() + ".history.zst", HistoryFilename: fmtHistoryName(2),
		IsConflicting: spec.IsConflicting,
	}
	if !spec.IsLegacy {
		seed.ParentTimelineID = new(1)
		seed.ForkLSN = new(promotionForkLSN.String())
	}

	return catalog.App.InsertPhysicalWalHistoryFile(t, seed)
}

type promotedWal struct {
	FormerSegments   map[int]*apptest.PhysicalWalSegmentSeed
	PromotedSegments map[int]*apptest.PhysicalWalSegmentSeed
}

// seedPromotedWal archives timeline 1 segments 1 to 3, timeline 1 segment 4 that only the former
// primary wrote after the fork point, and timeline 2 segments 3 to 6.
func (catalog *catalogFixture) seedPromotedWal(t *testing.T) promotedWal {
	t.Helper()

	wal := promotedWal{
		FormerSegments:   map[int]*apptest.PhysicalWalSegmentSeed{},
		PromotedSegments: map[int]*apptest.PhysicalWalSegmentSeed{},
	}
	for _, segment := range []int{1, 2, 3, 4} {
		wal.FormerSegments[segment] = catalog.seedCascadeWal(t, cascadeWalSpec{Segment: segment})
	}
	for _, segment := range []int{3, 4, 5, 6} {
		wal.PromotedSegments[segment] = catalog.seedCascadeWal(t, cascadeWalSpec{Segment: segment, TimelineID: 2})
	}

	return wal
}

func (catalog *catalogFixture) runCleanupStages(t *testing.T) {
	t.Helper()

	catalog.saveStorageConfig(t, catalog.StorageID)
	catalog.App.RunJobPass(t, "backups_physical_cleaner")
	catalog.App.RunJobPass(t, "storage_file_deletion")
}

func TestPhysicalWalRetentionAlongTimelineLine(t *testing.T) {
	t.Parallel()

	t.Run("keeps timeline 2 wal before any timeline 2 backup when its history is published", func(t *testing.T) {
		t.Parallel()
		catalog := newCatalog(t)
		catalog.seedCascadeFull(t, cascadeFullSpec{Segment: 1, Offset: 0x28})
		catalog.seedPromotionHistory(t, promotionHistorySpec{})
		wal := catalog.seedPromotedWal(t)

		catalog.runCleanupStages(t)

		for _, segment := range wal.PromotedSegments {
			catalog.assertWalPresent(t, segment.ID)
		}
		catalog.assertWalPresent(t, wal.FormerSegments[3].ID)
	})

	t.Run("keeps timeline 2 wal while its history is pending", func(t *testing.T) {
		t.Parallel()
		catalog := newCatalog(t)
		catalog.seedCascadeFull(t, cascadeFullSpec{Segment: 1, Offset: 0x28})
		wal := catalog.seedPromotedWal(t)

		catalog.runCleanupStages(t)

		for _, segment := range wal.PromotedSegments {
			catalog.assertWalPresent(t, segment.ID)
		}
	})

	for _, spec := range []promotionHistorySpec{{IsConflicting: true}, {IsLegacy: true}} {
		name := "removes timeline 2 wal behind a conflicting link"
		if spec.IsLegacy {
			name = "removes timeline 2 wal behind a link published without a fork point"
		}

		t.Run(name, func(t *testing.T) {
			t.Parallel()
			catalog := newCatalog(t)
			catalog.seedCascadeFull(t, cascadeFullSpec{Segment: 1, Offset: 0x28})
			catalog.seedPromotionHistory(t, spec)
			wal := catalog.seedPromotedWal(t)

			catalog.runCleanupStages(t)

			for _, segment := range wal.PromotedSegments {
				catalog.assertWalGone(t, segment.ID)
			}
			catalog.assertWalPresent(t, wal.FormerSegments[4].ID)
		})
	}

	t.Run("removes the former primary's wal past the fork point", func(t *testing.T) {
		t.Parallel()
		catalog := newCatalog(t)
		catalog.seedCascadeFull(t, cascadeFullSpec{Segment: 1, Offset: 0x28})
		catalog.seedPromotionHistory(t, promotionHistorySpec{})
		wal := catalog.seedPromotedWal(t)

		catalog.runCleanupStages(t)

		catalog.assertWalGone(t, wal.FormerSegments[4].ID)
		catalog.assertWalPresent(t, wal.FormerSegments[3].ID)
		catalog.assertWalPresent(t, wal.FormerSegments[2].ID)
	})

	t.Run("keeps the former primary's wal past the fork point for a full taken there", func(t *testing.T) {
		t.Parallel()
		catalog := newCatalog(t)
		catalog.seedCascadeFull(t, cascadeFullSpec{Segment: 1, Offset: 0x28})
		catalog.seedCascadeFull(t, cascadeFullSpec{Segment: 4, Offset: 0x28})
		catalog.seedPromotionHistory(t, promotionHistorySpec{})
		wal := catalog.seedPromotedWal(t)

		catalog.runCleanupStages(t)

		catalog.assertWalPresent(t, wal.FormerSegments[4].ID)
		catalog.assertWalPresent(t, wal.PromotedSegments[4].ID)
	})

	t.Run("issues a token for a timeline 2 target after both cleanup stages", func(t *testing.T) {
		t.Parallel()
		catalog := newCatalog(t)
		catalog.seedCascadeFull(t, cascadeFullSpec{Segment: 1, Offset: 0x28, Age: 2 * time.Hour})
		catalog.seedPromotionHistory(t, promotionHistorySpec{})
		for _, segment := range []int{1, 2} {
			catalog.seedCascadeWal(t, cascadeWalSpec{Segment: segment, Age: time.Hour})
		}
		for _, segment := range []int{3, 4} {
			wal := catalog.seedCascadeWal(t, cascadeWalSpec{Segment: segment, TimelineID: 2, Age: 10 * time.Minute})
			catalog.App.ExecuteSQL(
				t,
				"UPDATE backups_physical_wal_segments SET last_transaction_at=received_at WHERE id=?",
				wal.ID,
			)
		}

		catalog.runCleanupStages(t)

		response := catalog.call(t, catalogCall{
			Request: apptest.PhysicalPitrTokenRequest(
				catalog.DatabaseID,
				new(time.Now().UTC().Add(-30*time.Minute)),
			),
			Authorization: "Bearer " + catalog.Owner.Token,
			StatusCode:    http.StatusCreated,
		})
		require.Contains(t, string(response.Body), `"token"`)
	})

	t.Run("sets a closed chain's end time from timeline 2 wal received last", func(t *testing.T) {
		t.Parallel()
		cleaner := newCleanerCatalog(t)
		configureRetentionSchedule(&cleaner.Config)
		cleaner.Config.Retention.ChainsCount = 1
		older := cleaner.seedChainFull(t, chainFullSpec{StartSegment: 0, AgeHours: 40})
		cleaner.seedPromotionHistory(t, promotionHistorySpec{})
		newer := cleaner.seedChainFull(t, chainFullSpec{TimelineID: 2, StartSegment: 5, AgeHours: 20})
		cleaner.seedChainFull(t, chainFullSpec{TimelineID: 2, StartSegment: 8, AgeHours: 0})
		cleaner.seedCascadeWal(t, cascadeWalSpec{Segment: 4, TimelineID: 2, Age: 50 * time.Hour})

		cleaner.clean(t, apiclient.PhysicalRetentionPolicyCHAINS)

		cleaner.assertFullPresent(t, older.ID)
		cleaner.assertFullGone(t, newer.ID)
	})
}

func TestDeletePhysicalFullAlongTimelineLine(t *testing.T) {
	t.Parallel()

	t.Run(
		"keeps timeline 2 wal and history when deleting a timeline 2 full while a timeline 1 full remains",
		func(t *testing.T) {
			t.Parallel()
			catalog := newCatalog(t)
			catalog.seedCascadeFull(t, cascadeFullSpec{Segment: 1, Offset: 0x28, Age: time.Hour})
			history := catalog.seedPromotionHistory(t, promotionHistorySpec{})
			promotedFull := catalog.seedCascadeFull(t, cascadeFullSpec{Segment: 5, Offset: 0x28, TimelineID: 2})
			wal := catalog.seedPromotedWal(t)

			catalog.deleteFull(t, promotedFull.ID)

			for _, segment := range wal.PromotedSegments {
				catalog.assertWalPresent(t, segment.ID)
			}
			catalog.assertHistoryPresent(t, history.ID, true)
		},
	)

	t.Run(
		"removes only the ownership span when deleting the earliest full whose line crosses a fork point",
		func(t *testing.T) {
			t.Parallel()
			catalog := newCatalog(t)
			earliest := catalog.seedCascadeFull(t, cascadeFullSpec{Segment: 1, Offset: 0x28, Age: time.Hour})
			history := catalog.seedPromotionHistory(t, promotionHistorySpec{})
			catalog.seedCascadeFull(t, cascadeFullSpec{Segment: 5, Offset: 0x28, TimelineID: 2})
			wal := catalog.seedPromotedWal(t)

			catalog.deleteFull(t, earliest.ID)

			catalog.assertWalGone(t, wal.FormerSegments[2].ID)
			catalog.assertWalGone(t, wal.PromotedSegments[4].ID)
			for _, kept := range []*apptest.PhysicalWalSegmentSeed{
				wal.FormerSegments[1], wal.FormerSegments[3], wal.FormerSegments[4],
				wal.PromotedSegments[3], wal.PromotedSegments[5], wal.PromotedSegments[6],
			} {
				assert.True(t, catalog.isWalPresent(t, kept.ID), "%s", kept.WalFilename)
			}
			catalog.assertHistoryPresent(t, history.ID, true)
		},
	)
}

func TestDeleteLastPhysicalFullOnPromotedTimeline(t *testing.T) {
	t.Parallel()

	t.Run(
		"removes the timeline history and its leftover wal when the last full on its line is deleted",
		func(t *testing.T) {
			t.Parallel()
			catalog := newCatalog(t)
			history := catalog.seedPromotionHistory(t, promotionHistorySpec{})
			promotedFull := catalog.seedCascadeFull(t, cascadeFullSpec{Segment: 5, Offset: 0x28, TimelineID: 2})
			boundary := catalog.seedCascadeWal(t, cascadeWalSpec{Segment: 5, TimelineID: 2})
			owned := catalog.seedCascadeWal(t, cascadeWalSpec{Segment: 6, TimelineID: 2})

			catalog.deleteFull(t, promotedFull.ID)

			catalog.assertWalGone(t, owned.ID)
			catalog.assertWalGone(t, boundary.ID)
			catalog.assertHistoryPresent(t, history.ID, false)
		},
	)
}

func (catalog *catalogFixture) isWalPresent(t *testing.T, segmentID uuid.UUID) bool {
	t.Helper()

	var count int64
	catalog.App.QuerySQL(t, &count, "SELECT count(*) FROM backups_physical_wal_segments WHERE id=?", segmentID)

	return count == 1
}
