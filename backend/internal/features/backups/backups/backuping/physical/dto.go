package backuping_physical

import (
	"time"

	"github.com/google/uuid"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/features/backups/backups/core/physical/chain_view"
	backups_config_physical "databasus-backend/internal/features/backups/config/physical"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/storages"
)

type backupContext struct {
	Config *backups_config_physical.PhysicalBackupConfig

	Database databases.Database
	// Source reaches the cluster through the bastion, so everything the run touches — the size probe,
	// the executor's SourceDB, pg_basebackup — dials the forwarded port off one SSH session.
	Source databases.PostgresqlPhysicalConnectionSettings

	Storage   storages.Storage
	MasterKey string
	tunnel    *pgphysical.Tunnel
}

func (c *backupContext) Close() {
	c.tunnel.Close()
}

// chainCandidate pairs a non-extendable chain with its end timestamp so passes
// can order by recency without recomputing it.
type chainCandidate struct {
	view  *chain_view.ChainView
	endTs time.Time
}

// A WAL chain break is notified once per incident per database, so the throttle
// key carries the kind: a retention warning must not swallow the slot-rebuild
// alert that says a fresh full backup was requested.
type chainAlertKind string

const (
	chainAlertStreamerFailed chainAlertKind = "STREAMER_FAILED"
	chainAlertSlotRebuilt    chainAlertKind = "SLOT_REBUILT"
	chainAlertChainAtRisk    chainAlertKind = "CHAIN_AT_RISK"
	chainAlertArchiveStale   chainAlertKind = "WAL_ARCHIVE_STALE"
	chainAlertRotationDenied chainAlertKind = "WAL_ROTATION_DENIED"
)

type chainAlertKey struct {
	DatabaseID uuid.UUID
	Kind       chainAlertKind
}

type chainAlert struct {
	Kind    chainAlertKind
	Heading string
	Message string
}

type CancelInFlightBackupSpec struct {
	DatabaseID uuid.UUID
	BackupID   uuid.UUID
}
