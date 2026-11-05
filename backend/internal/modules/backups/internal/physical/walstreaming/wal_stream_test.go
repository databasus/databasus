package walstreaming

import (
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const instantReceiverExit = 100 * time.Millisecond

func Test_RecordMismatchAndDecideEscalation_WhenFirstMismatch_RetriesAfterRealign(t *testing.T) {
	var mismatchEscalator resumeMismatchEscalator

	require.Equal(t, resumeMismatchActionRetry, mismatchEscalator.recordMismatchAndDecideEscalation(),
		"a realign clears the resume path, so the next spawn deserves a chance before the chain is broken")
}

func Test_RecordMismatchAndDecideEscalation_WhenSecondMismatch_RequestsSlotRebuild(t *testing.T) {
	var mismatchEscalator resumeMismatchEscalator

	require.Equal(t, resumeMismatchActionRetry, mismatchEscalator.recordMismatchAndDecideEscalation())
	require.Equal(t, resumeMismatchActionRebuildSlot, mismatchEscalator.recordMismatchAndDecideEscalation(),
		"a realigned queue that still asks for recycled WAL means the slot itself no longer covers the source")
}

func Test_RecordMismatchAndDecideEscalation_WhenHealthyRunBetweenMismatches_ResetsCounter(t *testing.T) {
	var mismatchEscalator resumeMismatchEscalator

	require.Equal(t, resumeMismatchActionRetry, mismatchEscalator.recordMismatchAndDecideEscalation())

	mismatchEscalator.reset()

	require.Equal(t, resumeMismatchActionRetry, mismatchEscalator.recordMismatchAndDecideEscalation(),
		"mismatches must be back-to-back to escalate; a healthy run in between clears the incident")
}

func Test_RecordMismatchAndDecideEscalation_AfterRebuild_StartsCountingFromScratch(t *testing.T) {
	var mismatchEscalator resumeMismatchEscalator

	mismatchEscalator.recordMismatchAndDecideEscalation()
	require.Equal(t, resumeMismatchActionRebuildSlot, mismatchEscalator.recordMismatchAndDecideEscalation())

	require.Equal(t, resumeMismatchActionRetry, mismatchEscalator.recordMismatchAndDecideEscalation(),
		"a rebuild is expensive, so the next one needs its own pair of mismatches")
}

func fatalReceiverRun() receiverRunResult {
	return receiverRunResult{
		Exit:     receiverFatal,
		RanFor:   instantReceiverExit,
		FatalErr: errors.New("pg_receivewal fatal error: could not write 8192 bytes to WAL file"),
	}
}

func Test_DemoteFatalExitWhenBastionUnreachable_WhenTheBastionIsUnreachable_TurnsTheExitRetryable(
	t *testing.T,
) {
	demotedRun := demoteFatalExitWhenBastionUnreachable(
		t.Context(),
		slog.New(slog.DiscardHandler),
		fatalReceiverRun(),
		false,
	)

	assert.Equal(t, receiverRetryable, demotedRun.Exit)
}

func Test_DemoteFatalExitWhenBastionUnreachable_WhenTheBastionIsReachable_LeavesTheExitFatal(
	t *testing.T,
) {
	fatalRun := fatalReceiverRun()
	judgedRun := demoteFatalExitWhenBastionUnreachable(t.Context(), slog.New(slog.DiscardHandler), fatalRun, true)

	assert.Equal(t, receiverFatal, judgedRun.Exit)
	assert.Equal(t, fatalRun.FatalErr, judgedRun.FatalErr)
}

func Test_DemoteFatalExitWhenBastionUnreachable_WhenTheExitIsNotFatal_LeavesTheExitUnchanged(
	t *testing.T,
) {
	for _, exit := range []receiverExit{receiverCtxCancelled, receiverRetryable, receiverInternalRestart} {
		judgedRun := demoteFatalExitWhenBastionUnreachable(
			t.Context(),
			slog.New(slog.DiscardHandler),
			receiverRunResult{Exit: exit},
			false,
		)

		assert.Equal(t, exit, judgedRun.Exit)
	}
}

func Test_RecordExitAndDecideRetry_WhenTheBastionIsUnreachable_DoesNotEscalate(t *testing.T) {
	var failureEscalator rapidFailureEscalator

	for range receivewalMaxRapidFailures * 2 {
		require.False(t,
			failureEscalator.recordExitAndDecideRetry(instantReceiverExit, false).isEscalationRequired)
	}
}

func Test_RecordExitAndDecideRetry_WhenTheBastionIsReachable_EscalatesAfterTheRapidFailureLimit(
	t *testing.T,
) {
	var failureEscalator rapidFailureEscalator

	for range receivewalMaxRapidFailures - 1 {
		require.False(t,
			failureEscalator.recordExitAndDecideRetry(instantReceiverExit, true).isEscalationRequired)
	}

	require.True(t,
		failureEscalator.recordExitAndDecideRetry(instantReceiverExit, true).isEscalationRequired)
}

func Test_RecordExitAndDecideRetry_WhenAHealthyRunFollowsRapidFailures_ClearsTheCount(t *testing.T) {
	var failureEscalator rapidFailureEscalator

	for range receivewalMaxRapidFailures - 1 {
		failureEscalator.recordExitAndDecideRetry(instantReceiverExit, true)
	}

	require.False(t,
		failureEscalator.recordExitAndDecideRetry(receivewalMinHealthyUptime, true).isEscalationRequired)
	require.False(t,
		failureEscalator.recordExitAndDecideRetry(instantReceiverExit, true).isEscalationRequired)
}

func Test_RecordExitAndDecideRetry_WhenTheBastionIsUnreachable_DoesNotResetTheBackoff(t *testing.T) {
	var failureEscalator rapidFailureEscalator

	require.False(t,
		failureEscalator.recordExitAndDecideRetry(receivewalMinHealthyUptime, false).isBackoffResettable)
	require.True(t,
		failureEscalator.recordExitAndDecideRetry(receivewalMinHealthyUptime, true).isBackoffResettable)
	require.False(t,
		failureEscalator.recordExitAndDecideRetry(instantReceiverExit, true).isBackoffResettable)
}

func Test_RecordExitAndDecideRetry_WhenTheBastionIsUnreachable_CapsTheRespawnBackoff(t *testing.T) {
	var failureEscalator rapidFailureEscalator

	require.Equal(t, receivewalBastionDownMaxBackoff,
		failureEscalator.recordExitAndDecideRetry(instantReceiverExit, false).maxRespawnBackoff)
	require.Less(t, receivewalBastionDownMaxBackoff, receivewalRespawnMaxBackoff,
		"a cap that is not lower than the ordinary ceiling caps nothing")
}

func Test_RecordExitAndDecideRetry_WhenTheBastionIsReachable_KeepsTheOrdinaryBackoffCeiling(t *testing.T) {
	var failureEscalator rapidFailureEscalator

	require.Equal(t, receivewalRespawnMaxBackoff,
		failureEscalator.recordExitAndDecideRetry(instantReceiverExit, true).maxRespawnBackoff)
	require.Equal(t, receivewalRespawnMaxBackoff,
		failureEscalator.recordExitAndDecideRetry(receivewalMinHealthyUptime, true).maxRespawnBackoff)
}
