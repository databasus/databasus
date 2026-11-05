package iox

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ReadRetryPolicy_EveryFailedAttemptDeliveredBytes_NeverExhausts(t *testing.T) {
	policy := NewReadRetryPolicy(ReadRetryPolicySpec{MaxAttemptsWithoutProgress: 3})

	for range 100 {
		policy.RegisterFailedAttempt(true)
		require.False(t, policy.IsExhausted())
	}

	assert.Equal(t, 1, policy.GetAttemptsSinceProgress())
}

func Test_ReadRetryPolicy_FailedAttemptsWithoutProgress_ExhaustsAfterTheBudget(t *testing.T) {
	policy := NewReadRetryPolicy(ReadRetryPolicySpec{MaxAttemptsWithoutProgress: 3})

	for range 3 {
		policy.RegisterFailedAttempt(false)
		require.False(t, policy.IsExhausted())
	}

	policy.RegisterFailedAttempt(false)

	assert.True(t, policy.IsExhausted())
}

func Test_ReadRetryPolicy_ProgressAfterFailures_ResetsTheSpentBudget(t *testing.T) {
	policy := NewReadRetryPolicy(ReadRetryPolicySpec{MaxAttemptsWithoutProgress: 3})

	policy.RegisterFailedAttempt(false)
	policy.RegisterFailedAttempt(false)
	policy.RegisterFailedAttempt(true)

	assert.Equal(t, 1, policy.GetAttemptsSinceProgress())
}

func Test_ReadRetryPolicy_CompletedRead_ResetsAttempts(t *testing.T) {
	policy := NewReadRetryPolicy(ReadRetryPolicySpec{MaxAttemptsWithoutProgress: 3})

	policy.RegisterFailedAttempt(false)
	policy.ResetAttemptsAfterCompletedRead()

	assert.Equal(t, 0, policy.GetAttemptsSinceProgress())
}

func Test_ReadRetryPolicy_WaitBeforeRetryWithCancelledContext_ReturnsWithoutSleeping(t *testing.T) {
	policy := NewReadRetryPolicy(ReadRetryPolicySpec{BaseDelay: time.Hour})
	policy.RegisterFailedAttempt(false)

	cancelledCtx, cancel := context.WithCancel(t.Context())
	cancel()

	waitStart := time.Now()
	err := policy.WaitBeforeRetry(cancelledCtx)

	require.ErrorIs(t, err, context.Canceled)
	assert.Less(t, time.Since(waitStart), time.Second)
}

func Test_ReadRetryPolicy_ConsecutiveFailures_BackoffGrowsExponentially(t *testing.T) {
	policy := NewReadRetryPolicy(ReadRetryPolicySpec{
		MaxAttemptsWithoutProgress: 10,
		BaseDelay:                  20 * time.Millisecond,
	})

	for range 3 {
		policy.RegisterFailedAttempt(false)
	}

	waitStart := time.Now()
	require.NoError(t, policy.WaitBeforeRetry(t.Context()))

	assert.GreaterOrEqual(t, time.Since(waitStart), 80*time.Millisecond)
}
