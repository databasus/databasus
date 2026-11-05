package config

import (
	"github.com/google/uuid"

	"databasus-backend/internal/modules/storages"
)

type RemovedBackupHistory struct {
	HasHistory bool
	References []storages.StoredFileReference
}

type HistoryRemoval struct {
	UnfinishedStorageID *uuid.UUID
	DatabaseID          uuid.UUID
	TargetStorageID     uuid.UUID
	ShouldResetAll      bool
}
