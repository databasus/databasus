package backups_config_physical

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/intervals"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	users_models "databasus-backend/internal/features/users/models"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
)

type BackupConfigService struct {
	backupConfigRepository *BackupConfigRepository
	databaseService        *databases.DatabaseService
	storageService         *storages.StorageService
	notifierService        *notifiers.NotifierService
	workspaceService       *workspaces_services.WorkspaceService
	logger                 *slog.Logger

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

func (s *BackupConfigService) GetStorageAttachedDatabasesIDs(
	storageID uuid.UUID,
) ([]uuid.UUID, error) {
	databasesIDs, err := s.backupConfigRepository.GetDatabasesIDsByStorageID(storageID)
	if err != nil {
		return nil, err
	}

	return databasesIDs, nil
}

func (s *BackupConfigService) SaveBackupConfigWithAuth(
	ctx context.Context,
	user *users_models.User,
	backupConfig *PhysicalBackupConfig,
) (*PhysicalBackupConfig, error) {
	database, err := s.databaseService.GetDatabase(ctx, user, backupConfig.DatabaseID)
	if err != nil {
		return nil, err
	}

	if database.WorkspaceID == nil {
		return nil, errors.New("cannot save backup config for database without workspace")
	}

	canManage, err := s.workspaceService.CanUserManageDBs(ctx, *database.WorkspaceID, user)
	if err != nil {
		return nil, err
	}
	if !canManage {
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

	if backupConfig.Storage != nil && backupConfig.Storage.ID != uuid.Nil {
		storage, err := s.storageService.GetStorageByID(ctx, backupConfig.Storage.ID)
		if err != nil {
			return nil, err
		}
		if storage.WorkspaceID != *database.WorkspaceID {
			return nil, errors.New("storage does not belong to the same workspace as the database")
		}
	}

	return s.SaveBackupConfig(ctx, backupConfig)
}

func (s *BackupConfigService) SaveBackupConfig(
	ctx context.Context,
	backupConfig *PhysicalBackupConfig,
) (*PhysicalBackupConfig, error) {
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

	if existingConfig != nil {
		if s.dbStorageChangeListener != nil &&
			backupConfig.Storage != nil &&
			!storageIDsEqual(existingConfig.StorageID, &backupConfig.Storage.ID) {
			if err := s.dbStorageChangeListener.OnBeforeBackupsStorageChange(ctx,
				backupConfig.DatabaseID,
			); err != nil {
				return nil, err
			}
		}
	}

	savedConfig, err := s.backupConfigRepository.Save(backupConfig)
	if err != nil {
		return nil, err
	}

	if existingConfig != nil && s.backupCancellationListener != nil &&
		existingConfig.IsBackupsEnabled && !backupConfig.IsBackupsEnabled {
		s.backupCancellationListener.OnBackupsDisabled(ctx, backupConfig.DatabaseID)
	}

	return savedConfig, nil
}

func (s *BackupConfigService) GetAndRepairBackupConfigByDbIdWithAuth(
	ctx context.Context,
	user *users_models.User,
	databaseID uuid.UUID,
) (*PhysicalBackupConfig, error) {
	_, err := s.databaseService.GetDatabase(ctx, user, databaseID)
	if err != nil {
		return nil, err
	}

	backupConfig, err := s.GetBackupConfigByDbId(databaseID)
	if err != nil {
		return nil, err
	}

	return s.repairAndSaveConfigForBackupType(ctx, backupConfig)
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

func (s *BackupConfigService) GetBackupConfigsByDatabaseIDs(
	databaseIDs []uuid.UUID,
) ([]*PhysicalBackupConfig, error) {
	return s.backupConfigRepository.FindByDatabaseIDs(databaseIDs)
}

func (s *BackupConfigService) IsStorageInUse(
	ctx context.Context,
	user *users_models.User,
	storageID uuid.UUID,
) (bool, error) {
	_, err := s.storageService.GetStorage(ctx, user, storageID)
	if err != nil {
		return false, err
	}

	return s.storageService.IsStorageInUse(ctx, storageID)
}

func (s *BackupConfigService) CountDatabasesForStorage(
	ctx context.Context,
	user *users_models.User,
	storageID uuid.UUID,
) (int, error) {
	_, err := s.storageService.GetStorage(ctx, user, storageID)
	if err != nil {
		return 0, err
	}

	return s.storageService.CountDatabasesForStorage(ctx, storageID)
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

	if _, err := s.backupConfigRepository.Save(backupConfig); err != nil {
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
	user *users_models.User,
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

	canManageSource, err := s.workspaceService.CanUserManageDBs(ctx, *database.WorkspaceID, user)
	if err != nil {
		return err
	}
	if !canManageSource {
		return ErrInsufficientPermissionsInSourceWorkspace
	}

	canManageTarget, err := s.workspaceService.CanUserManageDBs(ctx, request.TargetWorkspaceID, user)
	if err != nil {
		return err
	}
	if !canManageTarget {
		return ErrInsufficientPermissionsInTargetWorkspace
	}

	if err := s.validateTargetNotifiers(ctx, request); err != nil {
		return err
	}

	backupConfig, err := s.GetBackupConfigByDbId(databaseID)
	if err != nil {
		return err
	}

	if request.IsTransferWithNotifiers {
		s.transferNotifiers(ctx, user, database, request.TargetWorkspaceID)
	}

	switch {
	case request.IsTransferWithStorage:
		if backupConfig.StorageID == nil {
			return ErrDatabaseHasNoStorage
		}

		attachedDatabasesIDs, err := s.storageService.GetStorageAttachedDatabasesIDs(
			*backupConfig.StorageID,
		)
		if err != nil {
			return err
		}

		for _, dbID := range attachedDatabasesIDs {
			if dbID != databaseID {
				return ErrStorageHasOtherAttachedDatabases
			}
		}

		err = s.storageService.TransferStorageToWorkspace(
			ctx,
			user,
			*backupConfig.StorageID,
			request.TargetWorkspaceID,
			&databaseID,
		)
		if err != nil {
			return err
		}
	case request.TargetStorageID != nil:
		targetStorage, err := s.storageService.GetStorageByID(ctx, *request.TargetStorageID)
		if err != nil {
			return err
		}

		if targetStorage.WorkspaceID != request.TargetWorkspaceID {
			return ErrTargetStorageNotInTargetWorkspace
		}

		backupConfig.StorageID = request.TargetStorageID
		backupConfig.Storage = targetStorage

		_, err = s.backupConfigRepository.Save(backupConfig)
		if err != nil {
			return err
		}
	default:
		return ErrTargetStorageNotSpecified
	}

	err = s.databaseService.TransferDatabaseToWorkspace(ctx, databaseID, request.TargetWorkspaceID)
	if err != nil {
		return err
	}

	if len(request.TargetNotifierIDs) > 0 {
		if err := s.assignTargetNotifiers(ctx, databaseID, request.TargetNotifierIDs); err != nil {
			return err
		}
	}

	return nil
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

	repairedConfig, err := s.backupConfigRepository.Save(backupConfig)
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

	_, err = s.backupConfigRepository.Save(defaultConfig)

	return err
}

func (s *BackupConfigService) transferNotifiers(
	ctx context.Context,
	user *users_models.User,
	database *databases.Database,
	targetWorkspaceID uuid.UUID,
) {
	for _, notifier := range database.Notifiers {
		_ = s.notifierService.TransferNotifierToWorkspace(
			ctx,
			user,
			notifier.ID,
			targetWorkspaceID,
			&database.ID,
		)
	}
}

func (s *BackupConfigService) validateTargetNotifiers(ctx context.Context, request *TransferDatabaseRequest) error {
	for _, notifierID := range request.TargetNotifierIDs {
		notifier, err := s.notifierService.GetNotifierByID(ctx, notifierID)
		if err != nil {
			return err
		}

		if notifier.WorkspaceID != request.TargetWorkspaceID {
			return ErrTargetNotifierNotInTargetWorkspace
		}
	}

	return nil
}

func (s *BackupConfigService) assignTargetNotifiers(
	ctx context.Context,
	databaseID uuid.UUID,
	notifierIDs []uuid.UUID,
) error {
	targetNotifiers := make([]notifiers.Notifier, 0, len(notifierIDs))

	for _, notifierID := range notifierIDs {
		notifier, err := s.notifierService.GetNotifierByID(ctx, notifierID)
		if err != nil {
			return err
		}

		targetNotifiers = append(targetNotifiers, *notifier)
	}

	return s.databaseService.UpdateDatabaseNotifiers(databaseID, targetNotifiers)
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
