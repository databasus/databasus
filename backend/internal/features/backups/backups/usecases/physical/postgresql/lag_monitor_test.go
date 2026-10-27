package usecases_physical_postgresql

import (
	"context"
	"log/slog"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/config"
	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/engines/postgresql/physical/walmath"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	physical_testing "databasus-backend/internal/features/backups/backups/core/physical/testing"
	"databasus-backend/internal/features/backups/backups/core/physical/wal_health"
	databases_bridge "databasus-backend/internal/features/databases"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/util/logger"
)

func runningSlotBreakSample(state *pgphysical.Slot, walLagThresholdBytes int64, observedAt time.Time) slotBreakSample {
	return slotBreakSample{
		SlotState:            state,
		WalLagThresholdBytes: walLagThresholdBytes,
		ReceiverRunStartedAt: observedAt.Add(-time.Minute),
		ObservedAt:           observedAt,
	}
}

const lagWindowTestThresholdBytes int64 = 100 * 1024 * 1024

var lagWindowTestStartedAt = time.Date(2026, time.October, 19, 12, 0, 0, 0, time.UTC)

func getThresholdPercentBytes(percent float64) int64 {
	return int64(float64(lagWindowTestThresholdBytes) * percent / 100)
}

// Samples one lag monitor tick apart, the first at lagWindowTestStartedAt.
type lagSampleTimeline struct {
	walLagThresholdBytes int64
	samples              []slotBreakSample
}

func newLagSampleTimeline() *lagSampleTimeline {
	return &lagSampleTimeline{walLagThresholdBytes: lagWindowTestThresholdBytes}
}

func (l *lagSampleTimeline) getNextObservedAt() time.Time {
	return lagWindowTestStartedAt.Add(time.Duration(len(l.samples)) * wal_health.SampleInterval)
}

// A zero runStartedAt is a tick with no receiver running.
func (l *lagSampleTimeline) addSample(runStartedAt time.Time, walStatus string, lagBytes int64) {
	l.samples = append(l.samples, slotBreakSample{
		SlotState: &pgphysical.Slot{
			CurrentLSN: new(walmath.LSN(0)),
			WalStatus:  walStatus,
			LagBytes:   new(int64(lagBytes)),
		},
		WalLagThresholdBytes: l.walLagThresholdBytes,
		ReceiverRunStartedAt: runStartedAt,
		ObservedAt:           l.getNextObservedAt(),
	})
}

func (l *lagSampleTimeline) addRunningSample(runStartedAt time.Time, lagBytes int64) {
	l.addSample(runStartedAt, "reserved", lagBytes)
}

func (l *lagSampleTimeline) addStoppedSample(lagBytes int64) {
	l.addSample(time.Time{}, "reserved", lagBytes)
}

type classifiedBreak struct {
	Reason walBreakReason
	Action slotBreakAction
}

func classifyEachSample(classifier *slotBreakClassifier, samples []slotBreakSample) []classifiedBreak {
	classifiedBreaks := make([]classifiedBreak, 0, len(samples))

	for _, sample := range samples {
		reason, action := classifier.recordSampleAndClassifyBreak(sample)
		classifiedBreaks = append(classifiedBreaks, classifiedBreak{Reason: reason, Action: action})
	}

	return classifiedBreaks
}

func getFirstBreakIndex(classifiedBreaks []classifiedBreak, action slotBreakAction) int {
	return slices.IndexFunc(classifiedBreaks, func(classified classifiedBreak) bool {
		return classified.Action == action
	})
}

// With one sample per tick, the hold of running time is covered by this many
// intervals between samples of the same run.
var lagWindowHoldIntervals = int(wal_health.WarningSlotStatusHoldPeriod / wal_health.SampleInterval)

func requireLagRebuildAt(t *testing.T, classifiedBreaks []classifiedBreak, expectedIndex int, message string) {
	t.Helper()

	require.Equal(t, expectedIndex, getFirstBreakIndex(classifiedBreaks, slotBreakActionRebuild), message)
	require.Equal(t, breakReasonWalLag, classifiedBreaks[expectedIndex].Reason)
}

func getFirstWalLagRebuildIndex(classifiedBreaks []classifiedBreak) int {
	return slices.IndexFunc(classifiedBreaks, func(classified classifiedBreak) bool {
		return classified.Action == slotBreakActionRebuild && classified.Reason == breakReasonWalLag
	})
}

func requireNoRebuild(t *testing.T, classifiedBreaks []classifiedBreak, message string) {
	t.Helper()

	require.Equal(t, -1, getFirstBreakIndex(classifiedBreaks, slotBreakActionRebuild), message)
}

