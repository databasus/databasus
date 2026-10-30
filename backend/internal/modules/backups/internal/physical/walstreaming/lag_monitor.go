package walstreaming

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/engines/postgresql/physical/walmath"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/backups/internal/physical/streamhealth"
)

const (
	slotRebuildMaxAttemptsPerHour = 3

	rebuildReceiverStopTimeout = 30 * time.Second
	rebuildReceiverStopPoll    = 1 * time.Second
)

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

	ReceiverRunStartedAt time.Time
	ObservedAt           time.Time
}

type slotBreakClassifier struct {
	warningStatusSince time.Time
	lagWindow          slotLagWindow
}

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

func (m *lagMonitorState) rememberSourcePosition(probe slotProbe) {
	m.previousSourceLSN = nil

	if probe.IsReachableWithSlot() {
		m.previousSourceLSN = probe.State.CurrentLSN
	}
}

func (c *slotBreakClassifier) recordSampleAndClassifyBreak(
	sample slotBreakSample,
) (walBreakReason, slotBreakAction) {
	state := sample.SlotState

	if state == nil {
		return "", slotBreakActionNone
	}

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

		if sample.ObservedAt.Sub(c.warningStatusSince) > streamhealth.WarningSlotStatusHoldPeriod {
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

func (s *Streamer) runLagMonitor(ctx context.Context, logger *slog.Logger) {
	ticker := time.NewTicker(streamhealth.SampleInterval)
	defer ticker.Stop()

	monitorState := s.newLagMonitorState(ctx, logger)

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			s.runLagMonitorTick(ctx, logger, monitorState, s.spec.Clock.Now())
		}
	}
}

func (s *Streamer) newLagMonitorState(ctx context.Context, logger *slog.Logger) *lagMonitorState {
	monitorState := &lagMonitorState{}

	if s.spec.WalStreamSampleRepo == nil {
		return monitorState
	}

	newestSample, _, err := s.spec.WalStreamSampleRepo.FindNewestByDatabase(ctx, s.spec.DatabaseID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to read the newest wal stream sample", slog.Any("error", err))

		return monitorState
	}

	if newestSample != nil &&
		newestSample.SourceState == catalog.WalStreamSourceStateUnreachable {
		monitorState.sourceUnreachableSince = newestSample.SourceUnreachableSince
	}

	return monitorState
}

