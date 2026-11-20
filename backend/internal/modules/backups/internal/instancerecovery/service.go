// Package instancerecovery normalizes the backups tables of a snapshot that a recovery of the
// instance staged, before it is published.
package instancerecovery

import (
	"context"
	"time"
)

const interruptedMessage = "Interrupted when the instance was recovered from a Databasus backup. " +
	"Trigger a new one."

type Service struct {
	repositories Repositories
}

func NewService(repositories Repositories) *Service {
	return &Service{repositories: repositories}
}

// NormalizeRecoveredState ends the backups and restores the lost instance had running, revokes its
// download tokens, drops its WAL streaming claims and requests a new full chain for every physical
// database, because the local WAL queue did not survive.
func (s *Service) NormalizeRecoveredState(ctx context.Context, now time.Time) error {
	return s.repositories.state.endRecoveredWork(ctx, now, interruptedMessage)
}
