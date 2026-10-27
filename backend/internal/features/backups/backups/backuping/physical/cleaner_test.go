package backuping_physical

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	physical_testing "databasus-backend/internal/features/backups/backups/core/physical/testing"
	"databasus-backend/internal/features/backups/backups/core/physical/wal_health"
	postgresql_executor "databasus-backend/internal/features/backups/backups/usecases/physical/postgresql"
	"databasus-backend/internal/lib/schedule"
	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/storage"
	"databasus-backend/internal/util/logger"
)

const (
	firstRecordOffset = physical_testing.FirstRecordOffset
	fullLSNSpan       = physical_testing.FullLSNSpan
)

// Test_RunStartupSlotCleanup_DropsOrphanSlot is the end-to-end proof of the
// "Databasus crashed mid-backup" path: the source keeps the per-backup slot and
// the in-flight claim survives the crash. In single-instance mode the backup that
// held the slot died with the process, so the startup sweep treats every claimed
// slot as a reclaimable orphan and drops it against the real source PG.
func Test_RunStartupSlotCleanup_DropsOrphanSlot(t *testing.T) {
	fixture := postgresql_executor.SetupPhysicalDBForBackup(t)
	cleaner := CreateTestPhysicalCleaner()

	slotName := fixture.SourceDB.BackupSlotName
	adminConn := postgresql_executor.OpenAdminConn(t, fixture)

	_, err := adminConn.Exec(context.Background(),
		"SELECT pg_create_physical_replication_slot($1, true)", slotName)
	require.NoError(t, err, "pre-create the orphan slot a crashed backup left behind")
	t.Cleanup(func() {
		_, _ = adminConn.Exec(context.Background(),
			`SELECT pg_drop_replication_slot(slot_name)
			   FROM pg_replication_slots WHERE slot_name = $1`, slotName)
	})

	require.NoError(t, physical_repositories.GetInFlightBackupRepository().Release(fixture.DB.ID))
	_, err = physical_repositories.GetInFlightBackupRepository().Claim(
		storage.GetDb(), physical_repositories.ClaimSpec{
			DatabaseID: fixture.DB.ID,
			BackupType: physical_enums.PhysicalBackupTypeFull,
			BackupID:   fixture.BackupID,
		})
	require.NoError(t, err)

	cleaner.runStartupSlotCleanup(t.Context())

	assert.False(t, postgresql_executor.SlotExists(t, adminConn, slotName),
		"a per-backup orphan slot must be dropped at startup in single-instance mode")
}

// seedChainFull seeds a COMPLETED FULL at a start segment and age so tests can
// build multiple distinct chains with deterministic ordering (higher segment =
// newer = the active head).
func seedChainFull(
	t *testing.T,
	prereqs *backupPrereqs,
	startSegment, ageHours int,
) *physical_models.PhysicalFullBackup {
	t.Helper()

	full := physical_testing.NewTestCompletedFullBackup(
		prereqs.DB.ID, prereqs.Storage.ID, 1, testLSN(startSegment), testLSN(startSegment+1))

	at := time.Now().UTC().Add(-time.Duration(ageHours) * time.Hour)
	full.CreatedAt = at
	full.CompletedAt = &at
	full.BackupSizeMb = new(1000.0)

	return physical_testing.CreateTestFullBackup(t, full)
}

// shortGraceConfig sets hourly cadences so the per-chain grace is 2 h — chains
// older than 2 h become evictable, which keeps retention tests deterministic.
func shortGraceConfig(backupConfig *backups.PhysicalBackupConfig) {
	backupConfig.FullBackupSchedule = schedule.Schedule{Type: schedule.TypeHourly}
	backupConfig.IncrementalBackupSchedule = &schedule.Schedule{Type: schedule.TypeHourly}
}

func fullExists(t *testing.T, id uuid.UUID) bool {
	t.Helper()

	full, err := physical_repositories.GetFullBackupRepository().FindByID(id)
	require.NoError(t, err)

	return full != nil
}

