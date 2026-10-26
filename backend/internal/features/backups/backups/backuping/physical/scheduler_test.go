package backuping_physical

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	physical_testing "databasus-backend/internal/features/backups/backups/core/physical/testing"
	backups_config_physical "databasus-backend/internal/features/backups/config/physical"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	"databasus-backend/internal/features/intervals"
	storages_bridge "databasus-backend/internal/features/storages"
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/storage"
	"databasus-backend/internal/util/logger"
	"databasus-backend/internal/util/walmath"
)

const testSegmentMB = 16

func testLSN(segments int) walmath.LSN {
	return walmath.LSN(segments * testSegmentMB * 1024 * 1024)
}

// makeIncrementalDue mutates the in-memory config so the FULL interval is not
// due (generic weekly) but the INCR interval is (hourly) and enables INCRs.
func makeIncrementalDue(prereqs *backupPrereqs) {
	prereqs.Config.PostgresqlPhysical.BackupType = postgresql_physical.BackupTypeFullAndIncremental
	prereqs.Config.FullBackupInterval = intervals.Interval{Type: intervals.IntervalWeekly}
	prereqs.Config.IncrementalBackupInterval = intervals.Interval{Type: intervals.IntervalHourly}
}

// seedFullWithStatusAndAge persists a FULL of the given status, aged by ageHours,
// so the cadence and chain-state decision tests are deterministic.
func seedFullWithStatusAndAge(
	t *testing.T,
	prereqs *backupPrereqs,
	status physical_enums.PhysicalBackupStatus,
	startSegment, ageHours int,
) *physical_models.PhysicalFullBackup {
	t.Helper()

	full := physical_testing.NewTestCompletedFullBackup(
		prereqs.DB.ID, prereqs.Storage.ID, 1, testLSN(startSegment), testLSN(startSegment+1))

	full.CreatedAt = time.Now().UTC().Add(-time.Duration(ageHours) * time.Hour)
	full.Status = status

	return physical_testing.CreateTestFullBackup(t, full)
}

// seedIncrWithStatusAndAge persists an INCR of the given status on a chain root,
// aged by ageHours. Used to assert how a terminal incr status steers the next
// decision (ERROR/CANCELED keep the chain extendable; CHAIN_BROKEN does not).
func seedIncrWithStatusAndAge(
	t *testing.T,
	prereqs *backupPrereqs,
	rootFullBackupID uuid.UUID,
	status physical_enums.PhysicalBackupStatus,
	startSegment, ageHours int,
) *physical_models.PhysicalIncrementalBackup {
	t.Helper()

	incr := physical_testing.NewTestCompletedIncrementalBackup(
		prereqs.DB.ID, prereqs.Storage.ID, rootFullBackupID, nil, 1, testLSN(startSegment), testLSN(startSegment+1))

	incr.CreatedAt = time.Now().UTC().Add(-time.Duration(ageHours) * time.Hour)
	incr.Status = status

	return physical_testing.CreateTestIncrementalBackup(t, incr)
}

func setFullErrorReason(
	t *testing.T,
	full *physical_models.PhysicalFullBackup,
	reason physical_enums.PhysicalBackupErrorReason,
) {
	t.Helper()

	full.ErrorReason = &reason
	require.NoError(t, storage.GetDb().Save(full).Error)
}

func setIncrementalErrorReason(
	t *testing.T,
	incremental *physical_models.PhysicalIncrementalBackup,
	reason physical_enums.PhysicalBackupErrorReason,
) {
	t.Helper()

	incremental.ErrorReason = &reason
	require.NoError(t, storage.GetDb().Save(incremental).Error)
}

func Test_DecideBackupKind_WhenIncrementalDetectsTimelineSwitch_SchedulesImmediateFull(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	makeIncrementalDue(prereqs)
	rootFull := seedFullWithStatusAndAge(t, prereqs, physical_enums.PhysicalBackupStatusCompleted, 1, 1)
	brokenIncremental := seedIncrWithStatusAndAge(
		t, prereqs, rootFull.ID, physical_enums.PhysicalBackupStatusChainBroken, 2, 0,
	)
	setIncrementalErrorReason(t, brokenIncremental, physical_enums.PhysicalBackupErrorTimelineSwitchDetected)

	decision, canSchedule := CreateTestPhysicalScheduler().decideBackupKind(
		logger.GetLogger(), time.Now().UTC(), prereqs.Config,
	)

	require.True(t, canSchedule)
	require.Equal(t, physical_enums.PhysicalBackupTypeFull, decision.kind)
	require.NotNil(t, decision.immediateTrigger)
	require.Equal(t, brokenIncremental.ID, decision.immediateTrigger.backupID)
}

func Test_DecideBackupKind_WhenFullFailsDuringFailover_SchedulesImmediateFull(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	prereqs.Config.FullBackupInterval = intervals.Interval{Type: intervals.IntervalWeekly}
	failedFull := seedFullWithStatusAndAge(t, prereqs, physical_enums.PhysicalBackupStatusError, 1, 0)
	setFullErrorReason(t, failedFull, physical_enums.PhysicalBackupErrorFailoverDuringBackup)

	decision, canSchedule := CreateTestPhysicalScheduler().decideBackupKind(
		logger.GetLogger(), time.Now().UTC(), prereqs.Config,
	)

	require.True(t, canSchedule)
	require.NotNil(t, decision.immediateTrigger)
	require.Equal(t, failedFull.ID, decision.immediateTrigger.backupID)
}

