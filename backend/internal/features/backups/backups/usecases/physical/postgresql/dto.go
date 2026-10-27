package usecases_physical_postgresql

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/storages"
)

type timelineIdentityProbe func(context.Context, int) (*TimelineDecision, error)

type CommonBackupSpec struct {
	SourceDB      databases.PostgresqlPhysicalConnectionSettings
	DatabaseName  string
	StorageID     uuid.UUID
	FileStore     FileStore
	Encryption    backups_core_enums.BackupEncryption
	MasterKey     *crypto.MasterKey
	FullRepo      *physical_repositories.PhysicalFullBackupRepository
	HistoryRepo   *physical_repositories.PhysicalWalHistoryRepository
	Logger        *slog.Logger
	timelineProbe timelineIdentityProbe

	// ProgressListener, when set, receives the streamed size and elapsed time
	// periodically during the backup so the row's size grows in near-real-time.
	// nil disables progress reporting.
	ProgressListener func(completedMb float64, elapsedMs int64)
}

type FullBackupSpec struct {
	CommonBackupSpec
	Backup *physical_models.PhysicalFullBackup
}

type IncrementalBackupSpec struct {
	CommonBackupSpec

	Backup         *physical_models.PhysicalIncrementalBackup
	ParentManifest ParentManifestRef
	// StorageReader reads the parent manifest, which sits in the same storage the increment goes to.
	StorageReader storages.StorageReader
	IncrRepo      *physical_repositories.PhysicalIncrementalBackupRepository

	// IncrementalCadence is the configured INCR interval, used only to size the
	// bounded summarizer wait window (min(cadence/4, cap)). Zero is acceptable —
	// the executor then waits only up to its own cap.
	IncrementalCadence time.Duration
}

type PhysicalBackupResult struct {
	Status       physical_enums.PhysicalBackupStatus
	ErrorReason  *physical_enums.PhysicalBackupErrorReason
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

	EncryptionAlgo backups_core_enums.BackupEncryption
	EncryptionSalt string
	EncryptionIV   string

	Compression physical_enums.PhysicalBackupCompression

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
	Encryption backups_core_enums.BackupEncryption
	Salt       string
	IV         string
	StopLSN    walmath.LSN
}

// Values of ChainRiskReport.Reason. Exported because the wiring package routes
// each one to its own notification kind, and the walBreakReason type it comes
// from is unexported.
const (
	ChainRiskReasonSlotRetention  = "SLOT_WAL_RETENTION"
	ChainRiskReasonArchiveStale   = "WAL_ARCHIVE_STALE"
	ChainRiskReasonRotationDenied = "WAL_ROTATION_DENIED"
)

// Payload of WalStreamSpec.OnChainAtRisk. SlotWalStatus and LagBytes describe the
// slot; LastArchivedWalAt is set only for ChainRiskReasonArchiveStale, where the
// question is how far the recovery point has fallen behind.
type ChainRiskReport struct {
	Reason            string
	SlotWalStatus     string
	LagBytes          int64
	LastArchivedWalAt *time.Time
}
