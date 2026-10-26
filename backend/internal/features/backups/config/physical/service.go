package backups_config_physical

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	backups_config_storage_usage "databasus-backend/internal/features/backups/config/storage_usage"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/intervals"
	workspaces_bridge "databasus-backend/internal/features/workspaces"
	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/modules/notifiers"
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
	"databasus-backend/internal/storage"
)

type BackupConfigService struct {
	backupConfigRepository *BackupConfigRepository
	databaseService        *databases.DatabaseService
	// getStorages and getNotifiers are read per call: each test application installs its own modules.
	getStorages      func() storages.Module
	getNotifiers     func() notifiers.Module
	workspaceService *workspaces_bridge.Access
	logger           *slog.Logger

	dbStorageChangeListener    BackupConfigStorageChangeListener
	backupCancellationListener BackupCancellationListener
}

func (s *BackupConfigService) SetDatabaseStorageChangeListener(
	dbStorageChangeListener BackupConfigStorageChangeListener,
) {
	s.dbStorageChangeListener = dbStorageChangeListener
}

func (s *BackupConfigService) SetBackupCancellationListener(
	backupCancellationListener BackupCancellationListener,
) {
	s.backupCancellationListener = backupCancellationListener
}

func (s *BackupConfigService) SaveBackupConfigWithAuth(
	ctx context.Context,
	actor users.Actor,
	backupConfig *PhysicalBackupConfig,
) (*PhysicalBackupConfig, error) {
	database, err := s.databaseService.GetDatabase(ctx, actor, backupConfig.DatabaseID)
	if err != nil {
		return nil, err
	}

	if database.WorkspaceID == nil {
		return nil, errors.New("cannot save backup config for database without workspace")
	}

	isAllowed, err := s.workspaceService.IsActorAllowed(ctx, actor, *database.WorkspaceID, workspaces.ActionEditContent)
	if err != nil {
		return nil, err
	}
	if !isAllowed {
		return nil, errors.New("insufficient permissions to modify backup configuration")
	}

	if database.PostgresqlPhysical == nil {
		return nil, errors.New(
			"physical backup config requires the owning database to be of type POSTGRES_PHYSICAL",
		)
	}

	backupConfig.PostgresqlPhysical = database.PostgresqlPhysical

	if err := backupConfig.Validate(); err != nil {
		return nil, err
	}

	if err := takeStorageIDFromEmbeddedStorage(backupConfig); err != nil {
		return nil, err
	}

	if err := s.checkStorageWorkspace(ctx, backupConfig, *database.WorkspaceID); err != nil {
		return nil, err
	}

	savedConfig, err := s.saveBackupConfig(ctx, backupConfig, database.WorkspaceID)
	if err != nil {
		return nil, err
	}

	return savedConfig, s.fillStorage(ctx, savedConfig)
}

func (s *BackupConfigService) SaveBackupConfig(
	ctx context.Context,
	backupConfig *PhysicalBackupConfig,
) (*PhysicalBackupConfig, error) {
	return s.saveBackupConfig(ctx, backupConfig, nil)
}

func (s *BackupConfigService) GetAndRepairBackupConfigByDbIdWithAuth(
	ctx context.Context,
	actor users.Actor,
	databaseID uuid.UUID,
) (*PhysicalBackupConfig, error) {
	_, err := s.databaseService.GetDatabase(ctx, actor, databaseID)
	if err != nil {
		return nil, err
	}

	backupConfig, err := s.GetBackupConfigByDbId(databaseID)
	if err != nil {
		return nil, err
	}

	repairedConfig, err := s.repairAndSaveConfigForBackupType(ctx, backupConfig)
	if err != nil {
		return nil, err
	}

	return repairedConfig, s.fillStorage(ctx, repairedConfig)
}

func (s *BackupConfigService) GetBackupConfigByDbId(
	databaseID uuid.UUID,
) (*PhysicalBackupConfig, error) {
	config, err := s.backupConfigRepository.FindByDatabaseID(databaseID)
	if err != nil {
		return nil, err
	}

	if config == nil {
		if err := s.initializeDefaultConfig(databaseID); err != nil {
			return nil, err
		}

		return s.backupConfigRepository.FindByDatabaseID(databaseID)
	}

	return config, nil
}