func Test_ClaimAndInsert_WhenImmediateTriggerWasReplaced_RejectsStaleDecision(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	makeIncrementalDue(prereqs)
	rootFull := seedFullWithStatusAndAge(t, prereqs, physical_enums.PhysicalBackupStatusCompleted, 1, 2)
	brokenIncremental := seedIncrWithStatusAndAge(
		t, prereqs, rootFull.ID, physical_enums.PhysicalBackupStatusChainBroken, 2, 1,
	)
	setIncrementalErrorReason(t, brokenIncremental, physical_enums.PhysicalBackupErrorTimelineSwitchDetected)

	scheduler := CreateTestPhysicalScheduler()
	staleDecision, canSchedule := scheduler.decideBackupKind(logger.GetLogger(), time.Now().UTC(), prereqs.Config)
	require.True(t, canSchedule)

	replacementFull := seedFullWithStatusAndAge(t, prereqs, physical_enums.PhysicalBackupStatusError, 3, 0)
	replacementFull.CreatedAt = brokenIncremental.CompletedAt.Add(time.Second)
	require.NoError(t, storage.GetDb().Save(replacementFull).Error)

	claimed, err := scheduler.claimAndInsert(prereqs.Config, uuid.New(), staleDecision)

	require.NoError(t, err)
	require.False(t, claimed)
}

func Test_ClaimAndInsert_WhenMismatchAppearsAfterAutomaticDecision_RejectsStaleDecision(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	scheduler := CreateTestPhysicalScheduler()
	automaticDecision, canSchedule := scheduler.decideBackupKind(
		logger.GetLogger(), time.Now().UTC(), prereqs.Config,
	)
	require.True(t, canSchedule)

	mismatchedFull := seedFullWithStatusAndAge(
		t, prereqs, physical_enums.PhysicalBackupStatusChainBroken, 1, 0,
	)
	setFullErrorReason(t, mismatchedFull, physical_enums.PhysicalBackupErrorSystemIdentifierMismatch)

	claimed, err := scheduler.claimAndInsert(prereqs.Config, uuid.New(), automaticDecision)

	require.NoError(t, err)
	require.False(t, claimed)
}

func Test_ClaimAndInsert_WhenMismatchCommitsWhileClaimWaits_RejectsStaleAutomaticDecision(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	scheduler := CreateTestPhysicalScheduler()
	automaticDecision, canSchedule := scheduler.decideBackupKind(
		logger.GetLogger(), time.Now().UTC(), prereqs.Config,
	)
	require.True(t, canSchedule)

	terminalTransaction := storage.GetDb().Begin()
	require.NoError(t, terminalTransaction.Error)
	t.Cleanup(func() { _ = terminalTransaction.Rollback().Error })

	mismatchedFull := physical_testing.NewTestCompletedFullBackup(
		prereqs.DB.ID,
		prereqs.Storage.ID,
		1,
		testLSN(1),
		testLSN(2),
	)
	mismatchReason := physical_enums.PhysicalBackupErrorSystemIdentifierMismatch
	mismatchedFull.Status = physical_enums.PhysicalBackupStatusChainBroken
	mismatchedFull.ErrorReason = &mismatchReason
	require.NoError(t, terminalTransaction.Create(mismatchedFull).Error)

	isOccupied, err := scheduler.inFlightRepo.Claim(terminalTransaction, physical_repositories.ClaimSpec{
		DatabaseID: prereqs.DB.ID,
		BackupType: physical_enums.PhysicalBackupTypeFull,
		BackupID:   uuid.New(),
	})
	require.NoError(t, err)
	require.True(t, isOccupied)

	type claimResult struct {
		isClaimed bool
		err       error
	}
	claimResults := make(chan claimResult, 1)
	go func() {
		isClaimed, claimErr := scheduler.claimAndInsert(
			prereqs.Config,
			uuid.New(),
			automaticDecision,
		)
		claimResults <- claimResult{isClaimed: isClaimed, err: claimErr}
	}()

	select {
	case <-claimResults:
		t.Fatal("automatic claim must wait for the pending terminal transaction")
	case <-time.After(100 * time.Millisecond):
	}

	require.NoError(t, terminalTransaction.Delete(
		&physical_models.PhysicalInFlightBackup{},
		"database_id = ?",
		prereqs.DB.ID,
	).Error)
	require.NoError(t, terminalTransaction.Commit().Error)

	result := <-claimResults
	require.NoError(t, result.err)
	require.False(t, result.isClaimed)

	inFlightBackup, err := scheduler.inFlightRepo.FindByDatabaseID(prereqs.DB.ID)
	require.NoError(t, err)
	require.Nil(t, inFlightBackup)

	inProgressFulls, err := scheduler.fullRepo.FindAllInProgress()
	require.NoError(t, err)
	for _, full := range inProgressFulls {
		require.NotEqual(t, prereqs.DB.ID, full.DatabaseID)
	}
}

func Test_DecideBackupKind_WhenSystemIdentifierMismatchUnresolved_SuppressesAutomaticBackup(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	failedFull := seedFullWithStatusAndAge(t, prereqs, physical_enums.PhysicalBackupStatusChainBroken, 1, 1)
	setFullErrorReason(t, failedFull, physical_enums.PhysicalBackupErrorSystemIdentifierMismatch)

	_, canSchedule := CreateTestPhysicalScheduler().decideBackupKind(
		logger.GetLogger(), time.Now().UTC(), prereqs.Config,
	)

	require.False(t, canSchedule)
}