func Test_RecordSampleAndClassifyBreak_WhenWalStatusLost_Rebuilds(t *testing.T) {
	var classifier slotBreakClassifier

	reason, action := classifier.recordSampleAndClassifyBreak(
		runningSlotBreakSample(
			&pgphysical.Slot{CurrentLSN: new(walmath.LSN(0)), LagBytes: new(int64(0)), WalStatus: "lost"},
			0,
			time.Now().UTC(),
		),
	)

	require.Equal(t, slotBreakActionRebuild, action)
	require.Equal(t, breakReasonSlotLost, reason)
}

func Test_RecordSampleAndClassifyBreak_WhenWalStatusUnreserved_AlertsWithoutRebuilding(t *testing.T) {
	var classifier slotBreakClassifier

	firstSeenAt := time.Now().UTC()
	state := &pgphysical.Slot{CurrentLSN: new(walmath.LSN(0)), LagBytes: new(int64(0)), WalStatus: "unreserved"}

	_, immediateAction := classifier.recordSampleAndClassifyBreak(runningSlotBreakSample(state, 0, firstSeenAt))
	require.Equal(t, slotBreakActionNone, immediateAction,
		"unreserved is a warning; it must hold before it is worth telling anyone about")

	reason, heldAction := classifier.recordSampleAndClassifyBreak(
		runningSlotBreakSample(state, 0, firstSeenAt.Add(wal_health.WarningSlotStatusHoldPeriod+time.Second)),
	)

	require.Equal(t, slotBreakActionAlert, heldAction,
		"PG trims that WAL at the next checkpoint, so dropping the slot would only add a gap")
	require.Equal(t, breakReasonSlotRetention, reason)
}

func Test_RecordSampleAndClassifyBreak_WhenWalStatusExtendedPastHold_AlertsWithoutRebuilding(t *testing.T) {
	var classifier slotBreakClassifier

	firstSeenAt := time.Now().UTC()
	state := &pgphysical.Slot{CurrentLSN: new(walmath.LSN(0)), LagBytes: new(int64(0)), WalStatus: "extended"}

	classifier.recordSampleAndClassifyBreak(runningSlotBreakSample(state, 0, firstSeenAt))

	reason, action := classifier.recordSampleAndClassifyBreak(
		runningSlotBreakSample(state, 0, firstSeenAt.Add(wal_health.WarningSlotStatusHoldPeriod+time.Second)),
	)

	require.Equal(t, slotBreakActionAlert, action,
		"retaining WAL beyond max_wal_size is routine on a busy cluster, not a reason to drop the slot")
	require.Equal(t, breakReasonSlotRetention, reason)
}

func Test_RecordSampleAndClassifyBreak_WhenReceiverResumesWithLagAboveThreshold_DoesNotRebuild(t *testing.T) {
	timeline := newLagSampleTimeline()
	timeline.addStoppedSample(2 * lagWindowTestThresholdBytes)

	resumedAt := timeline.getNextObservedAt()
	for sampleIndex := range 2*lagWindowHoldIntervals + 1 {
		timeline.addRunningSample(
			resumedAt,
			2*lagWindowTestThresholdBytes-int64(sampleIndex)*getThresholdPercentBytes(2),
		)
	}

	var classifier slotBreakClassifier

	requireNoRebuild(t, classifyEachSample(&classifier, timeline.samples),
		"a receiver catching up after a back-pressure pause must not cost the chain")
}

func Test_RecordSampleAndClassifyBreak_WhenFirstSampleOfSessionShowsShrinkingLag_DoesNotRebuild(t *testing.T) {
	timeline := newLagSampleTimeline()

	runStartedAt := lagWindowTestStartedAt.Add(-time.Second)
	for sampleIndex := range 2*lagWindowHoldIntervals + 1 {
		timeline.addRunningSample(
			runStartedAt,
			3*lagWindowTestThresholdBytes-int64(sampleIndex)*getThresholdPercentBytes(2),
		)
	}

	var classifier slotBreakClassifier

	requireNoRebuild(t, classifyEachSample(&classifier, timeline.samples),
		"the backlog left by Databasus being down drains on its own and must not cost the chain")
}

func Test_RecordSampleAndClassifyBreak_WhenSlowCatchUpOutlastsHold_DoesNotRebuild(t *testing.T) {
	timeline := newLagSampleTimeline()

	runStartedAt := lagWindowTestStartedAt
	for sampleIndex := range 4*lagWindowHoldIntervals + 1 {
		timeline.addRunningSample(
			runStartedAt,
			2*lagWindowTestThresholdBytes-int64(sampleIndex)*getThresholdPercentBytes(0.6),
		)
	}

	var classifier slotBreakClassifier

	requireNoRebuild(t, classifyEachSample(&classifier, timeline.samples),
		"a lag shrinking by more than 5% of the threshold every 5 minutes is a receiver draining")
}