func (s *BackupConfigService) GetBackupConfigsWithEnabledBackups() (
	[]*PhysicalBackupConfig,
	error,
) {
	return s.backupConfigRepository.GetWithEnabledBackups()
}

func (s *BackupConfigService) RequestFullBackupNow(databaseID uuid.UUID) error {
	return s.backupConfigRepository.RequestFullBackupNow(databaseID)
}

func (s *BackupConfigService) ClearFullBackupRequest(databaseID uuid.UUID, requestedAt *time.Time) error {
	return s.backupConfigRepository.ClearFullBackupRequest(databaseID, requestedAt)
}

func (s *BackupConfigService) RequestIncrementalBackupNow(databaseID uuid.UUID) error {
	return s.backupConfigRepository.RequestIncrementalBackupNow(databaseID)
}

func (s *BackupConfigService) ClearIncrementalBackupRequest(databaseID uuid.UUID, requestedAt *time.Time) error {
	return s.backupConfigRepository.ClearIncrementalBackupRequest(databaseID, requestedAt)
}

func (s *BackupConfigService) OnDatabaseCopied(ctx context.Context, originalDatabaseID, newDatabaseID uuid.UUID) {
	originalConfig, err := s.backupConfigRepository.FindByDatabaseID(originalDatabaseID)
	if err != nil || originalConfig == nil {
		return
	}

	newConfig := originalConfig.Copy(newDatabaseID)

	_, _ = s.SaveBackupConfig(ctx, newConfig)
}

func (s *BackupConfigService) OnBackupTypeChanged(
	ctx context.Context,
	change databases.BackupTypeChange,
) {
	logger := s.logger.With("database_id", change.DatabaseID)

	// Before the config write and regardless of its outcome: a streamer left
	// running keeps pinning WAL on the source cluster.
	if s.backupCancellationListener != nil &&
		change.OldBackupType.IsWalStreaming() && !change.NewBackupType.IsWalStreaming() {
		s.backupCancellationListener.OnWalStreamingDisabled(ctx, change.DatabaseID)
	}

	backupConfig, err := s.backupConfigRepository.FindByDatabaseID(change.DatabaseID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to load backup config after backup type change", "error", err)

		return
	}

	if backupConfig == nil {
		return
	}

	backupConfig.CoerceFieldsForBackupType(change.NewBackupType)

	if err := backupConfig.Validate(); err != nil {
		logger.ErrorContext(ctx, "backup config is not valid for the new backup type", "error", err)

		return
	}

	if _, err := s.backupConfigRepository.Save(ctx, backupConfig); err != nil {
		logger.ErrorContext(ctx, "failed to save backup config after backup type change", "error", err)

		return
	}

	logger.InfoContext(ctx, fmt.Sprintf("backup config updated for backup type change: %s -> %s",
		change.OldBackupType, change.NewBackupType))
}

func (s *BackupConfigService) CreateDisabledBackupConfig(databaseID uuid.UUID) error {
	return s.initializeDefaultConfig(databaseID)
}

func (s *BackupConfigService) TransferDatabaseToWorkspace(
	ctx context.Context,
	actor users.Actor,
	databaseID uuid.UUID,
	request *TransferDatabaseRequest,
) error {
	database, err := s.databaseService.GetDatabaseByID(databaseID)
	if err != nil {
		return err
	}

	if database.WorkspaceID == nil {
		return ErrDatabaseHasNoWorkspace
	}

	isAllowedInSource, err := s.workspaceService.IsActorAllowed(
		ctx,
		actor,
		*database.WorkspaceID,
		workspaces.ActionEditContent,
	)
	if err != nil {
		return err
	}
	if !isAllowedInSource {
		return ErrInsufficientPermissionsInSourceWorkspace
	}

	isAllowedInTarget, err := s.workspaceService.IsActorAllowed(
		ctx,
		actor,
		request.TargetWorkspaceID,
		workspaces.ActionEditContent,
	)
	if err != nil {
		return err
	}
	if !isAllowedInTarget {
		return ErrInsufficientPermissionsInTargetWorkspace
	}

	if err := s.validateTargetNotifiers(ctx, request); err != nil {
		return err
	}

	backupConfig, err := s.GetBackupConfigByDbId(databaseID)
	if err != nil {
		return err
	}

	switch {
	case request.IsTransferWithStorage:
		err = s.transferDatabaseWithStorage(ctx, actor, backupConfig, request.TargetWorkspaceID)
		if err != nil {
			return err
		}
	case request.TargetStorageID != nil:
		err = s.moveToTargetStorage(ctx, backupConfig, request)
		if err != nil {
			return err
		}

		err = s.databaseService.TransferDatabaseToWorkspace(ctx, databaseID, request.TargetWorkspaceID)
		if err != nil {
			return err
		}
	default:
		return ErrTargetStorageNotSpecified
	}

	if request.IsTransferWithNotifiers {
		err = s.transferNotifiers(ctx, actor, database, request.TargetWorkspaceID)
		if err != nil {
			return err
		}
	}

	if len(request.TargetNotifierIDs) > 0 {
		if err := s.databaseService.UpdateDatabaseNotifiers(databaseID, request.TargetNotifierIDs); err != nil {
			return err
		}
	}

	return nil
}