func Test_DecideBackupKind_WhenSystemIdentifierMismatchUnresolved_AllowsForcedFull(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	failedFull := seedFullWithStatusAndAge(t, prereqs, physical_enums.PhysicalBackupStatusChainBroken, 1, 1)
	setFullErrorReason(t, failedFull, physical_enums.PhysicalBackupErrorSystemIdentifierMismatch)
	requestedAt := time.Now().UTC()
	prereqs.Config.ForceFullRequestedAt = &requestedAt

	decision, canSchedule := CreateTestPhysicalScheduler().decideBackupKind(
		logger.GetLogger(), time.Now().UTC(), prereqs.Config,
	)

	require.True(t, canSchedule)
	require.Equal(t, physical_enums.PhysicalBackupTypeFull, decision.kind)
	require.NotNil(t, decision.forceFullRequestedAt)
}

func Test_DecideBackupKind_WhenCompletedFullFollowsMismatch_ResumesAutomaticScheduling(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	prereqs.Config.FullBackupInterval = intervals.Interval{Type: intervals.IntervalHourly}
	failedFull := seedFullWithStatusAndAge(t, prereqs, physical_enums.PhysicalBackupStatusChainBroken, 1, 3)
	setFullErrorReason(t, failedFull, physical_enums.PhysicalBackupErrorSystemIdentifierMismatch)
	seedFullWithStatusAndAge(t, prereqs, physical_enums.PhysicalBackupStatusCompleted, 2, 2)

	decision, canSchedule := CreateTestPhysicalScheduler().decideBackupKind(
		logger.GetLogger(), time.Now().UTC(), prereqs.Config,
	)

	require.True(t, canSchedule)
	require.Equal(t, physical_enums.PhysicalBackupTypeFull, decision.kind)
}

func Test_DecideBackupKind_WhenUnsuccessfulFullFollowsMismatch_KeepsAutomaticSchedulingSuppressed(t *testing.T) {
	for _, terminalStatus := range []physical_enums.PhysicalBackupStatus{
		physical_enums.PhysicalBackupStatusError,
		physical_enums.PhysicalBackupStatusCanceled,
	} {
		t.Run(string(terminalStatus), func(t *testing.T) {
			prereqs := seedBackupPrereqs(t)
			mismatchedFull := seedFullWithStatusAndAge(
				t, prereqs, physical_enums.PhysicalBackupStatusChainBroken, 1, 3,
			)
			setFullErrorReason(t, mismatchedFull, physical_enums.PhysicalBackupErrorSystemIdentifierMismatch)
			seedFullWithStatusAndAge(t, prereqs, terminalStatus, 2, 2)

			_, canSchedule := CreateTestPhysicalScheduler().decideBackupKind(
				logger.GetLogger(), time.Now().UTC(), prereqs.Config,
			)

			require.False(t, canSchedule)
		})
	}
}

func Test_DecideBackupKind_WhenRecentlyCompletedFullAndIncrNotDue_SchedulesNothing(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	makeIncrementalDue(prereqs) // full weekly, incr hourly, INCR enabled
	seedFullWithStatusAndAge(t, prereqs, physical_enums.PhysicalBackupStatusCompleted, 1, 0)

	scheduler := CreateTestPhysicalScheduler()

	_, ok := scheduler.decideBackupKind(logger.GetLogger(), time.Now().UTC(), prereqs.Config)

	assert.False(t, ok, "a just-completed chain with neither cadence due schedules nothing")
}

func Test_DecideBackupKind_WhenLastFullErroredAndFullCadenceNotDue_SchedulesNothing(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	prereqs.Config.FullBackupInterval = intervals.Interval{Type: intervals.IntervalWeekly}
	seedFullWithStatusAndAge(t, prereqs, physical_enums.PhysicalBackupStatusError, 1, 2)

	scheduler := CreateTestPhysicalScheduler()

	_, ok := scheduler.decideBackupKind(logger.GetLogger(), time.Now().UTC(), prereqs.Config)

	assert.False(t, ok, "a freshly errored full is not retried before its cadence — no tight retry loop")
}

func Test_DecideBackupKind_WhenLastFullErroredAndFullCadenceDue_RetriesFull(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	prereqs.Config.FullBackupInterval = intervals.Interval{Type: intervals.IntervalHourly}
	seedFullWithStatusAndAge(t, prereqs, physical_enums.PhysicalBackupStatusError, 1, 2)

	scheduler := CreateTestPhysicalScheduler()

	decision, ok := scheduler.decideBackupKind(logger.GetLogger(), time.Now().UTC(), prereqs.Config)

	require.True(t, ok)
	assert.Equal(t, physical_enums.PhysicalBackupTypeFull, decision.kind,
		"ERROR is transient: the next cadence-due tick retries the same kind")
}

func Test_DecideBackupKind_WhenLastIncrErroredAndChainExtendable_RetriesIncremental(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	makeIncrementalDue(prereqs)
	full := seedFullWithStatusAndAge(t, prereqs, physical_enums.PhysicalBackupStatusCompleted, 1, 3)
	seedIncrWithStatusAndAge(t, prereqs, full.ID, physical_enums.PhysicalBackupStatusError, 2, 2)

	scheduler := CreateTestPhysicalScheduler()

	decision, ok := scheduler.decideBackupKind(logger.GetLogger(), time.Now().UTC(), prereqs.Config)

	require.True(t, ok)
	assert.Equal(t, physical_enums.PhysicalBackupTypeIncremental, decision.kind,
		"an ERROR incr keeps the chain extendable, so the retry stays an INCR")
	assert.Equal(t, full.ID, decision.incrRootFullBackupID)
	assert.Nil(t, decision.incrParentIncrID, "an errored incr is not a COMPLETED parent")
}

