// Package instancerecovery normalizes the storages tables of a snapshot that a recovery of the
// instance staged, before it is published.
package instancerecovery

import "context"

type Service struct {
	repositories Repositories
}

func NewService(repositories Repositories) *Service {
	return &Service{repositories: repositories}
}

// NormalizeRecoveredState discards the unfinished file writes and deletions of the lost instance
// instead of replaying them against storages it may still use. It returns how many it discarded;
// their files may need manual cleanup.
func (s *Service) NormalizeRecoveredState(ctx context.Context) (int, error) {
	return s.repositories.state.discardPendingFileWork(ctx)
}
