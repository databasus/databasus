package catalog_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestPhysicalBackupChains(t *testing.T) {
	t.Parallel()

	t.Run("accepts an incremental when the chain has a healthy full", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		catalog.seedCascadeFull(t, cascadeFullSpec{})
		catalog.seedCascadeWal(t, cascadeWalSpec{Segment: 1})
		catalog.triggerIncremental(t, http.StatusAccepted)
	})
	t.Run("refuses an incremental and retains the old chain when its incremental is broken", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		full := catalog.seedCascadeFull(t, cascadeFullSpec{})
		incremental := catalog.seedCascadeIncremental(t, full.ID, 1)
		catalog.App.ExecuteSQL(
			t,
			"UPDATE backups_physical SET status='CHAIN_BROKEN',error_reason='SUMMARIES_EXPIRED' WHERE id=?",
			incremental.ID,
		)

		catalog.triggerIncremental(t, http.StatusConflict)
		catalog.assertFullPresent(t, full.ID)
	})
	t.Run("keeps a chain extendable and reports the gap when intermediate WAL is missing", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		catalog.seedCascadeFull(t, cascadeFullSpec{})
		catalog.seedCascadeWal(t, cascadeWalSpec{Segment: 1})
		catalog.seedCascadeWal(t, cascadeWalSpec{Segment: 4})

		catalog.triggerIncremental(t, http.StatusAccepted)
		page := getBackups(t, catalog, "?type=WAL")
		require.Len(t, page.Backups, 2)
		assertWalGap(t, page, 2, 4)
	})
	t.Run(
		"retains the newest healthy chain and permits an incremental after deleting the older full",
		func(t *testing.T) {
			t.Parallel()

			catalog := newCatalog(t)
			older := catalog.seedCascadeFull(t, cascadeFullSpec{Age: time.Hour})
			newer := catalog.seedCascadeFull(t, cascadeFullSpec{Segment: 4})
			catalog.deleteFull(t, older.ID)
			catalog.assertFullPresent(t, newer.ID)
			catalog.triggerIncremental(t, http.StatusAccepted)
		},
	)
	t.Run("ends the removable WAL span at the next full start", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		older := catalog.seedCascadeFull(t, cascadeFullSpec{Age: time.Hour})
		newer := catalog.seedCascadeFull(t, cascadeFullSpec{Segment: 4})
		catalog.deleteFull(t, older.ID)
		catalog.assertFullPresent(t, newer.ID)
		assert.Empty(t, getBackups(t, catalog, "?type=WAL").Backups)
	})
	t.Run("removes WAL beyond the full stop when no successor bounds the span", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		full := catalog.seedCascadeFull(t, cascadeFullSpec{Segment: 4})
		wal := catalog.seedCascadeWal(t, cascadeWalSpec{Segment: 5})
		catalog.deleteFull(t, full.ID)
		catalog.assertWalGone(t, wal.ID)
	})
	t.Run("orders inactive chains by the latest WAL receive time for retention", func(t *testing.T) {
		t.Parallel()

		var catalog *catalogFixture
		cleaner := newCleanerCatalog(t)
		catalog = cleaner.catalogFixture
		configureRetentionSchedule(&cleaner.Config)
		cleaner.Config.Retention.ChainsCount = 1

		older := cleaner.seedChainFull(t, chainFullSpec{StartSegment: 0, AgeHours: 40})
		newer := cleaner.seedChainFull(t, chainFullSpec{StartSegment: 4, AgeHours: 20})
		cleaner.seedChainFull(t, chainFullSpec{StartSegment: 8, AgeHours: 0})
		incremental := catalog.seedCascadeIncremental(t, older.ID, 1)
		catalog.App.ExecuteSQL(
			t,
			"UPDATE backups_physical SET completed_at=? WHERE id=?",
			time.Now().UTC().Add(-70*time.Hour),
			incremental.ID,
		)
		catalog.seedCascadeWal(t, cascadeWalSpec{Segment: 1, Age: 60 * time.Hour})

		cleaner.clean(t, apiclient.PhysicalRetentionPolicyCHAINS)
		catalog.assertFullPresent(t, older.ID)
		catalog.assertFullGone(t, newer.ID)
	})
	t.Run("removes only the orphan before the full start when sweeping WAL", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		catalog.seedCascadeFull(t, cascadeFullSpec{Segment: 4})
		orphan := catalog.seedCascadeWal(t, cascadeWalSpec{Segment: 1})
		anchored := catalog.seedCascadeWal(t, cascadeWalSpec{Segment: 5})

		catalog.saveStorageConfig(t, catalog.StorageID)
		catalog.App.RunJobPass(t, "backups_physical_cleaner")

		catalog.assertWalGone(t, orphan.ID)
		catalog.assertWalPresent(t, anchored.ID)
	})
	t.Run("keeps the full boundary segment when the full starts inside it", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		catalog.seedCascadeFull(t, cascadeFullSpec{Segment: 4, Offset: 0x28})
		boundary := catalog.seedCascadeWal(t, cascadeWalSpec{Segment: 4})
		catalog.saveStorageConfig(t, catalog.StorageID)
		catalog.App.RunJobPass(t, "backups_physical_cleaner")
		catalog.assertWalPresent(t, boundary.ID)
	})
	t.Run("reports both WAL gaps when two missing ranges separate three runs", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		catalog.seedCascadeFull(t, cascadeFullSpec{})
		for _, segment := range []int{1, 4, 7} {
			catalog.seedCascadeWal(t, cascadeWalSpec{Segment: segment})
		}
		page := getBackups(t, catalog, "?type=WAL")
		require.Len(t, page.Backups, 3)
		assertWalGap(t, page, 2, 4)
		assertWalGap(t, page, 5, 7)
	})
	t.Run("orders inactive chains by the latest incremental when there is no WAL", func(t *testing.T) {
		t.Parallel()

		var catalog *catalogFixture
		cleaner := newCleanerCatalog(t)
		catalog = cleaner.catalogFixture
		configureRetentionSchedule(&cleaner.Config)
		cleaner.Config.Retention.ChainsCount = 1

		older := cleaner.seedChainFull(t, chainFullSpec{StartSegment: 0, AgeHours: 40})
		newer := cleaner.seedChainFull(t, chainFullSpec{StartSegment: 4, AgeHours: 20})
		cleaner.seedChainFull(t, chainFullSpec{StartSegment: 8, AgeHours: 0})
		incremental := catalog.seedCascadeIncremental(t, older.ID, 1)
		catalog.App.ExecuteSQL(
			t,
			"UPDATE backups_physical SET completed_at=? WHERE id=?",
			time.Now().UTC().Add(-60*time.Hour),
			incremental.ID,
		)

		cleaner.clean(t, apiclient.PhysicalRetentionPolicyCHAINS)
		catalog.assertFullPresent(t, older.ID)
		catalog.assertFullGone(t, newer.ID)
	})
	t.Run("refuses to extend an older chain when the newest chain is broken", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		older := catalog.seedCascadeFull(t, cascadeFullSpec{Age: time.Hour})
		newer := catalog.seedCascadeFull(t, cascadeFullSpec{Segment: 4})
		incremental := catalog.seedCascadeIncremental(t, newer.ID, 5)

		catalog.App.ExecuteSQL(
			t,
			"UPDATE backups_physical SET status='CHAIN_BROKEN',error_reason='SUMMARIES_EXPIRED' WHERE id=?",
			incremental.ID,
		)

		catalog.triggerIncremental(t, http.StatusConflict)
		page := getBackups(t, catalog, "?type=FULL")
		require.Len(t, page.Backups, 2)
		assert.Equal(t, newer.ID, page.Backups[0].ID)
		assert.Equal(t, older.ID, page.Backups[1].ID)
	})
	t.Run("keeps the middle chain boundary segments when orphan cleanup examines three chains", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		for _, segment := range []int{0, 4, 8} {
			catalog.seedCascadeFull(t, cascadeFullSpec{Segment: segment, Offset: 0x28})
		}

		segments := catalog.insertContiguousWal(
			t,
			contiguousWalSpec{
				TimelineID:      1,
				FirstSegmentNo:  0,
				Count:           10,
				FirstReceivedAt: time.Now().UTC().Add(-time.Hour),
			},
		)

		catalog.saveStorageConfig(t, catalog.StorageID)
		catalog.App.RunJobPass(t, "backups_physical_cleaner")

		for _, wal := range segments {
			catalog.assertWalPresent(t, wal.ID)
		}
	})
	t.Run("deletes the segment ending at a chain boundary only with its owning predecessor", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		predecessor := catalog.seedCascadeFull(t, cascadeFullSpec{})
		successor := catalog.seedCascadeFull(t, cascadeFullSpec{Segment: 4})
		before := catalog.seedCascadeWal(t, cascadeWalSpec{Segment: 3})
		after := catalog.seedCascadeWal(t, cascadeWalSpec{Segment: 4})

		catalog.deleteFull(t, predecessor.ID)
		catalog.assertWalGone(t, before.ID)
		catalog.assertWalPresent(t, after.ID)
		catalog.assertFullPresent(t, successor.ID)
	})
	t.Run("keeps matching segments from another timeline and database when deleting a chain", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		full := catalog.seedCascadeFull(t, cascadeFullSpec{Segment: 4, Offset: 0x28})
		segments := catalog.insertContiguousWal(
			t,
			contiguousWalSpec{
				TimelineID:      1,
				FirstSegmentNo:  3,
				Count:           3,
				FirstReceivedAt: time.Now().UTC().Add(-time.Hour),
			},
		)
		otherTimeline := catalog.seedCascadeWal(t, cascadeWalSpec{Segment: 4, TimelineID: 2})
		otherDatabase := catalog.App.InsertPhysicalDatabase(t, catalog.WorkspaceID)
		foreign := catalog.insertWal(
			t,
			newWalSeed(
				walSeedSpec{
					DatabaseID:  otherDatabase,
					StorageID:   catalog.StorageID,
					TimelineID:  1,
					WalFilename: fmtWalName(1, 4),
					StartLSN:    catalogLSN(4) + 0x14,
					EndLSN:      catalogLSN(5) + 0x14,
				},
			),
		)

		catalog.deleteFull(t, full.ID)
		catalog.assertWalPresent(t, otherTimeline.ID)
		catalog.assertWalPresent(t, foreign.ID)
		catalog.assertWalPresent(t, segments[1].ID)
		catalog.assertWalGone(t, segments[2].ID)
		catalog.assertWalPresent(t, segments[0].ID)
	})
}

func (catalog *catalogFixture) triggerIncremental(t *testing.T, status int) {
	t.Helper()
	catalog.call(
		t,
		catalogCall{
			Request:       apptest.PhysicalStartRequest(catalog.DatabaseID, "INCREMENTAL"),
			Authorization: "Bearer " + catalog.Owner.Token,
			StatusCode:    status,
		},
	)
}

func assertWalGap(t *testing.T, page backupPage, endBeforeGap, startAfterGap int) {
	t.Helper()
	hasBefore := false
	hasAfter := false
	for _, wal := range page.Backups {
		if wal.StopLSN == catalogLSN(endBeforeGap).String() {
			hasBefore = true
		}
		if wal.StartLSN == catalogLSN(startAfterGap).String() {
			hasAfter = wal.HasGapBefore
		}
	}
	assert.True(t, hasBefore)
	assert.True(t, hasAfter)
}
