package backuping

import (
	"databasus-backend/internal/engines/postgresql/physical/walmath"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/storages"
)

// streamOutcome is the codec-agnostic result of one runStream attempt. Exactly
// one of three shapes is valid, discriminated by the fields below:
//
//  1. isCompressionUnsupported == true — the source build rejected this codec
//     before streaming any data. Retryable: the Execute-level loop advances to
//     the next codec. Only Stderr is meaningful.
//  2. Status == Completed — success; the LSN / size / encryption / compression /
//     manifest fields are populated and ErrorReason is nil.
//  3. otherwise — terminal failure already classified into
//     (Status, ErrorReason, ErrorMessage); the thin FULL/INCR adapters copy
//     these straight onto their typed result.
type streamOutcome struct {
	// isCompressionUnsupported is a retry signal, not a terminal status and not a
	// COMPLETED outcome — it is consumed only by the codec-fallback loop and never
	// persisted, so it stays a plain flag rather than a PhysicalBackupErrorReason.
	isCompressionUnsupported bool

	Status       catalog.BackupStatus
	ErrorReason  *catalog.BackupErrorReason
	ErrorMessage string

	TimelineID int
	StartLSN   walmath.LSN
	StopLSN    walmath.LSN

	BackupSizeMb float64

	EncryptionAlgo catalog.BackupEncryption
	EncryptionSalt string
	EncryptionIV   string

	Compression catalog.BackupCompression

	// Receipts are spent by the transaction that publishes the backup. An outcome
	// that carries none leaves every file it wrote for cleanup.
	Receipts []storages.WriteReceipt

	// Reconstructed-manifest sidecar. Salt/IV are the manifest's OWN fresh values
	// (never the tar's), so they live on dedicated row columns.
	ManifestFileName       string
	ManifestEncryptionSalt string
	ManifestEncryptionIV   string

	Stderr []byte
}
