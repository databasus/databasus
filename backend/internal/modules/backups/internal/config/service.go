package config

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

type Deps struct {
	Repositories Repositories
	Transactor   *database.Transactor
	Logger       *slog.Logger
	Clock        clock.Source
	Workspaces   workspaces.Module
	Storages     storages.Module
	Databases    databases.Module
	// LogicalBackups removes a database's logical backups when its config moves to another storage.
	LogicalBackups      BackupHistoryRemoverFunc
	PhysicalBackups     BackupHistoryRemoverFunc
	PhysicalConfigSaved PhysicalConfigSavedReactionFunc
}

// Service reads, lists and saves the backup configs. A config that was never saved reads as the
// default of its database's method, and nothing but a save writes one.
type Service struct {
	repository          *configRepository
	transactor          *database.Transactor
	logger              *slog.Logger
	clock               clock.Source
	workspaces          workspaces.Module
	storages            storages.Module
	databases           databases.Module
	logicalBackups      BackupHistoryRemoverFunc
	physicalBackups     BackupHistoryRemoverFunc
	physicalConfigSaved PhysicalConfigSavedReactionFunc
}

func NewService(deps Deps) *Service {
	return &Service{
		repository:          deps.Repositories.configs,
		transactor:          deps.Transactor,
		logger:              deps.Logger,
		clock:               deps.Clock,
		workspaces:          deps.Workspaces,
		storages:            deps.Storages,
		databases:           deps.Databases,
		logicalBackups:      deps.LogicalBackups,
		physicalBackups:     deps.PhysicalBackups,
		physicalConfigSaved: deps.PhysicalConfigSaved,
	}
}

