package usecases_physical_postgresql

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// BackupStatusRecorder is the part of the databases module that records a published WAL segment as
// the database's last backup, in the transaction that publishes it.
type BackupStatusRecorder interface {
	RecordBackupSuccess(ctx context.Context, databaseID uuid.UUID, backupTime time.Time) error
}
