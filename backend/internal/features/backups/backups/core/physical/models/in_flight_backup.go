package physical_models

import (
	"time"

	"github.com/google/uuid"

	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
)

// Cross-table single-in-flight invariant: at most one in-flight backup per
// database, across both FULL and INCR. The database_id PK is the cross-table
// claim; the typed row INSERT and this table's claim happen in the same
// transaction.
type PhysicalInFlightBackup struct {
	DatabaseID uuid.UUID                         `json:"databaseId" gorm:"column:database_id;type:uuid;primaryKey"`
	BackupType physical_enums.PhysicalBackupType `json:"backupType" gorm:"column:backup_type;type:text;not null"`
	BackupID   uuid.UUID                         `json:"backupId"   gorm:"column:backup_id;type:uuid;not null"`
	ClaimedAt  time.Time                         `json:"claimedAt"  gorm:"column:claimed_at"`
}

func (PhysicalInFlightBackup) TableName() string {
	return "backups_physical_in_flight"
}
