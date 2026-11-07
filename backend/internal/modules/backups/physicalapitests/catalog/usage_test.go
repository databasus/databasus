package catalog_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestPhysicalBackupUsage(t *testing.T) {
	t.Parallel()

	t.Run("returns separate database totals when two physical databases share a storage", func(t *testing.T) {
		t.Parallel()
		catalog := newCatalog(t)
		otherCatalog := *catalog
		otherCatalog.DatabaseID = createSecondPhysicalDatabase(t, catalog).ID
		full := catalog.seedCascadeFull(t, cascadeFullSpec{})
		otherFull := otherCatalog.seedCascadeFull(t, cascadeFullSpec{})
		catalog.seedCascadeWal(t, cascadeWalSpec{Segment: 1})
		otherCatalog.seedCascadeWal(t, cascadeWalSpec{Segment: 1})

		catalog.App.ExecuteSQL(t, "UPDATE backups_physical SET backup_size_mb = 100 WHERE id = ?", full.ID)
		catalog.App.ExecuteSQL(t, "UPDATE backups_physical SET backup_size_mb = 300 WHERE id = ?", otherFull.ID)
		catalog.App.ExecuteSQL(
			t,
			"UPDATE backups_physical_wal_segments SET compressed_size_mb = 4 WHERE storage_id = ?",
			catalog.StorageID,
		)

		assert.InDelta(t, 104.0, getBackups(t, catalog, "").TotalUsageMb, 0.000001)
		assert.InDelta(t, 304.0, getBackups(t, &otherCatalog, "").TotalUsageMb, 0.000001)

		storageUsage, err := catalog.Owner.Client.GetStorageBackupUsageWithResponse(t.Context(), catalog.StorageID)
		require.NoError(t, err)
		require.NotNil(t, storageUsage.JSON200, "%s", storageUsage.Body)
		assert.InDelta(t, 408.0, storageUsage.JSON200.TotalUsageMb, 0.000001)
	})

	t.Run("excludes deleted catalog records before cleanup while counting shared WAL once", func(t *testing.T) {
		t.Parallel()
		catalog := newCatalog(t)
		predecessor := catalog.seedCascadeFull(t, cascadeFullSpec{Age: time.Hour})
		successor := catalog.seedCascadeFull(t, cascadeFullSpec{Segment: 2, Age: 55 * time.Minute})
		incremental := catalog.seedCascadeIncremental(t, successor.ID, 3)

		for _, backup := range []struct {
			ID     uuid.UUID
			SizeMb float64
		}{
			{ID: predecessor.ID, SizeMb: 100},
			{ID: successor.ID, SizeMb: 50},
			{ID: incremental.ID, SizeMb: 30},
		} {
			catalog.App.ExecuteSQL(
				t,
				"UPDATE backups_physical SET backup_size_mb = ? WHERE id = ?",
				backup.SizeMb,
				backup.ID,
			)
		}

		deletedNames := []string{
			*successor.FileName, *successor.ManifestFileName, *successor.FileName + ".metadata",
			*incremental.FileName, *incremental.ManifestFileName, *incremental.FileName + ".metadata",
		}
		catalog.publishNames(t, deletedNames...)

		segments := make([]*apptest.PhysicalWalSegmentSeed, 0, 3)
		for segment := 1; segment <= 3; segment++ {
			wal := catalog.seedCascadeWal(
				t,
				cascadeWalSpec{Segment: segment, Age: time.Hour - time.Duration(4+(segment-1)*3)*time.Minute},
			)
			catalog.publishNames(t, *wal.FileName)
			segments = append(segments, wal)
		}
		catalog.App.ExecuteSQL(
			t,
			"UPDATE backups_physical_wal_segments SET compressed_size_mb = 4 WHERE database_id = ?",
			catalog.DatabaseID,
		)

		assertUsageTotals(t, catalog, 192)

		catalog.deleteFull(t, successor.ID)

		catalog.assertFullGone(t, successor.ID)
		catalog.assertIncrementalGone(t, incremental.ID)
		catalog.assertFullPresent(t, predecessor.ID)
		assertUsageTotals(t, catalog, 112)

		pending := catalog.pendingFiles(t, catalog.StorageID)
		assert.ElementsMatch(t, deletedNames, pending)

		for _, name := range deletedNames {
			assert.True(t, catalog.App.IsLocalStorageFilePresent(t, name))
		}

		for _, wal := range segments {
			catalog.assertWalPresent(t, wal.ID)
			assert.True(t, catalog.App.IsLocalStorageFilePresent(t, *wal.FileName))
			assert.NotContains(t, pending, *wal.FileName)
		}
	})

	t.Run("returns eligible stored sizes across filtered pages without counting claimed WAL", func(t *testing.T) {
		t.Parallel()
		catalog := newCatalog(t)
		full, incremental, wal := seedOneOfEachType(t, catalog)
		catalog.App.ExecuteSQL(
			t,
			"UPDATE backups_physical SET backup_size_mb = 40 WHERE id IN (?, ?)",
			full.ID,
			incremental.ID,
		)
		catalog.App.ExecuteSQL(
			t,
			"UPDATE backups_physical_wal_segments SET compressed_size_mb = 4 WHERE id = ?",
			wal.ID,
		)
		claimed := newWalSeed(
			walSeedSpec{
				DatabaseID:  catalog.DatabaseID,
				StorageID:   catalog.StorageID,
				TimelineID:  1,
				WalFilename: "000000010000000000000003",
				StartLSN:    fixtureLSN(3 * segmentBytes),
				EndLSN:      fixtureLSN(4 * segmentBytes),
			},
		)
		claimed.FileName = nil
		claimed.CompressedSizeMb = 900
		catalog.insertWal(t, claimed)
		params := apiclient.ListPhysicalBackupsParams{
			DatabaseId: catalog.DatabaseID,
			Type:       &[]apiclient.PhysicalBackupListRowType{apiclient.PhysicalBackupListRowTypeFULL},
			Status:     nil, CreatedBefore: nil, Limit: new(1), After: nil,
		}

		filtered, err := catalog.Owner.Client.ListPhysicalBackupsWithResponse(t.Context(), &params)
		require.NoError(t, err)
		require.NotNil(t, filtered.JSON200, "%s", filtered.Body)
		require.Len(t, filtered.JSON200.Items, 1)
		assert.InDelta(t, 84.0, filtered.JSON200.TotalUsageMb, 0.000001)

		params.Type = nil

		firstPage, err := catalog.Owner.Client.ListPhysicalBackupsWithResponse(t.Context(), &params)
		require.NoError(t, err)
		require.NotNil(t, firstPage.JSON200, "%s", firstPage.Body)
		require.NotNil(t, firstPage.JSON200.NextCursor)

		params.After = firstPage.JSON200.NextCursor

		secondPage, err := catalog.Owner.Client.ListPhysicalBackupsWithResponse(t.Context(), &params)
		require.NoError(t, err)
		require.NotNil(t, secondPage.JSON200, "%s", secondPage.Body)
		assert.InDelta(t, 84.0, secondPage.JSON200.TotalUsageMb, 0.000001)

		catalog.App.ExecuteSQL(t, "UPDATE backups_physical SET status = 'ERROR' WHERE id = ?", incremental.ID)

		afterFailure, err := catalog.Owner.Client.ListPhysicalBackupsWithResponse(t.Context(), &params)
		require.NoError(t, err)
		require.NotNil(t, afterFailure.JSON200, "%s", afterFailure.Body)
		assert.InDelta(t, 44.0, afterFailure.JSON200.TotalUsageMb, 0.000001)
	})

	for _, status := range []apiclient.PhysicalBackupStatus{
		apiclient.PhysicalBackupStatusERROR,
		apiclient.PhysicalBackupStatusINPROGRESS,
		apiclient.PhysicalBackupStatusCANCELED,
		apiclient.PhysicalBackupStatusCHAINBROKEN,
	} {
		for _, backupKind := range []string{"full", "incremental"} {
			t.Run(
				"excludes noncompleted stored sizes with "+string(status)+" "+backupKind+" backups",
				func(t *testing.T) {
					t.Parallel()
					catalog := newCatalog(t)
					completed := catalog.seedCascadeFull(t, cascadeFullSpec{})
					catalog.seedCascadeWal(t, cascadeWalSpec{Segment: 1})
					var excludedBackupID uuid.UUID

					if backupKind == "full" {
						excludedBackupID = catalog.seedCascadeFull(t, cascadeFullSpec{Segment: 10}).ID
					} else {
						excludedBackupID = catalog.seedCascadeIncremental(t, completed.ID, 10).ID
					}

					catalog.App.ExecuteSQL(
						t,
						"UPDATE backups_physical SET backup_size_mb = 100 WHERE id = ?",
						completed.ID,
					)
					catalog.App.ExecuteSQL(
						t,
						"UPDATE backups_physical_wal_segments SET compressed_size_mb = 4 WHERE database_id = ?",
						catalog.DatabaseID,
					)
					catalog.App.ExecuteSQL(
						t,
						"UPDATE backups_physical SET status = ?, backup_size_mb = 900 WHERE id = ?",
						status,
						excludedBackupID,
					)

					assertUsageTotals(t, catalog, 104)
				},
			)
		}
	}
}

func assertUsageTotals(t *testing.T, catalog *catalogFixture, totalUsageMb float64) {
	t.Helper()
	page := getBackups(t, catalog, "")
	assert.InDelta(t, totalUsageMb, page.TotalUsageMb, 0.000001)

	storageUsage, err := catalog.Owner.Client.GetStorageBackupUsageWithResponse(t.Context(), catalog.StorageID)
	require.NoError(t, err)
	require.NotNil(t, storageUsage.JSON200, "%s", storageUsage.Body)
	assert.InDelta(t, totalUsageMb, storageUsage.JSON200.TotalUsageMb, 0.000001)
}