func Test_RecordSampleAndClassifyBreak_WhenShortRunsDrainBetweenPausesThatGrowLag_DoesNotRebuild(t *testing.T) {
	timeline := newLagSampleTimeline()

	lagBytes := 2 * lagWindowTestThresholdBytes

	for range 10 {
		runStartedAt := timeline.getNextObservedAt()

		for range 6 {
			timeline.addRunningSample(runStartedAt, lagBytes)
			lagBytes -= getThresholdPercentBytes(1)
		}

		for range 2 {
			timeline.addStoppedSample(lagBytes)
			lagBytes += getThresholdPercentBytes(5)
		}
	}

	require.Greater(t, lagBytes, 2*lagWindowTestThresholdBytes, "the pauses grow the lag more than the runs drain it")

	var classifier slotBreakClassifier

	requireNoRebuild(t, classifyEachSample(&classifier, timeline.samples),
		"lag that grows while back pressure holds the receiver down must never count against it")
}

func Test_RecordSampleAndClassifyBreak_WhenRunningReceiverCannotKeepUp_RebuildsOnceHoldOfRunningTimePasses(
	t *testing.T,
) {
	timeline := newLagSampleTimeline()

	runStartedAt := lagWindowTestStartedAt
	for sampleIndex := range lagWindowHoldIntervals + 1 {
		timeline.addRunningSample(
			runStartedAt,
			lagWindowTestThresholdBytes+int64(sampleIndex+1)*getThresholdPercentBytes(1),
		)
	}

	var classifier slotBreakClassifier

	requireLagRebuildAt(t, classifyEachSample(&classifier, timeline.samples), lagWindowHoldIntervals,
		"a running receiver falling behind is rebuilt on the first sample after 5 minutes of running time")
}

func Test_RecordSampleAndClassifyBreak_WhenLagFluctuatesAboveThreshold_RebuildsOnceHoldOfRunningTimePasses(
	t *testing.T,
) {
	timeline := newLagSampleTimeline()

	runStartedAt := lagWindowTestStartedAt
	for sampleIndex := range lagWindowHoldIntervals + 1 {
		swingBytes := int64(sampleIndex%2) * getThresholdPercentBytes(3)
		trendBytes := int64(sampleIndex) * getThresholdPercentBytes(0.4)

		timeline.addRunningSample(runStartedAt, 3*lagWindowTestThresholdBytes/2+swingBytes-trendBytes)
	}

	var classifier slotBreakClassifier

	requireLagRebuildAt(t, classifyEachSample(&classifier, timeline.samples), lagWindowHoldIntervals,
		"a lag that only wobbles is not drained, however often it dips")
}

func Test_RecordSampleAndClassifyBreak_WhenStalledReceiverIsRestartedEvery70Seconds_RebuildsAfterHoldOfRunTime(
	t *testing.T,
) {
	const restartInterval = 70 * time.Second

	timeline := newLagSampleTimeline()

	for sampleIndex := range 30 {
		sinceStart := timeline.getNextObservedAt().Sub(lagWindowTestStartedAt)
		runStartedAt := lagWindowTestStartedAt.Add(sinceStart / restartInterval * restartInterval)

		timeline.addRunningSample(
			runStartedAt,
			lagWindowTestThresholdBytes+int64(sampleIndex+1)*getThresholdPercentBytes(1),
		)
	}

	var classifier slotBreakClassifier

	// Samples at 0 s..480 s fall into runs starting every 70 s; the intervals
	// inside one run add up to 300 s at the sample taken 480 s in.
	requireLagRebuildAt(t, classifyEachSample(&classifier, timeline.samples), 16,
		"restarts must not reset the running time, or a stalled receiver is never rebuilt")
}

func Test_RecordSampleAndClassifyBreak_WhenReceiverStoppedWithLagOverThreshold_DoesNotRebuildOrCountTheTime(
	t *testing.T,
) {
	timeline := newLagSampleTimeline()

	lagBytes := 2 * lagWindowTestThresholdBytes
	for range 2 * lagWindowHoldIntervals {
		timeline.addStoppedSample(lagBytes)
		lagBytes += getThresholdPercentBytes(1)
	}

	stoppedSampleCount := len(timeline.samples)

	runStartedAt := timeline.getNextObservedAt()
	for range lagWindowHoldIntervals + 1 {
		timeline.addRunningSample(runStartedAt, lagBytes)
		lagBytes += getThresholdPercentBytes(1)
	}

	var classifier slotBreakClassifier

	requireLagRebuildAt(t, classifyEachSample(&classifier, timeline.samples), stoppedSampleCount+lagWindowHoldIntervals,
		"lag we caused by holding the receiver down must never cost the chain, nor shorten the hold afterwards")
}