func incrementalExists(t *testing.T, id uuid.UUID) bool {
	t.Helper()

	incremental, err := physical_repositories.GetIncrementalBackupRepository().FindByID(id)
	require.NoError(t, err)

	return incremental != nil
}

func walExists(t *testing.T, id uuid.UUID) bool {
	t.Helper()

	segment, err := physical_repositories.GetWalSegmentRepository().FindByID(id)
	require.NoError(t, err)

	return segment != nil
}

func Test_CleanByChains_KeepsNLatestNonExtendableChains(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	shortGraceConfig(&prereqs.Config)
	prereqs.Config.Retention.ChainsCount = 2

	active := seedChainFull(t, prereqs, 8, 0) // newest → extendable head
	keptA := seedChainFull(t, prereqs, 6, 3)  // non-extendable, kept by count
	keptB := seedChainFull(t, prereqs, 4, 6)  // non-extendable, kept by count
	oldA := seedChainFull(t, prereqs, 2, 9)   // non-extendable, evicted
	oldB := seedChainFull(t, prereqs, 0, 12)  // non-extendable, evicted

	cleaner := CreateTestPhysicalCleaner()
	require.NoError(t, cleaner.cleanByChains(context.Background(), logger.GetLogger(), prereqs.Config))

	assert.True(t, fullExists(t, active.ID), "active chain is never a candidate")
	assert.True(t, fullExists(t, keptA.ID))
	assert.True(t, fullExists(t, keptB.ID))
	assert.False(t, fullExists(t, oldA.ID))
	assert.False(t, fullExists(t, oldB.ID))
}

func Test_CleanByChains_GracePeriodProtectsRecentChainBeyondKeepCount(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	shortGraceConfig(&prereqs.Config) // 2 h grace
	prereqs.Config.Retention.ChainsCount = 1

	active := seedChainFull(t, prereqs, 6, 0)
	recentKept := seedChainFull(t, prereqs, 4, 0)   // within grace, kept by count anyway
	recentBeyond := seedChainFull(t, prereqs, 2, 1) // within grace, beyond count → grace saves it

	cleaner := CreateTestPhysicalCleaner()
	require.NoError(t, cleaner.cleanByChains(context.Background(), logger.GetLogger(), prereqs.Config))

	assert.True(t, fullExists(t, active.ID))
	assert.True(t, fullExists(t, recentKept.ID))
	assert.True(t, fullExists(t, recentBeyond.ID), "a chain younger than the grace period is never evicted")
}

func Test_CleanByFullsLastN_KeepsNewestFullsAndDropsTheirIncrAndWal(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	shortGraceConfig(&prereqs.Config)
	prereqs.Config.Retention.FullBackups = backups.FullBackupsRetention{
		Policy: backups.FullBackupsRetentionPolicyLastN,
		Count:  2,
	}

	active := seedChainFull(t, prereqs, 9, 0)
	keptFull := seedChainFull(t, prereqs, 6, 3) // non-extendable but in newest-2 fulls
	droppedFull := seedChainFull(t, prereqs, 3, 6)

	// keptFull's chain gets an INCR + WAL that LAST_N must shed while keeping the
	// FULL. Age their timestamps to 3 h so the chain-end stays outside the grace
	// window (a fresh INCR/WAL would make the whole chain grace-protected).
	threeHoursAgo := time.Now().UTC().Add(-3 * time.Hour)

	incrModel := physical_testing.NewTestCompletedIncrementalBackup(
		prereqs.DB.ID, prereqs.Storage.ID, keptFull.ID, nil, 1, testLSN(7), testLSN(8))
	incrModel.CreatedAt = threeHoursAgo
	incrModel.CompletedAt = &threeHoursAgo
	keptIncr := physical_testing.CreateTestIncrementalBackup(t, incrModel)

	walModel := physical_testing.NewTestWalSegment(
		prereqs.DB.ID, prereqs.Storage.ID, 1, "000000010000000000000007", testLSN(7), testLSN(8))
	walModel.ReceivedAt = threeHoursAgo
	keptWal := physical_testing.CreateTestWalSegment(t, walModel)

	cleaner := CreateTestPhysicalCleaner()
	require.NoError(t, cleaner.cleanByFulls(context.Background(), logger.GetLogger(), prereqs.Config))

	assert.True(t, fullExists(t, active.ID))
	assert.True(t, fullExists(t, keptFull.ID), "newest-N full kept")
	assert.False(t, incrementalExists(t, keptIncr.ID), "kept full's incr dropped")
	assert.False(t, walExists(t, keptWal.ID), "kept full's wal dropped")
	assert.False(t, fullExists(t, droppedFull.ID), "non-kept chain deleted entirely")
}

