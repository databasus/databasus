package usecases_physical_postgresql

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

// Only a finalized segment is uploaded, so a database that writes a few bytes an
// hour would keep its newest WAL local until the segment fills — hours or days of
// recovery point sitting on one disk. Forcing a rotation on this cadence bounds
// that, at the cost of one mostly-empty segment per interval while the source is
// being written to. Exported because the wiring layer builds WalStreamSpec.
const DefaultForcedRotationInterval = 5 * time.Minute

const forcedRotationPollInterval = 30 * time.Second

type walRotationSample struct {
	CurrentLSN walmath.LSN
	ObservedAt time.Time
}

type walRotationTracker struct {
	lastRotatedLSN walmath.LSN
	lastRotatedAt  time.Time
}

// A source that has written nothing since the last rotation must not be rotated
// again: otherwise a database that takes one write per day would still archive a
// padded segment every interval. The first sample only starts the clock, so the
// very first rotation waits one interval like every other.
func (t *walRotationTracker) recordSampleAndDecideRotation(
	sample walRotationSample,
	rotationInterval time.Duration,
) (shouldRotate bool) {
	if t.lastRotatedAt.IsZero() {
		t.lastRotatedAt = sample.ObservedAt

		return false
	}

	if sample.CurrentLSN <= t.lastRotatedLSN {
		return false
	}

	return sample.ObservedAt.Sub(t.lastRotatedAt) >= rotationInterval
}

// The post-rotation position, not the position we decided on: pg_switch_wal moves
// the insert point into the new segment, and treating that move as fresh WAL would
// rotate again every interval on an otherwise idle source.
func (t *walRotationTracker) recordRotation(currentLSNAfterRotation walmath.LSN, rotatedAt time.Time) {
	t.lastRotatedLSN = currentLSNAfterRotation
	t.lastRotatedAt = rotatedAt
}

// Rotation is best effort: pg_switch_wal is superuser-only unless EXECUTE is
// granted, while physical backups only require REPLICATION. A refusal is
// permanent, so the loop reports it once and stops instead of polling forever.
func (s *WalStreamSupervisor) runForcedWalRotation(ctx context.Context, logger *slog.Logger) {
	if s.spec.ForcedRotationInterval <= 0 {
		return
	}

	ticker := time.NewTicker(min(forcedRotationPollInterval, s.spec.ForcedRotationInterval))
	defer ticker.Stop()

	var rotationTracker walRotationTracker

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			currentLSN, isSourceReachable := s.getCurrentWalLsnIfReachable(ctx, logger)
			if !isSourceReachable {
				continue
			}

			isRotationDue := rotationTracker.recordSampleAndDecideRotation(walRotationSample{
				CurrentLSN: currentLSN,
				ObservedAt: time.Now().UTC(),
			}, s.spec.ForcedRotationInterval)

			if !isRotationDue {
				continue
			}

			currentLSNAfterRotation, err := s.forceWalRotation(ctx)
			if err != nil {
				if !errors.Is(err, pgphysical.ErrWalSwitchDenied) {
					logger.WarnContext(ctx, "forced wal rotation failed; will retry", "error", err)

					continue
				}

				logger.WarnContext(ctx, "source refuses pg_switch_wal; forced wal rotation disabled for this streamer",
					"error", err)

				s.isForcedRotationDenied.Store(true)

				s.reportChainAtRisk(logger, breakReasonRotationDenied, nil)

				return
			}

			rotationTracker.recordRotation(currentLSNAfterRotation, time.Now().UTC())

			logger.DebugContext(ctx, "forced a wal segment switch so the recovery point keeps advancing")
		}
	}
}

func (s *WalStreamSupervisor) getCurrentWalLsnIfReachable(
	ctx context.Context,
	logger *slog.Logger,
) (currentLSN walmath.LSN, isSourceReachable bool) {
	conn, err := pgphysical.NewSource(s.spec.SourceDB.Engine).OpenInspection(ctx)
	if err != nil {
		logger.DebugContext(ctx, "forced wal rotation: source unreachable this tick", "error", err)

		return 0, false
	}
	defer func() { _ = conn.Close(context.Background()) }()

	currentLSN, err = conn.GetCurrentWalLSN(ctx)
	if err != nil {
		return 0, false
	}

	return currentLSN, true
}

func (s *WalStreamSupervisor) forceWalRotation(ctx context.Context) (currentLSNAfterRotation walmath.LSN, err error) {
	conn, err := pgphysical.NewSource(s.spec.SourceDB.Engine).OpenInspection(ctx)
	if err != nil {
		return 0, fmt.Errorf("open conn for wal switch: %w", err)
	}
	defer func() { _ = conn.Close(context.Background()) }()

	return conn.SwitchWal(ctx)
}