func Test_RecordSampleAndClassifyBreak_WhenLagDipsBelowThreshold_RestartsHoldOfRunningTime(t *testing.T) {
	timeline := newLagSampleTimeline()

	runStartedAt := lagWindowTestStartedAt
	lagBytes := lagWindowTestThresholdBytes + getThresholdPercentBytes(1)

	for range lagWindowHoldIntervals - 1 {
		timeline.addRunningSample(runStartedAt, lagBytes)
		lagBytes += getThresholdPercentBytes(1)
	}

	timeline.addRunningSample(runStartedAt, lagWindowTestThresholdBytes)
	dipIndex := len(timeline.samples) - 1

	for range lagWindowHoldIntervals + 1 {
		timeline.addRunningSample(runStartedAt, lagBytes)
		lagBytes += getThresholdPercentBytes(1)
	}

	var classifier slotBreakClassifier

	requireLagRebuildAt(t, classifyEachSample(&classifier, timeline.samples), dipIndex+1+lagWindowHoldIntervals,
		"a lag at the threshold ends the window; the next one needs its own 5 minutes")
}

func Test_RecordSampleAndClassifyBreak_WhenThresholdIsZero_NeverRebuildsForLag(t *testing.T) {
	timeline := newLagSampleTimeline()
	timeline.walLagThresholdBytes = 0

	runStartedAt := lagWindowTestStartedAt
	for sampleIndex := range 4 * lagWindowHoldIntervals {
		timeline.addRunningSample(runStartedAt, int64(sampleIndex+1)*lagWindowTestThresholdBytes)
	}

	var classifier slotBreakClassifier

	requireNoRebuild(t, classifyEachSample(&classifier, timeline.samples), "a zero threshold turns the lag rule off")
}

func Test_RecordSampleAndClassifyBreak_WhenSlotStatusEpisodeEndsAboveThreshold_RestartsHoldOfRunningTime(
	t *testing.T,
) {
	episodes := []struct {
		walStatus   string
		sampleCount int
	}{
		{walStatus: "extended", sampleCount: 6},
		{walStatus: "unreserved", sampleCount: 6},
		{walStatus: "lost", sampleCount: 1},
	}

	for _, episode := range episodes {
		t.Run(episode.walStatus, func(t *testing.T) {
			timeline := newLagSampleTimeline()

			runStartedAt := lagWindowTestStartedAt
			lagBytes := lagWindowTestThresholdBytes + getThresholdPercentBytes(1)

			for range lagWindowHoldIntervals - 2 {
				timeline.addRunningSample(runStartedAt, lagBytes)
				lagBytes += getThresholdPercentBytes(1)
			}

			for range episode.sampleCount {
				timeline.addSample(runStartedAt, episode.walStatus, lagBytes)
				lagBytes += getThresholdPercentBytes(1)
			}

			returnedToReservedIndex := len(timeline.samples)

			for range lagWindowHoldIntervals + 1 {
				timeline.addRunningSample(runStartedAt, lagBytes)
				lagBytes += getThresholdPercentBytes(1)
			}

			var classifier slotBreakClassifier

			classifiedBreaks := classifyEachSample(&classifier, timeline.samples)

			require.Equal(t, returnedToReservedIndex+lagWindowHoldIntervals,
				getFirstWalLagRebuildIndex(classifiedBreaks),
				"lag seen before and during a slot status episode must not count once the slot returns to reserved")
		})
	}
}

func Test_RecordSampleAndClassifyBreak_WhenReceiverStallsAfterDrainingHold_RebuildsOneHoldAfterTheDrain(
	t *testing.T,
) {
	timeline := newLagSampleTimeline()

	runStartedAt := lagWindowTestStartedAt
	lagBytes := 2 * lagWindowTestThresholdBytes

	for range lagWindowHoldIntervals + 1 {
		timeline.addRunningSample(runStartedAt, lagBytes)
		lagBytes -= getThresholdPercentBytes(2)
	}

	drainedHoldEndIndex := len(timeline.samples) - 1

	for range lagWindowHoldIntervals + 1 {
		timeline.addRunningSample(runStartedAt, lagBytes)
		lagBytes += getThresholdPercentBytes(1)
	}

	var classifier slotBreakClassifier

	requireLagRebuildAt(t, classifyEachSample(&classifier, timeline.samples),
		drainedHoldEndIndex+lagWindowHoldIntervals,
		"a drained hold starts the window again, so a stall after it is judged over its own 5 minutes")
}