func Test_CleanByFullsGfs_KeepsBucketRepresentativesDropsExtras(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	shortGraceConfig(&prereqs.Config)
	prereqs.Config.Retention.FullBackups = backups.FullBackupsRetention{
		Policy: backups.FullBackupsRetentionPolicyGfs,
		Gfs:    backups.GfsRetention{Hours: 2},
	}

	// 4 distinct hourly buckets; GFS keeps the 2 newest (active + age2).
	active := seedChainFull(t, prereqs, 9, 0)
	keptHour := seedChainFull(t, prereqs, 6, 2)
	droppedA := seedChainFull(t, prereqs, 3, 3)
	droppedB := seedChainFull(t, prereqs, 0, 4)

	cleaner := CreateTestPhysicalCleaner()
	require.NoError(t, cleaner.cleanByFulls(context.Background(), logger.GetLogger(), prereqs.Config))

	assert.True(t, fullExists(t, active.ID))
	assert.True(t, fullExists(t, keptHour.ID), "GFS keeps the 2nd-newest hourly bucket")
	assert.False(t, fullExists(t, droppedA.ID))
	assert.False(t, fullExists(t, droppedB.ID))
}

func Test_CleanByCombined_KeepsUnionOfChainsAndFulls(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	shortGraceConfig(&prereqs.Config)
	prereqs.Config.Retention.ChainsCount = 1
	prereqs.Config.Retention.FullBackups = backups.FullBackupsRetention{
		Policy: backups.FullBackupsRetentionPolicyLastN,
		Count:  3,
	}

	active := seedChainFull(t, prereqs, 9, 0)
	keptByChains := seedChainFull(t, prereqs, 6, 3) // newest non-ext → CHAINS keeps it
	keptByFulls := seedChainFull(t, prereqs, 3, 6)  // 3rd-newest full → FULL_BACKUPS keeps it
	dropped := seedChainFull(t, prereqs, 0, 9)      // in neither keep-set

	cleaner := CreateTestPhysicalCleaner()
	require.NoError(t, cleaner.cleanByCombined(context.Background(), logger.GetLogger(), prereqs.Config))

	assert.True(t, fullExists(t, active.ID))
	assert.True(t, fullExists(t, keptByChains.ID), "kept by CHAINS policy")
	assert.True(t, fullExists(t, keptByFulls.ID), "kept by FULL_BACKUPS policy even though CHAINS count=1")
	assert.False(t, fullExists(t, dropped.ID))
}

func Test_CleanOrphanWalForDatabase_WhenWalOutsideAllChains_DeletesOrphan(t *testing.T) {
	prereqs := seedBackupPrereqs(t)

	// No FULL covers this segment, so it is orphan WAL the cleaner must reclaim.
	orphan := physical_testing.CreateTestWalSegment(t, physical_testing.NewTestWalSegment(
		prereqs.DB.ID, prereqs.Storage.ID, 1, "000000010000000000000005", testLSN(5), testLSN(6)))

	cleaner := CreateTestPhysicalCleaner()
	cleaner.cleanOrphanWalForDatabase(context.Background(), logger.GetLogger(), prereqs.DB.ID)

	assert.False(t, walExists(t, orphan.ID), "WAL not covered by any chain is deleted")
}

