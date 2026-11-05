package walstreaming

import (
	"context"
	"log/slog"
	"time"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

const (
	slotLsnWatcherPollInterval = 10 * time.Second

	slotLsnStallTimeout = 60 * time.Second

	slotProbeTimeout = 10 * time.Second
)

type slotLivenessSample struct {
	RestartLSN        walmath.LSN
	LagBytes          int64
	IsReceiverRunning bool
	ObservedAt        time.Time
}

type stallTracker struct {
	lastRestartLSN walmath.LSN
	lastAdvanceAt  time.Time
	hasSample      bool
}

func (t *stallTracker) recordSampleAndDetectStall(
	sample slotLivenessSample,
	stallTimeout time.Duration,
) bool {
	if sample.LagBytes == 0 || !sample.IsReceiverRunning {
		t.reset()

		return false
	}

	if !t.hasSample || sample.RestartLSN != t.lastRestartLSN {
		t.lastRestartLSN = sample.RestartLSN
		t.lastAdvanceAt = sample.ObservedAt
		t.hasSample = true

		return false
	}

	if sample.ObservedAt.Sub(t.lastAdvanceAt) > stallTimeout {
		t.lastAdvanceAt = sample.ObservedAt

		return true
	}

	return false
}

func (t *stallTracker) reset() {
	t.hasSample = false
	t.lastRestartLSN = 0
	t.lastAdvanceAt = time.Time{}
}

type slotProbeOutcome int

const (
	slotProbeOutcomeReachable slotProbeOutcome = iota

	slotProbeOutcomeSlotMissing

	slotProbeOutcomeUnreachable
)

type slotProbe struct {
	Outcome slotProbeOutcome
	State   *pgphysical.Slot
}

func (p slotProbe) IsReachableWithSlot() bool {
	return p.Outcome == slotProbeOutcomeReachable
}

func (s *Streamer) probeSlot(ctx context.Context, logger *slog.Logger) slotProbe {
	probe := s.inspectSourceSlot(ctx, logger)
	s.latestProbe.Store(&timedSlotProbe{probe: probe, probedAt: s.spec.Clock.Now()})

	return probe
}

func (s *Streamer) inspectSourceSlot(ctx context.Context, logger *slog.Logger) slotProbe {
	probeCtx, cancel := context.WithTimeout(ctx, slotProbeTimeout)
	defer cancel()

	conn, err := pgphysical.NewSource(s.spec.SourceDB.Engine).OpenInspection(probeCtx)
	if err != nil {
		logger.DebugContext(ctx, "source unreachable during the slot probe", slog.Any("error", err))

		return slotProbe{Outcome: slotProbeOutcomeUnreachable}
	}
	defer func() { _ = conn.Close(context.WithoutCancel(ctx)) }()

	state, err := conn.GetSlot(probeCtx, s.slotName)
	probe := classifySlotInspection(slotInspection{State: state, Error: err})

	if probe.Outcome == slotProbeOutcomeUnreachable {
		inspectionError := err
		if inspectionError == nil {
			inspectionError = errSourceWalPositionUnavailable
		}
		logger.DebugContext(ctx, "failed to inspect the replication slot", slog.Any("error", inspectionError))
		return probe
	}
	if probe.Outcome == slotProbeOutcomeSlotMissing {
		logger.WarnContext(ctx, "the replication slot no longer exists on the source")
		return probe
	}

	if err := conn.Ping(probeCtx); err != nil {
		logger.DebugContext(ctx, "source stopped responding during the slot probe", slog.Any("error", err))

		return slotProbe{Outcome: slotProbeOutcomeUnreachable}
	}

	return slotProbe{Outcome: slotProbeOutcomeReachable, State: state}
}

func (s *Streamer) runSlotLsnWatcher(ctx context.Context, logger *slog.Logger) {
	ticker := time.NewTicker(slotLsnWatcherPollInterval)
	defer ticker.Stop()

	var tracker stallTracker

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			probe := s.probeSlot(ctx, logger)
			if !probe.IsReachableWithSlot() {
				continue
			}

			state := probe.State

			sample := slotLivenessSample{
				RestartLSN:        state.RestartLSN,
				LagBytes:          *state.LagBytes,
				IsReceiverRunning: s.isReceiverRunning(),
				ObservedAt:        s.spec.Clock.Now(),
			}

			if tracker.recordSampleAndDetectStall(sample, slotLsnStallTimeout) {
				logger.WarnContext(
					ctx,
					"slot restart_lsn stalled; restarting pg_receivewal",
					slog.Any("lag_bytes", *state.LagBytes),
					slog.Any("restart_lsn", state.RestartLSN.String()),
				)

				s.signalRestart()
			}
		}
	}
}

type slotInspection struct {
	State *pgphysical.Slot
	Error error
}

func classifySlotInspection(inspection slotInspection) slotProbe {
	if inspection.Error != nil || (inspection.State != nil && inspection.State.CurrentLSN == nil) {
		return slotProbe{Outcome: slotProbeOutcomeUnreachable}
	}
	if inspection.State == nil {
		return slotProbe{Outcome: slotProbeOutcomeSlotMissing}
	}
	return slotProbe{Outcome: slotProbeOutcomeReachable, State: inspection.State}
}