// saveBackupConfig checks the storage again under its share lock when databaseWorkspaceID names the
// workspace it must belong to, in the transaction that writes the config. The listeners run after
// that transaction commits, so a backup they start reads the new config.
func (s *BackupConfigService) saveBackupConfig(
	ctx context.Context,
	backupConfig *PhysicalBackupConfig,
	databaseWorkspaceID *uuid.UUID,
) (*PhysicalBackupConfig, error) {
	if err := takeStorageIDFromEmbeddedStorage(backupConfig); err != nil {
		return nil, err
	}

	if backupConfig.PostgresqlPhysical == nil {
		database, err := s.databaseService.GetDatabaseByID(backupConfig.DatabaseID)
		if err != nil {
			return nil, err
		}

		if database.PostgresqlPhysical == nil {
			return nil, errors.New(
				"physical backup config requires the owning database to be of type POSTGRES_PHYSICAL",
			)
		}

		backupConfig.PostgresqlPhysical = database.PostgresqlPhysical
	}

	if err := backupConfig.Validate(); err != nil {
		return nil, err
	}

	existingConfig, err := s.GetBackupConfigByDbId(backupConfig.DatabaseID)
	if err != nil {
		return nil, err
	}

	savedConfig, err := s.writeBackupConfig(ctx, backupConfig, databaseWorkspaceID)
	if err != nil {
		return nil, err
	}

	if existingConfig != nil && s.backupCancellationListener != nil &&
		existingConfig.IsBackupsEnabled && !backupConfig.IsBackupsEnabled {
		s.backupCancellationListener.OnBackupsDisabled(ctx, backupConfig.DatabaseID)
	}

	isStorageChanged := existingConfig != nil &&
		backupConfig.StorageID != nil &&
		!storageIDsEqual(existingConfig.StorageID, backupConfig.StorageID)

	if isStorageChanged && s.backupCancellationListener != nil {
		s.backupCancellationListener.OnBackupsStorageChanged(ctx, backupConfig.DatabaseID)
	}

	// Runs after the save, so a full backup or a streamer that starts meanwhile already writes to
	// the new storage instead of surviving the change on the old one.
	if s.dbStorageChangeListener != nil && backupConfig.StorageID != nil {
		if err := s.dbStorageChangeListener.OnBackupsStorageChanged(
			ctx,
			backupConfig.DatabaseID,
			*backupConfig.StorageID,
		); err != nil {
			return nil, fmt.Errorf("remove backups kept in a previous storage: %w", err)
		}
	}

	return savedConfig, nil
}

// transferDatabaseWithStorage moves the database before its storage, so the storages module's
// Transferring check finds the database in the target workspace already and lets the storage through.
func (s *BackupConfigService) transferDatabaseWithStorage(
	ctx context.Context,
	actor users.Actor,
	backupConfig *PhysicalBackupConfig,
	targetWorkspaceID uuid.UUID,
) error {
	if backupConfig.StorageID == nil {
		return ErrDatabaseHasNoStorage
	}

	databaseIDs, err := backups_config_storage_usage.GetDatabaseIDsUsingStorage(ctx, *backupConfig.StorageID)
	if err != nil {
		return err
	}

	for _, databaseID := range databaseIDs {
		if databaseID != backupConfig.DatabaseID {
			return ErrStorageHasOtherAttachedDatabases
		}
	}

	err = s.databaseService.TransferDatabaseToWorkspace(ctx, backupConfig.DatabaseID, targetWorkspaceID)
	if err != nil {
		return err
	}

	return s.getStorages().TransferStorage(ctx, actor, storages.StorageTransfer{
		StorageID:         *backupConfig.StorageID,
		TargetWorkspaceID: targetWorkspaceID,
	})
}

