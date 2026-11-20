// Package instancerecovery normalizes the verification tables of a snapshot that a recovery of
// the instance staged, before it is published.
package instancerecovery

import (
	"context"
	"time"
)

const interruptedMessage = "Interrupted when the instance was recovered from a Databasus backup."

type Service struct {
	repositories Repositories
}

func NewService(repositories Repositories) *Service {
	return &Service{repositories: repositories}
}

// NormalizeRecoveredState fails the verifications agents were running for the lost instance and
// revokes every agent token, so an agent of the lost instance cannot act on this one; its owner
// issues a new token.
func (s *Service) NormalizeRecoveredState(ctx context.Context, now time.Time) error {
	return s.repositories.state.endRecoveredWork(ctx, now, interruptedMessage)
}