func Test_DecideBackupKind_WhenChainBrokenAndFullCadenceDue_ReAnchorsWithFull(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	makeIncrementalDue(prereqs)
	prereqs.Config.FullBackupInterval = intervals.Interval{Type: intervals.IntervalHourly}
	full := seedFullWithStatusAndAge(t, prereqs, physical_enums.PhysicalBackupStatusCompleted, 1, 2)
	seedIncrWithStatusAndAge(t, prereqs, full.ID, physical_enums.PhysicalBackupStatusChainBroken, 2, 1)

	scheduler := CreateTestPhysicalScheduler()

	decision, ok := scheduler.decideBackupKind(logger.GetLogger(), time.Now().UTC(), prereqs.Config)

	require.True(t, ok)
	assert.Equal(t, physical_enums.PhysicalBackupTypeFull, decision.kind,
		"a CHAIN_BROKEN chain re-anchors with a new FULL, never an INCR")
}

func Test_DecideBackupKind_WhenChainBrokenAndIncrDueButFullNotDue_ReAnchorsWithFull(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	makeIncrementalDue(prereqs) // full weekly (not due), incr hourly
	full := seedFullWithStatusAndAge(t, prereqs, physical_enums.PhysicalBackupStatusCompleted, 1, 3)
	seedIncrWithStatusAndAge(t, prereqs, full.ID, physical_enums.PhysicalBackupStatusChainBroken, 2, 2)

	scheduler := CreateTestPhysicalScheduler()

	decision, ok := scheduler.decideBackupKind(logger.GetLogger(), time.Now().UTC(), prereqs.Config)

	require.True(t, ok)
	assert.Equal(t, physical_enums.PhysicalBackupTypeFull, decision.kind,
		"a broken chain re-anchors on the incremental cadence instead of idling until the full one")
}

func Test_DecideBackupKind_WhenChainBrokenAndRecentFullsAllErrored_SchedulesNothing(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	makeIncrementalDue(prereqs)
	rootFull := seedFullWithStatusAndAge(t, prereqs, physical_enums.PhysicalBackupStatusCompleted, 1, 10)
	seedIncrWithStatusAndAge(t, prereqs, rootFull.ID, physical_enums.PhysicalBackupStatusChainBroken, 2, 9)

	for index := range recentFullAttemptsWindow {
		seedFullWithStatusAndAge(t, prereqs, physical_enums.PhysicalBackupStatusError, 3+index, 4-index)
	}

	scheduler := CreateTestPhysicalScheduler()

	_, ok := scheduler.decideBackupKind(logger.GetLogger(), time.Now().UTC(), prereqs.Config)

	assert.False(t, ok,
		"a source whose recent fulls all failed is broken itself; re-anchoring at incr speed only burns it")
}

func Test_DecideBackupKind_WhenChainBrokenAndRecentFullsCanceled_SchedulesNothing(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	makeIncrementalDue(prereqs)
	rootFull := seedFullWithStatusAndAge(t, prereqs, physical_enums.PhysicalBackupStatusCompleted, 1, 10)
	seedIncrWithStatusAndAge(t, prereqs, rootFull.ID, physical_enums.PhysicalBackupStatusChainBroken, 2, 9)

	for index := range recentFullAttemptsWindow {
		seedFullWithStatusAndAge(t, prereqs, physical_enums.PhysicalBackupStatusCanceled, 3+index, 4-index)
	}

	scheduler := CreateTestPhysicalScheduler()

	_, ok := scheduler.decideBackupKind(logger.GetLogger(), time.Now().UTC(), prereqs.Config)

	assert.False(t, ok, "a user who keeps cancelling fulls must not have one restarted on the incr cadence")
}

func Test_DecideBackupKind_WhenChainBrokenAndOneFullErrored_ReAnchorsWithFull(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	makeIncrementalDue(prereqs)
	rootFull := seedFullWithStatusAndAge(t, prereqs, physical_enums.PhysicalBackupStatusCompleted, 1, 10)
	seedIncrWithStatusAndAge(t, prereqs, rootFull.ID, physical_enums.PhysicalBackupStatusChainBroken, 2, 9)
	seedFullWithStatusAndAge(t, prereqs, physical_enums.PhysicalBackupStatusError, 3, 2)

	scheduler := CreateTestPhysicalScheduler()

	decision, ok := scheduler.decideBackupKind(logger.GetLogger(), time.Now().UTC(), prereqs.Config)

	require.True(t, ok)
	assert.Equal(t, physical_enums.PhysicalBackupTypeFull, decision.kind,
		"one failed full is not a broken source, so the re-anchor still goes ahead")
}

func Test_DecideBackupKind_WhenChainBrokenBySummarizerOff_WaitsForFullCadence(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	makeIncrementalDue(prereqs)
	rootFull := seedFullWithStatusAndAge(t, prereqs, physical_enums.PhysicalBackupStatusCompleted, 1, 10)

	brokenIncr := seedIncrWithStatusAndAge(t, prereqs, rootFull.ID,
		physical_enums.PhysicalBackupStatusChainBroken, 2, 9)
	summarizerOff := physical_enums.PhysicalBackupErrorSummarizerOff
	brokenIncr.ErrorReason = &summarizerOff
	require.NoError(t, storage.GetDb().Save(brokenIncr).Error)

	scheduler := CreateTestPhysicalScheduler()

	_, ok := scheduler.decideBackupKind(logger.GetLogger(), time.Now().UTC(), prereqs.Config)

	assert.False(t, ok,
		"a fresh FULL cannot fix summarize_wal being off, so re-anchoring would loop full backups forever")
}