func (s *BackupConfigService) moveToTargetStorage(
	ctx context.Context,
	backupConfig *PhysicalBackupConfig,
	request *TransferDatabaseRequest,
) error {
	targetStorages, err := s.getStorages().GetStorages(ctx, []uuid.UUID{*request.TargetStorageID})
	if err != nil {
		return err
	}

	if len(targetStorages) == 0 {
		return ErrStorageNotFound
	}

	if targetStorages[0].WorkspaceID != request.TargetWorkspaceID {
		return ErrTargetStorageNotInTargetWorkspace
	}

	backupConfig.StorageID = request.TargetStorageID

	_, err = s.backupConfigRepository.Save(ctx, backupConfig)

	return err
}

func (s *BackupConfigService) checkStorageWorkspace(
	ctx context.Context,
	backupConfig *PhysicalBackupConfig,
	databaseWorkspaceID uuid.UUID,
) error {
	if backupConfig.StorageID == nil {
		return nil
	}

	configStorages, err := s.getStorages().GetStorages(ctx, []uuid.UUID{*backupConfig.StorageID})
	if err != nil {
		return err
	}

	if len(configStorages) == 0 {
		return ErrStorageNotFound
	}

	return checkStorageInWorkspace(configStorages[0], true, databaseWorkspaceID)
}

func checkStorageInWorkspace(configStorage storages.Storage, isFound bool, databaseWorkspaceID uuid.UUID) error {
	if !isFound {
		return ErrStorageNotFound
	}

	if configStorage.WorkspaceID != databaseWorkspaceID {
		return ErrStorageNotInDatabaseWorkspace
	}

	return nil
}

// fillStorage names the config's storage for the answers of /backup-configs.
func (s *BackupConfigService) fillStorage(ctx context.Context, backupConfig *PhysicalBackupConfig) error {
	backupConfig.Storage = nil

	if backupConfig.StorageID == nil {
		return nil
	}

	configStorages, err := s.getStorages().GetStorages(ctx, []uuid.UUID{*backupConfig.StorageID})
	if err != nil {
		return err
	}

	for _, configStorage := range configStorages {
		backupConfig.Storage = &BackupConfigStorage{
			ID:   configStorage.ID,
			Name: configStorage.Name,
			Type: configStorage.Type,
		}
	}

	return nil
}

// writeBackupConfig takes the storage's share lock when databaseWorkspaceID is set, see
// saveBackupConfig.
func (s *BackupConfigService) writeBackupConfig(
	ctx context.Context,
	backupConfig *PhysicalBackupConfig,
	databaseWorkspaceID *uuid.UUID,
) (*PhysicalBackupConfig, error) {
	if databaseWorkspaceID == nil || backupConfig.StorageID == nil {
		return s.backupConfigRepository.Save(ctx, backupConfig)
	}

	var savedConfig *PhysicalBackupConfig

	// The share lock on the storage keeps a transfer of it waiting until the config is written, so
	// the transfer sees this config, or this save sees the storage in its new workspace.
	err := storage.GetTransactor().Transaction(ctx, func(ctx context.Context) error {
		configStorage, isFound, err := s.getStorages().GetStorageBlockingTransfer(ctx, *backupConfig.StorageID)
		if err != nil {
			return err
		}

		err = checkStorageInWorkspace(configStorage, isFound, *databaseWorkspaceID)
		if err != nil {
			return err
		}

		savedConfig, err = s.backupConfigRepository.Save(ctx, backupConfig)

		return err
	})
	if err != nil {
		return nil, err
	}

	return savedConfig, nil
}