func Test_CleanOrphanWalForDatabase_WhenWalCoveredByChain_KeepsIt(t *testing.T) {
	prereqs := seedBackupPrereqs(t)

	// A COMPLETED FULL at segment 1 covers everything from its start_lsn onward on
	// the same timeline, so the segment at 2 is in-chain, not an orphan.
	seedChainFull(t, prereqs, 1, 0)
	covered := physical_testing.CreateTestWalSegment(t, physical_testing.NewTestWalSegment(
		prereqs.DB.ID, prereqs.Storage.ID, 1, "000000010000000000000002", testLSN(2), testLSN(3)))

	cleaner := CreateTestPhysicalCleaner()
	cleaner.cleanOrphanWalForDatabase(context.Background(), logger.GetLogger(), prereqs.DB.ID)

	assert.True(t, walExists(t, covered.ID), "chain-covered WAL must never be caught by the orphan pass")
}

func Test_CleanOrphanWalForDatabase_WhenFirstFullInProgress_KeepsWalArchivedDuringBackup(t *testing.T) {
	prereqs := seedBackupPrereqs(t)

	inProgressFull := physical_testing.CreateTestFullBackup(t, physical_testing.NewTestInProgressFullBackup(
		prereqs.DB.ID,
		prereqs.Storage.ID,
		1,
	))
	physical_testing.CreateTestInFlightClaim(
		t,
		prereqs.DB.ID,
		inProgressFull.ID,
		physical_enums.PhysicalBackupTypeFull,
	)

	walArchivedDuringBackup := physical_testing.CreateTestWalSegment(t, physical_testing.NewTestWalSegment(
		prereqs.DB.ID,
		prereqs.Storage.ID,
		1,
		"000000010000000000000001",
		testLSN(1),
		testLSN(2),
	))

	cleaner := CreateTestPhysicalCleaner()
	cleaner.cleanOrphanWalForDatabase(t.Context(), logger.GetLogger(), prereqs.DB.ID)

	assert.True(t, walExists(t, walArchivedDuringBackup.ID),
		"WAL archived while the first full is running must remain available when the full completes")
}

func Test_CleanOrphanWalForDatabase_WhenInProgressFullHasNoClaim_DeletesOrphanWal(t *testing.T) {
	prereqs := seedBackupPrereqs(t)

	physical_testing.CreateTestFullBackup(t, physical_testing.NewTestInProgressFullBackup(
		prereqs.DB.ID,
		prereqs.Storage.ID,
		1,
	))
	orphanWal := physical_testing.CreateTestWalSegment(t, physical_testing.NewTestWalSegment(
		prereqs.DB.ID,
		prereqs.Storage.ID,
		1,
		"000000010000000000000001",
		testLSN(1),
		testLSN(2),
	))

	cleaner := CreateTestPhysicalCleaner()
	cleaner.cleanOrphanWalForDatabase(t.Context(), logger.GetLogger(), prereqs.DB.ID)

	assert.False(t, walExists(t, orphanWal.ID))
}

// A FULL taken right after a segment switch reports a start_lsn at
// firstRecordOffset. The segment's own bounds come from its filename and are
// file-aligned, so it starts below the FULL and must still be kept: it carries the
// FULL's own start and stop positions.
func Test_CleanOrphanWalForDatabase_WhenFullStartsMidSegment_KeepsBoundarySegment(t *testing.T) {
	prereqs := seedBackupPrereqs(t)

	physical_testing.CreateTestFullBackup(t, physical_testing.NewTestCompletedFullBackup(
		prereqs.DB.ID, prereqs.Storage.ID, 1,
		testLSN(1)+firstRecordOffset, testLSN(1)+firstRecordOffset+fullLSNSpan))

	boundarySegment := physical_testing.CreateTestWalSegment(t, physical_testing.NewTestWalSegment(
		prereqs.DB.ID, prereqs.Storage.ID, 1, "000000010000000000000001", testLSN(1), testLSN(2)))

	cleaner := CreateTestPhysicalCleaner()
	cleaner.cleanOrphanWalForDatabase(t.Context(), logger.GetLogger(), prereqs.DB.ID)

	assert.True(t, walExists(t, boundarySegment.ID),
		"the segment holding the FULL's start_lsn and stop_lsn must survive the orphan sweep")
}

