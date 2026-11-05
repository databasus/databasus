// Package logical starts, runs, lists, cancels, keeps and deletes the logical backups of databases,
// and serves their files to the restores, the downloads and the verification runs.
package logical

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	files_utils "databasus-backend/internal/lib/filename"
	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/backups/internal/pipeline"
	"databasus-backend/internal/modules/storages"
)

type Status string

const (
	StatusInProgress Status = "IN_PROGRESS"
	StatusCompleted  Status = "COMPLETED"
	StatusFailed     Status = "FAILED"
	StatusCanceled   Status = "CANCELED"
)

// VerificationStatus is the outcome of the backup's last restore verification.
type VerificationStatus string

const (
	VerificationStatusNotVerified        VerificationStatus = "NOT_VERIFIED"
	VerificationStatusVerifiedSuccessful VerificationStatus = "VERIFIED_SUCCESSFUL"
	VerificationStatusVerificationFailed VerificationStatus = "VERIFICATION_FAILED"
)

// metadataSuffix names the sidecar file that holds the encryption metadata next to the backup file.
const metadataSuffix = ".metadata"

type LogicalBackup struct {
	ID         uuid.UUID
	DatabaseID uuid.UUID
	StorageID  uuid.UUID
	// FileName is unique per attempt, so a retry never addresses a file a pending deletion owns.
	FileName    string
	Status      Status
	FailMessage *string
	// The sizes are megabytes of the stored file and of the database before the dump.
	BackupSizeMb      float64
	BackupRawDbSizeMb float64
	BackupDurationMs  int64
	Encryption        config.Encryption
	EncryptionSalt    *string
	EncryptionIV      *string
	// TimescaledbVersion is the source's TimescaleDB version at backup time, empty without the
	// extension. A non-empty one makes the restore run the TimescaleDB procedure.
	TimescaledbVersion        string
	RestoreVerificationStatus VerificationStatus
	CreatedAt                 time.Time
}

// GetStoredBackup names the stored file of the backup for the pipeline. An encrypted backup without
// its header reads as one without encryption and fails to decrypt nothing, so it is an error.
func (b LogicalBackup) GetStoredBackup() (pipeline.StoredBackup, error) {
	storedBackup := pipeline.StoredBackup{
		BackupID:         b.ID,
		StorageID:        b.StorageID,
		FileName:         b.FileName,
		EncryptionHeader: nil,
	}

	if b.Encryption != config.EncryptionEncrypted {
		return storedBackup, nil
	}

	if b.EncryptionSalt == nil || b.EncryptionIV == nil {
		return pipeline.StoredBackup{}, errEncryptionMetadataMissing
	}

	storedBackup.EncryptionHeader = &pipeline.EncryptionHeader{Salt: *b.EncryptionSalt, Nonce: *b.EncryptionIV}

	return storedBackup, nil
}

// getStoredFiles names the backup file and its metadata sidecar, which every removal deletes
// together.
func (b LogicalBackup) getStoredFiles() []storages.StoredFileReference {
	return []storages.StoredFileReference{
		{StorageID: b.StorageID, FileName: b.FileName},
		{StorageID: b.StorageID, FileName: b.FileName + metadataSuffix},
	}
}

// newFileName names the stored file of one backup attempt. The ID makes it unique per attempt.
func newFileName(databaseName string, backupID uuid.UUID, startedAt time.Time) string {
	return fmt.Sprintf("%s-%s-%s",
		files_utils.SanitizeFilename(databaseName), startedAt.UTC().Format("20060102-150405"), backupID)
}
