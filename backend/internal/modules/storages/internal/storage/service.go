package storage

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"unicode/utf8"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/storages/internal/files"
	"databasus-backend/internal/modules/storages/internal/providers"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

const maxStorageNameLength = 255

type Deps struct {
	Repositories Repositories
	Settings     *SettingsReader
	Registry     *providers.Registry

	FileDeletions *files.Store
	Transactor    *database.Transactor
	Logger        *slog.Logger
	AuditLog      *audit.Log
	Users         users.Module
	Workspaces    workspaces.Module
}

// Service keeps storages with their provider settings and deletes them together with their pending
// files.
type Service struct {
	repository    *storageRepository
	settings      *SettingsReader
	registry      *providers.Registry
	fileDeletions *files.Store
	transactor    *database.Transactor
	logger        *slog.Logger
	audit         auditRecorder
	workspaces    workspaces.Module
	// deletingHandlers check a deletion before the storages' own references and cleanup.
	deletingHandlers []DeletingHandler
}

func NewService(deps Deps) *Service {
	return &Service{
		repository:       deps.Repositories.storages,
		settings:         deps.Settings,
		registry:         deps.Registry,
		fileDeletions:    deps.FileDeletions,
		transactor:       deps.Transactor,
		logger:           deps.Logger,
		audit:            auditRecorder{auditLog: deps.AuditLog, users: deps.Users},
		workspaces:       deps.Workspaces,
		deletingHandlers: nil,
	}
}

// SubscribeToDeleting adds a handler after the ones subscribed before it. internal/app subscribes
// while it builds the application, before any request runs.
func (s *Service) SubscribeToDeleting(handler DeletingHandler) {
	s.deletingHandlers = append(s.deletingHandlers, handler)
}

// ListStorages pages through the storages of a workspace, ordered by name.
func (s *Service) ListStorages(ctx context.Context, actor users.Actor, query ListQuery) (Page, error) {
	limit := defaultStoragePageLimit
	if query.Limit != nil {
		limit = *query.Limit
	}

	if limit < 1 || limit > maxStoragePageLimit {
		return Page{}, apierror.NewInvalidFieldsProblem([]apierror.FieldError{{
			Field:  "limit",
			Detail: fmt.Sprintf("must be between 1 and %d", maxStoragePageLimit),
		}})
	}

	var after *database.NameCursor

	if query.After != nil {
		cursor, err := database.DecodeNameCursor(*query.After)
		if err != nil {
			return Page{}, err
		}

		after = &cursor
	}

	err := s.workspaces.Authorize(ctx, actor, query.WorkspaceID, workspaces.ActionView)
	if err != nil {
		return Page{}, err
	}

	listedStorages, err := s.repository.list(ctx, storageListFilter{
		workspaceID: query.WorkspaceID,
		limit:       limit + 1,
		after:       after,
	})
	if err != nil {
		return Page{}, err
	}

	pageStorages, err := s.settings.load(ctx, listedStorages[:min(len(listedStorages), limit)])
	if err != nil {
		return Page{}, err
	}

	page := Page{Storages: pageStorages, NextCursor: nil}

	if len(listedStorages) > limit {
		lastStorage := listedStorages[limit-1]
		nextCursor := database.NameCursor{Name: lastStorage.Name, ID: lastStorage.ID}.Encode()
		page.NextCursor = &nextCursor
	}

	return page, nil
}

