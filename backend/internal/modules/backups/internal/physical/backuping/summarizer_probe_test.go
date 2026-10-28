package backuping

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

func TestResolveSummarizerProbe(t *testing.T) {
	for _, testCase := range []struct {
		name             string
		decisions        []SummarizerDecision
		expectedDecision SummarizerDecision
		reason           *catalog.BackupErrorReason
	}{
		{"proceeds when summaries catch up during the wait", []SummarizerDecision{DecisionWait, DecisionWait, DecisionGoIncremental}, DecisionGoIncremental, nil},
		{"retries on the next cadence when the wait expires", []SummarizerDecision{DecisionWait}, DecisionRetryNextCadence, nil},
		{"starts a new chain when summaries expire during the wait", []SummarizerDecision{DecisionWait, DecisionFullNewChain}, DecisionFullNewChain, new(catalog.BackupErrorSummariesExpired)},
		{"returns immediately when the first probe is terminal", []SummarizerDecision{DecisionFullNewChain}, DecisionFullNewChain, new(catalog.BackupErrorSummarizerOff)},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			manualClock := clock.NewManual(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
			probeCount := 0
			spec := summarizerWaitSpec{
				Clock: manualClock,
				Probe: func(context.Context) (SummarizerResult, error) {
					decision := testCase.decisions[min(probeCount, len(testCase.decisions)-1)]
					probeCount++
					manualClock.Advance(time.Minute)
					return SummarizerResult{
						Decision:  decision,
						WaitFor:   2 * time.Minute,
						PollEvery: time.Nanosecond,
						Reason:    testCase.reason,
					}, nil
				},
			}

			resolved, err := resolveSummarizerProbe(t.Context(), spec)

			require.NoError(t, err)
			assert.Equal(t, testCase.expectedDecision, resolved.Decision)
			if testCase.expectedDecision == DecisionFullNewChain {
				assert.Equal(t, testCase.reason, resolved.Reason)
			}
			if testCase.decisions[0] != DecisionWait {
				assert.Equal(t, 1, probeCount)
			}
		})
	}
	t.Run("returns cancellation when its context ends during a wait", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		_, err := resolveSummarizerProbe(ctx, summarizerWaitSpec{
			Clock: clock.NewManual(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
			Probe: func(context.Context) (SummarizerResult, error) {
				return SummarizerResult{Decision: DecisionWait, WaitFor: time.Hour, PollEvery: time.Hour}, nil
			},
		})
		assert.ErrorIs(t, err, context.Canceled)
	})
	t.Run("returns a probe failure when the source cannot be inspected", func(t *testing.T) {
		sourceError := errors.New("inspection failed")
		_, err := resolveSummarizerProbe(t.Context(), summarizerWaitSpec{
			Clock: clock.NewManual(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
			Probe: func(context.Context) (SummarizerResult, error) { return SummarizerResult{}, sourceError },
		})
		assert.ErrorIs(t, err, sourceError)
	})
}
