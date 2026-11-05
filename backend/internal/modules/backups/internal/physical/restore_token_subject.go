package physical

import (
	"time"

	"github.com/google/uuid"
)

type RestoreTokenSubject struct {
	DatabaseID uuid.UUID
	TargetTime *time.Time
	BackupID   *uuid.UUID
}
