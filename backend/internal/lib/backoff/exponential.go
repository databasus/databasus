// Package backoff computes how long to wait before a retry.
package backoff

import "time"

// Exponential waits First before the first retry and twice as long before each next one, up to Max,
// so an unreachable service settles into one attempt per Max instead of an ever-widening gap.
type Exponential struct {
	First time.Duration
	Max   time.Duration
}

// GetDelay is the wait after attempt number attemptCount, counted from 1; a smaller count counts as
// 1. A large count stops at Max without overflowing.
func (e Exponential) GetDelay(attemptCount int) time.Duration {
	delay := e.First

	for range attemptCount - 1 {
		if delay >= e.Max/2 {
			return e.Max
		}

		delay *= 2
	}

	return min(delay, e.Max)
}
