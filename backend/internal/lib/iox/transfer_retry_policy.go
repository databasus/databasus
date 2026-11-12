package iox

import (
	"context"
	"time"
)

const (
	defaultMaxAttemptsWithoutProgress = 10
	defaultRetryBaseDelay             = 1 * time.Second
	retryMaxDelay                     = 30 * time.Second
)

type TransferRetryPolicySpec struct {
	MaxAttemptsWithoutProgress int
	BaseDelay                  time.Duration
}

type TransferRetryPolicy struct {
	maxAttemptsWithoutProgress int
	baseDelay                  time.Duration
	attemptsSinceProgress      int
}

func NewTransferRetryPolicy(spec TransferRetryPolicySpec) *TransferRetryPolicy {
	if spec.MaxAttemptsWithoutProgress == 0 {
		spec.MaxAttemptsWithoutProgress = defaultMaxAttemptsWithoutProgress
	}

	if spec.BaseDelay == 0 {
		spec.BaseDelay = defaultRetryBaseDelay
	}

	return &TransferRetryPolicy{
		maxAttemptsWithoutProgress: spec.MaxAttemptsWithoutProgress,
		baseDelay:                  spec.BaseDelay,
	}
}

// The budget counts attempts since the last progress, such as a delivered byte or an accepted
// request, not attempts per transfer: a multi-hour stream collects far more sparse failures than
// any per-transfer budget can cover.
func (p *TransferRetryPolicy) RegisterFailedAttempt(hasMadeProgress bool) {
	if hasMadeProgress {
		p.attemptsSinceProgress = 1

		return
	}

	p.attemptsSinceProgress++
}

func (p *TransferRetryPolicy) IsExhausted() bool {
	return p.attemptsSinceProgress > p.maxAttemptsWithoutProgress
}

func (p *TransferRetryPolicy) GetAttemptsSinceProgress() int {
	return p.attemptsSinceProgress
}

func (p *TransferRetryPolicy) GetMaxAttemptsWithoutProgress() int {
	return p.maxAttemptsWithoutProgress
}

func (p *TransferRetryPolicy) ResetAttempts() {
	p.attemptsSinceProgress = 0
}

func (p *TransferRetryPolicy) WaitBeforeRetry(ctx context.Context) error {
	delay := retryMaxDelay
	if shift := p.attemptsSinceProgress - 1; shift < 32 {
		delay = min(p.baseDelay<<shift, retryMaxDelay)
	}

	retryTimer := time.NewTimer(delay)
	defer retryTimer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-retryTimer.C:
		return nil
	}
}
