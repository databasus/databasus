package storageswitch

import (
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/storages"
)

type StorageSwitch struct {
	ID                uuid.UUID
	DatabaseID        uuid.UUID
	WorkspaceID       uuid.UUID
	PreviousStorageID *uuid.UUID
	TargetStorageID   uuid.UUID
	DeletionBatchID   uuid.UUID
	CreatedAt         time.Time
}

func (StorageSwitch) TableName() string { return "backups_storage_switches" }

type Acceptance struct {
	DatabaseID        uuid.UUID
	WorkspaceID       uuid.UUID
	PreviousStorageID *uuid.UUID
	TargetStorageID   uuid.UUID
	References        []storages.StoredFileReference
}
