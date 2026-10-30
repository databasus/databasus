package config

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/backups/internal/storageswitch"
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

func (s *Service) acceptStorageSwitch(
	ctx context.Context,
	save storageSwitchSave,
) (*storageswitch.Outcome, error) {
	scope, previousStorageID, savedConfig := save.Scope, save.PreviousStorageID, save.Config
	common := savedConfig.GetBackupConfig()
	if common.StorageID == nil || (!scope.IsTransfer && isSameStorage(previousStorageID, common.StorageID)) {
		return nil, nil
	}
	lastSelectedStorageID, hasLastSelectedStorage, err := s.repository.getLastSelectedStorageID(ctx, common.DatabaseID)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	if previousStorageID == nil && !scope.IsTransfer && !history.HasHistory && len(history.References) == 0 {
		return nil, nil
	}
	return s.switches.Accept(
		ctx,
		storageswitch.Acceptance{
			DatabaseID:        common.DatabaseID,
			WorkspaceID:       scope.WorkspaceID,
			PreviousStorageID: previousStorageID,
			TargetStorageID:   *common.StorageID,
			References:        history.References,
		},
	)
}
