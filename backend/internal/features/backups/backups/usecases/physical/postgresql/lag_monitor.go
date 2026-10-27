package usecases_physical_postgresql

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/engines/postgresql/physical/walmath"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	"databasus-backend/internal/features/backups/backups/core/physical/wal_health"
)

const (
	// Beyond this many rebuild ATTEMPTS in a sliding hour (counted regardless of
	// outcome), mechanical retry won't help (creds rotated, pg_hba changed,
	// source dead); stop and surface the condition instead of dropping+recreating
	// in a loop.
	slotRebuildMaxAttemptsPerHour = 3

	// How long to wait for our own pg_receivewal to release the slot during a
	// rebuild before concluding another consumer holds it.
	rebuildReceiverStopTimeout = 30 * time.Second
	rebuildReceiverStopPoll    = 1 * time.Second
)

// NOT a catalog enum: WAL chain breaks are derived from LSN gaps between segment
// rows, never stored — the log carries the human-readable "why".
type walBreakReason string

const (
	breakReasonSlotLost       walBreakReason = "SLOT_LOST"
	breakReasonWalLag         walBreakReason = "WAL_LAG_THRESHOLD"
	breakReasonSlotStolen     walBreakReason = "SLOT_STOLEN"
	breakReasonSlotRetention  walBreakReason = ChainRiskReasonSlotRetention
	breakReasonRotationDenied walBreakReason = ChainRiskReasonRotationDenied
)

type slotBreakAction int

const (
	slotBreakActionNone slotBreakAction = iota
	slotBreakActionAlert
	slotBreakActionRebuild
)

type slotBreakSample struct {
	SlotState            *pgphysical.Slot
	WalLagThresholdBytes int64
	// Zero when no receiver ran at the tick.
	ReceiverRunStartedAt time.Time
	ObservedAt           time.Time
}

type slotBreakClassifier struct {
	warningStatusSince time.Time
	lagWindow          slotLagWindow
}

// What the lag monitor keeps from one tick to the next within a session. The
// trackers decide the notifications, and their start times go into every
// sample, so the samples and the notifications agree about when a condition
// began.
type lagMonitorState struct {
	classifier       slotBreakClassifier
	stalenessTracker walArchiveStalenessTracker

	previousSourceLSN      *walmath.LSN
	sourceUnreachableSince *time.Time
	hasRecordedSample      bool
}

func (m *lagMonitorState) recordProbeOutcome(probe slotProbe, now time.Time) {
	if probe.Outcome != slotProbeOutcomeUnreachable {
		m.sourceUnreachableSince = nil

		return
	}

	if m.sourceUnreachableSince == nil {
		m.sourceUnreachableSince = new(now)
	}
}

// A tick without source-side values forgets the position, so the next sample
// has no write volume instead of one that spans the gap.
func (m *lagMonitorState) rememberSourcePosition(probe slotProbe) {
	m.previousSourceLSN = nil

	if probe.IsReachableWithSlot() {
		m.previousSourceLSN = probe.State.CurrentLSN
	}
}

// Only 'lost' and a stolen slot are unrecoverable enough to justify dropping the
// slot, which always costs a WAL gap and a fresh FULL. 'extended' merely means
// the slot retains WAL beyond max_wal_size — routine on a busy cluster, and
// guaranteed whenever back pressure pauses us — and 'unreserved' means PG will
// trim that WAL at the next checkpoint, which already protects the primary. Both
// alert instead. The operator's WalLagThresholdBytes stays the explicit
// "sacrifice the chain to protect the primary" knob, but fires only when the
// running receiver fails to reduce the lag (slot_lag_window.go), never against
// lag we caused by holding the receiver down ourselves. Lag seen during a
// warning status never counts afterwards.
func (c *slotBreakClassifier) recordSampleAndClassifyBreak(
	sample slotBreakSample,
) (walBreakReason, slotBreakAction) {
	state := sample.SlotState

	if state == nil {
		return "", slotBreakActionNone
	}

	// A foreign backend holding our slot (active, but not one of our own
	// pg_receivewal processes) blocks our receiver from ever attaching. Surface
	// it as SLOT_STOLEN and let the rebuild path decide — terminateOwnedSlotBackend
	// refuses to drop a slot held by a consumer we cannot attribute, so a genuine
	// third party trips loop-protection rather than getting force-dropped.
	if state.IsActive && !isOwnedReceiverBackend(state) {
		return breakReasonSlotStolen, slotBreakActionRebuild
	}

	if state.WalStatus == "lost" {
		c.warningStatusSince = time.Time{}
		c.lagWindow.clear()

		return breakReasonSlotLost, slotBreakActionRebuild
	}

	if state.WalStatus == "extended" || state.WalStatus == "unreserved" {
		c.lagWindow.clear()

		if c.warningStatusSince.IsZero() {
			c.warningStatusSince = sample.ObservedAt
		}

		if sample.ObservedAt.Sub(c.warningStatusSince) > wal_health.WarningSlotStatusHoldPeriod {
			return breakReasonSlotRetention, slotBreakActionAlert
		}

		return "", slotBreakActionNone
	}

	c.warningStatusSince = time.Time{}

	if c.lagWindow.recordSampleAndDetectUnreducedLag(sample) {
		return breakReasonWalLag, slotBreakActionRebuild
	}

	return "", slotBreakActionNone
}