func (s *Service) CreateStorage(ctx context.Context, actor users.Actor, creation Creation) (WithSettings, error) {
	err := validateName(creation.Name)
	if err != nil {
		return WithSettings{}, err
	}

	err = s.workspaces.Authorize(ctx, actor, creation.WorkspaceID, workspaces.ActionEditContent)
	if err != nil {
		return WithSettings{}, err
	}

	provider, err := s.registry.GetProvider(creation.Settings.GetType())
	if err != nil {
		return WithSettings{}, err
	}

	err = provider.Validate(creation.Settings)
	if err != nil {
		return WithSettings{}, err
	}

	createdStorage := Storage{
		ID:          uuid.New(),
		WorkspaceID: creation.WorkspaceID,
		Name:        creation.Name,
		Type:        provider.GetType(),

		RecoveryValidationStatus: RecoveryValidationStatusAvailable,
	}

	err = s.transactor.Transaction(ctx, func(ctx context.Context) error {
		err := s.repository.insertWithSettings(ctx, createdStorage, providerSettings{
			repository: provider.GetRepository(),
			settings:   creation.Settings,
		})
		if err != nil {
			return err
		}

		return s.audit.record(ctx, actor, createdStorage.WorkspaceID, "Storage created: "+createdStorage.Name)
	})
	if err != nil {
		return WithSettings{}, err
	}

	// The repository encrypted the secrets on save; the stored settings are what the caller gets back.
	return s.settings.loadOne(ctx, createdStorage)
}

func (s *Service) GetStorage(ctx context.Context, actor users.Actor, storageID uuid.UUID) (WithSettings, error) {
	foundStorage, err := s.getAuthorized(ctx, actor, storageID, workspaces.ActionView, noRowLock)
	if err != nil {
		return WithSettings{}, err
	}

	return s.settings.loadOne(ctx, foundStorage)
}

// UpdateStorage keeps the type of the storage: settings of another type are refused before anything
// is written. The stored settings are read, merged and written under the row lock, so two edits
// cannot lose each other's secrets.
func (s *Service) UpdateStorage(ctx context.Context, actor users.Actor, change Change) (WithSettings, error) {
	err := validateName(change.Name)
	if err != nil {
		return WithSettings{}, err
	}

	var updatedStorage WithSettings

	err = s.transactor.Transaction(ctx, func(ctx context.Context) error {
		storedStorage, err := s.getAuthorized(
			ctx,
			actor,
			change.StorageID,
			workspaces.ActionEditContent,
			forUpdateRowLock,
		)
		if err != nil {
			return err
		}

		mergedSettings, err := s.mergeAndValidate(ctx, storedStorage, change.Settings)
		if err != nil {
			return err
		}

		provider, err := s.registry.GetProvider(storedStorage.Type)
		if err != nil {
			return err
		}

		err = s.repository.updateNameAndSettings(ctx, storedStorage.ID, change.Name, providerSettings{
			repository: provider.GetRepository(),
			settings:   mergedSettings,
		})
		if err != nil {
			return err
		}

		message := "Storage updated: " + change.Name
		if storedStorage.Name != change.Name {
			message = fmt.Sprintf("Storage renamed from '%s' to '%s'", storedStorage.Name, change.Name)
		}

		err = s.audit.record(ctx, actor, storedStorage.WorkspaceID, message)
		if err != nil {
			return err
		}

		storedStorage.Name = change.Name

		updatedStorage, err = s.settings.loadOne(ctx, storedStorage)

		return err
	})
	if err != nil {
		return WithSettings{}, err
	}

	return updatedStorage, nil
}

// GetStorages leaves out the IDs that match no storage.
func (s *Service) GetStorages(ctx context.Context, storageIDs []uuid.UUID) ([]Storage, error) {
	if len(storageIDs) == 0 {
		return []Storage{}, nil
	}

	return s.repository.listByIDs(ctx, storageIDs)
}

// GetStoragesBlockingChanges leaves out the IDs that match no storage and holds a share lock on the
// found rows until the transaction in ctx ends, so their deletion and transfer wait for it.
func (s *Service) GetStoragesBlockingChanges(ctx context.Context, storageIDs []uuid.UUID) ([]Storage, error) {
	if len(storageIDs) == 0 {
		return []Storage{}, nil
	}

	return s.repository.listByIDsBlockingChanges(ctx, storageIDs)
}

// GetAllStorages returns every storage of the instance, ordered by name and ID.
func (s *Service) GetAllStorages(ctx context.Context) ([]Storage, error) {
	return s.repository.listAll(ctx)
}

