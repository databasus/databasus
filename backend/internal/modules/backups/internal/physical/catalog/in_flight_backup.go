package catalog

import (
	"time"

	"github.com/google/uuid"
)

// Cross-table single-in-flight invariant: at most one in-flight backup per
// database, across both FULL and INCR. The database_id PK is the cross-table
// claim; the typed row INSERT and this table's claim happen in the same
// transaction.
type InFlightBackup struct {
	DatabaseID uuid.UUID  `gorm:"column:database_id;type:uuid;primaryKey"`
	BackupType BackupType `gorm:"column:backup_type;type:text;not null"`
	BackupID   uuid.UUID  `gorm:"column:backup_id;type:uuid;not null"`
	ClaimedAt  time.Time  `gorm:"column:claimed_at"`
}

func (InFlightBackup) TableName() string {
	return "backups_physical_in_flight"
}
