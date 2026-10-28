package apptest

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

type PhysicalWalStreamerSeed struct {
	DatabaseID      uuid.UUID `json:"databaseId"      gorm:"column:database_id;type:uuid;primaryKey"`
	StartedAt       time.Time `json:"startedAt"       gorm:"column:started_at"`
	LastHeartbeatAt time.Time `json:"lastHeartbeatAt" gorm:"column:last_heartbeat_at"`
	Status          string    `json:"status"          gorm:"column:status;type:text;not null"`

	FailingSince *time.Time `json:"failingSince" gorm:"column:failing_since"`
}

func (a *App) InsertPhysicalWalStreamer(tb testing.TB, seed PhysicalWalStreamerSeed) PhysicalWalStreamerSeed {
	tb.Helper()
	if seed.Status == "" {
		seed.Status = "RUNNING"
	}
	if seed.StartedAt.IsZero() {
		seed.StartedAt = time.Now().UTC()
	}
	if seed.LastHeartbeatAt.IsZero() {
		seed.LastHeartbeatAt = seed.StartedAt
	}
	a.insertPhysicalCatalogRow(tb, "backups_physical_wal_streamers", &seed)
	return seed
}
