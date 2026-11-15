package storage

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/recoverypause"
	"databasus-backend/internal/modules/storages/internal/providers"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

type ConnectionTestServiceDeps struct {
	Storages   *Service
	Registry   *providers.Registry
	Workspaces workspaces.Module
	// RecoveryPause blocks every test while a recovered instance is paused, because the tests write
	// probes to the storage.
	RecoveryPause *recoverypause.Gate
}

// SettingsTest is a test of settings that are not saved. StorageID names the saved storage whose
// stored secrets fill the ones the settings leave out, nil for a new storage.
type SettingsTest struct {
	WorkspaceID uuid.UUID
	StorageID   *uuid.UUID
	Settings    providers.Settings
}

type ConnectionTestService struct {
	storages      *Service
	registry      *providers.Registry
	workspaces    workspaces.Module
	recoveryPause *recoverypause.Gate
}

func NewConnectionTestService(deps ConnectionTestServiceDeps) *ConnectionTestService {
	return &ConnectionTestService{
		storages:      deps.Storages,
		registry:      deps.Registry,
		workspaces:    deps.Workspaces,
		recoveryPause: deps.RecoveryPause,
	}
}

// TestStorage tests the stored settings. A viewer may test, as the settings never leave the backend
// for anywhere but the storage itself. A recovered local storage is also probed in its backup
// folder, and an editor's successful test of unchanged settings makes it available; a viewer's
// test of it stays diagnostic.
func (s *ConnectionTestService) TestStorage(
	ctx context.Context,
	actor users.Actor,
	storageID uuid.UUID,
) (TestResult, error) {
	testedStorage, err := s.storages.getAuthorized(ctx, actor, storageID, workspaces.ActionView, noRowLock)
	if err != nil {
		return TestResult{}, err
	}

	if err := s.recoveryPause.Check(); err != nil {
		return TestResult{}, err
	}

	testedStorageWithSettings, err := s.storages.settings.loadOne(ctx, testedStorage)
	if err != nil {
		return TestResult{}, err
	}

	location := providers.Location{StorageID: storageID, Settings: testedStorageWithSettings.Settings}
	status := testedStorage.RecoveryValidationStatus

	provider, err := s.registry.GetProvider(location.Settings.GetType())
	if err != nil {
		return TestResult{}, err
	}

	if testedStorage.IsAvailable() {
		return newTestResult(provider.TestConnection(ctx, location), &status), nil
	}

	testErr := testRecoveredConnection(ctx, provider, location)
	isEditor := s.workspaces.Authorize(ctx, actor, testedStorage.WorkspaceID, workspaces.ActionEditContent) == nil

	if testErr == nil && isEditor {
		status, err = s.storages.markRecoveryValidated(ctx, actor, testedStorageWithSettings)
		if err != nil {
			return TestResult{}, err
		}
	}

	return newTestResult(testErr, &status), nil
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

	if err := s.recoveryPause.Check(); err != nil {
		return TestResult{}, err
	}

	location, err := s.prepareLocation(ctx, settingsTest)
	if err != nil {
		return TestResult{}, err
	}

	provider, err := s.registry.GetProvider(location.Settings.GetType())
	if err != nil {
		return TestResult{}, err
	}

	return newTestResult(provider.TestConnection(ctx, location), nil), nil
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

// testRecoveredConnection adds the provider's recovery probe to its ordinary test. Both must pass.
func testRecoveredConnection(ctx context.Context, provider providers.Provider, location providers.Location) error {
	if err := provider.TestConnection(ctx, location); err != nil {
		return err
	}

	validator, isValidator := provider.(providers.RecoveryValidator)
	if !isValidator {
		return nil
	}

	return validator.TestRecoveredConnection(ctx, location)
}