func Test_DecideBackupKind_WhenCanceledIncrRecent_SchedulesNothing(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	makeIncrementalDue(prereqs)
	full := seedFullWithStatusAndAge(t, prereqs, physical_enums.PhysicalBackupStatusCompleted, 1, 3)
	seedIncrWithStatusAndAge(t, prereqs, full.ID, physical_enums.PhysicalBackupStatusCanceled, 2, 0)

	scheduler := CreateTestPhysicalScheduler()

	_, ok := scheduler.decideBackupKind(logger.GetLogger(), time.Now().UTC(), prereqs.Config)

	assert.False(t, ok, "a just-canceled incr is not auto-retried; the chain resumes on cadence")
}

func Test_DecideBackupKind_WhenFullAndIncrBothDue_ChoosesFull(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	makeIncrementalDue(prereqs) // full weekly, incr hourly
	seedFullWithStatusAndAge(t, prereqs, physical_enums.PhysicalBackupStatusCompleted, 1, 24*8)

	scheduler := CreateTestPhysicalScheduler()

	decision, ok := scheduler.decideBackupKind(logger.GetLogger(), time.Now().UTC(), prereqs.Config)

	require.True(t, ok)
	assert.Equal(t, physical_enums.PhysicalBackupTypeFull, decision.kind,
		"when both cadences are due the same tick, FULL takes precedence")
}

func Test_DecideBackupKind_WhenNoFullExists_ChoosesFull(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	scheduler := CreateTestPhysicalScheduler()

	decision, ok := scheduler.decideBackupKind(logger.GetLogger(), time.Now().UTC(), prereqs.Config)

	require.True(t, ok)
	assert.Equal(t, physical_enums.PhysicalBackupTypeFull, decision.kind)
}

func Test_DecideBackupKind_WhenExtendableChainAndIncrDue_ChoosesIncrementalWithFullParent(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	makeIncrementalDue(prereqs)

	fullModel := physical_testing.NewTestCompletedFullBackup(
		prereqs.DB.ID,
		prereqs.Storage.ID,
		1,
		testLSN(0),
		testLSN(1),
	)
	fullModel.CreatedAt = time.Now().UTC().Add(-2 * time.Hour)
	full := physical_testing.CreateTestFullBackup(t, fullModel)

	scheduler := CreateTestPhysicalScheduler()

	decision, ok := scheduler.decideBackupKind(logger.GetLogger(), time.Now().UTC(), prereqs.Config)

	require.True(t, ok)
	assert.Equal(t, physical_enums.PhysicalBackupTypeIncremental, decision.kind)
	assert.Equal(t, full.ID, decision.incrRootFullBackupID)
	assert.Nil(t, decision.incrParentIncrID, "first INCR has no INCR parent — resolves to the FULL at read time")
}

func Test_DecideBackupKind_WhenChainHasCompletedIncr_ChoosesIncrementalWithIncrParent(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	makeIncrementalDue(prereqs)

	fullModel := physical_testing.NewTestCompletedFullBackup(
		prereqs.DB.ID,
		prereqs.Storage.ID,
		1,
		testLSN(0),
		testLSN(1),
	)
	fullModel.CreatedAt = time.Now().UTC().Add(-3 * time.Hour)
	full := physical_testing.CreateTestFullBackup(t, fullModel)

	incrModel := physical_testing.NewTestCompletedIncrementalBackup(
		prereqs.DB.ID, prereqs.Storage.ID, full.ID, nil, 1, testLSN(1), testLSN(2))
	incrModel.CreatedAt = time.Now().UTC().Add(-2 * time.Hour)
	incr := physical_testing.CreateTestIncrementalBackup(t, incrModel)

	scheduler := CreateTestPhysicalScheduler()

	decision, ok := scheduler.decideBackupKind(logger.GetLogger(), time.Now().UTC(), prereqs.Config)

	require.True(t, ok)
	assert.Equal(t, physical_enums.PhysicalBackupTypeIncremental, decision.kind)
	assert.Equal(t, full.ID, decision.incrRootFullBackupID)
	require.NotNil(t, decision.incrParentIncrID)
	assert.Equal(t, incr.ID, *decision.incrParentIncrID)
}

func Test_DecideBackupKind_WhenFullOnlyConfig_NeverChoosesIncremental(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	// FULL_ONLY (default), FULL not due (generic weekly), INCR interval is set
	// but must be ignored.
	prereqs.Config.FullBackupInterval = intervals.Interval{Type: intervals.IntervalWeekly}
	prereqs.Config.IncrementalBackupInterval = intervals.Interval{Type: intervals.IntervalHourly}

	fullModel := physical_testing.NewTestCompletedFullBackup(
		prereqs.DB.ID,
		prereqs.Storage.ID,
		1,
		testLSN(0),
		testLSN(1),
	)
	fullModel.CreatedAt = time.Now().UTC().Add(-2 * time.Hour)
	physical_testing.CreateTestFullBackup(t, fullModel)

	scheduler := CreateTestPhysicalScheduler()

	_, ok := scheduler.decideBackupKind(logger.GetLogger(), time.Now().UTC(), prereqs.Config)

	assert.False(t, ok, "FULL_ONLY config must never schedule an incremental")
}

