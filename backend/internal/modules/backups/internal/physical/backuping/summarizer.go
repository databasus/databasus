package backuping

import (
	"context"
	"time"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/engines/postgresql/physical/walmath"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

// SummarizerDecision encodes the outcome of a per-tick incremental pre-check.
// The caller maps each value to one of: run the INCR, poll and recheck, abandon
// this attempt so the INCR cadence brings the next one, or spawn a FULL
// anchoring a new chain.
type SummarizerDecision int

const (
	DecisionGoIncremental SummarizerDecision = iota
	DecisionWait
	DecisionRetryNextCadence
	DecisionFullNewChain
)

// SummarizerResult carries the decision plus the inputs the caller needs to
// act on it: wait/poll cadence for DecisionWait, error_reason for the
// chain-killing DecisionFullNewChain branches.
type SummarizerResult struct {
	Decision  SummarizerDecision
	WaitFor   time.Duration
	PollEvery time.Duration
	Reason    *catalog.BackupErrorReason
}

const (
	// summarizerWaitPollInterval bounds how often the bounded wait re-probes.
	// Five seconds is tight enough that recovery is felt quickly, loose enough
	// that we don't hammer the source catalog.
	summarizerWaitPollInterval = 5 * time.Second

	// summarizerWaitCap bounds the DecisionWait timeout, which is
	// min(cadence/4, this).
	summarizerWaitCap = 30 * time.Minute
)

// CheckSummarizerReadiness classifies the state of the WAL summarizer relative to
// prevStopLSN (the parent backup's stop_lsn, or — for the WAL-gap fallback
// path — the current LSN). The conn must be an ordinary, non-replication
// connection.
//
// WAL summary files close only at checkpoint records, so a summarizer that has
// read every byte still publishes nothing until the next checkpoint.
// pg_basebackup --incremental forces its own checkpoint at backup start, which
// closes a summary file at exactly the LSN it then waits for, so how far the
// published summaries trail current WAL never decides the outcome. Liveness is
// all that is worth probing here: a summarizer that absorbs no WAL record for a
// minute is caught by PostgreSQL itself, which fails the backup with
// "WAL summarization is not progressing".
func CheckSummarizerReadiness(
	ctx context.Context,
	conn *pgphysical.Inspection,
	prevStopLSN walmath.LSN,
	incrementalCadence time.Duration,
) (SummarizerResult, error) {
	state, err := conn.GetWalSummarizerState(ctx)
	if err != nil {
		return SummarizerResult{}, err
	}

	if !state.IsEnabled {
		reason := catalog.BackupErrorSummarizerOff

		return SummarizerResult{
			Decision: DecisionFullNewChain,
			Reason:   &reason,
		}, nil
	}

	// A parent stop_lsn below the oldest retained summary has aged out for good — no
	// future summary will ever cover it, so the chain must re-anchor on a fresh FULL.
	if state.OldestStartLSN != nil && prevStopLSN < *state.OldestStartLSN {
		reason := catalog.BackupErrorSummariesExpired

		return SummarizerResult{
			Decision: DecisionFullNewChain,
			Reason:   &reason,
		}, nil
	}

	// Summarizer on but nothing published yet: not expiry, just not ready.
	if state.OldestStartLSN == nil {
		return newWaitResult(incrementalCadence), nil
	}

	if !state.IsRunning {
		return newWaitResult(incrementalCadence), nil
	}

	return SummarizerResult{Decision: DecisionGoIncremental}, nil
}

func newWaitResult(incrementalCadence time.Duration) SummarizerResult {
	return SummarizerResult{
		Decision:  DecisionWait,
		WaitFor:   waitWindowForCadence(incrementalCadence),
		PollEvery: summarizerWaitPollInterval,
	}
}

// resolveSummarizerDecision runs the readiness probe and, when it reports
// DecisionWait (no summaries published yet, or the summarizer process is gone),
// polls until the state resolves or the bounded window elapses. The returned
// decision is never DecisionWait: a window that expires unresolved collapses to
// DecisionRetryNextCadence, so the caller fails this attempt instead of racing a
// summarizer that is not coming back within the tick.
func resolveSummarizerDecision(
	ctx context.Context,
	conn *pgphysical.Inspection,
	prevStopLSN walmath.LSN,
	cadence time.Duration,
	sourceClock clock.Source,
) (SummarizerResult, error) {
	return resolveSummarizerProbe(ctx, summarizerWaitSpec{
		Probe: func(probeCtx context.Context) (SummarizerResult, error) {
			return CheckSummarizerReadiness(probeCtx, conn, prevStopLSN, cadence)
		},
		Clock: sourceClock,
	})
}

type summarizerWaitSpec struct {
	Probe func(context.Context) (SummarizerResult, error)
	Clock clock.Source
}

func resolveSummarizerProbe(ctx context.Context, spec summarizerWaitSpec) (SummarizerResult, error) {
	result, err := spec.Probe(ctx)
	if err != nil {
		return SummarizerResult{}, err
	}
	if result.Decision != DecisionWait {
		return result, nil
	}
	return waitForSummarizer(ctx, spec, result)
}

// waitForSummarizer polls the readiness probe on result.PollEvery until the
// summarizer reaches a terminal decision or result.WaitFor elapses. It never
// writes any catalog state — the INCR row stays IN_PROGRESS throughout, so a
// recovery within the window proceeds to a normal incremental with no
// intermediate failure row.
func waitForSummarizer(
	ctx context.Context,
	spec summarizerWaitSpec,
	initial SummarizerResult,
) (SummarizerResult, error) {
	deadline := spec.Clock.Now().Add(initial.WaitFor)

	ticker := time.NewTicker(initial.PollEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return SummarizerResult{}, ctx.Err()

		case <-ticker.C:
			result, err := spec.Probe(ctx)
			if err != nil {
				return SummarizerResult{}, err
			}

			if result.Decision != DecisionWait {
				return result, nil
			}

			if spec.Clock.Now().After(deadline) {
				return SummarizerResult{Decision: DecisionRetryNextCadence}, nil
			}
		}
	}
}

func waitWindowForCadence(cadence time.Duration) time.Duration {
	// ApproxPeriod returns 0 for cron intervals; a zero window would skip the
	// wait entirely.
	if cadence == 0 {
		return summarizerWaitCap
	}

	quarter := cadence / 4
	if quarter > summarizerWaitCap {
		return summarizerWaitCap
	}

	return quarter
}