// Source-side slot state only; consumer-side liveness is the slot-LSN watcher's
// job (slot_lsn_watcher.go).
func (s *WalStreamSupervisor) runLagMonitor(ctx context.Context, logger *slog.Logger) {
	ticker := time.NewTicker(wal_health.SampleInterval)
	defer ticker.Stop()

	monitorState := s.newLagMonitorState(logger)

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			s.runLagMonitorTick(ctx, logger, monitorState, time.Now().UTC())
		}
	}
}

// A down source makes every session fail within a minute, so the time it has
// been unreachable is seeded from the newest stored sample instead of starting
// over with each session.
func (s *WalStreamSupervisor) newLagMonitorState(logger *slog.Logger) *lagMonitorState {
	monitorState := &lagMonitorState{}

	if s.spec.WalStreamSampleRepo == nil {
		return monitorState
	}

	newestSample, err := s.spec.WalStreamSampleRepo.FindNewestByDatabase(s.spec.DatabaseID)
	if err != nil {
		logger.Error("failed to read the newest wal stream sample", "error", err)

		return monitorState
	}

	if newestSample != nil &&
		newestSample.SourceState == physical_enums.PhysicalWalStreamSourceStateUnreachable {
		monitorState.sourceUnreachableSince = newestSample.SourceUnreachableSince
	}

	return monitorState
}

// The sample is recorded after classifying and before acting, so a rebuild in
// this tick leaves the slot as it was probed in the sample rather than the fresh
// slot it is replaced with. The upload snapshot resets on read, so it is taken
// once and shared by the log line and the sample.
func (s *WalStreamSupervisor) runLagMonitorTick(
	ctx context.Context,
	logger *slog.Logger,
	monitorState *lagMonitorState,
	now time.Time,
) {
	probe := s.probeSlot(ctx, logger)
	snapshot := s.uploader.TakeArchivingSnapshot()

	newestCommitted, lastCommittedWalAt := s.getLastCommittedWal(logger)
	archiveProgressLSN := getArchiveProgressPoint(s.getLowestQueuedSegment(), newestCommitted)

	var (
		reason walBreakReason
		action slotBreakAction
	)

	if probe.IsReachableWithSlot() {
		state := probe.State

		s.reportArchivingProgress(ctx, logger, state, snapshot)

		s.reportArchiveStalenessIfDue(logger, &monitorState.stalenessTracker, state, walArchiveSample{
			SourceCurrentLSN:     *state.CurrentLSN,
			ArchiveProgressPoint: archiveProgressLSN,
			ObservedAt:           now,
			StalenessThreshold:   s.spec.ArchiveStalenessThreshold,
		}, lastCommittedWalAt)

		reason, action = monitorState.classifier.recordSampleAndClassifyBreak(s.buildSlotBreakSample(state, now))
	}

	monitorState.recordProbeOutcome(probe, now)

	s.recordWalStreamSample(logger, monitorState, walStreamSampleInput{
		DatabaseID:             s.spec.DatabaseID,
		SampledAt:              now,
		SessionStartedAt:       s.sessionStartedAt,
		Probe:                  probe,
		SourceUnreachableSince: monitorState.sourceUnreachableSince,
		PreviousSourceLSN:      monitorState.previousSourceLSN,
		ArchiveProgressLSN:     archiveProgressLSN,
		Snapshot:               snapshot,
		WalSegmentSizeBytes:    walSegmentSizeBytes(s.spec.SourceDB),
		SlotExtraWalSince:      monitorState.classifier.warningStatusSince,
		ArchiveStalledSince:    monitorState.stalenessTracker.fallingBehindSince,
		LocalQueueBytes:        s.backlogBytes(),
		LocalQueuePauseBytes:   s.highWatermarkBytes.Load(),
		IsReceiverRunning:      s.isReceiverRunning(),
		IsForcedRotationDenied: s.isForcedRotationDenied.Load(),
	})

	monitorState.rememberSourcePosition(probe)

	switch action {
	case slotBreakActionNone:
		return

	case slotBreakActionAlert:
		s.reportChainAtRisk(logger, reason, probe.State)

	case slotBreakActionRebuild:
		logger.WarnContext(ctx, "wal stream break observed", "reason", string(reason),
			"lag_bytes", *probe.State.LagBytes, "wal_status", probe.State.WalStatus)

		if err := s.rebuildSlot(ctx, logger, reason); err != nil {
			logger.ErrorContext(ctx, "slot rebuild failed", "reason", string(reason), "error", err)
		}
	}
}

