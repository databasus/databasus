package usecases_physical_postgresql

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	physical_dto "databasus-backend/internal/features/backups/backups/core/physical/dto"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/storages"
)

// The `.metadata` sidecar is a few KB of JSON, so its deadline covers endpoint
// latency rather than transfer time: a slow S3-compatible provider is the only
// realistic way this upload runs long, and losing it fails the whole backup.
// Aligned with walSegmentUploadTimeout and the manifest sidecar's floor.
const metadataUploadTimeout = 2 * time.Minute

// uploadFullMetadata writes the `<artifact>.metadata` sidecar describing a
// completed FULL, next to the artifact / manifest / history the stream already
// uploaded. Living in the executor keeps the backuper out of the upload path and
// makes the sidecar atomic with the COMPLETED result.
func uploadFullMetadata(
	logger *slog.Logger,
	fileStore FileStore,
	storageID uuid.UUID,
	sourceDB databases.PostgresqlPhysicalConnectionSettings,
	fullBackup *physical_models.PhysicalFullBackup,
	result PhysicalBackupResult,
) (storages.WriteReceipt, error) {
	if result.FileName == "" {
		return storages.WriteReceipt{}, errors.New("cannot upload metadata: file_name is empty")
	}

	metadata := physical_dto.PhysicalBackupMetadata{
		BackupID:              fullBackup.ID,
		DatabaseID:            fullBackup.DatabaseID,
		BackupType:            physical_enums.PhysicalBackupTypeFull,
		SystemIdentifier:      sourceDB.GetSystemIdentifierUint64(),
		PgVersion:             pgVersionFromTag(sourceDB.Version),
		TimelineID:            result.TimelineID,
		StartLSN:              result.StartLSN.String(),
		StopLSN:               result.StopLSN.String(),
		UncompressedSizeBytes: 0,
		CompressedSizeBytes:   int64(result.BackupSizeMb * 1024 * 1024),
		Encryption:            result.EncryptionAlgo,
		EncryptionSalt:        result.EncryptionSalt,
		EncryptionIV:          result.EncryptionIV,
		Compression:           result.Compression,
		CreatedAt:             fullBackup.CreatedAt,
		CompletedAt:           result.CompletedAt,
	}

	return uploadMetadata(logger, fileStore, storageID, result.FileName, metadata)
}

// uploadIncrMetadata writes the `<artifact>.metadata` sidecar for a completed
// INCR, carrying the chain references (root full + parent incremental) restore
// needs to walk the chain.
func uploadIncrMetadata(
	logger *slog.Logger,
	fileStore FileStore,
	storageID uuid.UUID,
	sourceDB databases.PostgresqlPhysicalConnectionSettings,
	incrBackup *physical_models.PhysicalIncrementalBackup,
	result PhysicalBackupResult,
) (storages.WriteReceipt, error) {
	if result.FileName == "" {
		return storages.WriteReceipt{}, errors.New("cannot upload metadata: file_name is empty")
	}

	rootID := incrBackup.RootFullBackupID

	metadata := physical_dto.PhysicalBackupMetadata{
		BackupID:                  incrBackup.ID,
		DatabaseID:                incrBackup.DatabaseID,
		BackupType:                physical_enums.PhysicalBackupTypeIncremental,
		SystemIdentifier:          sourceDB.GetSystemIdentifierUint64(),
		PgVersion:                 pgVersionFromTag(sourceDB.Version),
		TimelineID:                result.TimelineID,
		StartLSN:                  result.StartLSN.String(),
		StopLSN:                   result.StopLSN.String(),
		UncompressedSizeBytes:     0,
		CompressedSizeBytes:       int64(result.BackupSizeMb * 1024 * 1024),
		RootFullBackupID:          &rootID,
		ParentIncrementalBackupID: incrBackup.ParentIncrementalBackupID,
		Encryption:                result.EncryptionAlgo,
		EncryptionSalt:            result.EncryptionSalt,
		EncryptionIV:              result.EncryptionIV,
		Compression:               result.Compression,
		CreatedAt:                 incrBackup.CreatedAt,
		CompletedAt:               result.CompletedAt,
	}

	return uploadMetadata(logger, fileStore, storageID, result.FileName, metadata)
}

// uploadMetadata marshals the metadata and PUTs `<artifact>.metadata`. It uses a
// fresh Background context with its own timeout: the stream is already done, so a
// cancelled backup context must not abort the sidecar write.
func uploadMetadata(
	logger *slog.Logger,
	fileStore FileStore,
	storageID uuid.UUID,
	artifactFileName string,
	metadata physical_dto.PhysicalBackupMetadata,
) (storages.WriteReceipt, error) {
	body, err := json.Marshal(metadata)
	if err != nil {
		return storages.WriteReceipt{}, fmt.Errorf("marshal metadata JSON: %w", err)
	}

	metadataName := artifactFileName + metadataSuffix

	ctx, cancel := context.WithTimeout(context.Background(), metadataUploadTimeout)
	defer cancel()

	logger.DebugContext(ctx, "uploading backup metadata sidecar", "file_name", metadataName)

	receipt, err := fileStore.WriteFile(
		ctx,
		storages.StoredFileReference{StorageID: storageID, FileName: metadataName},
		bytes.NewReader(body),
	)
	if err != nil {
		return storages.WriteReceipt{}, fmt.Errorf("upload metadata: %w", err)
	}

	return receipt, nil
}

// pgVersionFromTag converts the pgshared.Version enum ("17", "18") into
// the canonical server_version_num style (170000, 180000) so the metadata
// carries a value pg_combinebackup can compare against on restore. The exact
// patch level is unknown at this layer — major.minor is enough for the check.
func pgVersionFromTag(v pgshared.Version) int {
	major, err := strconv.Atoi(strings.TrimSpace(string(v)))
	if err != nil {
		return 0
	}

	return major * 10000
}
