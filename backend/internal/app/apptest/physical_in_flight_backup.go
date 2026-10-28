package apptest

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

type PhysicalInFlightBackupSeed struct {
	DatabaseID uuid.UUID `json:"databaseId" gorm:"column:database_id;type:uuid;primaryKey"`
	BackupType string    `json:"backupType" gorm:"column:backup_type;type:text;not null"`
	BackupID   uuid.UUID `json:"backupId"   gorm:"column:backup_id;type:uuid;not null"`
	ClaimedAt  time.Time `json:"claimedAt"  gorm:"column:claimed_at"`
}

func (a *App) InsertPhysicalInFlightBackup(tb testing.TB, seed PhysicalInFlightBackupSeed) PhysicalInFlightBackupSeed {
	tb.Helper()
	if seed.ClaimedAt.IsZero() {
		seed.ClaimedAt = time.Now().UTC()
	}
	a.insertPhysicalCatalogRow(tb, "backups_physical_in_flight", &seed)
	return seed
}
