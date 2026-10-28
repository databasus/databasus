package walstreaming

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type BackupStatusRecorder interface {
	RecordBackupSuccess(ctx context.Context, databaseID uuid.UUID, backupTime time.Time) error
}
