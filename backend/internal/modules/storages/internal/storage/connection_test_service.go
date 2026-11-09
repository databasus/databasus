package storage

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/storages/internal/providers"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

type ConnectionTestServiceDeps struct {
	Storages   *Service
	Registry   *providers.Registry
	Workspaces workspaces.Module
}

// SettingsTest is a test of settings that are not saved. StorageID names the saved storage whose
// stored secrets fill the ones the settings leave out, nil for a new storage.
type SettingsTest struct {
	WorkspaceID uuid.UUID
	StorageID   *uuid.UUID
	Settings    providers.Settings
}

type ConnectionTestService struct {
	storages   *Service
	registry   *providers.Registry
	workspaces workspaces.Module
}

func NewConnectionTestService(deps ConnectionTestServiceDeps) *ConnectionTestService {
	return &ConnectionTestService{storages: deps.Storages, registry: deps.Registry, workspaces: deps.Workspaces}
}

// TestStorage tests the stored settings. A viewer may test, as the settings never leave the backend
// for anywhere but the storage itself.
func (s *ConnectionTestService) TestStorage(
	ctx context.Context,
	actor users.Actor,
	storageID uuid.UUID,
) (TestResult, error) {
	testedStorage, err := s.storages.getAuthorized(ctx, actor, storageID, workspaces.ActionView, noRowLock)
	if err != nil {
		return TestResult{}, err
	}

	testedStorageWithSettings, err := s.storages.settings.loadOne(ctx, testedStorage)
	if err != nil {
		return TestResult{}, err
	}

	return s.testConnection(ctx, providers.Location{StorageID: storageID, Settings: testedStorageWithSettings.Settings})
}

// TestSettings needs the right to change the workspace's content, because the test connects wherever
// the settings point, and a saved storage's ID fills in its stored secrets while the resource address
// stays the stored one. Every refusal comes before anything leaves the backend, and nothing is saved.
func (s *ConnectionTestService) TestSettings(
	ctx context.Context,
	actor users.Actor,
	settingsTest SettingsTest,
) (TestResult, error) {
	err := s.workspaces.Authorize(ctx, actor, settingsTest.WorkspaceID, workspaces.ActionEditContent)
	if err != nil {
		return TestResult{}, err
	}

	location, err := s.prepareLocation(ctx, settingsTest)
	if err != nil {
		return TestResult{}, err
	}

	return s.testConnection(ctx, location)
}

// prepareLocation refuses a named storage of another workspace as a missing one, and settings that a
// save of that storage would refuse, so its stored secrets never reach them.
func (s *ConnectionTestService) prepareLocation(
	ctx context.Context,
	settingsTest SettingsTest,
) (providers.Location, error) {
	if settingsTest.StorageID == nil {
		provider, err := s.registry.GetProvider(settingsTest.Settings.GetType())
		if err != nil {
			return providers.Location{}, err
		}

		err = provider.Validate(settingsTest.Settings)
		if err != nil {
			return providers.Location{}, err
		}

		return providers.Location{StorageID: uuid.Nil, Settings: settingsTest.Settings}, nil
	}

	storedStorage, isFound, err := s.storages.repository.get(ctx, *settingsTest.StorageID, noRowLock)
	if err != nil {
		return providers.Location{}, err
	}

	if !isFound || storedStorage.WorkspaceID != settingsTest.WorkspaceID {
		return providers.Location{}, ErrStorageNotFound
	}

	mergedSettings, err := s.storages.mergeAndValidate(ctx, storedStorage, settingsTest.Settings)
	if err != nil {
		return providers.Location{}, err
	}

	return providers.Location{StorageID: storedStorage.ID, Settings: mergedSettings}, nil
}

// testConnection answers the provider's refusal as the outcome of the test; each provider bounds its
// own test with a deadline.
func (s *ConnectionTestService) testConnection(ctx context.Context, location providers.Location) (TestResult, error) {
	provider, err := s.registry.GetProvider(location.Settings.GetType())
	if err != nil {
		return TestResult{}, err
	}

	return newTestResult(provider.TestConnection(ctx, location)), nil
}
