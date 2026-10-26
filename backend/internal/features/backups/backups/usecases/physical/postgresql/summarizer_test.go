package usecases_physical_postgresql

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	"databasus-backend/internal/util/testing/containers"
	"databasus-backend/internal/util/walmath"
)

// Test_CheckSummarizerReadiness_WhenSummarizerOff_FallsBackToFullNewChain proves
// the first WAL-gap fallback: with summarize_wal off the source can never produce
// the summaries an INCR needs, so the readiness check must steer to a fresh FULL
// (new chain) and tag it SUMMARIZER_OFF rather than attempting a doomed INCR.
//
// It targets the dedicated no-summary cluster: summarize_wal is off at the
// postmaster level there, which is the only way to reach this branch — ALTER
// SYSTEM cannot override a command-line GUC on the standard fixture.
func Test_CheckSummarizerReadiness_WhenSummarizerOff_FallsBackToFullNewChain(t *testing.T) {
	source := containers.StartPhysicalPostgres(t, "postgres:17", containers.WithoutSummarizer())
	ctx := context.Background()

	conn := openPhysicalTestConn(t, ctx, source)

	enabled, err := isSummarizerEnabled(ctx, conn)
	require.NoError(t, err)
	require.False(t, enabled, "the no-summary cluster must report summarize_wal off")

	result, err := CheckSummarizerReadiness(ctx, conn, walmath.LSN(0), time.Hour)
	require.NoError(t, err)

	require.Equal(t, DecisionFullNewChain, result.Decision)
	require.NotNil(t, result.Reason)
	require.Equal(t, physical_enums.PhysicalBackupErrorSummarizerOff, *result.Reason)
}

// Test_CheckSummarizerReadiness_WhenStopLsnPredatesOldestSummary_FallsBackToFullNewChain
// covers genuine expiry: the summarizer is on but the parent's stop LSN sits below
// the oldest retained summary, so no summary can ever cover it again — the check
// must steer to a fresh FULL and tag it SUMMARIES_EXPIRED. ExpireWalSummaries is
// applied first so the path mirrors production expiry rather than a synthetic LSN
// alone.
func Test_CheckSummarizerReadiness_WhenStopLsnPredatesOldestSummary_FallsBackToFullNewChain(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)
	conn := OpenAdminConn(t, fixture)

	ExpireWalSummaries(t, conn)

	enabled, err := isSummarizerEnabled(context.Background(), conn)
	require.NoError(t, err)
	require.True(t, enabled, "the standard fixture runs with summarize_wal on")

	// LSN 0/1 sits before any summary the running cluster can hold, so it predates
	// the oldest retained summary regardless of where the summarizer currently starts.
	expiredTargetLSN := walmath.LSN(1)

	result, err := CheckSummarizerReadiness(context.Background(), conn, expiredTargetLSN, time.Hour)
	require.NoError(t, err)

	require.Equal(t, DecisionFullNewChain, result.Decision)
	require.NotNil(t, result.Reason)
	require.Equal(t, physical_enums.PhysicalBackupErrorSummariesExpired, *result.Reason)
}

// Test_CheckSummarizerReadiness_WhenStopLsnAheadOfSummaries_GoesIncremental is the
// regression guard for the field report: right after a FULL on a near-idle DB the
// WAL summarizer trails the FULL's stop_lsn (it never summarizes the active segment).
// A stop_lsn the summarizer simply hasn't reached yet is NOT expiry, and with the
// summarizer process alive there is nothing left to wait for either. Runs on both PG
// majors since PG 18 is the reported engine.
func Test_CheckSummarizerReadiness_WhenStopLsnAheadOfSummaries_GoesIncremental(t *testing.T) {
	for _, version := range []string{"17", "18"} {
		t.Run("PostgreSQL "+version, func(t *testing.T) {
			fixture := SetupPhysicalDBForBackupVersion(t, version)
			conn := OpenAdminConn(t, fixture)

			ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
			defer cancel()

			// Advance WAL a little so the current tip sits past every flushed summary,
			// reproducing the "summarizer behind a fresh FULL's stop_lsn" state.
			_, err := GenerateWalActivity(ctx, conn, 1)
			require.NoError(t, err)

			var walTipLSN walmath.LSN
			require.NoError(t, conn.QueryRow(ctx, "SELECT pg_current_wal_lsn()::text").Scan(&walTipLSN))

			result, err := CheckSummarizerReadiness(ctx, conn, walTipLSN, time.Hour)
			require.NoError(t, err)

			require.Equal(t, DecisionGoIncremental, result.Decision,
				"a stop_lsn the summarizer has not reached yet must neither stall nor break the chain")
			require.Nil(t, result.Reason)
		})
	}
}

// stubSummarizerCheck installs a scripted readiness probe in place of the live
// CheckSummarizerReadiness and restores the original when the test ends. Each
// call returns the next scripted step; the final step repeats once the script
// is exhausted so an over-eager wait loop sees a stable terminal value rather
// than a panic.
func stubSummarizerCheck(t *testing.T, steps ...summarizerStep) {
	t.Helper()

	original := summarizerCheck
	t.Cleanup(func() { summarizerCheck = original })

	callIndex := 0
	summarizerCheck = func(
		_ context.Context, _ *pgx.Conn, _ walmath.LSN, _ time.Duration,
	) (SummarizerResult, error) {
		step := steps[min(callIndex, len(steps)-1)]
		callIndex++

		return step.result, step.err
	}
}

