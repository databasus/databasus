// Package ratelimit counts attempts per scope and identifier in the memory of the one backend
// process and tells whether the latest one stays within its limit.
package ratelimit

import (
	"context"
	"time"
)

type Attempt struct {
	Scope      string
	Identifier string
	Limit      int
	Window     time.Duration
}

type Counter interface {
	RecordAttemptAndCheckIsAllowed(ctx context.Context, attempt Attempt) (bool, error)
}
