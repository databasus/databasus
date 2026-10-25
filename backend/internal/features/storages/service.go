package storages

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	audit_logs "databasus-backend/internal/features/audit_logs"
	audit_logs_models "databasus-backend/internal/features/audit_logs/models"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/util/encryption"
)

type StorageService struct {
	storageRepository       *StorageRepository
	workspaceService        *workspaces_services.WorkspaceService
	auditLogService         *audit_logs.AuditLogService
	fieldEncryptor          encryption.FieldEncryptor
	storageDatabaseCounters []StorageDatabaseCounter
	storageBackupCounters   []StorageBackupCounter
}

func (s *StorageService) AddStorageDatabaseCounter(counter StorageDatabaseCounter) {
	s.storageDatabaseCounters = append(s.storageDatabaseCounters, counter)
}

func (s *StorageService) AddStorageBackupCounter(counter StorageBackupCounter) {
	s.storageBackupCounters = append(s.storageBackupCounters, counter)
}

func (s *StorageService) GetStorageBackupReferenceCount(storageID uuid.UUID) (int64, error) {
	var total int64

	for _, counter := range s.storageBackupCounters {
		count, err := counter.GetStorageBackupReferenceCount(storageID)
		if err != nil {
			return 0, err
		}

		total += count
	}

	return total, nil
}

func (s *StorageService) GetStorageAttachedDatabasesIDs(
	storageID uuid.UUID,
) ([]uuid.UUID, error) {
	seen := make(map[uuid.UUID]struct{})
	merged := make([]uuid.UUID, 0)

	for _, counter := range s.storageDatabaseCounters {
		ids, err := counter.GetStorageAttachedDatabasesIDs(storageID)
		if err != nil {
			return nil, err
		}

		for _, id := range ids {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			merged = append(merged, id)
		}
	}

	return merged, nil
}

func (s *StorageService) IsStorageInUse(ctx context.Context, storageID uuid.UUID) (bool, error) {
	ids, err := s.GetStorageAttachedDatabasesIDs(storageID)
	if err != nil {
		return false, err
	}

	return len(ids) > 0, nil
}

func (s *StorageService) CountDatabasesForStorage(ctx context.Context, storageID uuid.UUID) (int, error) {
	ids, err := s.GetStorageAttachedDatabasesIDs(storageID)
	if err != nil {
		return 0, err
	}

	return len(ids), nil
}

func (s *StorageService) OnBeforeWorkspaceDeletion(workspaceID uuid.UUID) error {
	storages, err := s.storageRepository.FindByWorkspaceID(workspaceID)
	if err != nil {
		return fmt.Errorf("failed to get storages for workspace deletion: %w", err)
	}

	// The listener contract carries no context, and the drain needs one that is not
	// already cancelled by the finished request.
	ctx := context.Background()

	for _, storage := range storages {
		s.drainPendingDeletions(ctx, storage)

		if err := s.storageRepository.Delete(storage); err != nil {
			return fmt.Errorf("failed to delete storage %s: %w", storage.ID, err)
		}
	}

	return nil
}

