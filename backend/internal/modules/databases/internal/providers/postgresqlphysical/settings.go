// Package postgresqlphysical backs up a whole PostgreSQL cluster with pg_basebackup and streams its
// WAL with pg_receivewal.
package postgresqlphysical

import (
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/modules/databases/internal/providers"
)

type Settings struct {
	// Version is what the server reported at the last save.
	Version  pgshared.Version
	Host     string
	Port     int
	Username string
	// Password is a secret.
	Password   string
	TLS        providers.PostgresqlTLSSettings
	SSHTunnel  providers.SSHTunnelSettings
	BackupType providers.PhysicalBackupType
	// ReplicationSlotName is set by the provider when the database is created and never changes.
	ReplicationSlotName string
	// BackupSlotName names the replication slot each pg_basebackup holds while it runs. The provider
	// sets it when the database is created and never changes it, so a slot an interrupted backup left
	// behind is found by the next one.
	BackupSlotName string
	// SystemIdentifier is detected at the first save and kept, so a change of the cluster behind the
	// host is refused.
	SystemIdentifier *string
	// WalSegmentSizeBytes is detected at the first save and kept, like SystemIdentifier.
	WalSegmentSizeBytes *int64
}

func (Settings) GetType() providers.Type {
	return providers.TypePostgresqlPhysical
}