// GetStorageBlockingTransfer holds a share lock on the storage's row until the transaction in ctx
// ends, so a transfer of the storage waits for that transaction.
func (s *Service) GetStorageBlockingTransfer(ctx context.Context, storageID uuid.UUID) (Storage, bool, error) {
	return s.repository.get(ctx, storageID, forShareRowLock)
}

func (s *Service) GetStorageTypes(ctx context.Context) ([]Type, error) {
	return s.repository.listTypes(ctx)
}

// getAuthorized answers a storage of a workspace the actor is not a member of exactly as a missing
// storage, so its existence stays hidden.
func (s *Service) getAuthorized(
	ctx context.Context,
	actor users.Actor,
	storageID uuid.UUID,
	action workspaces.Action,
	lock rowLock,
) (Storage, error) {
	foundStorage, isFound, err := s.repository.get(ctx, storageID, lock)
	if err != nil {
		return Storage{}, err
	}

	if !isFound {
		return Storage{}, ErrStorageNotFound
	}

	err = s.workspaces.Authorize(ctx, actor, foundStorage.WorkspaceID, action)

	var problem *apierror.Problem
	if errors.As(err, &problem) && problem.Code == apierror.CodeNotFound {
		return Storage{}, ErrStorageNotFound
	}

	if err != nil {
		return Storage{}, err
	}

	return foundStorage, nil
}

// mergeAndValidate resolves secrets under the provider's selection and retention policy, then
// validates the result. Settings of another type than the stored storage's are
// refused, as is a change of a setting fixed at creation.
func (s *Service) mergeAndValidate(
	ctx context.Context,
	storedStorage Storage,
	incomingSettings providers.Settings,
) (providers.Settings, error) {
	if incomingSettings.GetType() != storedStorage.Type {
		return nil, ErrTypeChanged
	}

	storedStorageWithSettings, err := s.settings.loadOne(ctx, storedStorage)
	if err != nil {
		return nil, err
	}

	provider, err := s.registry.GetProvider(storedStorage.Type)
	if err != nil {
		return nil, err
	}

	mergedSettings, err := provider.MergeStoredSettings(providers.SettingsMerge{
		Incoming: incomingSettings,
		Stored:   storedStorageWithSettings.Settings,
	})
	if err != nil {
		return nil, err
	}

	err = provider.Validate(mergedSettings)
	if err != nil {
		return nil, err
	}

	return mergedSettings, nil
}

func validateName(name string) error {
	nameLength := utf8.RuneCountInString(name)
	if nameLength < 1 || nameLength > maxStorageNameLength {
		return ErrInvalidName
	}

	return nil
}

// markRecoveryValidated makes a recovered storage available after an editor's successful test, but
// only while its saved settings are still the ones the test probed. The status is the one column
// it changes.
func (s *Service) markRecoveryValidated(
	ctx context.Context,
	actor users.Actor,
	tested WithSettings,
) (RecoveryValidationStatus, error) {
	status := tested.RecoveryValidationStatus

	err := s.transactor.Transaction(ctx, func(txCtx context.Context) error {
		lockedStorage, isFound, err := s.repository.get(txCtx, tested.ID, forUpdateRowLock)
		if err != nil {
			return err
		}

		if !isFound {
			return ErrStorageNotFound
		}

		current, err := s.settings.loadOne(txCtx, lockedStorage)
		if err != nil {
			return err
		}

		status = lockedStorage.RecoveryValidationStatus
		if lockedStorage.IsAvailable() || !reflect.DeepEqual(current.Settings, tested.Settings) {
			return nil
		}

		err = s.repository.updateRecoveryValidationStatus(txCtx, tested.ID, RecoveryValidationStatusAvailable)
		if err != nil {
			return err
		}

		status = RecoveryValidationStatusAvailable

		return s.audit.record(txCtx, actor, lockedStorage.WorkspaceID,
			"Storage validated on this host after recovery: "+lockedStorage.Name)
	})
	if err != nil {
		return "", err
	}

	return status, nil
}