func (s *Streamer) runLagMonitorTick(
	ctx context.Context,
	logger *slog.Logger,
	monitorState *lagMonitorState,
	now time.Time,
) {
	probe := s.probeSlot(ctx, logger)
	snapshot := s.uploader.TakeArchivingSnapshot()

	newestCommitted, lastCommittedWalAt := s.getLastCommittedWal(ctx, logger)
	archiveProgressLSN := getArchiveProgressPoint(s.getLowestQueuedSegment(), newestCommitted)

	var (
		reason walBreakReason
		action slotBreakAction
	)

	if probe.IsReachableWithSlot() {
		state := probe.State

		s.reportArchivingProgress(ctx, logger, state, snapshot)

		s.reportArchiveStalenessIfDue(ctx, logger, &monitorState.stalenessTracker, state, walArchiveSample{
			SourceCurrentLSN:     *state.CurrentLSN,
			ArchiveProgressPoint: archiveProgressLSN,
			ObservedAt:           now,
			StalenessThreshold:   s.spec.ArchiveStalenessThreshold,
		}, lastCommittedWalAt)

		reason, action = monitorState.classifier.recordSampleAndClassifyBreak(s.buildSlotBreakSample(state, now))
	}

	monitorState.recordProbeOutcome(probe, now)

	s.recordWalStreamSample(ctx, logger, monitorState, walStreamSampleInput{
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
		s.reportChainAtRisk(ctx, logger, reason, probe.State)

	case slotBreakActionRebuild:
		logger.WarnContext(
			ctx,
			"wal stream break observed",
			slog.Any("reason", string(reason)),
			slog.Any("lag_bytes", *probe.State.LagBytes),
			slog.Any("wal_status", probe.State.WalStatus),
		)

		if err := s.rebuildSlot(ctx, logger, reason); err != nil {
			logger.ErrorContext(ctx, "slot rebuild failed", slog.Any("reason", string(reason)), slog.Any("error", err))
		}
	}
}

func (s *Streamer) buildSlotBreakSample(state *pgphysical.Slot, now time.Time) slotBreakSample {
	return slotBreakSample{
		SlotState:            state,
		WalLagThresholdBytes: s.spec.WalLagThresholdBytes,
		ReceiverRunStartedAt: s.getReceiverRunStartedAt(),
		ObservedAt:           now,
	}
}

func (s *Streamer) recordWalStreamSample(ctx context.Context,
	logger *slog.Logger,
	monitorState *lagMonitorState,
	input walStreamSampleInput,
) {
	if s.spec.WalStreamSampleRepo == nil {
		return
	}

	input.ReceiverRestarts, input.ReceiverPauses = s.receiverRespawns.takeCounts()

	if err := s.spec.Transactor.Transaction(ctx, func(ctx context.Context) error {
		if err := s.spec.Publication.Validate(ctx); err != nil {
			return err
		}
		return s.spec.WalStreamSampleRepo.Insert(ctx, buildWalStreamSample(input))
	}); err != nil {
		logger.ErrorContext(ctx, "failed to record a wal stream sample", slog.Any("error", err))

		return
	}

	if monitorState.hasRecordedSample {
		return
	}

	monitorState.hasRecordedSample = true

	if s.spec.WalStreamerRepo == nil {
		return
	}

	if err := s.spec.WalStreamerRepo.ClearFailingSince(ctx, s.spec.ClaimID); err != nil {
		logger.ErrorContext(ctx, "failed to clear the wal streamer's failing time", slog.Any("error", err))
	}
}

func (s *Streamer) reportArchivingProgress(
	ctx context.Context,
	logger *slog.Logger,
	state *pgphysical.Slot,
	snapshot ArchivingSnapshot,
) {
	if snapshot.ArchivedSegments == 0 {
		logger.DebugContext(
			ctx,
			"no wal archived since the last report",
			slog.Any("lag_bytes", *state.LagBytes),
			slog.Any("wal_status", state.WalStatus),
		)

		return
	}

	logger.InfoContext(
		ctx,
		"wal archiving progress",
		slog.Any("archived_segments", snapshot.ArchivedSegments),
		slog.Any("archived_mb", float64(snapshot.ArchivedBytes)/(1024*1024)),
		slog.Any("highest_concurrent_uploads", snapshot.HighestConcurrentUploads),
		slog.Any("claim_timing", formatUploadStepTiming(snapshot.Claim, snapshot.ArchivedSegments)),
		slog.Any("segment_write_timing", formatUploadStepTiming(snapshot.SegmentWrite, snapshot.ArchivedSegments)),
		slog.Any("sidecar_write_timing", formatUploadStepTiming(snapshot.SidecarWrite, snapshot.ArchivedSegments)),
		slog.Any("commit_timing", formatUploadStepTiming(snapshot.Commit, snapshot.ArchivedSegments)),
		slog.Any("lag_bytes", *state.LagBytes),
		slog.Any("wal_status", state.WalStatus),
	)
}

func formatUploadStepTiming(timing uploadStepTiming, archivedSegments int64) string {
	return fmt.Sprintf("avg %s max %s",
		timing.GetAverage(archivedSegments).Round(time.Millisecond),
		timing.Longest.Round(time.Millisecond))
}

func (s *Streamer) reportArchiveStalenessIfDue(ctx context.Context,
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

	logger.WarnContext(
		ctx,
		"wal archiving fell behind the source; the recovery point is no longer advancing",
		slog.Any("reason", ChainRiskReasonArchiveStale),
		slog.Any("lag_bytes", *state.LagBytes),
		slog.Any("wal_status", state.WalStatus),
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

func (s *Streamer) reportChainAtRisk(
	ctx context.Context,
	logger *slog.Logger,
	reason walBreakReason,
	state *pgphysical.Slot,
) {
	report := ChainRiskReport{Reason: string(reason)}

	if state != nil {
		report.SlotWalStatus = state.WalStatus
		report.LagBytes = *state.LagBytes
	}

	logger.WarnContext(
		ctx,
		"wal chain at risk",
		slog.Any("wal_status", report.SlotWalStatus),
		slog.Any("lag_bytes", report.LagBytes),
		slog.Any("reason", string(reason)),
	)

	if s.spec.OnChainAtRisk == nil {
		return
	}

	s.spec.OnChainAtRisk(report)
}
