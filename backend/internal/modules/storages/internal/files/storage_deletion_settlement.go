package files

import (
	"time"

	"github.com/google/uuid"
)

type StorageDeletionSettlement struct {
	StorageID      uuid.UUID
	ConfirmedFiles []string
	Now            time.Time
}
