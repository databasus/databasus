package backups_config_physical

import (
	"context"

	"github.com/google/uuid"
)

// BackupConfigStorageChangeListener is called after every save of a configuration that names a
// storage, not only when the storage changed, so a removal that failed on an earlier save is
// retried by the next one.
type BackupConfigStorageChangeListener interface {
	OnBackupsStorageChanged(ctx context.Context, databaseID, storageID uuid.UUID) error
}

type BackupCancellationListener interface {
	OnBackupsDisabled(ctx context.Context, databaseID uuid.UUID)
	OnWalStreamingDisabled(ctx context.Context, databaseID uuid.UUID)
	OnBackupsStorageChanged(ctx context.Context, databaseID uuid.UUID)
}