func Test_RecordSampleAndClassifyBreak_WhenExtendedSlotLagsAboveThreshold_AlertsWithoutRebuilding(t *testing.T) {
	timeline := newLagSampleTimeline()

	runStartedAt := lagWindowTestStartedAt
	for sampleIndex := range 2*lagWindowHoldIntervals + 1 {
		timeline.addSample(
			runStartedAt,
			"extended",
			lagWindowTestThresholdBytes+int64(sampleIndex+1)*getThresholdPercentBytes(1),
		)
	}

	var classifier slotBreakClassifier

	classifiedBreaks := classifyEachSample(&classifier, timeline.samples)

	requireNoRebuild(t, classifiedBreaks, "an extended slot alerts instead of costing the chain")

	firstAlertIndex := getFirstBreakIndex(classifiedBreaks, slotBreakActionAlert)
	require.Equal(t, lagWindowHoldIntervals+1, firstAlertIndex,
		"the retention alert comes on the first sample after the status has lasted 5 minutes")
	require.Equal(t, breakReasonSlotRetention, classifiedBreaks[firstAlertIndex].Reason)
}

func Test_RecordSampleAndClassifyBreak_WhenForeignConsumerHoldsSlot_Rebuilds(t *testing.T) {
	var classifier slotBreakClassifier

	state := &pgphysical.Slot{
		CurrentLSN: new(walmath.LSN(0)), LagBytes: new(int64(0)),
		WalStatus:       "reserved",
		IsActive:        true,
		ActivePID:       new(4242),
		ApplicationName: "some_other_consumer",
	}

	reason, action := classifier.recordSampleAndClassifyBreak(runningSlotBreakSample(state, 0, time.Now().UTC()))

	require.Equal(t, slotBreakActionRebuild, action)
	require.Equal(t, breakReasonSlotStolen, reason)
}

func Test_RecordSampleAndClassifyBreak_WhenOurReceiverActive_DoesNotRebuild(t *testing.T) {
	var classifier slotBreakClassifier

	state := &pgphysical.Slot{
		CurrentLSN:      new(walmath.LSN(0)),
		WalStatus:       "reserved",
		IsActive:        true,
		ActivePID:       new(17),
		ApplicationName: pgphysical.WalReceiverApplicationNamePrefix + "db-1",
		LagBytes:        new(int64(10)),
	}

	_, action := classifier.recordSampleAndClassifyBreak(runningSlotBreakSample(state, 100, time.Now().UTC()))

	require.Equal(t, slotBreakActionNone, action)
}

func Test_RecordSampleAndClassifyBreak_WhenSlotHealthy_DoesNotRebuildAndClearsWarningClock(t *testing.T) {
	var classifier slotBreakClassifier

	firstSeenAt := time.Now().UTC()

	classifier.recordSampleAndClassifyBreak(
		runningSlotBreakSample(
			&pgphysical.Slot{CurrentLSN: new(walmath.LSN(0)), LagBytes: new(int64(0)), WalStatus: "extended"},
			100,
			firstSeenAt,
		),
	)

	_, healthyAction := classifier.recordSampleAndClassifyBreak(
		runningSlotBreakSample(
			&pgphysical.Slot{CurrentLSN: new(walmath.LSN(0)), WalStatus: "reserved", LagBytes: new(int64(10))},
			100,
			firstSeenAt.Add(time.Minute),
		),
	)
	require.Equal(t, slotBreakActionNone, healthyAction)

	_, actionAfterRecovery := classifier.recordSampleAndClassifyBreak(
		runningSlotBreakSample(
			&pgphysical.Slot{
				CurrentLSN: new(walmath.LSN(0)),
				LagBytes:   new(int64(0)),
				WalStatus:  "extended",
			},
			100,
			firstSeenAt.Add(wal_health.WarningSlotStatusHoldPeriod+time.Second),
		),
	)

	require.Equal(t, slotBreakActionNone, actionAfterRecovery,
		"a recovered slot restarts the hold window instead of inheriting the previous one")
}

