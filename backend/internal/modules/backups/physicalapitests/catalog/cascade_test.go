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

func TestCascadePhysicalBackup(t *testing.T) {
	t.Parallel()

	t.Run("deletes all chain rows and stored objects while another timeline survives", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		full := catalog.seedCascadeFull(t, cascadeFullSpec{})
		first := catalog.seedCascadeIncremental(t, full.ID, 1)
		second := catalog.seedCascadeIncremental(t, full.ID, 2)
		history := catalog.seedHistory(t, 1)
		successor := catalog.seedCascadeFull(t, cascadeFullSpec{Segment: 10, TimelineID: 2})

		names := []string{
			*full.FileName,
			*full.ManifestFileName,
			*full.FileName + ".metadata",
			history.FileName,
			history.FileName + ".metadata",
		}
		for _, incremental := range []*apptest.PhysicalIncrementalBackupSeed{first, second} {
			names = append(
				names,
				*incremental.FileName,
				*incremental.ManifestFileName,
				*incremental.FileName+".metadata",
			)
		}
		segments := []*apptest.PhysicalWalSegmentSeed{}
		for index := 1; index <= 3; index++ {
			wal := catalog.seedCascadeWal(t, cascadeWalSpec{Segment: index})
			segments = append(segments, wal)
			names = append(names, *wal.FileName, *wal.FileName+".metadata")
		}

		catalog.publishNames(t, names...)
		catalog.deleteFull(t, full.ID)

		catalog.assertFullGone(t, full.ID)
		catalog.assertIncrementalGone(t, first.ID)
		catalog.assertIncrementalGone(t, second.ID)
		for _, wal := range segments {
			catalog.assertWalGone(t, wal.ID)
		}
		catalog.assertHistoryPresent(t, history.ID, false)
		catalog.assertFilesDeleted(t, names...)
		catalog.assertFullPresent(t, successor.ID)
	})
	t.Run("keeps the next full boundary segment when deleting its predecessor", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		full := catalog.seedCascadeFull(t, cascadeFullSpec{})
		catalog.seedCascadeFull(t, cascadeFullSpec{Segment: 4, Offset: 0x28})
		owned := catalog.seedCascadeWal(t, cascadeWalSpec{Segment: 1})
		boundary := catalog.seedCascadeWal(t, cascadeWalSpec{Segment: 4})
		catalog.publishNames(t, *owned.FileName, *boundary.FileName)

		catalog.deleteFull(t, full.ID)

		catalog.assertWalGone(t, owned.ID)
		catalog.assertWalPresent(t, boundary.ID)
		assert.True(t, catalog.App.IsLocalStorageFilePresent(t, *boundary.FileName))
	})
	t.Run("keeps shared WAL restorable when an older full survives", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		predecessor := catalog.seedCascadeFull(t, cascadeFullSpec{Age: time.Hour})
		successor := catalog.seedCascadeFull(t, cascadeFullSpec{Segment: 2, Age: 55 * time.Minute})

		segments := []*apptest.PhysicalWalSegmentSeed{}
		for index := 1; index <= 3; index++ {
			wal := catalog.seedCascadeWal(
				t,
				cascadeWalSpec{Segment: index, Age: time.Hour - time.Duration(4+(index-1)*3)*time.Minute},
			)
			catalog.publishNames(t, *wal.FileName)
			segments = append(segments, wal)
		}

		catalog.deleteFull(t, successor.ID)
		for _, wal := range segments {
			catalog.assertWalPresent(t, wal.ID)
			assert.True(t, catalog.App.IsLocalStorageFilePresent(t, *wal.FileName))
		}
		catalog.assertFullPresent(t, predecessor.ID)

		var token struct {
			Token string `json:"token"`
		}
		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalPitrTokenRequest(catalog.DatabaseID, nil),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusCreated,
				Destination:   &token,
			},
		)
		assert.NotEmpty(t, token.Token)
	})
	t.Run("keeps WAL and history when another full is running", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		full := catalog.seedCascadeFull(t, cascadeFullSpec{})
		wal := catalog.seedCascadeWal(t, cascadeWalSpec{Segment: 1})
		history := catalog.seedHistory(t, 1)
		catalog.insertFull(
			t,
			newInProgressFull(
				inProgressFullSpec{DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID, TimelineID: 1},
			),
		)

		catalog.deleteFull(t, full.ID)
		catalog.assertWalPresent(t, wal.ID)
		catalog.assertHistoryPresent(t, history.ID, true)
	})
	t.Run("removes an incremental with no stored file when deleting its full", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		full := catalog.seedCascadeFull(t, cascadeFullSpec{})
		incremental := catalog.insertIncremental(
			t,
			&apptest.PhysicalIncrementalBackupSeed{
				DatabaseID:       catalog.DatabaseID,
				StorageID:        catalog.StorageID,
				RootFullBackupID: full.ID,
				TimelineID:       1,
				Status:           "CHAIN_BROKEN",
				ErrorReason:      new("SUMMARIZER_OFF"),
			},
		)

		catalog.deleteFull(t, full.ID)

		catalog.assertFullGone(t, full.ID)
		catalog.assertIncrementalGone(t, incremental.ID)
	})
	t.Run("deletes only part of the WAL within the cleaner budget and keeps its full", func(t *testing.T) {
		t.Parallel()

		var catalog *catalogFixture
		cleaner := newCleanerCatalog(t)
		catalog = cleaner.catalogFixture
		configureRetentionSchedule(&cleaner.Config)
		cleaner.Config.Retention.ChainsCount = 1

		full := cleaner.seedChainFull(t, chainFullSpec{StartSegment: 0, AgeHours: 20})
		cleaner.seedChainFull(t, chainFullSpec{StartSegment: 100, AgeHours: 10})
		active := cleaner.seedChainFull(t, chainFullSpec{StartSegment: 200, AgeHours: 0})
		catalog.App.ExecuteSQL(t, "UPDATE backups_physical SET backup_size_mb=1 WHERE id=?", active.ID)

		segments := []*apptest.PhysicalWalSegmentSeed{}
		for index := 1; index <= 5; index++ {
			wal := newWalSeed(
				walSeedSpec{
					DatabaseID:  catalog.DatabaseID,
					StorageID:   catalog.StorageID,
					TimelineID:  1,
					WalFilename: fmtWalName(1, index),
					StartLSN:    catalogLSN(index),
					EndLSN:      catalogLSN(index + 1),
				},
			)
			wal.CompressedSizeMb = 128
			wal.ReceivedAt = time.Now().UTC().Add(-80 * time.Hour)
			segments = append(segments, catalog.insertWal(t, wal))
		}

		cleaner.clean(t, apiclient.PhysicalRetentionPolicyCHAINS)

		catalog.assertFullPresent(t, full.ID)
		remaining := 0
		for _, wal := range segments {
			var count int64
			catalog.App.QuerySQL(t, &count, "SELECT count(*) FROM backups_physical_wal_segments WHERE id=?", wal.ID)
			remaining += int(count)
		}
		assert.Positive(t, remaining)
		assert.Less(t, remaining, 5)
	})
	t.Run("sheds dependent rows while keeping a retained full", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		full := catalog.seedCascadeFull(t, cascadeFullSpec{Age: 80 * time.Hour})
		catalog.seedCascadeFull(t, cascadeFullSpec{Segment: 10, Age: 60 * time.Hour})
		incremental := catalog.seedCascadeIncremental(t, full.ID, 1)
		wal := catalog.seedCascadeWal(t, cascadeWalSpec{Segment: 1, Age: 70 * time.Hour})
		catalog.App.ExecuteSQL(
			t,
			"UPDATE backups_physical SET created_at=?,completed_at=? WHERE id=?",
			time.Now().UTC().Add(-75*time.Hour),
			time.Now().UTC().Add(-74*time.Hour),
			incremental.ID,
		)

		catalog.keepFullPass(t)

		catalog.assertFullPresent(t, full.ID)
		catalog.assertIncrementalGone(t, incremental.ID)
		catalog.assertWalGone(t, wal.ID)
	})
	t.Run("hands every physical artifact to file cleanup after database deletion", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		full := catalog.seedCascadeFull(t, cascadeFullSpec{})
		history := catalog.seedHistory(t, 1)
		names := []string{*full.FileName, *full.ManifestFileName, *full.FileName + ".metadata", history.FileName}
		catalog.publishNames(t, names...)

		response, err := catalog.Owner.Client.DeleteDatabaseWithResponse(t.Context(), catalog.DatabaseID)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, response.StatusCode(), "%s", response.Body)
		catalog.assertFilesDeleted(t, names...)
	})
}
