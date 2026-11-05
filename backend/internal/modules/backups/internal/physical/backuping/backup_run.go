package backuping

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/backups/internal/physical/timelines"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/storages"
)

type timelineIdentityProbe func(context.Context, int) (*timelines.TimelineDecision, error)

type CommonBackupSpec struct {
	Publication    config.PublicationFence
	ExecutionClock clock.Source
	Clock          clock.Source
	Transactor     *database.Transactor
	ClientTools    pgshared.ClientTools
	SourceDB       databases.PostgresqlPhysicalConnectionSettings
	DatabaseName   string
	StorageID      uuid.UUID
	FileStore      storages.Module
	Encryption     catalog.BackupEncryption
	MasterKey      *crypto.MasterKey
	BackupRepo     *catalog.BackupRepository
	HistoryRepo    *catalog.WalHistoryRepository
	Logger         *slog.Logger
	timelineProbe  timelineIdentityProbe

	// ProgressListener, when set, receives the streamed size and elapsed time
	// periodically during the backup so the row's size grows in near-real-time.
	// nil disables progress reporting.
	ProgressListener func(completedMb float64, elapsedMs int64)
}

type FullBackupSpec struct {
	CommonBackupSpec
	Backup *catalog.Backup
}

type IncrementalBackupSpec struct {
	CommonBackupSpec

	Backup         *catalog.Backup
	ParentManifest ParentManifestRef
	// StorageReader reads the parent manifest, which sits in the same storage the increment goes to.
	StorageReader storages.StorageReader

	// IncrementalCadence is the configured INCR interval, used only to size the
	// bounded summarizer wait window (min(cadence/4, cap)). Zero is acceptable —
	// the executor then waits only up to its own cap.
	IncrementalCadence time.Duration
}

type Result struct {
	Status       catalog.BackupStatus
	ErrorReason  *catalog.BackupErrorReason
	ErrorMessage string

	// Receipts are spent by the transaction that publishes the backup. A result
	// that carries none leaves every file the attempt wrote for cleanup.
	Receipts []storages.WriteReceipt

	FileName string

	TimelineID int
	StartLSN   walmath.LSN
	StopLSN    walmath.LSN

	BackupSizeMb     float64
	BackupDurationMs int64

	EncryptionAlgo catalog.BackupEncryption
	EncryptionSalt string
	EncryptionIV   string

	Compression catalog.BackupCompression

	ManifestFileName       string
	ManifestEncryptionSalt string
	ManifestEncryptionIV   string

	CompletedAt time.Time
}

// ParentManifestRef is everything an incremental needs about its parent: the
// reconstructed manifest sidecar to fetch+decrypt, plus the parent's stop_lsn
// (the lower bound the WAL summarizer must cover before this INCR can run).
type ParentManifestRef struct {
	BackupID   uuid.UUID
	FileName   string
	Encryption catalog.BackupEncryption
	Salt       string
	IV         string
	StopLSN    walmath.LSN
}
