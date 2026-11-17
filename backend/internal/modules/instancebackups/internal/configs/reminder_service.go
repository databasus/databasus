package configs

import "context"

// Reminder is the initial setup reminder. It becomes eligible once the instance holds a user
// database or a storage, so a fresh installation is not nagged before it protects anything.
// It cannot be deferred: it shows while no destination is selected.
type Reminder struct {
	IsEligible bool
}

func (s *Service) GetReminder(ctx context.Context) (Reminder, error) {
	isEligible, err := s.isSetupEligible(ctx)
	if err != nil {
		return Reminder{}, err
	}

	return Reminder{IsEligible: isEligible}, nil
}

func (s *Service) isSetupEligible(ctx context.Context) (bool, error) {
	allStorages, err := s.storages.GetAllStorages(ctx)
	if err != nil {
		return false, err
	}

	if len(allStorages) > 0 {
		return true, nil
	}

	allDatabases, err := s.databases.ListAllDatabases(ctx)
	if err != nil {
		return false, err
	}

	return len(allDatabases) > 0, nil
}
