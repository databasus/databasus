package config

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/storages"
)

type storageSwitchScope struct {
	WorkspaceID uuid.UUID
	IsTransfer  bool
}

type storageSwitchSave struct {
	Scope             storageSwitchScope
	PreviousStorageID *uuid.UUID
	Config            AnyBackupConfig
}

func (s *Service) requestStorageChangeCleanup(
	ctx context.Context,
	save storageSwitchSave,
) (bool, error) {
	scope, previousStorageID, savedConfig := save.Scope, save.PreviousStorageID, save.Config
	common := savedConfig.GetBackupConfig()
	if common.StorageID == nil || (!scope.IsTransfer && isSameStorage(previousStorageID, common.StorageID)) {
		return false, nil
	}

	lastSelectedStorageID, hasLastSelectedStorage, err := s.repository.getLastSelectedStorageID(ctx, common.DatabaseID)
	if err != nil {
		return false, err
	}

	var unfinishedStorageID *uuid.UUID
	if hasLastSelectedStorage {
		unfinishedStorageID = &lastSelectedStorageID
	}

	removeHistory := s.logicalBackups
	if savedConfig.GetMethod() == MethodPhysical {
		removeHistory = s.physicalBackups
	}

	history, err := removeHistory(
		ctx,
		HistoryRemoval{
			UnfinishedStorageID: unfinishedStorageID, DatabaseID: common.DatabaseID,
			TargetStorageID: *common.StorageID,
			ShouldResetAll:  previousStorageID == nil,
		},
	)
	if err != nil {
		return false, err
	}

	if previousStorageID == nil && !scope.IsTransfer && !history.HasHistory && len(history.References) == 0 {
		return false, nil
	}

	err = s.storages.RequestScopedFileDeletions(ctx, storages.FileDeletionRequest{
		References: history.References,
		Scope:      &storages.CleanupScope{DatabaseID: common.DatabaseID, WorkspaceID: scope.WorkspaceID},
	})

	return true, err
}