// The counterpart of the boundary case: a segment that ends at or before the
// earliest COMPLETED FULL's start_lsn holds nothing any retained backup replays,
// so the sweep must still reclaim it.
func Test_CleanOrphanWalForDatabase_WhenWalEndsBeforeEarliestFull_DeletesOrphan(t *testing.T) {
	prereqs := seedBackupPrereqs(t)

	physical_testing.CreateTestFullBackup(t, physical_testing.NewTestCompletedFullBackup(
		prereqs.DB.ID, prereqs.Storage.ID, 1,
		testLSN(4)+firstRecordOffset, testLSN(4)+firstRecordOffset+fullLSNSpan))

	staleSegment := physical_testing.CreateTestWalSegment(t, physical_testing.NewTestWalSegment(
		prereqs.DB.ID, prereqs.Storage.ID, 1, "000000010000000000000001", testLSN(1), testLSN(2)))

	cleaner := CreateTestPhysicalCleaner()
	cleaner.cleanOrphanWalForDatabase(t.Context(), logger.GetLogger(), prereqs.DB.ID)

	assert.False(t, walExists(t, staleSegment.ID),
		"WAL that ends below the earliest retained FULL's start is still reclaimable")
}

func Test_ReapAbandonedWalClaims_WhenClaimOlderThanGrace_DeletesIt(t *testing.T) {
	prereqs := seedBackupPrereqs(t)

	// An insert-first claim (file_name NULL) whose upload never finished and that
	// has aged past WAL_CLAIM_GRACE (1h). NULL file_name ⇒ no storage object exists.
	twoHoursAgo := time.Now().UTC().Add(-2 * time.Hour)
	abandoned := &physical_models.PhysicalWalSegment{
		DatabaseID:  prereqs.DB.ID,
		StorageID:   prereqs.Storage.ID,
		TimelineID:  1,
		FileName:    nil,
		WalFilename: "000000010000000000000009",
		StartLSN:    testLSN(9),
		EndLSN:      testLSN(10),
		ReceivedAt:  twoHoursAgo,
		ClaimedAt:   twoHoursAgo,
	}
	require.NoError(t, physical_repositories.GetWalSegmentRepository().Insert(abandoned))

	cleaner := CreateTestPhysicalCleaner()
	cleaner.reapAbandonedWalClaims(t.Context(), logger.GetLogger(), prereqs.DB.ID)

	assert.False(
		t,
		walExists(t, abandoned.ID),
		"an abandoned NULL-file_name claim past grace is reaped (no storage I/O)",
	)
}

func Test_ReapAbandonedWalClaims_WhenClaimWithinGrace_KeepsIt(t *testing.T) {
	prereqs := seedBackupPrereqs(t)

	thirtyMinutesAgo := time.Now().UTC().Add(-30 * time.Minute)
	freshClaim := &physical_models.PhysicalWalSegment{
		DatabaseID:  prereqs.DB.ID,
		StorageID:   prereqs.Storage.ID,
		TimelineID:  1,
		FileName:    nil,
		WalFilename: "000000010000000000000009",
		StartLSN:    testLSN(9),
		EndLSN:      testLSN(10),
		ReceivedAt:  thirtyMinutesAgo,
		ClaimedAt:   thirtyMinutesAgo,
	}
	require.NoError(t, physical_repositories.GetWalSegmentRepository().Insert(freshClaim))

	cleaner := CreateTestPhysicalCleaner()
	cleaner.reapAbandonedWalClaims(t.Context(), logger.GetLogger(), prereqs.DB.ID)

	assert.True(t, walExists(t, freshClaim.ID), "a live in-flight claim within grace must survive")
}

func Test_CleanByChains_WhenKeepCountZero_DeletesNothing(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	shortGraceConfig(&prereqs.Config)
	prereqs.Config.Retention.ChainsCount = 0

	active := seedChainFull(t, prereqs, 4, 0)
	old := seedChainFull(t, prereqs, 2, 9)

	cleaner := CreateTestPhysicalCleaner()
	require.NoError(t, cleaner.cleanByChains(context.Background(), logger.GetLogger(), prereqs.Config))

	assert.True(t, fullExists(t, active.ID))
	assert.True(t, fullExists(t, old.ID), "keep-count 0 is treated as keep-all, not delete-all")
}

