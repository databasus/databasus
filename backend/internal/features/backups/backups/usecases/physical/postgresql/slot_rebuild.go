package usecases_physical_postgresql

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
)

func (s *WalStreamSupervisor) rebuildSlot(ctx context.Context, logger *slog.Logger, reason walBreakReason) error {
	s.rebuildMu.Lock()
	defer s.rebuildMu.Unlock()

	if !s.recordRebuildAttemptWithinCap() {
		// Repeated rebuilds each open a WAL gap, so hitting the cap means the chain is degrading and
		// the cause is upstream of this streamer.
		logger.ErrorContext(ctx, fmt.Sprintf(
			"rebuild loop-protection tripped: more than %d rebuilds in the last hour",
			slotRebuildMaxAttemptsPerHour), "reason", string(reason))

		return fmt.Errorf(
			"rebuild loop-protection tripped: more than %d rebuilds in the last hour", slotRebuildMaxAttemptsPerHour,
		)
	}

	logger.InfoContext(ctx, "starting slot rebuild", "reason", string(reason))

	s.isPaused.Store(true)
	defer s.isPaused.Store(false)

	s.signalRestart()

	released, err := s.waitForSlotReleased(ctx, logger)
	if err != nil {
		return fmt.Errorf("wait for our receiver to release slot: %w", err)
	}

	if !released {
		terminated, terminateErr := s.terminateOwnedSlotBackend(ctx, logger)
		if terminateErr != nil {
			return fmt.Errorf("terminate owned slot backend: %w", terminateErr)
		}

		if !terminated {
			logger.WarnContext(ctx,
				"refusing to rebuild: the slot is held by a consumer that is not ours")

			return errors.New(
				"slot still active after stopping our receiver; refusing to drop a slot held by another consumer",
			)
		}

		released, err = s.waitForSlotReleased(ctx, logger)
		if err != nil {
			return fmt.Errorf("wait for terminated receiver to release slot: %w", err)
		}

		if !released {
			return errors.New("owned slot backend did not release the slot after termination")
		}
	}

	if err := pgphysical.NewSource(s.spec.SourceDB.Engine).
		DropSlot(ctx, logger, s.spec.SourceDB.WalStreamSlotName); err != nil {
		return fmt.Errorf("drop slot: %w", err)
	}

	if err := s.createSlotUnlessDatabaseDeleted(ctx, logger); err != nil {
		return fmt.Errorf("recreate slot: %w", err)
	}

	// The new slot reserves from the cluster's current WAL position, so the queue
	// we just stopped writing to now sits below it and would drag pg_receivewal's
	// resume point into recycled WAL.
	s.realignResumePath(ctx, logger)

	if s.spec.OnSlotRebuilt != nil {
		if err := s.spec.OnSlotRebuilt(ctx, string(reason)); err != nil {
			return fmt.Errorf("request full after slot rebuild: %w", err)
		}
	}

	logger.InfoContext(ctx, "slot rebuild complete; resuming pg_receivewal on fresh slot")

	return nil
}

// A slot held by anything else (a third-party consumer, or a backend we cannot
// attribute) must never be force-terminated or dropped during a rebuild.
func isOwnedReceiverBackend(state *pgphysical.Slot) bool {
	return state.ActivePID != nil &&
		strings.HasPrefix(state.ApplicationName, pgphysical.WalReceiverApplicationNamePrefix)
}

func (s *WalStreamSupervisor) terminateOwnedSlotBackend(ctx context.Context, logger *slog.Logger) (bool, error) {
	conn, err := pgphysical.NewSource(s.spec.SourceDB.Engine).OpenInspection(ctx)
	if err != nil {
		return false, fmt.Errorf("open inspection connection: %w", err)
	}
	defer func() { _ = conn.Close(context.Background()) }()

	state, err := conn.GetSlot(ctx, s.slotName)
	if err == nil && state != nil && state.CurrentLSN == nil {
		err = errSourceWalPositionUnavailable
	}
	if err != nil {
		return false, err
	}

	if state == nil || !state.IsActive {
		return true, nil
	}

	if !isOwnedReceiverBackend(state) {
		return false, nil
	}

	terminated, err := conn.TerminateBackend(ctx, *state.ActivePID)
	if err != nil {
		return false, err
	}

	logger.WarnContext(ctx, "terminated Databasus-owned wal receiver backend", "active_pid", *state.ActivePID)

	return terminated, nil
}

func (s *WalStreamSupervisor) waitForSlotReleased(ctx context.Context, logger *slog.Logger) (bool, error) {
	deadline := time.Now().UTC().Add(rebuildReceiverStopTimeout)

	for {
		active, ok := s.isSlotActive(ctx, logger)
		if ok && !active {
			return true, nil
		}

		if time.Now().UTC().After(deadline) {
			return ok && !active, nil
		}

		if !sleepCtx(ctx, rebuildReceiverStopPoll) {
			return false, ctx.Err()
		}
	}
}

func (s *WalStreamSupervisor) isSlotActive(ctx context.Context, logger *slog.Logger) (active, ok bool) {
	conn, err := pgphysical.NewSource(s.spec.SourceDB.Engine).OpenInspection(ctx)
	if err != nil {
		logger.DebugContext(ctx, "rebuild: source unreachable while waiting for slot release", "error", err)

		return false, false
	}
	defer func() { _ = conn.Close(context.Background()) }()

	state, err := conn.GetSlot(ctx, s.slotName)
	if err == nil && state != nil && state.CurrentLSN == nil {
		err = errSourceWalPositionUnavailable
	}
	if err != nil {
		return false, false
	}

	if state == nil {
		return false, true
	}

	return state.IsActive, true
}

func (s *WalStreamSupervisor) recordRebuildAttemptWithinCap() bool {
	now := time.Now().UTC()
	cutoff := now.Add(-time.Hour)

	kept := s.rebuildTimestamps[:0]
	for _, ts := range s.rebuildTimestamps {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}

	s.rebuildTimestamps = kept

	if len(s.rebuildTimestamps) >= slotRebuildMaxAttemptsPerHour {
		return false
	}

	s.rebuildTimestamps = append(s.rebuildTimestamps, now)

	return true
}

func (s *WalStreamSupervisor) GetSlotRebuildTimestamps() []time.Time {
	s.rebuildMu.Lock()
	defer s.rebuildMu.Unlock()

	return slices.Clone(s.rebuildTimestamps)
}
