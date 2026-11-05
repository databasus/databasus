package walstreaming

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

const forcedRotationPollInterval = 30 * time.Second

type walRotationSample struct {
	CurrentLSN walmath.LSN
	ObservedAt time.Time
}

type walRotationTracker struct {
	lastRotatedLSN walmath.LSN
	lastRotatedAt  time.Time
}

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

func (t *walRotationTracker) recordRotation(currentLSNAfterRotation walmath.LSN, rotatedAt time.Time) {
	t.lastRotatedLSN = currentLSNAfterRotation
	t.lastRotatedAt = rotatedAt
}

func (s *Streamer) runForcedWalRotation(ctx context.Context, logger *slog.Logger) {
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
				ObservedAt: s.spec.Clock.Now(),
			}, s.spec.ForcedRotationInterval)

			if !isRotationDue {
				continue
			}

			currentLSNAfterRotation, err := s.forceWalRotation(ctx)
			if err != nil {
				if !errors.Is(err, pgphysical.ErrWalSwitchDenied) {
					logger.WarnContext(ctx, "forced wal rotation failed; will retry", slog.Any("error", err))

					continue
				}

				logger.WarnContext(
					ctx,
					"source refuses pg_switch_wal; forced wal rotation disabled for this streamer",
					slog.Any("error", err),
				)

				s.isForcedRotationDenied.Store(true)

				s.reportChainAtRisk(ctx, logger, breakReasonRotationDenied, nil)

				return
			}

			rotationTracker.recordRotation(currentLSNAfterRotation, s.spec.Clock.Now())

			logger.DebugContext(ctx, "forced a wal segment switch so the recovery point keeps advancing")
		}
	}
}

func (s *Streamer) getCurrentWalLsnIfReachable(
	ctx context.Context,
	logger *slog.Logger,
) (currentLSN walmath.LSN, isSourceReachable bool) {
	conn, err := pgphysical.NewSource(s.spec.SourceDB.Engine).OpenInspection(ctx)
	if err != nil {
		logger.DebugContext(ctx, "forced wal rotation: source unreachable this tick", slog.Any("error", err))

		return 0, false
	}
	defer func() { _ = conn.Close(context.WithoutCancel(ctx)) }()

	currentLSN, err = conn.GetCurrentWalLSN(ctx)
	if err != nil {
		return 0, false
	}

	return currentLSN, true
}

func (s *Streamer) forceWalRotation(ctx context.Context) (currentLSNAfterRotation walmath.LSN, err error) {
	conn, err := pgphysical.NewSource(s.spec.SourceDB.Engine).OpenInspection(ctx)
	if err != nil {
		return 0, fmt.Errorf("open conn for wal switch: %w", err)
	}
	defer func() { _ = conn.Close(context.WithoutCancel(ctx)) }()

	return conn.SwitchWal(ctx)
}
