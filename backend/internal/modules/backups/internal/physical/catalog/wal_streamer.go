package catalog

import (
	"time"

	"github.com/google/uuid"
)

// Sidecar row for the long-running WAL streamer. The PK is
// database_id (one streamer per database). Heartbeat > 90 s means the row is
// stale and a later manager pass reclaims it; heartbeat alone answers
// liveness, and runtime PG-slot semantics (active_pid != ours ⇒ refuse)
// prevent concurrent-claim races.
type WalStreamer struct {
	DatabaseID      uuid.UUID         `gorm:"column:database_id;type:uuid;primaryKey"`
	StartedAt       time.Time         `gorm:"column:started_at"`
	LastHeartbeatAt time.Time         `gorm:"column:last_heartbeat_at"`
	Status          WalStreamerStatus `gorm:"column:status;type:text;not null"`

	// The first failure since a session last recorded a sample. A crash loop keeps
	// it, so a streamer that never gets as far as its first sample still shows how
	// long it has been failing.
	FailingSince *time.Time `gorm:"column:failing_since"`
}

func (WalStreamer) TableName() string {
	return "backups_physical_wal_streamers"
}
