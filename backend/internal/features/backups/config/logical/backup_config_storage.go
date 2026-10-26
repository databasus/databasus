package backups_config_logical

import (
	"github.com/google/uuid"

	"databasus-backend/internal/modules/storages"
)

// BackupConfigStorage names the storage of a backup config in the answers of /backup-configs. The
// configuration keeps only StorageID; the service fills this from the storages module.
type BackupConfigStorage struct {
	ID   uuid.UUID     `json:"id"`
	Name string        `json:"name"`
	Type storages.Type `json:"type"`
}
