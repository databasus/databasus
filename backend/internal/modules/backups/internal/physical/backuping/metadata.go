package backuping

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/storages"
)

// The `.metadata` sidecar is a few KB of JSON, so its deadline covers endpoint
// latency rather than transfer time: a slow S3-compatible provider is the only
// realistic way this upload runs long, and losing it fails the whole backup.
// Aligned with walSegmentUploadTimeout and the manifest sidecar's floor.
const metadataUploadTimeout = 2 * time.Minute

func uploadMetadata(ctx context.Context, spec backupExecutionSpec, result Result) (storages.WriteReceipt, error) {
	if result.FileName == "" {
		return storages.WriteReceipt{}, errBackupFileNameMissing
	}

	metadata := catalog.BackupMetadata{
		BackupID:              spec.Backup.ID,
		DatabaseID:            spec.Backup.DatabaseID,
		BackupType:            catalog.BackupTypeFull,
		SystemIdentifier:      spec.SourceDB.GetSystemIdentifierUint64(),
		PgVersion:             pgVersionFromTag(spec.SourceDB.Version),
		TimelineID:            result.TimelineID,
		StartLSN:              result.StartLSN.String(),
		StopLSN:               result.StopLSN.String(),
		UncompressedSizeBytes: 0,
		CompressedSizeBytes:   int64(result.BackupSizeMb * 1024 * 1024),
		Encryption:            result.EncryptionAlgo,
		EncryptionSalt:        result.EncryptionSalt,
		EncryptionIV:          result.EncryptionIV,
		Compression:           result.Compression,
		CreatedAt:             spec.Backup.CreatedAt,
		CompletedAt:           result.CompletedAt,
	}

	if spec.Backup.Kind == catalog.BackupKindIncremental {
		metadata.BackupType = catalog.BackupTypeIncremental
		metadata.RootFullBackupID = spec.Backup.RootFullBackupID
		metadata.ParentIncrementalBackupID = spec.Backup.ParentBackupID

		if metadata.ParentIncrementalBackupID != nil && metadata.RootFullBackupID != nil &&
			*metadata.ParentIncrementalBackupID == *metadata.RootFullBackupID {
			metadata.ParentIncrementalBackupID = nil
		}
	}

	body, err := json.Marshal(metadata)
	if err != nil {
		return storages.WriteReceipt{}, fmt.Errorf("marshal metadata JSON: %w", err)
	}

	metadataName := result.FileName + catalog.MetadataSuffix

	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), metadataUploadTimeout)
	defer cancel()

	spec.Logger.DebugContext(ctx, "uploading backup metadata sidecar", slog.Any("file_name", metadataName))

	receipt, err := spec.FileStore.WriteFile(
		ctx,
		storages.StoredFileReference{StorageID: spec.StorageID, FileName: metadataName},
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
