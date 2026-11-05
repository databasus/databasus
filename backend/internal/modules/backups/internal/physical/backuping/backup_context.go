package backuping

import (
	"github.com/google/uuid"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/storages"
)

type backupContext struct {
	Config config.PhysicalBackupConfig

	Database databases.Database

	Source databases.PostgresqlPhysicalConnectionSettings

	Storage   storages.Storage
	MasterKey *crypto.MasterKey
	tunnel    *pgphysical.Tunnel
}

func (c *backupContext) Close() {
	c.tunnel.Close()
}

type CancelRunningBackupSpec struct {
	DatabaseID uuid.UUID
	BackupID   uuid.UUID
}