func Test_ClaimAndInsert_CreatesInProgressRowAndClaim(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	scheduler := CreateTestPhysicalScheduler()
	backupID := uuid.New()

	claimed, err := scheduler.claimAndInsert(prereqs.Config, backupID,
		backupDecision{kind: physical_enums.PhysicalBackupTypeFull})

	require.NoError(t, err)
	assert.True(t, claimed)

	full, _ := physical_repositories.GetFullBackupRepository().FindByID(backupID)
	require.NotNil(t, full)
	assert.Equal(t, physical_enums.PhysicalBackupStatusInProgress, full.Status)

	claim, _ := physical_repositories.GetInFlightBackupRepository().FindByDatabaseID(prereqs.DB.ID)
	require.NotNil(t, claim)
	assert.Equal(t, physical_enums.PhysicalBackupTypeFull, claim.BackupType)
	assert.Equal(t, backupID, claim.BackupID)
}

func Test_ClaimAndInsert_WhenConcurrentClaimSameDatabase_OnlyOneSucceeds(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	scheduler := CreateTestPhysicalScheduler()

	results := make([]bool, 2)
	backupIDs := []uuid.UUID{uuid.New(), uuid.New()}

	var waitGroup sync.WaitGroup
	for i := range 2 {
		waitGroup.Go(func() {
			claimed, _ := scheduler.claimAndInsert(prereqs.Config, backupIDs[i],
				backupDecision{kind: physical_enums.PhysicalBackupTypeFull})
			results[i] = claimed
		})
	}
	waitGroup.Wait()

	assert.True(t, results[0] != results[1], "exactly one of two concurrent claims must win")

	fulls, err := physical_repositories.GetFullBackupRepository().FindAllInProgress()
	require.NoError(t, err)

	inProgressForDB := 0
	for _, full := range fulls {
		if full.DatabaseID == prereqs.DB.ID {
			inProgressForDB++
		}
	}
	assert.Equal(t, 1, inProgressForDB, "only the winning claim inserts a typed row")
}

func Test_ClaimAndInsert_WhenOrphanCleanupLockIsHeld_WaitsForTransaction(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	scheduler := CreateTestPhysicalScheduler()
	cleanupTransaction := storage.GetDb().Begin()
	require.NoError(t, cleanupTransaction.Error)
	t.Cleanup(func() { _ = cleanupTransaction.Rollback().Error })
	require.NoError(t, physical_repositories.AcquireBackupAndOrphanCleanupLock(cleanupTransaction, prereqs.DB.ID))

	type claimResult struct {
		isClaimed bool
		err       error
	}
	claimResults := make(chan claimResult, 1)
	go func() {
		isClaimed, err := scheduler.claimAndInsert(
			prereqs.Config,
			uuid.New(),
			backupDecision{kind: physical_enums.PhysicalBackupTypeFull},
		)
		claimResults <- claimResult{isClaimed: isClaimed, err: err}
	}()

	select {
	case <-claimResults:
		t.Fatal("FULL claim must wait while orphan cleanup holds the database lock")
	case <-time.After(100 * time.Millisecond):
	}

	require.NoError(t, cleanupTransaction.Commit().Error)
	result := <-claimResults
	require.NoError(t, result.err)
	require.True(t, result.isClaimed)
}

func Test_EvaluateConfig_WhenInFlightClaimExists_SkipsWithoutNewRow(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	scheduler := CreateTestPhysicalScheduler()

	ok, err := physical_repositories.GetInFlightBackupRepository().Claim(
		storage.GetDb(), physical_repositories.ClaimSpec{
			DatabaseID: prereqs.DB.ID,
			BackupType: physical_enums.PhysicalBackupTypeFull,
			BackupID:   uuid.New(),
		})
	require.NoError(t, err)
	require.True(t, ok)

	scheduler.evaluateConfig(t.Context(), logger.GetLogger(), time.Now().UTC(), prereqs.Config)

	lastFull, _ := physical_repositories.GetFullBackupRepository().FindLastFullAnyStatusByDatabase(prereqs.DB.ID)
	assert.Nil(t, lastFull, "the existing claim must block a new typed row this tick")
}

func Test_DecideBackupKind_WhenFullRequested_ChoosesFullBeforeCadence(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	makeIncrementalDue(prereqs)
	seedFullWithStatusAndAge(t, prereqs, physical_enums.PhysicalBackupStatusCompleted, 1, 0)

	requestedAt := time.Now().UTC()
	prereqs.Config.ForceFullRequestedAt = &requestedAt

	scheduler := CreateTestPhysicalScheduler()

	decision, ok := scheduler.decideBackupKind(logger.GetLogger(), time.Now().UTC(), prereqs.Config)

	require.True(t, ok)
	assert.Equal(t, physical_enums.PhysicalBackupTypeFull, decision.kind)
	require.NotNil(t, decision.forceFullRequestedAt)
	assert.Equal(t, requestedAt, *decision.forceFullRequestedAt)
}

func Test_DecideBackupKind_WhenIncrementalRequested_ChoosesIncrementalOutOfCadence(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	makeIncrementalDue(prereqs)
	// A just-completed full means the INCR cadence is NOT due; the forced request
	// must override cadence and still pick an incremental on the existing chain.
	full := seedFullWithStatusAndAge(t, prereqs, physical_enums.PhysicalBackupStatusCompleted, 1, 0)

	requestedAt := time.Now().UTC()
	prereqs.Config.ForceIncrementalRequestedAt = &requestedAt

	scheduler := CreateTestPhysicalScheduler()

	decision, ok := scheduler.decideBackupKind(logger.GetLogger(), time.Now().UTC(), prereqs.Config)

	require.True(t, ok)
	assert.Equal(t, physical_enums.PhysicalBackupTypeIncremental, decision.kind)
	assert.Equal(t, full.ID, decision.incrRootFullBackupID)
	require.NotNil(t, decision.forceIncrementalRequestedAt)
	assert.Equal(t, requestedAt, *decision.forceIncrementalRequestedAt)
}