// Heals rows left unsaveable by a backup type switch that predates this repair.
// Owner-facing reads only: a background caller repairing a row it merely polled
// would race the owner's own save.
func (s *BackupConfigService) repairAndSaveConfigForBackupType(
	ctx context.Context,
	backupConfig *PhysicalBackupConfig,
) (*PhysicalBackupConfig, error) {
	if backupConfig.PostgresqlPhysical == nil || backupConfig.Validate() == nil {
		return backupConfig, nil
	}

	logger := s.logger.With("database_id", backupConfig.DatabaseID)

	backupConfig.CoerceFieldsForBackupType(backupConfig.PostgresqlPhysical.BackupType)

	if err := backupConfig.Validate(); err != nil {
		logger.WarnContext(ctx, "backup config cannot be repaired for its backup type", "error", err)

		return backupConfig, nil
	}

	repairedConfig, err := s.backupConfigRepository.Save(ctx, backupConfig)
	if err != nil {
		return nil, err
	}

	logger.InfoContext(ctx, "repaired backup config for its backup type")

	return repairedConfig, nil
}

func (s *BackupConfigService) initializeDefaultConfig(databaseID uuid.UUID) error {
	database, err := s.databaseService.GetDatabaseByID(databaseID)
	if err != nil {
		return err
	}

	defaultConfig := &PhysicalBackupConfig{
		DatabaseID:       databaseID,
		IsBackupsEnabled: false,
		FullBackupInterval: intervals.Interval{
			Type:      intervals.IntervalDaily,
			TimeOfDay: new(defaultBackupTimeOfDay),
		},
		Retention: RetentionFullBackups,
		FullBackupsRetention: FullBackupsRetention{
			Policy: FullBackupsRetentionPolicyLastN,
			Count:  defaultFullBackupsRetentionCount,
		},
		SendNotificationsOn: []BackupNotificationType{
			NotificationBackupFailed,
			NotificationBackupSuccess,
			NotificationChainBroken,
			NotificationWalGap,
		},
		Encryption: "NONE",
	}

	if database.PostgresqlPhysical != nil {
		defaultConfig.CoerceFieldsForBackupType(database.PostgresqlPhysical.BackupType)
	}

	_, err = s.backupConfigRepository.Save(context.Background(), defaultConfig)

	return err
}

// transferNotifiers runs after the database moved, so the notifiers module lets through the
// notifiers that only this database uses. A notifier that another database still uses is refused
// with in_use and stays in its workspace.
func (s *BackupConfigService) transferNotifiers(
	ctx context.Context,
	actor users.Actor,
	database *databases.Database,
	targetWorkspaceID uuid.UUID,
) error {
	for _, notifierID := range database.GetNotifierIDs() {
		err := s.getNotifiers().TransferNotifier(ctx, actor, notifiers.NotifierTransfer{
			NotifierID:        notifierID,
			TargetWorkspaceID: targetWorkspaceID,
		})

		var problem *apierror.Problem
		if errors.As(err, &problem) && problem.Code == apierror.CodeInUse {
			continue
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (s *BackupConfigService) validateTargetNotifiers(ctx context.Context, request *TransferDatabaseRequest) error {
	targetNotifiers, err := s.getNotifiers().GetNotifiers(ctx, request.TargetNotifierIDs)
	if err != nil {
		return err
	}

	targetNotifierIDs := make(map[uuid.UUID]bool, len(targetNotifiers))
	for _, targetNotifier := range targetNotifiers {
		if targetNotifier.WorkspaceID == request.TargetWorkspaceID {
			targetNotifierIDs[targetNotifier.ID] = true
		}
	}

	for _, notifierID := range request.TargetNotifierIDs {
		if !targetNotifierIDs[notifierID] {
			return ErrTargetNotifierNotInTargetWorkspace
		}
	}

	return nil
}

// A configuration refers to a storage only by its identifier. The storage endpoints are the only
// place that checks who may create or edit a storage, so an embedded storage object never reaches
// the database as anything but a reference.
func takeStorageIDFromEmbeddedStorage(backupConfig *PhysicalBackupConfig) error {
	if backupConfig.Storage == nil {
		return nil
	}

	if backupConfig.Storage.ID == uuid.Nil {
		return ErrStorageWithoutID
	}

	backupConfig.StorageID = &backupConfig.Storage.ID

	return nil
}

func storageIDsEqual(id1, id2 *uuid.UUID) bool {
	if id1 == nil && id2 == nil {
		return true
	}
	if id1 == nil || id2 == nil {
		return false
	}

	return *id1 == *id2
}
