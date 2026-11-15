package configs

import (
	"context"
	"time"

	"databasus-backend/internal/modules/users"
)

// Reminder is the initial setup reminder. It becomes eligible once the instance holds a user
// database or a storage, so a fresh installation is not nagged before it protects anything.
type Reminder struct {
	IsEligible    bool
	DeferredUntil *time.Time
}

func (s *Service) GetReminder(ctx context.Context) (Reminder, error) {
	config, err := s.configs.get(ctx)
	if err != nil {
		return Reminder{}, err
	}

	isEligible, err := s.isSetupEligible(ctx)
	if err != nil {
		return Reminder{}, err
	}

	return Reminder{IsEligible: isEligible, DeferredUntil: config.SetupDeferredUntil}, nil
}

// UpdateSetupReminder defers the initial setup reminder for 24 hours or shows it again.
func (s *Service) UpdateSetupReminder(ctx context.Context, actor users.Actor, isDeferred bool) (Reminder, error) {
	if _, err := s.users.RequireRootAdmin(ctx, actor); err != nil {
		return Reminder{}, err
	}

	var deferredUntil *time.Time
	if isDeferred {
		until := s.clock.Now().Add(setupDeferral)
		deferredUntil = &until
	}

	if err := s.configs.updateSetupDeferredUntil(ctx, deferredUntil); err != nil {
		return Reminder{}, err
	}

	return s.GetReminder(ctx)
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