func (s *StorageService) SaveStorage(
	ctx context.Context,
	actor users.Actor,
	workspaceID uuid.UUID,
	storage *Storage,
) error {
	canManage, err := s.workspaceService.CanUserManageDBs(ctx, workspaceID, actor)
	if err != nil {
		return err
	}
	if !canManage {
		return ErrInsufficientPermissionsToManageStorage
	}

	if storage.Type == StorageTypeRclone && actor.Role != users.RoleAdmin {
		return ErrRcloneStorageRequiresAdmin
	}

	isUpdate := storage.ID != uuid.Nil

	if isUpdate {
		existingStorage, err := s.storageRepository.FindByID(ctx, storage.ID)
		if err != nil {
			return err
		}

		if existingStorage.WorkspaceID != workspaceID {
			return ErrStorageDoesNotBelongToWorkspace
		}

		existingStorage.Update(storage)

		oldName := existingStorage.Name

		if err := existingStorage.EncryptSensitiveData(s.fieldEncryptor); err != nil {
			return err
		}

		if err := existingStorage.Validate(s.fieldEncryptor); err != nil {
			return err
		}

		_, err = s.storageRepository.Save(existingStorage)
		if err != nil {
			return err
		}

		if oldName != existingStorage.Name {
			s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
				Message:     fmt.Sprintf("Storage renamed from '%s' to '%s'", oldName, existingStorage.Name),
				UserID:      &actor.UserID,
				WorkspaceID: &workspaceID,
			})
		} else {
			s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
				Message:     fmt.Sprintf("Storage updated: %s", existingStorage.Name),
				UserID:      &actor.UserID,
				WorkspaceID: &workspaceID,
			})
		}
	} else {
		storage.WorkspaceID = workspaceID

		if err := storage.EncryptSensitiveData(s.fieldEncryptor); err != nil {
			return err
		}

		if err := storage.Validate(s.fieldEncryptor); err != nil {
			return err
		}

		_, err = s.storageRepository.Save(storage)
		if err != nil {
			return err
		}

		s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
			Message:     fmt.Sprintf("Storage created: %s", storage.Name),
			UserID:      &actor.UserID,
			WorkspaceID: &workspaceID,
		})
	}

	return nil
}

func (s *StorageService) DeleteStorage(
	ctx context.Context,
	actor users.Actor,
	storageID uuid.UUID,
) error {
	storage, err := s.storageRepository.FindByID(ctx, storageID)
	if err != nil {
		return err
	}

	canManage, err := s.workspaceService.CanUserManageDBs(ctx, storage.WorkspaceID, actor)
	if err != nil {
		return err
	}
	if !canManage {
		return ErrInsufficientPermissionsToManageStorage
	}

	attachedDatabasesIDs, err := s.GetStorageAttachedDatabasesIDs(storage.ID)
	if err != nil {
		return err
	}
	if len(attachedDatabasesIDs) > 0 {
		return ErrStorageHasAttachedDatabases
	}

	backupReferences, err := s.GetStorageBackupReferenceCount(storage.ID)
	if err != nil {
		return err
	}

	if backupReferences > 0 {
		return ErrStorageHasBackups
	}

	s.drainPendingDeletions(ctx, storage)

	err = s.storageRepository.Delete(storage)
	if err != nil {
		return err
	}

	s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     fmt.Sprintf("Storage deleted: %s", storage.Name),
		UserID:      &actor.UserID,
		WorkspaceID: &storage.WorkspaceID,
	})

	return nil
}

func (s *StorageService) GetStorage(
	ctx context.Context,
	actor users.Actor,
	id uuid.UUID,
) (*Storage, error) {
	storage, err := s.storageRepository.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	canView, _, err := s.workspaceService.CanUserAccessWorkspace(ctx, storage.WorkspaceID, actor)
	if err != nil {
		return nil, err
	}
	if !canView {
		return nil, ErrInsufficientPermissionsToViewStorage
	}

	storage.HideSensitiveData()

	return storage, nil
}

func (s *StorageService) GetStorages(
	ctx context.Context,
	actor users.Actor,
	workspaceID uuid.UUID,
) ([]*Storage, error) {
	canView, _, err := s.workspaceService.CanUserAccessWorkspace(ctx, workspaceID, actor)
	if err != nil {
		return nil, err
	}
	if !canView {
		return nil, ErrInsufficientPermissionsToViewStorages
	}

	storages, err := s.storageRepository.FindByWorkspaceID(workspaceID)
	if err != nil {
		return nil, err
	}

	for _, storage := range storages {
		storage.HideSensitiveData()
	}

	return storages, nil
}

func (s *StorageService) TestStorageConnection(
	ctx context.Context,
	actor users.Actor,
	storageID uuid.UUID,
) error {
	storage, err := s.storageRepository.FindByID(ctx, storageID)
	if err != nil {
		return err
	}

	canView, _, err := s.workspaceService.CanUserAccessWorkspace(ctx, storage.WorkspaceID, actor)
	if err != nil {
		return err
	}
	if !canView {
		return ErrInsufficientPermissionsToTestStorage
	}

	err = storage.TestConnection(s.fieldEncryptor)
	if err != nil {
		lastSaveError := err.Error()
		storage.LastSaveError = &lastSaveError
		return err
	}

	storage.LastSaveError = nil
	_, err = s.storageRepository.Save(storage)
	if err != nil {
		return err
	}

	return nil
}