func (s *Service) GetBackupConfig(
	ctx context.Context,
	actor users.Actor,
	databaseID uuid.UUID,
) (AnyBackupConfig, error) {
	configDatabase, err := s.getDatabase(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	err = s.authorizeDatabase(ctx, actor, configDatabase, workspaces.ActionView)
	if err != nil {
		return nil, err
	}

	readConfig, err := s.ReadBackupConfig(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	configs, err := s.withCleanup(ctx, actor, []AnyBackupConfig{readConfig})
	if err != nil {
		return nil, err
	}

	return configs[0], nil
}

// ReadBackupConfig returns the saved config or the default of the database's method, fitted to the
// database's physical backup type, and writes nothing.
func (s *Service) ReadBackupConfig(ctx context.Context, databaseID uuid.UUID) (AnyBackupConfig, error) {
	backupTypes, err := s.databases.GetPhysicalBackupTypes(ctx, []uuid.UUID{databaseID})
	if err != nil {
		return nil, err
	}

	backupType, isPhysical := backupTypes[databaseID]
	if isPhysical {
		return s.readPhysicalBackupConfig(ctx, databaseID, backupType)
	}

	logicalConfigs, err := s.repository.listLogical(ctx, []uuid.UUID{databaseID})
	if err != nil {
		return nil, err
	}

	if len(logicalConfigs) == 0 {
		return newDefaultLogicalBackupConfig(databaseID), nil
	}

	return logicalConfigs[0], nil
}

func (s *Service) GetLogicalBackupConfig(ctx context.Context, databaseID uuid.UUID) (LogicalBackupConfig, error) {
	_, err := s.getDatabase(ctx, databaseID)
	if err != nil {
		return LogicalBackupConfig{}, err
	}

	readConfig, err := s.ReadBackupConfig(ctx, databaseID)
	if err != nil {
		return LogicalBackupConfig{}, err
	}

	logicalConfig, isLogical := readConfig.(LogicalBackupConfig)
	if !isLogical {
		return LogicalBackupConfig{}, fmt.Errorf("%w: %s", ErrNotLogicalDatabase, databaseID)
	}

	return logicalConfig, nil
}

// LockAndGetLogicalBackupConfig reads the config for a backup start and holds a share lock on its saved
// row until the caller's transaction ends, so a save that changes the storage either commits first or
// waits for the started backup's row.
func (s *Service) LockAndGetLogicalBackupConfig(
	ctx context.Context,
	databaseID uuid.UUID,
) (LogicalBackupConfig, error) {
	if err := s.databases.LockDatabaseForBackup(ctx, databaseID); err != nil {
		return LogicalBackupConfig{}, err
	}
	if err := s.repository.lockDatabaseConfig(ctx, databaseID); err != nil {
		return LogicalBackupConfig{}, err
	}
	_, _, err := s.repository.get(ctx, databaseID, forShareRowLock)
	if err != nil {
		return LogicalBackupConfig{}, err
	}

	return s.GetLogicalBackupConfig(ctx, databaseID)
}

func (s *Service) GetPhysicalBackupConfig(ctx context.Context, databaseID uuid.UUID) (PhysicalBackupConfig, error) {
	readConfig, err := s.ReadBackupConfig(ctx, databaseID)
	if err != nil {
		return PhysicalBackupConfig{}, err
	}

	physicalConfig, isPhysical := readConfig.(PhysicalBackupConfig)
	if !isPhysical {
		return PhysicalBackupConfig{}, fmt.Errorf("%w: %s", ErrNotPhysicalDatabase, databaseID)
	}

	return physicalConfig, nil
}

func (s *Service) ListEnabledLogicalBackupConfigs(ctx context.Context) ([]LogicalBackupConfig, error) {
	return s.repository.listEnabledLogical(ctx)
}

func (s *Service) ListEnabledPhysicalBackupConfigs(ctx context.Context) ([]PhysicalBackupConfig, error) {
	enabledConfigs, err := s.repository.listEnabledPhysical(ctx)
	if err != nil {
		return nil, err
	}

	return s.fitToBackupTypes(ctx, enabledConfigs)
}

// ListPhysicalBackupConfigsWithPendingRequest leaves out a config with no storage, which cannot start its backup
// and which a full scheduler pass reports.
func (s *Service) ListPhysicalBackupConfigsWithPendingRequest(ctx context.Context) ([]PhysicalBackupConfig, error) {
	pendingConfigs, err := s.repository.listPhysicalWithPendingRequest(ctx)
	if err != nil {
		return nil, err
	}

	return s.fitToBackupTypes(ctx, pendingConfigs)
}

// ListBackupConfigs pages through the saved configs of a workspace's databases or of the databases
// that back up into a storage, ordered by database ID.
func (s *Service) ListBackupConfigs(ctx context.Context, actor users.Actor, query ListQuery) (Page, error) {
	if (query.WorkspaceID == nil) == (query.StorageID == nil) {
		return Page{}, ErrListFilterRequired
	}

	limit := defaultConfigPageLimit
	if query.Limit != nil {
		limit = *query.Limit
	}

	if limit < 1 || limit > maxConfigPageLimit {
		return Page{}, apierror.NewInvalidFieldsProblem([]apierror.FieldError{{
			Field:  "limit",
			Detail: fmt.Sprintf("must be between 1 and %d", maxConfigPageLimit),
		}})
	}

	filter := configListFilter{databaseIDs: nil, storageID: query.StorageID, afterID: nil, limit: limit + 1}

	if query.After != nil {
		afterID, err := decodeCursor(*query.After)
		if err != nil {
			return Page{}, err
		}

		filter.afterID = &afterID
	}

	if query.WorkspaceID != nil {
		databaseIDs, err := s.listWorkspaceDatabaseIDs(ctx, actor, *query.WorkspaceID)
		if err != nil {
			return Page{}, err
		}

		filter.databaseIDs = databaseIDs
	} else {
		err := s.authorizeStorage(ctx, actor, *query.StorageID)
		if err != nil {
			return Page{}, err
		}
	}

	if filter.databaseIDs != nil && len(filter.databaseIDs) == 0 {
		return Page{Configs: []AnyBackupConfig{}, NextCursor: nil}, nil
	}

	listedRows, err := s.repository.list(ctx, filter)
	if err != nil {
		return Page{}, err
	}

	pageRows := listedRows[:min(len(listedRows), limit)]

	pageConfigs, err := s.loadConfigs(ctx, pageRows)
	if err != nil {
		return Page{}, err
	}

	pageConfigs, err = s.withCleanup(ctx, actor, pageConfigs)
	if err != nil {
		return Page{}, err
	}

	page := Page{Configs: pageConfigs, NextCursor: nil}

	if len(listedRows) > limit {
		nextCursor := encodeCursor(pageRows[len(pageRows)-1].DatabaseID)
		page.NextCursor = &nextCursor
	}

	return page, nil
}

// A transfer can commit between the initial access check and the locks. The locked database decides
// both storage ownership and cleanup attribution.
func (s *Service) SaveBackupConfig(
	ctx context.Context,
	actor users.Actor,
	savedConfig AnyBackupConfig,
) (AnyBackupConfig, error) {
	databaseID := savedConfig.GetBackupConfig().DatabaseID

	configDatabase, err := s.getDatabase(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	err = s.authorizeDatabase(ctx, actor, configDatabase, workspaces.ActionEditContent)
	if err != nil {
		return nil, err
	}

	var saved AnyBackupConfig
	err = s.transactor.Transaction(ctx, func(ctx context.Context) error {
		if _, err := s.LockBackupConfig(ctx, databaseID); err != nil {
			return err
		}

		configDatabase, err := s.getDatabase(ctx, databaseID)
		if err != nil {
			return err
		}
		if err := s.authorizeDatabase(ctx, actor, configDatabase, workspaces.ActionEditContent); err != nil {
			return err
		}

		backupTypes, err := s.databases.GetPhysicalBackupTypes(ctx, []uuid.UUID{databaseID})
		if err != nil {
			return err
		}

		backupType, isPhysical := backupTypes[databaseID]
		scope := storageSwitchScope{WorkspaceID: configDatabase.WorkspaceID}

		switch typedConfig := savedConfig.(type) {
		case LogicalBackupConfig:
			if isPhysical {
				return ErrMethodMismatch
			}
			saved, err = s.saveLogicalBackupConfig(ctx, scope, typedConfig)
		case PhysicalBackupConfig:
			if !isPhysical {
				return ErrMethodMismatch
			}
			typedConfig.BackupType = backupType
			saved, err = s.savePhysicalBackupConfig(ctx, scope, typedConfig)
		default:
			return ErrMethodMismatch
		}
		if err != nil {
			return err
		}

		configs, err := s.withCleanup(ctx, actor, []AnyBackupConfig{saved})
		if err != nil {
			return err
		}

		saved = configs[0]

		return nil
	})
	if err != nil {
		return nil, err
	}

	return saved, nil
}

func (s *Service) RequestFullBackup(ctx context.Context, databaseID uuid.UUID) error {
	_, err := s.repository.updateForceFullRequestedAt(ctx, databaseID, s.clock.Now().UTC())

	return err
}

func (s *Service) ClearFullBackupRequest(ctx context.Context, databaseID uuid.UUID, requestedAt *time.Time) error {
	_, err := s.repository.clearForceFullRequestedAt(ctx, databaseID, requestedAt)

	return err
}

func (s *Service) RequestIncrementalBackup(ctx context.Context, databaseID uuid.UUID) error {
	_, err := s.repository.updateForceIncrementalRequestedAt(ctx, databaseID, s.clock.Now().UTC())

	return err
}

func (s *Service) ClearIncrementalBackupRequest(
	ctx context.Context,
	databaseID uuid.UUID,
	requestedAt *time.Time,
) error {
	_, err := s.repository.clearForceIncrementalRequestedAt(ctx, databaseID, requestedAt)

	return err
}

func (s *Service) saveLogicalBackupConfig(
	ctx context.Context,
	scope storageSwitchScope,
	savedConfig LogicalBackupConfig,
) (AnyBackupConfig, error) {
	savedConfig.NotificationTriggers = normalizeTriggers(savedConfig.NotificationTriggers)

	err := savedConfig.validate()
	if err != nil {
		return nil, err
	}

	err = s.transactor.Transaction(ctx, func(ctx context.Context) error {
		if err := s.databases.LockDatabaseForBackup(ctx, savedConfig.DatabaseID); err != nil {
			return err
		}
		if err := s.repository.lockDatabaseConfig(ctx, savedConfig.DatabaseID); err != nil {
			return err
		}
		previousConfig, isSaved, err := s.repository.get(ctx, savedConfig.DatabaseID, forUpdateRowLock)
		if err != nil {
			return err
		}

		err = s.checkStorageInWorkspace(ctx, savedConfig.StorageID, scope.WorkspaceID)
		if err != nil {
			return err
		}

		var previousStorageID *uuid.UUID
		if isSaved {
			previousStorageID = previousConfig.StorageID
		}

		hasReassignedStorage, err := s.requestStorageChangeCleanup(
			ctx,
			storageSwitchSave{Scope: scope, PreviousStorageID: previousStorageID, Config: savedConfig},
		)
		if err != nil {
			return err
		}

		if err := s.repository.upsertLogical(ctx, savedConfig); err != nil {
			return err
		}

		if hasReassignedStorage {
			return s.repository.incrementStorageGeneration(ctx, savedConfig.DatabaseID)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return savedConfig, nil
}

// The physical reaction runs after commit, so concurrent backups already see the new storage.
func (s *Service) savePhysicalBackupConfig(
	ctx context.Context,
	scope storageSwitchScope,
	savedConfig PhysicalBackupConfig,
) (AnyBackupConfig, error) {
	savedConfig.NotificationTriggers = normalizeTriggers(savedConfig.NotificationTriggers)

	err := savedConfig.validate()
	if err != nil {
		return nil, err
	}

	var previousConfig *PhysicalBackupConfig

	err = s.transactor.Transaction(ctx, func(ctx context.Context) error {
		if err := s.repository.lockPhysicalHistory(ctx, savedConfig.DatabaseID); err != nil {
			return err
		}
		if err := s.databases.LockDatabaseForBackup(ctx, savedConfig.DatabaseID); err != nil {
			return err
		}
		err := s.repository.lockDatabaseConfig(ctx, savedConfig.DatabaseID)
		if err != nil {
			return err
		}

		_, _, err = s.repository.get(ctx, savedConfig.DatabaseID, forUpdateRowLock)
		if err != nil {
			return err
		}

		// The type is read under the lock a type change also takes, so the config is validated against
		// the type that is committed when it is written.
		backupTypes, err := s.databases.GetPhysicalBackupTypes(ctx, []uuid.UUID{savedConfig.DatabaseID})
		if err != nil {
			return err
		}

		savedConfig.BackupType = backupTypes[savedConfig.DatabaseID]

		err = savedConfig.validate()
		if err != nil {
			return err
		}

		previousConfigs, err := s.repository.listPhysical(ctx, []uuid.UUID{savedConfig.DatabaseID})
		if err != nil {
			return err
		}

		if len(previousConfigs) > 0 {
			previousConfig = &previousConfigs[0]
			previousConfig.BackupType = savedConfig.BackupType
			savedConfig.ForceFullRequestedAt = previousConfig.ForceFullRequestedAt
			savedConfig.ForceIncrementalRequestedAt = previousConfig.ForceIncrementalRequestedAt
		}

		err = s.checkStorageInWorkspace(ctx, savedConfig.StorageID, scope.WorkspaceID)
		if err != nil {
			return err
		}

		var previousStorageID *uuid.UUID
		if previousConfig != nil {
			previousStorageID = previousConfig.StorageID
		}

		hasReassignedStorage, err := s.requestStorageChangeCleanup(
			ctx,
			storageSwitchSave{Scope: scope, PreviousStorageID: previousStorageID, Config: savedConfig},
		)
		if err != nil {
			return err
		}

		if err := s.repository.upsertPhysical(ctx, savedConfig); err != nil {
			return err
		}

		if hasReassignedStorage || !isSameStorage(previousStorageID, savedConfig.StorageID) {
			if err := s.repository.incrementStorageGeneration(ctx, savedConfig.DatabaseID); err != nil {
				return err
			}
		}

		if hasReassignedStorage {
			if err := s.databases.ResetBackupHistory(ctx, savedConfig.DatabaseID); err != nil {
				return err
			}
			if err := s.repository.resetPhysicalRequests(
				ctx,
				physicalRequestsReset{DatabaseID: savedConfig.DatabaseID, RequestedAt: s.clock.Now()},
			); err != nil {
				return err
			}
		}

		return database.AfterCommit(ctx, func(ctx context.Context) {
			if err := s.physicalConfigSaved.ReactToSavedConfig(ctx, previousConfig, savedConfig); err != nil {
				s.logger.ErrorContext(ctx, "failed to react to saved physical config", slog.Any("error", err))
			}
		})
	})
	if err != nil {
		return nil, err
	}

	return savedConfig, nil
}

// checkStorageInWorkspace answers a storage that does not exist like one of another workspace, so a
// save reveals nothing about storages outside the database's workspace.
func (s *Service) checkStorageInWorkspace(ctx context.Context, storageID *uuid.UUID, workspaceID uuid.UUID) error {
	if storageID == nil {
		return nil
	}

	configStorage, isFound, err := s.storages.GetStorageBlockingTransfer(ctx, *storageID)
	if err != nil {
		return err
	}

	if !isFound || configStorage.WorkspaceID != workspaceID {
		return ErrStorageNotInDatabaseWorkspace
	}

	return nil
}

func (s *Service) readPhysicalBackupConfig(
	ctx context.Context,
	databaseID uuid.UUID,
	backupType databases.PhysicalBackupType,
) (PhysicalBackupConfig, error) {
	physicalConfigs, err := s.repository.listPhysical(ctx, []uuid.UUID{databaseID})
	if err != nil {
		return PhysicalBackupConfig{}, err
	}

	if len(physicalConfigs) == 0 {
		return newDefaultPhysicalBackupConfig(databaseID, backupType), nil
	}

	return physicalConfigs[0].coerceToBackupType(backupType), nil
}

// fitToBackupTypes reads the backup types of the configs once and fits every config to its type. A
// config left unfit by an earlier release is returned fitted and stays stored as it is until its next
// save.
func (s *Service) fitToBackupTypes(
	ctx context.Context,
	physicalConfigs []PhysicalBackupConfig,
) ([]PhysicalBackupConfig, error) {
	databaseIDs := make([]uuid.UUID, 0, len(physicalConfigs))
	for _, physicalConfig := range physicalConfigs {
		databaseIDs = append(databaseIDs, physicalConfig.DatabaseID)
	}

	backupTypes, err := s.databases.GetPhysicalBackupTypes(ctx, databaseIDs)
	if err != nil {
		return nil, err
	}

	fittedConfigs := make([]PhysicalBackupConfig, 0, len(physicalConfigs))

	for _, physicalConfig := range physicalConfigs {
		backupType, isFound := backupTypes[physicalConfig.DatabaseID]
		if !isFound {
			return nil, fmt.Errorf("%w: %s", errUnknownBackupType, physicalConfig.DatabaseID)
		}

		fittedConfigs = append(fittedConfigs, physicalConfig.coerceToBackupType(backupType))
	}

	return fittedConfigs, nil
}

// loadConfigs reads the rows of each method in one batch and keeps the order of rows.
func (s *Service) loadConfigs(ctx context.Context, rows []backupConfigRow) ([]AnyBackupConfig, error) {
	var logicalDatabaseIDs, physicalDatabaseIDs []uuid.UUID

	for _, row := range rows {
		switch row.Method {
		case MethodLogical:
			logicalDatabaseIDs = append(logicalDatabaseIDs, row.DatabaseID)
		case MethodPhysical:
			physicalDatabaseIDs = append(physicalDatabaseIDs, row.DatabaseID)
		}
	}

	configsByDatabaseID := make(map[uuid.UUID]AnyBackupConfig, len(rows))

	if len(logicalDatabaseIDs) > 0 {
		logicalConfigs, err := s.repository.listLogical(ctx, logicalDatabaseIDs)
		if err != nil {
			return nil, err
		}

		for _, logicalConfig := range logicalConfigs {
			configsByDatabaseID[logicalConfig.DatabaseID] = logicalConfig
		}
	}

	if len(physicalDatabaseIDs) > 0 {
		physicalConfigs, err := s.repository.listPhysical(ctx, physicalDatabaseIDs)
		if err != nil {
			return nil, err
		}

		fittedConfigs, err := s.fitToBackupTypes(ctx, physicalConfigs)
		if err != nil {
			return nil, err
		}

		for _, fittedConfig := range fittedConfigs {
			configsByDatabaseID[fittedConfig.DatabaseID] = fittedConfig
		}
	}

	pageConfigs := make([]AnyBackupConfig, 0, len(rows))

	for _, row := range rows {
		loadedConfig, isFound := configsByDatabaseID[row.DatabaseID]
		if !isFound {
			return nil, fmt.Errorf("%w: %s", errMethodRowMissing, row.DatabaseID)
		}

		pageConfigs = append(pageConfigs, loadedConfig)
	}

	return pageConfigs, nil
}

func (s *Service) listWorkspaceDatabaseIDs(
	ctx context.Context,
	actor users.Actor,
	workspaceID uuid.UUID,
) ([]uuid.UUID, error) {
	err := s.workspaces.Authorize(ctx, actor, workspaceID, workspaces.ActionView)
	if err != nil {
		return nil, err
	}

	allDatabases, err := s.databases.ListAllDatabases(ctx)
	if err != nil {
		return nil, err
	}

	databaseIDs := make([]uuid.UUID, 0)

	for _, workspaceDatabase := range allDatabases {
		if workspaceDatabase.WorkspaceID == workspaceID {
			databaseIDs = append(databaseIDs, workspaceDatabase.ID)
		}
	}

	return databaseIDs, nil
}

// authorizeStorage answers a storage of a workspace the actor is not a member of like a missing one.
func (s *Service) authorizeStorage(ctx context.Context, actor users.Actor, storageID uuid.UUID) error {
	foundStorages, err := s.storages.GetStorages(ctx, []uuid.UUID{storageID})
	if err != nil {
		return err
	}

	if len(foundStorages) == 0 {
		return ErrStorageNotFound
	}

	err = s.workspaces.Authorize(ctx, actor, foundStorages[0].WorkspaceID, workspaces.ActionView)
	if isNotFound(err) {
		return ErrStorageNotFound
	}

	return err
}

// authorizeDatabase answers a database of a workspace the actor is not a member of like a missing one.
func (s *Service) authorizeDatabase(
	ctx context.Context,
	actor users.Actor,
	authorizedDatabase databases.Database,
	action workspaces.Action,
) error {
	err := s.workspaces.Authorize(ctx, actor, authorizedDatabase.WorkspaceID, action)
	if isNotFound(err) {
		return ErrDatabaseNotFound
	}

	return err
}

func isNotFound(err error) bool {
	var problem *apierror.Problem

	return errors.As(err, &problem) && problem.Code == apierror.CodeNotFound
}

func (s *Service) getDatabase(ctx context.Context, databaseID uuid.UUID) (databases.Database, error) {
	foundDatabases, err := s.databases.GetDatabases(ctx, []uuid.UUID{databaseID})
	if err != nil {
		return databases.Database{}, err
	}

	if len(foundDatabases) == 0 {
		return databases.Database{}, ErrDatabaseNotFound
	}

	return foundDatabases[0], nil
}

func isSameStorage(first, second *uuid.UUID) bool {
	if first == nil || second == nil {
		return first == second
	}

	return *first == *second
}

type PhysicalConfigSavedReactionFunc func(context.Context, *PhysicalBackupConfig, PhysicalBackupConfig) error

func (f PhysicalConfigSavedReactionFunc) ReactToSavedConfig(
	ctx context.Context,
	previous *PhysicalBackupConfig,
	saved PhysicalBackupConfig,
) error {
	return f(ctx, previous, saved)
}