func Test_DecideBackupKind_WhenIncrementalRequestedButNoExtendableChain_ClearsRequestAndFallsThrough(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	makeIncrementalDue(prereqs) // incrementals enabled, but no full backup exists yet

	require.NoError(t, backups_config_physical.GetBackupConfigService().RequestIncrementalBackupNow(prereqs.DB.ID))
	requested, err := backups_config_physical.GetBackupConfigService().GetBackupConfigByDbId(prereqs.DB.ID)
	require.NoError(t, err)
	require.NotNil(t, requested.ForceIncrementalRequestedAt)
	prereqs.Config.ForceIncrementalRequestedAt = requested.ForceIncrementalRequestedAt

	scheduler := CreateTestPhysicalScheduler()

	decision, ok := scheduler.decideBackupKind(logger.GetLogger(), time.Now().UTC(), prereqs.Config)

	require.True(t, ok)
	assert.Equal(t, physical_enums.PhysicalBackupTypeFull, decision.kind,
		"with no chain the forced incremental can't run, so the decision falls through to a bootstrap full")

	cleared, err := backups_config_physical.GetBackupConfigService().GetBackupConfigByDbId(prereqs.DB.ID)
	require.NoError(t, err)
	assert.Nil(t, cleared.ForceIncrementalRequestedAt,
		"an unsatisfiable forced incremental must be cleared so it can't loop forever")
}

func Test_DecideBackupKind_WhenIncrementalRequestedButIncrementalsDisabled_ClearsRequest(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	// Default config is FULL_ONLY (incrementals disabled). A completed full means a
	// chain exists, isolating the "disabled" clear branch from the "no chain" one.
	seedFullWithStatusAndAge(t, prereqs, physical_enums.PhysicalBackupStatusCompleted, 1, 0)

	require.NoError(t, backups_config_physical.GetBackupConfigService().RequestIncrementalBackupNow(prereqs.DB.ID))
	requested, err := backups_config_physical.GetBackupConfigService().GetBackupConfigByDbId(prereqs.DB.ID)
	require.NoError(t, err)
	require.NotNil(t, requested.ForceIncrementalRequestedAt)
	prereqs.Config.ForceIncrementalRequestedAt = requested.ForceIncrementalRequestedAt

	scheduler := CreateTestPhysicalScheduler()

	_, ok := scheduler.decideBackupKind(logger.GetLogger(), time.Now().UTC(), prereqs.Config)

	assert.False(t, ok, "a forced incremental on a FULL_ONLY config schedules nothing this tick")

	cleared, err := backups_config_physical.GetBackupConfigService().GetBackupConfigByDbId(prereqs.DB.ID)
	require.NoError(t, err)
	assert.Nil(t, cleared.ForceIncrementalRequestedAt,
		"a forced incremental on a FULL_ONLY config must be cleared")
}

func seedInProgressFullWithClaim(t *testing.T, prereqs *backupPrereqs) uuid.UUID {
	t.Helper()

	backupID := uuid.New()
	full := &physical_models.PhysicalFullBackup{
		ID:         backupID,
		DatabaseID: prereqs.DB.ID,
		StorageID:  prereqs.Storage.ID,
		Status:     physical_enums.PhysicalBackupStatusInProgress,
	}
	require.NoError(t, physical_repositories.GetFullBackupRepository().Save(full))

	ok, err := physical_repositories.GetInFlightBackupRepository().Claim(
		storage.GetDb(), physical_repositories.ClaimSpec{
			DatabaseID: prereqs.DB.ID,
			BackupType: physical_enums.PhysicalBackupTypeFull,
			BackupID:   backupID,
		})
	require.NoError(t, err)
	require.True(t, ok)

	return backupID
}

func Test_RecoverInFlightOnRestart_FailsAndReleasesClaim(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	scheduler := CreateTestPhysicalScheduler()
	backupID := seedInProgressFullWithClaim(t, prereqs)

	require.NoError(t, scheduler.recoverInFlightBackupsOnRestart(t.Context(), logger.GetLogger()))

	got, _ := physical_repositories.GetFullBackupRepository().FindByID(backupID)
	require.NotNil(t, got)
	assert.Equal(t, physical_enums.PhysicalBackupStatusError, got.Status)
	require.NotNil(t, got.ErrorReason)
	assert.Equal(t, physical_enums.PhysicalBackupErrorApplicationRestart, *got.ErrorReason)

	claim, _ := physical_repositories.GetInFlightBackupRepository().FindByDatabaseID(prereqs.DB.ID)
	assert.Nil(t, claim, "a backup orphaned by restart must be failed and released")
}

