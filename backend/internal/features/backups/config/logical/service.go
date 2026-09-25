package backups_config_logical

import (
	"context"
	"errors"

	"github.com/google/uuid"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/intervals"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	users_models "databasus-backend/internal/features/users/models"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/util/period"
)

type BackupConfigService struct {
	backupConfigRepository *BackupConfigRepository
	databaseService        *databases.DatabaseService
	storageService         *storages.StorageService
	notifierService        *notifiers.NotifierService
	workspaceService       *workspaces_services.WorkspaceService

	dbStorageChangeListener BackupConfigStorageChangeListener
}

func (s *BackupConfigService) SetDatabaseStorageChangeListener(
	dbStorageChangeListener BackupConfigStorageChangeListener,
) {
	s.dbStorageChangeListener = dbStorageChangeListener
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
	backupConfig *LogicalBackupConfig,
) (*LogicalBackupConfig, error) {
	if err := backupConfig.Validate(); err != nil {
		return nil, err
	}

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
	backupConfig *LogicalBackupConfig,
) (*LogicalBackupConfig, error) {
	if err := backupConfig.Validate(); err != nil {
		return nil, err
	}

	// Check if there's an existing backup config for this database
	existingConfig, err := s.GetBackupConfigByDbId(backupConfig.DatabaseID)
	if err != nil {
		return nil, err
	}

	if existingConfig != nil {
		// If storage is changing, notify the listener
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

	return s.backupConfigRepository.Save(backupConfig)
}

func (s *BackupConfigService) GetBackupConfigByDbIdWithAuth(
	ctx context.Context,
	user *users_models.User,
	databaseID uuid.UUID,
) (*LogicalBackupConfig, error) {
	_, err := s.databaseService.GetDatabase(ctx, user, databaseID)
	if err != nil {
		return nil, err
	}

	return s.GetBackupConfigByDbId(databaseID)
}

func (s *BackupConfigService) GetBackupConfigByDbId(
	databaseID uuid.UUID,
) (*LogicalBackupConfig, error) {
	config, err := s.backupConfigRepository.FindByDatabaseID(databaseID)
	if err != nil {
		return nil, err
	}

	if config == nil {
		err = s.initializeDefaultConfig(databaseID)
		if err != nil {
			return nil, err
		}

		return s.backupConfigRepository.FindByDatabaseID(databaseID)
	}

	return config, nil
}

func (s *BackupConfigService) GetBackupConfigsByDatabaseIDs(
	databaseIDs []uuid.UUID,
) ([]*LogicalBackupConfig, error) {
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

func (s *BackupConfigService) GetBackupConfigsWithEnabledBackups() ([]*LogicalBackupConfig, error) {
	return s.backupConfigRepository.GetWithEnabledBackups()
}

func (s *BackupConfigService) OnDatabaseCopied(ctx context.Context, originalDatabaseID, newDatabaseID uuid.UUID) {
	originalConfig, err := s.GetBackupConfigByDbId(originalDatabaseID)
	if err != nil {
		return
	}

	newConfig := originalConfig.Copy(newDatabaseID)

	_, err = s.SaveBackupConfig(ctx, newConfig)
	if err != nil {
		return
	}
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
		err = s.assignTargetNotifiers(ctx, databaseID, request.TargetNotifierIDs)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *BackupConfigService) initializeDefaultConfig(
	databaseID uuid.UUID,
) error {
	timeOfDay := "04:00"

	_, err := s.backupConfigRepository.Save(&LogicalBackupConfig{
		DatabaseID:          databaseID,
		IsBackupsEnabled:    false,
		RetentionPolicyType: RetentionPolicyTypeTimePeriod,
		RetentionTimePeriod: period.Period3Month,
		BackupInterval: intervals.Interval{
			Type:      intervals.IntervalDaily,
			TimeOfDay: &timeOfDay,
		},
		SendNotificationsOn: []BackupNotificationType{
			NotificationBackupFailed,
			NotificationBackupSuccess,
		},
		IsRetryIfFailed:     true,
		MaxFailedTriesCount: 3,
		Encryption:          backups_core_enums.BackupEncryptionNone,
	})

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