func newSupervisorReportingTo(log *capturedLog) (*WalStreamSupervisor, *slog.Logger) {
	return &WalStreamSupervisor{uploader: NewWalUploader(WalUploadDeps{})},
		slog.New(slog.NewTextHandler(log, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func Test_ReportArchivingProgress_AfterBusyInterval_LogsCountsStepTimingsAndConcurrency(t *testing.T) {
	log := &capturedLog{}
	supervisor, supervisorLogger := newSupervisorReportingTo(log)

	supervisor.uploader.stats.recordArchived(3*1024*1024, segmentUploadTimings{
		Claim:        4 * time.Millisecond,
		SegmentWrite: 300 * time.Millisecond,
		SidecarWrite: 60 * time.Millisecond,
		Commit:       8 * time.Millisecond,
	})
	supervisor.uploader.stats.recordArchived(1024*1024, segmentUploadTimings{
		Claim:        2 * time.Millisecond,
		SegmentWrite: 100 * time.Millisecond,
		SidecarWrite: 40 * time.Millisecond,
		Commit:       4 * time.Millisecond,
	})
	supervisor.uploader.RecordConcurrentUploads(4)

	supervisor.reportArchivingProgress(
		t.Context(),
		supervisorLogger,
		&pgphysical.Slot{CurrentLSN: new(walmath.LSN(0)), LagBytes: new(int64(2048)), WalStatus: "reserved"},
		supervisor.uploader.TakeArchivingSnapshot(),
	)

	logLine := log.String()
	require.Contains(t, logLine, "level=INFO")
	require.Contains(t, logLine, "archived 2 wal segments (4.0 MB)")
	require.Contains(t, logLine, "up to 4 uploads at once")
	require.Contains(t, logLine, "claim avg 3ms max 4ms")
	require.Contains(t, logLine, "segment write avg 200ms max 300ms")
	require.Contains(t, logLine, "sidecar write avg 50ms max 60ms")
	require.Contains(t, logLine, "commit avg 6ms max 8ms")
	require.Contains(t, logLine, "2048 bytes behind")
}

func Test_ReportArchivingProgress_AfterQuietInterval_LogsOnlyBytesBehindAtDebug(t *testing.T) {
	log := &capturedLog{}
	supervisor, supervisorLogger := newSupervisorReportingTo(log)

	supervisor.reportArchivingProgress(
		t.Context(),
		supervisorLogger,
		&pgphysical.Slot{CurrentLSN: new(walmath.LSN(0)), LagBytes: new(int64(4096)), WalStatus: "reserved"},
		supervisor.uploader.TakeArchivingSnapshot(),
	)

	logLine := log.String()
	require.Contains(t, logLine, "level=DEBUG")
	require.Contains(t, logLine, "no wal archived since the last report, 4096 bytes behind")
	require.NotContains(t, logLine, "avg")
}

var lagMonitorTickStartedAt = time.Date(2026, time.October, 18, 12, 0, 0, 0, time.UTC)

func newLagMonitorTickSupervisor(
	t *testing.T,
	fixture *PhysicalDBFixture,
	sourceDB databases.PostgresqlPhysicalConnectionSettings,
	walLagThresholdBytes int64,
) *WalStreamSupervisor {
	t.Helper()

	return NewWalStreamSupervisor(WalStreamSpec{
		ClientTools:          pgshared.ClientTools{Root: config.GetEnv().ClientToolsDir},
		DatabaseID:           fixture.DB.ID,
		BackupStatus:         databases_bridge.GetModule(),
		SourceDB:             sourceDB,
		StorageID:            fixture.Storage.ID,
		WalSegmentRepo:       physical_repositories.GetWalSegmentRepository(),
		HistoryRepo:          physical_repositories.GetWalHistoryRepository(),
		WalStreamSampleRepo:  physical_repositories.GetWalStreamSampleRepository(),
		WalStreamerRepo:      physical_repositories.GetWalStreamerRepository(),
		WatchDirRoot:         t.TempDir(),
		WalLagThresholdBytes: walLagThresholdBytes,
		Logger:               logger.GetLogger(),
	})
}

func createStreamerSlot(t *testing.T, fixture *PhysicalDBFixture) {
	t.Helper()

	adminConn := OpenAdminConn(t, fixture)

	require.NoError(
		t,
		pgphysical.NewSource(fixture.SourceDB.Engine).
			VerifySlot(t.Context(), logger.GetLogger(), fixture.SourceDB.WalStreamSlotName),
	)

	t.Cleanup(func() {
		DropReplicationSlotExternally(t, adminConn, fixture.SourceDB.WalStreamSlotName)
	})
}

// A committed segment below the source's position gives the tick an archive
// progress point, which the unarchived WAL is measured from.
func seedCommittedSegmentBelowCurrentPosition(t *testing.T, fixture *PhysicalDBFixture) {
	t.Helper()

	currentLSN, err := OpenTestInspection(t, fixture.SourceDB).GetCurrentWalLSN(t.Context())
	require.NoError(t, err)

	segmentSizeBytes := walSegmentSizeBytes(fixture.SourceDB)
	segmentEndLSN := walmath.LSN(uint64(currentLSN) / uint64(segmentSizeBytes) * uint64(segmentSizeBytes))
	segmentStartLSN := segmentEndLSN - walmath.LSN(segmentSizeBytes)

	physical_testing.CreateTestWalSegment(t, physical_testing.NewTestWalSegment(
		fixture.DB.ID, fixture.Storage.ID, 1, "seeded-segment", segmentStartLSN, segmentEndLSN,
	))
}

func runLagMonitorTicks(
	t *testing.T,
	supervisor *WalStreamSupervisor,
	monitorState *lagMonitorState,
	firstTickAt time.Time,
	tickCount int,
	beforeEachTick func(),
) {
	t.Helper()

	for tickIndex := range tickCount {
		if beforeEachTick != nil {
			beforeEachTick()
		}

		supervisor.runLagMonitorTick(
			t.Context(),
			logger.GetLogger(),
			monitorState,
			firstTickAt.Add(time.Duration(tickIndex)*wal_health.SampleInterval),
		)
	}
}

func getRecordedSamples(t *testing.T, fixture *PhysicalDBFixture) []*physical_models.PhysicalWalStreamSample {
	t.Helper()

	samples, err := physical_repositories.GetWalStreamSampleRepository().
		FindSinceByDatabase(fixture.DB.ID, time.Time{})
	require.NoError(t, err)

	return samples
}

func Test_RunLagMonitorTick_WhenSourceHealthy_RecordsReachableSamplesWithSlotAndQueueValues(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)
	adminConn := OpenAdminConn(t, fixture)

	createStreamerSlot(t, fixture)
	seedCommittedSegmentBelowCurrentPosition(t, fixture)

	supervisor := newLagMonitorTickSupervisor(t, fixture, fixture.SourceDB, 0)

	runLagMonitorTicks(t, supervisor, supervisor.newLagMonitorState(logger.GetLogger()), lagMonitorTickStartedAt, 4,
		func() {
			_, err := GenerateWalActivity(t.Context(), adminConn, 4096)
			require.NoError(t, err)
		})

	samples := getRecordedSamples(t, fixture)
	require.Len(t, samples, 4)

	for index, sample := range samples {
		assert.Equal(t, physical_enums.PhysicalWalStreamSourceStateReachable, sample.SourceState)
		assert.Equal(
			t,
			lagMonitorTickStartedAt.Add(time.Duration(index)*wal_health.SampleInterval),
			sample.SampledAt.UTC(),
		)
		assert.NotNil(t, sample.SourceCurrentLSN)
		assert.NotNil(t, sample.SlotWalStatus)
		assert.NotNil(t, sample.SlotLagBytes)
		assert.NotNil(t, sample.UnarchivedWalBytes)
		assert.Positive(t, sample.LocalQueuePauseBytes)

		if index > 0 {
			require.NotNil(t, sample.SourceWrittenBytes)
			assert.Positive(t, *sample.SourceWrittenBytes)
		}
	}

	assert.Nil(t, samples[0].SourceWrittenBytes)
}

func Test_RunLagMonitorTick_WhenSourceUnreachableAcrossSessions_KeepsOneUnreachableSinceTime(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	unreachableSourceDB := fixture.SourceDB
	unreachableSourceDB.Engine.Host = "127.0.0.1"
	unreachableSourceDB.Engine.Port = 1

	firstSession := newLagMonitorTickSupervisor(t, fixture, unreachableSourceDB, 0)
	runLagMonitorTicks(t, firstSession, firstSession.newLagMonitorState(logger.GetLogger()),
		lagMonitorTickStartedAt, 3, nil)

	secondSessionStartedAt := lagMonitorTickStartedAt.Add(5 * time.Minute)
	secondSession := newLagMonitorTickSupervisor(t, fixture, unreachableSourceDB, 0)
	runLagMonitorTicks(t, secondSession, secondSession.newLagMonitorState(logger.GetLogger()),
		secondSessionStartedAt, 1, nil)

	samples := getRecordedSamples(t, fixture)
	require.Len(t, samples, 4)

	for _, sample := range samples {
		assert.Equal(t, physical_enums.PhysicalWalStreamSourceStateUnreachable, sample.SourceState)
		require.NotNil(t, sample.SourceUnreachableSince)
		assert.Equal(t, lagMonitorTickStartedAt, sample.SourceUnreachableSince.UTC(),
			"the time a source has been unreachable survives the session restart its outage causes")
		assert.Nil(t, sample.SlotWalStatus)
		assert.Nil(t, sample.SlotLagBytes)
		assert.Nil(t, sample.SlotSafeWalSizeBytes)
		assert.Nil(t, sample.UnarchivedWalBytes)
		assert.Nil(t, sample.SourceWrittenBytes)
	}
}

func Test_RunLagMonitorTick_WhenSlotHeldExtended_RecordsTimeClassifierStartedItsHold(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)
	adminConn := OpenAdminConn(t, fixture)

	createStreamerSlot(t, fixture)
	setMaxWalSizeForTest(t, adminConn, "32MB")

	slotName := fixture.SourceDB.WalStreamSlotName

	inspection := OpenTestInspection(t, fixture.SourceDB)

	require.Eventually(t, func() bool {
		ForceReplicationLag(t, adminConn, 16*1024*1024)

		slotState, err := inspection.GetSlot(t.Context(), slotName)

		return err == nil && slotState != nil && slotState.WalStatus == slotWalStatusExtendedForTest
	}, 90*time.Second, time.Second, "a slot lagging past max_wal_size is reported as extended")

	supervisor := newLagMonitorTickSupervisor(t, fixture, fixture.SourceDB, 0)
	runLagMonitorTicks(
		t,
		supervisor,
		supervisor.newLagMonitorState(logger.GetLogger()),
		lagMonitorTickStartedAt,
		2,
		nil,
	)

	samples := getRecordedSamples(t, fixture)
	require.Len(t, samples, 2)

	for _, sample := range samples {
		require.NotNil(t, sample.SlotExtraWalSince)
		assert.Equal(t, lagMonitorTickStartedAt, sample.SlotExtraWalSince.UTC())
	}
}

const slotWalStatusExtendedForTest = "extended"

// max_wal_size is reloadable, so the test restores it instead of leaving the
// shared source checkpointing every two segments.
func setMaxWalSizeForTest(t *testing.T, adminConn *pgx.Conn, size string) {
	t.Helper()

	_, err := adminConn.Exec(t.Context(), "ALTER SYSTEM SET max_wal_size = '"+size+"'")
	require.NoError(t, err)

	_, err = adminConn.Exec(t.Context(), "SELECT pg_reload_conf()")
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = adminConn.Exec(context.Background(), "ALTER SYSTEM RESET max_wal_size")
		_, _ = adminConn.Exec(context.Background(), "SELECT pg_reload_conf()")
	})
}

