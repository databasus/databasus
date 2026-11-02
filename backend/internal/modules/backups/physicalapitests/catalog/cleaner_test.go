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

func TestCleanPhysicalBackups(t *testing.T) {
	t.Parallel()

	t.Run("keeps the newest inactive chains and the active chain when chain retention runs", func(t *testing.T) {
		t.Parallel()

		catalog := newCleanerCatalog(t)
		configureRetentionSchedule(&catalog.Config)
		catalog.Config.Retention.ChainsCount = 2

		active := catalog.seedChainFull(t, chainFullSpec{StartSegment: 8, AgeHours: 0}) // newest → extendable head
		keptA := catalog.seedChainFull(t, chainFullSpec{StartSegment: 6, AgeHours: 3})  // non-extendable, kept by count
		keptB := catalog.seedChainFull(t, chainFullSpec{StartSegment: 4, AgeHours: 6})  // non-extendable, kept by count
		oldA := catalog.seedChainFull(t, chainFullSpec{StartSegment: 2, AgeHours: 9})   // non-extendable, evicted
		oldB := catalog.seedChainFull(t, chainFullSpec{StartSegment: 0, AgeHours: 12})  // non-extendable, evicted

		catalog.clean(t, "CHAINS")

		assert.True(t, catalog.fullExists(t, active.ID), "active chain is never a candidate")
		assert.True(t, catalog.fullExists(t, keptA.ID))
		assert.True(t, catalog.fullExists(t, keptB.ID))
		assert.False(t, catalog.fullExists(t, oldA.ID))
		assert.False(t, catalog.fullExists(t, oldB.ID))
	})
	t.Run("keeps a recent chain beyond the count when the grace period has not elapsed", func(t *testing.T) {
		t.Parallel()

		catalog := newCleanerCatalog(t)
		catalog.hasRecentChains = true
		configureRetentionSchedule(&catalog.Config)
		catalog.Config.Retention.ChainsCount = 1

		active := catalog.seedChainFull(t, chainFullSpec{StartSegment: 6, AgeHours: 0})
		recentKept := catalog.seedChainFull(
			t,
			chainFullSpec{StartSegment: 4, AgeHours: 0},
		) // within grace, kept by count anyway
		recentBeyond := catalog.seedChainFull(
			t,
			chainFullSpec{StartSegment: 2, AgeHours: 1},
		) // within grace, beyond count → grace saves it

		catalog.clean(t, "CHAINS")

		assert.True(t, catalog.fullExists(t, active.ID))
		assert.True(t, catalog.fullExists(t, recentKept.ID))
		assert.True(t, catalog.fullExists(t, recentBeyond.ID), "a chain younger than the grace period is never evicted")
	})
	t.Run("keeps the newest fulls and removes their dependents when full retention runs", func(t *testing.T) {
		t.Parallel()

		catalog := newCleanerCatalog(t)
		configureRetentionSchedule(&catalog.Config)
		catalog.Config.Retention.FullBackups = &apiclient.FullBackupsRetention{
			Policy: apiclient.FullBackupsRetentionPolicyLASTN,
			Count:  2,
		}

		active := catalog.seedChainFull(t, chainFullSpec{StartSegment: 9, AgeHours: 0})
		keptFull := catalog.seedChainFull(
			t,
			chainFullSpec{StartSegment: 6, AgeHours: 3},
		) // non-extendable but in newest-2 fulls
		droppedFull := catalog.seedChainFull(t, chainFullSpec{StartSegment: 3, AgeHours: 6})

		threeHoursAgo := time.Now().UTC().Add(-51 * time.Hour)

		incrModel := newIncrementalSeed(
			incrementalSeedSpec{
				DatabaseID:                catalog.DatabaseID,
				StorageID:                 catalog.StorageID,
				RootFullBackupID:          keptFull.ID,
				ParentIncrementalBackupID: nil,
				TimelineID:                1,
				StartLSN:                  cleanerLSN(7),
				StopLSN:                   cleanerLSN(8),
			},
		)
		incrModel.CreatedAt = threeHoursAgo
		incrModel.CompletedAt = &threeHoursAgo
		keptIncr := catalog.insertIncremental(t, incrModel)

		walModel := newWalSeed(
			walSeedSpec{
				DatabaseID:  catalog.DatabaseID,
				StorageID:   catalog.StorageID,
				TimelineID:  1,
				WalFilename: "000000010000000000000007",
				StartLSN:    cleanerLSN(7),
				EndLSN:      cleanerLSN(8),
			},
		)
		walModel.ReceivedAt = threeHoursAgo
		keptWal := catalog.insertWal(t, walModel)

		catalog.clean(t, "FULL_BACKUPS")

		assert.True(t, catalog.fullExists(t, active.ID))
		assert.True(t, catalog.fullExists(t, keptFull.ID), "newest-N full kept")
		assert.False(t, catalog.incrementalExists(t, keptIncr.ID), "kept full's incr dropped")
		assert.False(t, catalog.walExists(t, keptWal.ID), "kept full's wal dropped")
		assert.False(t, catalog.fullExists(t, droppedFull.ID), "non-kept chain deleted entirely")
	})
	t.Run("keeps hourly bucket representatives when GFS retention runs", func(t *testing.T) {
		t.Parallel()

		catalog := newCleanerCatalog(t)
		configureRetentionSchedule(&catalog.Config)
		catalog.Config.Retention.FullBackups = &apiclient.FullBackupsRetention{
			Policy:   apiclient.FullBackupsRetentionPolicyGFS,
			GfsHours: 2,
		}

		active := catalog.seedChainFull(t, chainFullSpec{StartSegment: 9, AgeHours: 0})
		keptHour := catalog.seedChainFull(t, chainFullSpec{StartSegment: 6, AgeHours: 1})
		droppedA := catalog.seedChainFull(t, chainFullSpec{StartSegment: 3, AgeHours: 3})
		droppedB := catalog.seedChainFull(t, chainFullSpec{StartSegment: 0, AgeHours: 4})

		catalog.clean(t, "FULL_BACKUPS")

		assert.True(t, catalog.fullExists(t, active.ID))
		assert.True(t, catalog.fullExists(t, keptHour.ID), "GFS keeps the 2nd-newest hourly bucket")
		assert.False(t, catalog.fullExists(t, droppedA.ID))
		assert.False(t, catalog.fullExists(t, droppedB.ID))
	})
	t.Run("keeps the union of fulls and chains when combined retention runs", func(t *testing.T) {
		t.Parallel()

		catalog := newCleanerCatalog(t)
		configureRetentionSchedule(&catalog.Config)
		catalog.Config.Retention.ChainsCount = 1
		catalog.Config.Retention.FullBackups = &apiclient.FullBackupsRetention{
			Policy: apiclient.FullBackupsRetentionPolicyLASTN,
			Count:  3,
		}

		active := catalog.seedChainFull(t, chainFullSpec{StartSegment: 9, AgeHours: 0})
		keptByChains := catalog.seedChainFull(
			t,
			chainFullSpec{StartSegment: 6, AgeHours: 3},
		) // newest non-ext → CHAINS keeps it
		keptByFulls := catalog.seedChainFull(
			t,
			chainFullSpec{StartSegment: 3, AgeHours: 6},
		) // 3rd-newest full → FULL_BACKUPS keeps it
		dropped := catalog.seedChainFull(t, chainFullSpec{StartSegment: 0, AgeHours: 9}) // in neither keep-set

		catalog.clean(t, "CHAINS_AND_FULL_BACKUPS")

		assert.True(t, catalog.fullExists(t, active.ID))
		assert.True(t, catalog.fullExists(t, keptByChains.ID), "kept by CHAINS policy")
		assert.True(t, catalog.fullExists(t, keptByFulls.ID), "kept by FULL_BACKUPS policy even though CHAINS count=1")
		assert.False(t, catalog.fullExists(t, dropped.ID))
	})
	t.Run("removes WAL when no chain covers it", func(t *testing.T) {
		t.Parallel()

		catalog := newCleanerCatalog(t)

		orphan := catalog.insertWal(
			t,
			newWalSeed(
				walSeedSpec{
					DatabaseID:  catalog.DatabaseID,
					StorageID:   catalog.StorageID,
					TimelineID:  1,
					WalFilename: "000000010000000000000005",
					StartLSN:    cleanerLSN(5),
					EndLSN:      cleanerLSN(6),
				},
			),
		)

		catalog.clean(t, "CHAINS")

		assert.False(t, catalog.walExists(t, orphan.ID), "WAL not covered by any chain is deleted")
	})
	t.Run("keeps WAL when a completed full covers it", func(t *testing.T) {
		t.Parallel()

		catalog := newCleanerCatalog(t)

		catalog.seedChainFull(t, chainFullSpec{StartSegment: 1, AgeHours: 0})
		covered := catalog.insertWal(
			t,
			newWalSeed(
				walSeedSpec{
					DatabaseID:  catalog.DatabaseID,
					StorageID:   catalog.StorageID,
					TimelineID:  1,
					WalFilename: "000000010000000000000002",
					StartLSN:    cleanerLSN(2),
					EndLSN:      cleanerLSN(3),
				},
			),
		)

		catalog.clean(t, "CHAINS")

		assert.True(t, catalog.walExists(t, covered.ID), "chain-covered WAL must never be caught by the orphan pass")
	})
	t.Run("keeps archived WAL when the first full is running", func(t *testing.T) {
		t.Parallel()

		catalog := newCleanerCatalog(t)

		catalog.insertFull(
			t,
			newInProgressFull(
				inProgressFullSpec{DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID, TimelineID: 1},
			),
		)

		walArchivedDuringBackup := catalog.insertWal(
			t,
			newWalSeed(
				walSeedSpec{
					DatabaseID:  catalog.DatabaseID,
					StorageID:   catalog.StorageID,
					TimelineID:  1,
					WalFilename: "000000010000000000000001",
					StartLSN:    cleanerLSN(1),
					EndLSN:      cleanerLSN(2),
				},
			),
		)

		catalog.clean(t, "CHAINS")

		assert.True(t, catalog.walExists(t, walArchivedDuringBackup.ID),
			"WAL archived while the first full is running must remain available when the full completes")
	})

	t.Run("keeps the boundary segment when a full starts inside it", func(t *testing.T) {
		t.Parallel()

		catalog := newCleanerCatalog(t)

		catalog.insertFull(
			t,
			newFullSeed(
				fullSeedSpec{
					DatabaseID: catalog.DatabaseID,
					StorageID:  catalog.StorageID,
					TimelineID: 1,
					StartLSN:   cleanerLSN(1) + 0x28,
					StopLSN:    cleanerLSN(1) + 0x28 + 0x100,
				},
			),
		)

		boundarySegment := catalog.insertWal(
			t,
			newWalSeed(
				walSeedSpec{
					DatabaseID:  catalog.DatabaseID,
					StorageID:   catalog.StorageID,
					TimelineID:  1,
					WalFilename: "000000010000000000000001",
					StartLSN:    cleanerLSN(1),
					EndLSN:      cleanerLSN(2),
				},
			),
		)

		catalog.clean(t, "CHAINS")

		assert.True(t, catalog.walExists(t, boundarySegment.ID),
			"the segment holding the FULL's start_lsn and stop_lsn must survive the orphan sweep")
	})
	t.Run("removes WAL ending below the first retained full when the orphan pass runs", func(t *testing.T) {
		t.Parallel()

		catalog := newCleanerCatalog(t)

		catalog.insertFull(
			t,
			newFullSeed(
				fullSeedSpec{
					DatabaseID: catalog.DatabaseID,
					StorageID:  catalog.StorageID,
					TimelineID: 1,
					StartLSN:   cleanerLSN(4) + 0x28,
					StopLSN:    cleanerLSN(4) + 0x28 + 0x100,
				},
			),
		)

		staleSegment := catalog.insertWal(
			t,
			newWalSeed(
				walSeedSpec{
					DatabaseID:  catalog.DatabaseID,
					StorageID:   catalog.StorageID,
					TimelineID:  1,
					WalFilename: "000000010000000000000001",
					StartLSN:    cleanerLSN(1),
					EndLSN:      cleanerLSN(2),
				},
			),
		)

		catalog.clean(t, "CHAINS")

		assert.False(t, catalog.walExists(t, staleSegment.ID),
			"WAL that ends below the earliest retained FULL's start is still reclaimable")
	})
	t.Run("removes an abandoned WAL claim when it is older than the grace period", func(t *testing.T) {
		t.Parallel()

		catalog := newCleanerCatalog(t)

		catalog.seedChainFull(t, chainFullSpec{StartSegment: 0})
		twoHoursAgo := time.Now().UTC().Add(-2 * time.Hour)
		abandoned := &apptest.PhysicalWalSegmentSeed{
			DatabaseID:  catalog.DatabaseID,
			StorageID:   catalog.StorageID,
			TimelineID:  1,
			FileName:    nil,
			WalFilename: "000000010000000000000009",
			StartLSN:    cleanerLSN(9).String(),
			EndLSN:      cleanerLSN(10).String(),
			ReceivedAt:  twoHoursAgo,
			ClaimedAt:   twoHoursAgo,
		}
		catalog.insertWal(t, abandoned)

		catalog.clean(t, "CHAINS")

		assert.False(
			t,
			catalog.walExists(t, abandoned.ID),
			"an abandoned NULL-file_name claim past grace is reaped (no storage I/O)",
		)
	})
	t.Run("keeps an uncommitted WAL claim when it is inside the grace period", func(t *testing.T) {
		t.Parallel()

		catalog := newCleanerCatalog(t)

		catalog.seedChainFull(t, chainFullSpec{StartSegment: 0})
		thirtyMinutesAgo := time.Now().UTC().Add(-30 * time.Minute)
		freshClaim := &apptest.PhysicalWalSegmentSeed{
			DatabaseID:  catalog.DatabaseID,
			StorageID:   catalog.StorageID,
			TimelineID:  1,
			FileName:    nil,
			WalFilename: "000000010000000000000009",
			StartLSN:    cleanerLSN(9).String(),
			EndLSN:      cleanerLSN(10).String(),
			ReceivedAt:  thirtyMinutesAgo,
			ClaimedAt:   thirtyMinutesAgo,
		}
		catalog.insertWal(t, freshClaim)

		catalog.clean(t, "CHAINS")

		assert.True(t, catalog.walExists(t, freshClaim.ID), "an active WAL upload within grace must survive")
	})
	t.Run("keeps the running full while evicting expired completed chains", func(t *testing.T) {
		t.Parallel()

		catalog := newCleanerCatalog(t)
		configureRetentionSchedule(&catalog.Config)
		catalog.Config.Retention.ChainsCount = 1

		inProgress := catalog.seedFullWithStatusAndAge(
			t,
			agedFullSpec{Status: "IN_PROGRESS", StartSegment: 9, AgeHours: 1},
		)
		active := catalog.seedChainFull(
			t,
			chainFullSpec{StartSegment: 6, AgeHours: 3},
		) // newest COMPLETED → extendable head
		keptByCount := catalog.seedChainFull(t, chainFullSpec{StartSegment: 4, AgeHours: 6})
		evicted := catalog.seedChainFull(t, chainFullSpec{StartSegment: 2, AgeHours: 9})

		catalog.clean(t, "CHAINS")

		assert.True(t, catalog.fullExists(t, inProgress.ID), "an IN_PROGRESS full is never a retention candidate")
		assert.True(t, catalog.fullExists(t, active.ID))
		assert.True(t, catalog.fullExists(t, keptByCount.ID))
		assert.False(t, catalog.fullExists(t, evicted.ID), "completed chains beyond the keep-count are still evicted")
	})
	t.Run("deletes only expired samples when the cleaner pass runs", func(t *testing.T) {
		t.Parallel()

		catalog := newCleanerCatalog(t)
		now := time.Now().UTC()

		expiredSample := catalog.insertCleanerSample(t,
			newCleanerSample(catalog.DatabaseID, now.Add(-7*24*time.Hour-time.Hour)))
		recentSample := catalog.insertCleanerSample(t,
			newCleanerSample(catalog.DatabaseID, now.Add(-24*time.Hour)))

		catalog.clean(t, "CHAINS")

		var remainingSamples []struct{ ID uuid.UUID }
		catalog.App.QuerySQL(
			t,
			&remainingSamples,
			"SELECT id FROM backups_physical_wal_stream_samples WHERE database_id = ? ORDER BY sampled_at",
			catalog.DatabaseID,
		)
		require.Len(t, remainingSamples, 1)
		require.Equal(t, recentSample.ID, remainingSamples[0].ID)
		require.NotEqual(t, expiredSample.ID, remainingSamples[0].ID)
	})
	t.Run("deletes every expired sample when the database stopped streaming eight days ago", func(t *testing.T) {
		t.Parallel()

		catalog := newCleanerCatalog(t)
		now := time.Now().UTC()
		stoppedStreamingAt := now.Add(-8 * 24 * time.Hour)

		for sampleIndex := range 3 {
			catalog.insertCleanerSample(t, newCleanerSample(
				catalog.DatabaseID, stoppedStreamingAt.Add(-time.Duration(sampleIndex)*30*time.Second)))
		}

		catalog.clean(t, "CHAINS")

		var remainingSamples []struct{ ID uuid.UUID }
		catalog.App.QuerySQL(
			t,
			&remainingSamples,
			"SELECT id FROM backups_physical_wal_stream_samples WHERE database_id = ? ORDER BY sampled_at",
			catalog.DatabaseID,
		)
		require.Empty(t, remainingSamples, "samples expire whether or not the database still streams")
	})
}