func Test_RecoverInFlightOnRestart_WhenFullHasNamedArtifact_ArtifactRemoved(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	scheduler := CreateTestPhysicalScheduler()
	backupID := seedInProgressFullWithClaim(t, prereqs)
	fileName := "restart-interrupted-full-" + backupID.String()

	fullBackup, err := physical_repositories.GetFullBackupRepository().FindByID(backupID)
	require.NoError(t, err)
	require.NotNil(t, fullBackup)
	fullBackup.FileName = &fileName
	require.NoError(t, physical_repositories.GetFullBackupRepository().Save(fullBackup))
	require.NoError(
		t,
		storages_bridge.WriteTestFile(
			t.Context(),
			storages.StoredFileReference{StorageID: prereqs.Storage.ID, FileName: fileName},
			strings.NewReader("partial physical backup"),
		),
	)

	require.NoError(t, scheduler.recoverInFlightBackupsOnRestart(t.Context(), logger.GetLogger()))

	persistedFullBackup, err := physical_repositories.GetFullBackupRepository().FindByID(backupID)
	require.NoError(t, err)
	require.NotNil(t, persistedFullBackup)
	assert.Equal(t, physical_enums.PhysicalBackupStatusError, persistedFullBackup.Status)
	require.NotNil(t, persistedFullBackup.FileName)
	assert.Equal(t, fileName, *persistedFullBackup.FileName)

	require.NoError(t, storages_bridge.DrainStorageFileDeletions(t.Context(),
		storages.StoredFileReference{StorageID: prereqs.Storage.ID, FileName: fileName},
	))

	_, err = storages_bridge.OpenTestFile(
		t.Context(),
		storages.StoredFileReference{StorageID: prereqs.Storage.ID, FileName: fileName},
	)
	assert.Error(t, err, "a backup interrupted by a restart must not leave its artifact behind")
}

func Test_ClaimAndInsert_WhenConcurrentFullAndIncrSameDatabase_OnlyOneSucceeds(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	// The INCR claimer needs a real chain root to satisfy the FK; the FULL claimer
	// needs nothing. The race is on the single cross-table in-flight slot.
	rootFull := seedFullWithStatusAndAge(t, prereqs, physical_enums.PhysicalBackupStatusCompleted, 0, 1)
	scheduler := CreateTestPhysicalScheduler()

	results := make([]bool, 2)
	fullID := uuid.New()
	incrID := uuid.New()

	var waitGroup sync.WaitGroup
	waitGroup.Go(func() {
		claimed, _ := scheduler.claimAndInsert(prereqs.Config, fullID,
			backupDecision{kind: physical_enums.PhysicalBackupTypeFull})
		results[0] = claimed
	})
	waitGroup.Go(func() {
		claimed, _ := scheduler.claimAndInsert(
			prereqs.Config,
			incrID,
			backupDecision{
				kind:                 physical_enums.PhysicalBackupTypeIncremental,
				incrRootFullBackupID: rootFull.ID,
			},
		)
		results[1] = claimed
	})
	waitGroup.Wait()

	assert.True(t, results[0] != results[1], "exactly one of a concurrent FULL/INCR claim wins the slot")

	inProgressFulls, err := physical_repositories.GetFullBackupRepository().FindAllInProgress()
	require.NoError(t, err)
	inProgressIncrs, err := physical_repositories.GetIncrementalBackupRepository().FindAllInProgress()
	require.NoError(t, err)

	inProgressForDB := 0
	for _, full := range inProgressFulls {
		if full.DatabaseID == prereqs.DB.ID {
			inProgressForDB++
		}
	}
	for _, incr := range inProgressIncrs {
		if incr.DatabaseID == prereqs.DB.ID {
			inProgressForDB++
		}
	}
	assert.Equal(t, 1, inProgressForDB, "the loser inserts no row in either typed table")
}

func Test_EvaluateConfig_WhenStorageIDNil_SkipsWithoutScheduling(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	prereqs.Config.StorageID = nil
	scheduler := CreateTestPhysicalScheduler()

	scheduler.evaluateConfig(t.Context(), logger.GetLogger(), time.Now().UTC(), prereqs.Config)

	lastFull, _ := physical_repositories.GetFullBackupRepository().FindLastFullAnyStatusByDatabase(prereqs.DB.ID)
	assert.Nil(t, lastFull, "a config without a storage id must schedule nothing")

	claim, _ := physical_repositories.GetInFlightBackupRepository().FindByDatabaseID(prereqs.DB.ID)
	assert.Nil(t, claim)
}

func Test_SchedulerRun_WhenCalledTwice_Panics(t *testing.T) {
	scheduler := CreateTestPhysicalScheduler()

	// An already-cancelled context lets the first Run run startup recovery, flip
	// hasRun, then return at the ctx.Err() guard without entering the ticker loop.
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	scheduler.Run(cancelledCtx)

	assert.Panics(t, func() { scheduler.Run(context.Background()) },
		"a second Run on the same scheduler must panic via the hasRun guard")
}

func Test_FailBackupAndReleaseClaim_WhenFullRolledBack_LeavesNoNamedArtifact(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	scheduler := CreateTestPhysicalScheduler()
	backupID := seedInProgressFullWithClaim(t, prereqs)

	require.NoError(t, scheduler.failBackupAndReleaseClaim(t.Context(), orphanedBackupSpec{
		Kind:       physical_enums.PhysicalBackupTypeFull,
		BackupID:   backupID,
		DatabaseID: prereqs.DB.ID,
		StorageID:  prereqs.Storage.ID,
	}, physical_enums.PhysicalBackupErrorApplicationRestart, "restart interrupted backup"))

	full, _ := physical_repositories.GetFullBackupRepository().FindByID(backupID)
	require.NotNil(t, full)
	assert.Equal(t, physical_enums.PhysicalBackupStatusError, full.Status)
	require.NotNil(t, full.FailMessage)
	assert.Equal(t, "restart interrupted backup", *full.FailMessage)
	assert.Nil(t, full.FileName, "a FULL rolled back before its first attempt never named a file")

	claim, _ := physical_repositories.GetInFlightBackupRepository().FindByDatabaseID(prereqs.DB.ID)
	assert.Nil(t, claim, "the in-flight claim must be released in the same tx as the status flip")
}