func Test_CleanByFulls_WhenNoEffectiveConfig_DeletesNothing(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	shortGraceConfig(&prereqs.Config)
	// LAST_N with count 0 yields no keep-set; the policy must no-op rather than
	// interpret an empty keep-set as "delete every chain".
	prereqs.Config.Retention.FullBackups = backups.FullBackupsRetention{
		Policy: backups.FullBackupsRetentionPolicyLastN,
		Count:  0,
	}

	active := seedChainFull(t, prereqs, 4, 0)
	old := seedChainFull(t, prereqs, 2, 9)

	cleaner := CreateTestPhysicalCleaner()
	require.NoError(t, cleaner.cleanByFulls(context.Background(), logger.GetLogger(), prereqs.Config))

	assert.True(t, fullExists(t, active.ID))
	assert.True(t, fullExists(t, old.ID), "an empty keep-set must never delete everything")
}

func Test_CleanByChains_WhenInProgressFullExists_NeverDeletesIt(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	shortGraceConfig(&prereqs.Config)
	prereqs.Config.Retention.ChainsCount = 1

	inProgress := seedFullWithStatusAndAge(t, prereqs, physical_enums.PhysicalBackupStatusInProgress, 9, 1)
	active := seedChainFull(t, prereqs, 6, 3) // newest COMPLETED → extendable head
	keptByCount := seedChainFull(t, prereqs, 4, 6)
	evicted := seedChainFull(t, prereqs, 2, 9)

	cleaner := CreateTestPhysicalCleaner()
	require.NoError(t, cleaner.cleanByChains(context.Background(), logger.GetLogger(), prereqs.Config))

	assert.True(t, fullExists(t, inProgress.ID), "an IN_PROGRESS full is never a retention candidate")
	assert.True(t, fullExists(t, active.ID))
	assert.True(t, fullExists(t, keptByCount.ID))
	assert.False(t, fullExists(t, evicted.ID), "completed chains beyond the keep-count are still evicted")
}

func Test_DeleteExpiredWalStreamSamples_WhenSamplesOlderThanRetention_DeletesOnlyThose(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	now := time.Now().UTC()

	expiredSample := physical_testing.CreateTestWalStreamSample(t,
		physical_testing.NewTestWalStreamSample(prereqs.DB.ID, now.Add(-wal_health.SampleRetention-time.Hour)))
	recentSample := physical_testing.CreateTestWalStreamSample(t,
		physical_testing.NewTestWalStreamSample(prereqs.DB.ID, now.Add(-24*time.Hour)))

	CreateTestPhysicalCleaner().deleteExpiredWalStreamSamples(t.Context(), logger.GetLogger(), now)

	remainingSamples, err := physical_repositories.GetWalStreamSampleRepository().
		FindSinceByDatabase(prereqs.DB.ID, time.Time{})
	require.NoError(t, err)
	require.Len(t, remainingSamples, 1)
	require.Equal(t, recentSample.ID, remainingSamples[0].ID)
	require.NotEqual(t, expiredSample.ID, remainingSamples[0].ID)
}

func Test_DeleteExpiredWalStreamSamples_WhenDatabaseStoppedStreamingEightDaysAgo_DeletesAllItsSamples(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	now := time.Now().UTC()
	stoppedStreamingAt := now.Add(-8 * 24 * time.Hour)

	for sampleIndex := range 3 {
		physical_testing.CreateTestWalStreamSample(t, physical_testing.NewTestWalStreamSample(
			prereqs.DB.ID, stoppedStreamingAt.Add(-time.Duration(sampleIndex)*wal_health.SampleInterval)))
	}

	CreateTestPhysicalCleaner().deleteExpiredWalStreamSamples(t.Context(), logger.GetLogger(), now)

	remainingSamples, err := physical_repositories.GetWalStreamSampleRepository().
		FindSinceByDatabase(prereqs.DB.ID, time.Time{})
	require.NoError(t, err)
	require.Empty(t, remainingSamples, "samples expire whether or not the database still streams")
}
