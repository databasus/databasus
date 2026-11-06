package backuping

import (
	"databasus-backend/internal/engines/postgresql/physical/walmath"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/storages"
)

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
