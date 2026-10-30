package iox

import (
	"context"
	"time"
)

const (
	defaultMaxReadAttemptsWithoutProgress = 10
	defaultReadRetryBaseDelay             = 1 * time.Second
	readRetryMaxDelay                     = 30 * time.Second
)

type ReadRetryPolicySpec struct {
	MaxAttemptsWithoutProgress int
	BaseDelay                  time.Duration
}

type ReadRetryPolicy struct {
	maxAttemptsWithoutProgress int
	baseDelay                  time.Duration
	attemptsSinceProgress      int
}

func NewReadRetryPolicy(spec ReadRetryPolicySpec) *ReadRetryPolicy {
	if spec.MaxAttemptsWithoutProgress == 0 {
		spec.MaxAttemptsWithoutProgress = defaultMaxReadAttemptsWithoutProgress
	}

	if spec.BaseDelay == 0 {
		spec.BaseDelay = defaultReadRetryBaseDelay
	}

	return &ReadRetryPolicy{
		maxAttemptsWithoutProgress: spec.MaxAttemptsWithoutProgress,
		baseDelay:                  spec.BaseDelay,
	}
}

// The budget counts attempts since the last byte delivered, not attempts per transfer: a restore
// runs at the target's ingestion rate, so a multi-hour stream collects far more sparse drops than
// any per-transfer budget can cover.
func (p *ReadRetryPolicy) RegisterFailedAttempt(hasDeliveredBytes bool) {
	if hasDeliveredBytes {
		p.attemptsSinceProgress = 1

		return
	}

	p.attemptsSinceProgress++
}

func (p *ReadRetryPolicy) IsExhausted() bool {
	return p.attemptsSinceProgress > p.maxAttemptsWithoutProgress
}

func (p *ReadRetryPolicy) GetAttemptsSinceProgress() int {
	return p.attemptsSinceProgress
}

func (p *ReadRetryPolicy) GetMaxAttemptsWithoutProgress() int {
	return p.maxAttemptsWithoutProgress
}

func (p *ReadRetryPolicy) ResetAttemptsAfterCompletedRead() {
	p.attemptsSinceProgress = 0
}

func (p *ReadRetryPolicy) WaitBeforeRetry(ctx context.Context) error {
	delay := readRetryMaxDelay
	if shift := p.attemptsSinceProgress - 1; shift < 32 {
		delay = min(p.baseDelay<<shift, readRetryMaxDelay)
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