func (s *StorageService) TestStorageConnectionDirect(
	ctx context.Context,
	actor users.Actor,
	storage *Storage,
) error {
	if storage.Type == StorageTypeRclone && actor.Role != users.RoleAdmin {
		return ErrRcloneStorageRequiresAdmin
	}

	var usingStorage *Storage

	if storage.ID != uuid.Nil {
		existingStorage, err := s.storageRepository.FindByID(ctx, storage.ID)
		if err != nil {
			return err
		}

		if existingStorage.WorkspaceID != storage.WorkspaceID {
			return ErrStorageDoesNotBelongToWorkspace
		}

		existingStorage.Update(storage)

		if err := existingStorage.Validate(s.fieldEncryptor); err != nil {
			return err
		}

		usingStorage = existingStorage
	} else {
		usingStorage = storage
	}

	return usingStorage.TestConnection(s.fieldEncryptor)
}

func (s *StorageService) GetStorageByID(
	ctx context.Context,
	id uuid.UUID,
) (*Storage, error) {
	return s.storageRepository.FindByID(ctx, id)
}

func (s *StorageService) GetAllStorages() ([]*Storage, error) {
	return s.storageRepository.GetAllStorages()
}

func (s *StorageService) TransferStorageToWorkspace(
	ctx context.Context,
	actor users.Actor,
	storageID uuid.UUID,
	targetWorkspaceID uuid.UUID,
	transferingWithDbID *uuid.UUID,
) error {
	existingStorage, err := s.storageRepository.FindByID(ctx, storageID)
	if err != nil {
		return err
	}

	canManageSource, err := s.workspaceService.CanUserManageDBs(ctx, existingStorage.WorkspaceID, actor)
	if err != nil {
		return err
	}
	if !canManageSource {
		return ErrInsufficientPermissionsInSourceWorkspace
	}

	canManageTarget, err := s.workspaceService.CanUserManageDBs(ctx, targetWorkspaceID, actor)
	if err != nil {
		return err
	}
	if !canManageTarget {
		return ErrInsufficientPermissionsInTargetWorkspace
	}

	attachedDatabasesIDs, err := s.GetStorageAttachedDatabasesIDs(existingStorage.ID)
	if err != nil {
		return err
	}

	if transferingWithDbID != nil {
		for _, dbID := range attachedDatabasesIDs {
			if dbID != *transferingWithDbID {
				return ErrStorageHasOtherAttachedDatabasesCannotTransfer
			}
		}
	} else if len(attachedDatabasesIDs) > 0 {
		return ErrStorageHasAttachedDatabasesCannotTransfer
	}

	sourceWorkspaceID := existingStorage.WorkspaceID
	existingStorage.WorkspaceID = targetWorkspaceID

	_, err = s.storageRepository.Save(existingStorage)
	if err != nil {
		return err
	}

	sourceWorkspace, err := s.workspaceService.GetWorkspaceByID(sourceWorkspaceID)
	if err != nil {
		return fmt.Errorf("failed to get source workspace: %w", err)
	}

	targetWorkspace, err := s.workspaceService.GetWorkspaceByID(targetWorkspaceID)
	if err != nil {
		return fmt.Errorf("failed to get target workspace: %w", err)
	}

	s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
		Message: fmt.Sprintf(
			"Storage transferred out: %s to workspace '%s'",
			existingStorage.Name,
			targetWorkspace.Name,
		),
		UserID:      &actor.UserID,
		WorkspaceID: &sourceWorkspaceID,
	})

	s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
		Message: fmt.Sprintf(
			"Storage transferred in: %s from workspace '%s'",
			existingStorage.Name,
			sourceWorkspace.Name,
		),
		UserID:      &actor.UserID,
		WorkspaceID: &targetWorkspaceID,
	})

	return nil
}
