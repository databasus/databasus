package files

import (
	"time"

	"github.com/google/uuid"
)

type DeletionBatchStatus string

const (
	CleanupPending        DeletionBatchStatus = "CLEANUP_PENDING"
	CleanupCompleted      DeletionBatchStatus = "COMPLETED"
	CleanupWithRemainders DeletionBatchStatus = "COMPLETED_WITH_REMAINDERS"
)

type DeletionBatch struct {
	ID              uuid.UUID
	Status          DeletionBatchStatus
	CreatedAt       time.Time
	CleanupDeadline time.Time
	CompletedAt     *time.Time
	Total           int64
	Pending         int64
	Deleted         int64
	Remaining       int64
}

func (DeletionBatch) TableName() string { return "storages_deletion_batches" }

type DeletionBatchRequest struct {
	ID         uuid.UUID
	References []StoredFileReference
}
