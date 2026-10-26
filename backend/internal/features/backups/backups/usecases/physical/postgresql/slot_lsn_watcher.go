package usecases_physical_postgresql

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"databasus-backend/internal/util/walmath"
)

const (
	// Tighter than the lag monitor because it tests whether OUR pg_receivewal is
	// actually flushing.
	slotLsnWatcherPollInterval = 10 * time.Second

	// restart_lsn unchanged this long while WAL is waiting means pg_receivewal is
	// stuck (alive but flushing nothing); restart it locally.
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

// A frozen restart_lsn is only a stall when the source has WAL we have not taken
// yet and a receiver is up to take it: an idle database legitimately parks
// restart_lsn, and so does a receiver we are deliberately holding down for a
// rebuild or back pressure. A positive result re-arms the clock, so the caller
// restarts at most once per stallTimeout window.
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

	// The source answered, but our slot is gone from it, so WAL is no longer
	// retained for us and the chain is already broken.
	slotProbeOutcomeSlotMissing

	slotProbeOutcomeUnreachable
)

// Only a reachable probe carries a slot state.
type slotProbe struct {
	Outcome slotProbeOutcome
	State   *SlotState
}

func (p slotProbe) IsReachableWithSlot() bool {
	return p.Outcome == slotProbeOutcomeReachable
}

// Every probe is remembered with its time, so the live state can show the
// source as of the newest probe without connecting to it again.
func (s *WalStreamSupervisor) probeSlot(ctx context.Context, logger *slog.Logger) slotProbe {
	probe := s.inspectSourceSlot(ctx, logger)
	s.latestProbe.Store(&timedSlotProbe{probe: probe, probedAt: time.Now().UTC()})

	return probe
}

// A source that accepts connections but never answers would otherwise hold
// every caller's tick for as long as the TCP connection lives.
func (s *WalStreamSupervisor) inspectSourceSlot(ctx context.Context, logger *slog.Logger) slotProbe {
	probeCtx, cancel := context.WithTimeout(ctx, slotProbeTimeout)
	defer cancel()

	conn, err := openInspectionConn(probeCtx, s.spec.SourceDB)
	if err != nil {
		logger.DebugContext(ctx, "source unreachable during the slot probe", "error", err)

		return slotProbe{Outcome: slotProbeOutcomeUnreachable}
	}
	defer func() { _ = conn.Close(context.Background()) }()

	state, err := InspectSlot(probeCtx, conn, s.slotName)
	if err != nil {
		logger.DebugContext(ctx, "failed to inspect the replication slot", "error", err)

		return slotProbe{Outcome: slotProbeOutcomeUnreachable}
	}

	if state == nil {
		logger.WarnContext(ctx, "the replication slot no longer exists on the source")

		return slotProbe{Outcome: slotProbeOutcomeSlotMissing}
	}

	var alive int
	if err := conn.QueryRow(probeCtx, "SELECT 1").Scan(&alive); err != nil {
		logger.DebugContext(ctx, "source stopped responding during the slot probe", "error", err)

		return slotProbe{Outcome: slotProbeOutcomeUnreachable}
	}

	return slotProbe{Outcome: slotProbeOutcomeReachable, State: state}
}

// A stall with an unreachable server is left to the lag monitor (slot loss /
// network down).
func (s *WalStreamSupervisor) runSlotLsnWatcher(ctx context.Context, logger *slog.Logger) {
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
				LagBytes:          state.LagBytes,
				IsReceiverRunning: s.isReceiverRunning(),
				ObservedAt:        time.Now().UTC(),
			}

			if tracker.recordSampleAndDetectStall(sample, slotLsnStallTimeout) {
				logger.WarnContext(
					ctx,
					fmt.Sprintf("slot restart_lsn stalled with %d bytes pending; restarting pg_receivewal",
						state.LagBytes),
					"restart_lsn", state.RestartLSN.String(),
				)

				s.signalRestart()
			}
		}
	}
}