func Test_RunLagMonitorTick_WhenTickRebuildsSlot_RecordsSlotAsProbedBeforeRebuild(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)
	adminConn := OpenAdminConn(t, fixture)

	createStreamerSlot(t, fixture)

	const walLagThresholdBytes = 64 * 1024

	slotName := fixture.SourceDB.WalStreamSlotName
	ForceReplicationLag(t, adminConn, 4*walLagThresholdBytes)
	WaitUntilSlotLag(t, adminConn, slotName, 2*walLagThresholdBytes, 30*time.Second)

	supervisor := newLagMonitorTickSupervisor(t, fixture, fixture.SourceDB, walLagThresholdBytes)
	supervisor.receiverRunStartedAtUnixNano.Store(lagMonitorTickStartedAt.UnixNano())

	sampleCountAtRebuild := -1
	supervisor.spec.OnSlotRebuilt = func(context.Context, string) error {
		sampleCountAtRebuild = len(getRecordedSamples(t, fixture))

		return nil
	}

	// No receiver drains the slot, so the lag stays above the threshold until
	// the hold of running time ends on the last tick.
	tickCount := int(wal_health.WarningSlotStatusHoldPeriod/wal_health.SampleInterval) + 1

	runLagMonitorTicks(
		t,
		supervisor,
		supervisor.newLagMonitorState(logger.GetLogger()),
		lagMonitorTickStartedAt,
		tickCount,
		nil,
	)

	require.Equal(t, tickCount, sampleCountAtRebuild, "the tick records its sample before it rebuilds the slot")

	samples := getRecordedSamples(t, fixture)
	require.Len(t, samples, tickCount)

	rebuildSample := samples[len(samples)-1]
	require.NotNil(t, rebuildSample.SlotLagBytes)
	assert.Greater(t, *rebuildSample.SlotLagBytes, int64(walLagThresholdBytes),
		"the sample shows the slot that caused the rebuild")
}

func Test_RunLagMonitorTick_OnFirstSampleOfSession_ClearsStreamerFailingTime(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)

	streamerRepository := physical_repositories.GetWalStreamerRepository()
	t.Cleanup(func() {
		_ = streamerRepository.DeleteByDatabaseID(fixture.DB.ID)
	})

	require.NoError(t, streamerRepository.Claim(fixture.DB.ID))
	require.NoError(t, streamerRepository.MarkFailed(fixture.DB.ID))

	failedStreamer, err := streamerRepository.FindByDatabaseID(fixture.DB.ID)
	require.NoError(t, err)
	require.NotNil(t, failedStreamer.FailingSince)

	createStreamerSlot(t, fixture)

	supervisor := newLagMonitorTickSupervisor(t, fixture, fixture.SourceDB, 0)
	runLagMonitorTicks(
		t,
		supervisor,
		supervisor.newLagMonitorState(logger.GetLogger()),
		lagMonitorTickStartedAt,
		1,
		nil,
	)

	recoveredStreamer, err := streamerRepository.FindByDatabaseID(fixture.DB.ID)
	require.NoError(t, err)
	assert.Nil(t, recoveredStreamer.FailingSince)
}