func (s *WalStreamSupervisor) buildSlotBreakSample(state *pgphysical.Slot, now time.Time) slotBreakSample {
	return slotBreakSample{
		SlotState:            state,
		WalLagThresholdBytes: s.spec.WalLagThresholdBytes,
		ReceiverRunStartedAt: s.getReceiverRunStartedAt(),
		ObservedAt:           now,
	}
}

// Without a sample repository, as in tests that drive the streamer alone,
// nothing is recorded and the receiver counters stay for the test to read.
func (s *WalStreamSupervisor) recordWalStreamSample(
	logger *slog.Logger,
	monitorState *lagMonitorState,
	input walStreamSampleInput,
) {
	if s.spec.WalStreamSampleRepo == nil {
		return
	}

	input.ReceiverRestarts, input.ReceiverPauses = s.receiverRespawns.takeCounts()

	if err := s.spec.WalStreamSampleRepo.Insert(buildWalStreamSample(input)); err != nil {
		logger.Error("failed to record a wal stream sample", "error", err)

		return
	}

	if monitorState.hasRecordedSample {
		return
	}

	monitorState.hasRecordedSample = true

	if s.spec.WalStreamerRepo == nil {
		return
	}

	if err := s.spec.WalStreamerRepo.ClearFailingSince(s.spec.DatabaseID); err != nil {
		logger.Error("failed to clear the wal streamer's failing time", "error", err)
	}
}

// Archiving throughput and lag are only visible today once something has already broken, which
// leaves no history to read after an incident. This rides the existing poll rather than adding a
// ticker, and stays at INFO only when segments actually moved. The step timings tell latency-bound
// archiving from bandwidth-bound: a segment write near its size divided by the link speed means
// more uploads at once will not help, a much longer one means the time goes to round trips.
func (s *WalStreamSupervisor) reportArchivingProgress(
	ctx context.Context,
	logger *slog.Logger,
	state *pgphysical.Slot,
	snapshot ArchivingSnapshot,
) {
	if snapshot.ArchivedSegments == 0 {
		logger.DebugContext(ctx, fmt.Sprintf("no wal archived since the last report, %d bytes behind",
			*state.LagBytes), "wal_status", state.WalStatus)

		return
	}

	logger.InfoContext(ctx, fmt.Sprintf(
		"archived %d wal segments (%.1f MB), up to %d uploads at once; "+
			"claim %s, segment write %s, sidecar write %s, commit %s; %d bytes behind",
		snapshot.ArchivedSegments,
		float64(snapshot.ArchivedBytes)/(1024*1024),
		snapshot.HighestConcurrentUploads,
		formatUploadStepTiming(snapshot.Claim, snapshot.ArchivedSegments),
		formatUploadStepTiming(snapshot.SegmentWrite, snapshot.ArchivedSegments),
		formatUploadStepTiming(snapshot.SidecarWrite, snapshot.ArchivedSegments),
		formatUploadStepTiming(snapshot.Commit, snapshot.ArchivedSegments),
		*state.LagBytes,
	), "wal_status", state.WalStatus)
}

func formatUploadStepTiming(timing uploadStepTiming, archivedSegments int64) string {
	return fmt.Sprintf("avg %s max %s",
		timing.GetAverage(archivedSegments).Round(time.Millisecond),
		timing.Longest.Round(time.Millisecond))
}

func (s *WalStreamSupervisor) reportArchiveStalenessIfDue(
	logger *slog.Logger,
	stalenessTracker *walArchiveStalenessTracker,
	state *pgphysical.Slot,
	archiveSample walArchiveSample,
	lastCommittedWalAt *time.Time,
) {
	isArchiveStale := stalenessTracker.recordSampleAndDetectStaleness(archiveSample)

	if !isArchiveStale {
		return
	}

	logger.Warn("wal archiving fell behind the source; the recovery point is no longer advancing",
		"reason", ChainRiskReasonArchiveStale,
		"lag_bytes", *state.LagBytes,
		"wal_status", state.WalStatus,
	)

	if s.spec.OnChainAtRisk == nil {
		return
	}

	s.spec.OnChainAtRisk(ChainRiskReport{
		Reason:            ChainRiskReasonArchiveStale,
		SlotWalStatus:     state.WalStatus,
		LagBytes:          *state.LagBytes,
		LastArchivedWalAt: lastCommittedWalAt,
	})
}

// state is nil for a risk that is not about the slot at all (a refused rotation).
func (s *WalStreamSupervisor) reportChainAtRisk(logger *slog.Logger, reason walBreakReason, state *pgphysical.Slot) {
	report := ChainRiskReport{Reason: string(reason)}

	if state != nil {
		report.SlotWalStatus = state.WalStatus
		report.LagBytes = *state.LagBytes
	}

	logger.Warn(
		fmt.Sprintf("wal chain at risk: slot wal_status=%s, %d bytes behind", report.SlotWalStatus, report.LagBytes),
		"reason", string(reason),
	)

	if s.spec.OnChainAtRisk == nil {
		return
	}

	s.spec.OnChainAtRisk(report)
}