type summarizerStep struct {
	result SummarizerResult
	err    error
}

func waitStep(waitFor, pollEvery time.Duration) summarizerStep {
	return summarizerStep{result: SummarizerResult{
		Decision:  DecisionWait,
		WaitFor:   waitFor,
		PollEvery: pollEvery,
	}}
}

func decisionStep(decision SummarizerDecision, reason *physical_enums.PhysicalBackupErrorReason) summarizerStep {
	return summarizerStep{result: SummarizerResult{Decision: decision, Reason: reason}}
}

func Test_WaitForSummarizer_LaggingThenCatchesUp_ProceedsNoChainBroken(t *testing.T) {
	stubSummarizerCheck(t,
		waitStep(2*time.Second, 5*time.Millisecond), // first probe: lagging, opens a generous window
		waitStep(2*time.Second, 5*time.Millisecond), // still lagging
		decisionStep(DecisionGoIncremental, nil),    // caught up within the window
	)

	result, err := resolveSummarizerDecision(context.Background(), nil, walmath.LSN(0), time.Hour)

	require.NoError(t, err)
	require.Equal(t, DecisionGoIncremental, result.Decision,
		"a summarizer that catches up within the window must proceed with the incremental")
}

func Test_WaitForSummarizer_WhenWaitDecisionOutlivesDeadline_FallsBackToRetryNextCadence(t *testing.T) {
	// WaitFor is tiny and every probe keeps reporting Wait, so the window expires
	// with the summarizer still publishing nothing.
	stubSummarizerCheck(t, waitStep(20*time.Millisecond, 5*time.Millisecond))

	result, err := resolveSummarizerDecision(context.Background(), nil, walmath.LSN(0), time.Hour)

	require.NoError(t, err)
	require.Equal(t, DecisionRetryNextCadence, result.Decision,
		"a wait window that expires unresolved must hand the attempt to the next cadence")
}

func Test_WaitWindowForCadence_WhenCronIntervalHasNoPeriod_FallsBackToCap(t *testing.T) {
	require.Equal(t, summarizerWaitCap, waitWindowForCadence(0),
		"a cron interval reports no period, and a zero window would skip the wait entirely")
}

func Test_WaitWindowForCadence_WhenHourlyCadence_UsesQuarterOfCadence(t *testing.T) {
	require.Equal(t, 15*time.Minute, waitWindowForCadence(time.Hour))
}

func Test_WaitWindowForCadence_WhenDailyCadence_ClampsToCap(t *testing.T) {
	require.Equal(t, summarizerWaitCap, waitWindowForCadence(24*time.Hour))
}

func Test_IsSummarizerRunning_OnClusterWithSummarizerOn_ReturnsTrue(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)
	conn := OpenAdminConn(t, fixture)

	isRunning, err := isSummarizerRunning(context.Background(), conn)

	require.NoError(t, err)
	require.True(t, isRunning)
}

func Test_IsSummarizerRunning_OnClusterWithoutSummarizer_ReturnsFalse(t *testing.T) {
	source := containers.StartPhysicalPostgres(t, "postgres:17", containers.WithoutSummarizer())
	ctx := context.Background()

	conn := openPhysicalTestConn(t, ctx, source)

	isRunning, err := isSummarizerRunning(ctx, conn)

	require.NoError(t, err)
	require.False(t, isRunning, "the summarizer process never starts when summarize_wal is off")
}

func Test_WaitForSummarizer_SummariesExpireMidWait_FallsBackToFullNewChain(t *testing.T) {
	expired := physical_enums.PhysicalBackupErrorSummariesExpired

	stubSummarizerCheck(t,
		waitStep(2*time.Second, 5*time.Millisecond),
		decisionStep(DecisionFullNewChain, &expired), // summaries aged out during the wait
	)

	result, err := resolveSummarizerDecision(context.Background(), nil, walmath.LSN(0), time.Hour)

	require.NoError(t, err)
	require.Equal(t, DecisionFullNewChain, result.Decision)
	require.NotNil(t, result.Reason)
	require.Equal(t, expired, *result.Reason,
		"a terminal decision reached mid-wait must propagate its reason verbatim")
}

func Test_WaitForSummarizer_ContextCanceled_ReturnsError(t *testing.T) {
	stubSummarizerCheck(t, waitStep(time.Hour, time.Hour)) // open window; only ctx ends the wait

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := resolveSummarizerDecision(ctx, nil, walmath.LSN(0), time.Hour)

	require.True(t, errors.Is(err, context.Canceled),
		"a cancelled context during the wait must surface as context.Canceled")
}

func Test_ResolveSummarizerDecision_TerminalOnFirstProbe_DoesNotWait(t *testing.T) {
	off := physical_enums.PhysicalBackupErrorSummarizerOff

	stubSummarizerCheck(t, decisionStep(DecisionFullNewChain, &off))

	result, err := resolveSummarizerDecision(context.Background(), nil, walmath.LSN(0), time.Hour)

	require.NoError(t, err)
	require.Equal(t, DecisionFullNewChain, result.Decision)
	require.Equal(t, off, *result.Reason)
}
