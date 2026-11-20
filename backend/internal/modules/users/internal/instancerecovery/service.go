package instancerecovery

import (
	"context"

	"databasus-backend/internal/modules/users/internal/settings"
)

type Service struct {
	repositories Repositories
}

func NewService(repositories Repositories) *Service {
	return &Service{repositories: repositories}
}

func (s *Service) Inspect(ctx context.Context) (State, error) {
	hasAccounts, err := s.repositories.state.hasAccounts(ctx)
	if err != nil {
		return State{}, err
	}

	settingsRows, err := s.repositories.state.listSettings(ctx)
	if err != nil {
		return State{}, err
	}

	state := State{HasAccounts: hasAccounts, HasChangedSettings: false, AutomaticTables: []string{}}

	switch {
	case len(settingsRows) == 0:
	case len(settingsRows) == 1 && settingsRows[0].GetSwitches() == settings.DefaultSwitches():
		state.AutomaticTables = append(state.AutomaticTables, settingsTable)
	default:
		state.HasChangedSettings = true
	}

	return state, nil
}

// NormalizeRecoveredState ends every session, pending sign-in and emailed code of a recovered
// snapshot, so nobody signed in to the lost instance is signed in to this one.
func (s *Service) NormalizeRecoveredState(ctx context.Context) error {
	return s.repositories.state.endSessionsAndCodes(ctx)
}
