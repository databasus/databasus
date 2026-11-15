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

// NormalizeRecoveredState makes every local storage require validation, because the Databasus
// backup did not restore the host folders it points at, and discards the unfinished file writes
// and deletions of the lost instance instead of replaying them against storages it may still use.
// It returns how many it discarded; their files may need manual cleanup.
func (s *Service) NormalizeRecoveredState(ctx context.Context) (int, error) {
	if err := s.repositories.state.requireLocalStorageValidation(ctx); err != nil {
		return 0, err
	}

	return s.repositories.state.discardPendingFileWork(ctx)
}
