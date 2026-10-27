package usecases_physical_postgresql

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/google/uuid"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/modules/storages"
)

type runStreamParams struct {
	FileStore  FileStore
	StorageID  uuid.UUID
	Logger     *slog.Logger
	Source     *pgphysical.Source
	FileName   string
	Encryption backups_core_enums.BackupEncryption
	MasterKey  *crypto.MasterKey
	BackupID   uuid.UUID
	Codec      physical_enums.PhysicalBackupCompression
}

type streamErrorClassifier func(streamErr error, stderr []byte) streamOutcome

type streamSpec struct {
	Params   runStreamParams
	Backup   pgphysical.BaseBackupSpec
	Classify streamErrorClassifier
}

func runStream(ctx context.Context, spec streamSpec) (streamOutcome, error) {
	p := spec.Params
	streamCtx, cancelStream := context.WithCancelCause(ctx)
	defer cancelStream(nil)
	storageReader, storageWriter := io.Pipe()
	defer func() { _ = storageReader.Close() }()
	finalWriter, encryptionWriter, encSalt, encNonce, err := setupEncryption(
		storageWriter,
		p.Encryption,
		p.MasterKey,
		p.BackupID,
	)
	if err != nil {
		_ = storageWriter.Close()
		return streamOutcome{}, err
	}

	fileWrite := p.FileStore.StartBackgroundWrite(streamCtx, storages.BackgroundWriteSpec{
		Reference: storages.StoredFileReference{
			StorageID: p.StorageID,
			FileName:  p.FileName,
		},
		Source:    storageReader,
		OnFailure: cancelStream,
	})
	backup, streamErr := p.Source.StreamBaseBackup(streamCtx, spec.Backup, finalWriter)
	if encryptionWriter != nil {
		_ = encryptionWriter.Close()
	}
	_ = storageWriter.Close()
	saveErr := <-fileWrite.Errors
	var failure *pgphysical.BaseBackupError
	if errors.As(streamErr, &failure) {
		switch failure.Kind {
		case pgphysical.BaseBackupStalled:
			return outcomeNetworkStall(failure.Stderr), nil
		case pgphysical.BaseBackupCodecRejected:
			return streamOutcome{isCompressionUnsupported: true, Stderr: failure.Stderr}, nil
		case pgphysical.BaseBackupManifestBroken:
			return outcomeManifestCorrupted(failure.Cause, failure.Stderr), nil
		case pgphysical.BaseBackupContextEnded, pgphysical.BaseBackupToolFailed:
		}
	}

	if saveErr != nil {
		return outcomeStorageUploadFailed(saveErr), nil
	}

	if streamErr != nil {
		if failure != nil {
			return spec.Classify(failure.Cause, failure.Stderr), nil
		}
		return streamOutcome{}, streamErr
	}

	manifestFileName := p.FileName + manifestSuffix
	manifest, err := saveManifestSidecar(
		ctx,
		manifestSidecarSpec{Params: p, FileName: manifestFileName, Manifest: backup.Manifest},
	)
	if err != nil {
		return outcomeManifestCorrupted(fmt.Errorf("save manifest sidecar: %w", err), nil), nil
	}

	encAlgo := backups_core_enums.BackupEncryptionNone
	if encryptionWriter != nil {
		encAlgo = backups_core_enums.BackupEncryptionEncrypted
	}

	return streamOutcome{
		Status:                 physical_enums.PhysicalBackupStatusCompleted,
		TimelineID:             backup.TimelineID,
		StartLSN:               backup.StartLSN,
		StopLSN:                backup.StopLSN,
		BackupSizeMb:           float64(backup.StreamedSize) / (1024 * 1024),
		EncryptionAlgo:         encAlgo,
		EncryptionSalt:         encSalt,
		EncryptionIV:           encNonce,
		Compression:            p.Codec,
		ManifestFileName:       manifestFileName,
		ManifestEncryptionSalt: manifest.SaltBase64,
		ManifestEncryptionIV:   manifest.NonceBase64,
		Receipts:               []storages.WriteReceipt{<-fileWrite.Receipts, manifest.Receipt},
	}, nil
}

func outcomeNetworkStall(stderr []byte) streamOutcome {
	reason := physical_enums.PhysicalBackupErrorNetworkStallTimeout

	return streamOutcome{
		Status:       physical_enums.PhysicalBackupStatusError,
		ErrorReason:  &reason,
		ErrorMessage: "byte-stall watcher cancelled pg_basebackup",
		Stderr:       stderr,
	}
}

func outcomeStorageUploadFailed(saveErr error) streamOutcome {
	reason := physical_enums.PhysicalBackupErrorStorageUploadFailed

	return streamOutcome{
		Status:       physical_enums.PhysicalBackupStatusError,
		ErrorReason:  &reason,
		ErrorMessage: fmt.Sprintf("save to storage: %v", saveErr),
	}
}

func outcomeManifestCorrupted(cause error, stderr []byte) streamOutcome {
	reason := physical_enums.PhysicalBackupErrorManifestCorrupted

	return streamOutcome{
		Status:       physical_enums.PhysicalBackupStatusChainBroken,
		ErrorReason:  &reason,
		ErrorMessage: cause.Error(),
		Stderr:       stderr,
	}
}

// compressionExhaustedOutcome is the terminal failure when every codec —
// including `none`, which never legitimately raises the capability error — was
// rejected. That is not a real "unsupported" case but a genuine pg_basebackup
// failure, so it maps to PG_BASEBACKUP_FAILED.
func compressionExhaustedOutcome(stderr []byte) streamOutcome {
	reason := physical_enums.PhysicalBackupErrorPgBasebackupFailed

	return streamOutcome{
		Status:      physical_enums.PhysicalBackupStatusError,
		ErrorReason: &reason,
		ErrorMessage: fmt.Sprintf(
			"all compression codecs rejected by source build; stderr: %s",
			truncateStderr(stderr),
		),
	}
}

func truncateStderr(stderr []byte) string {
	const limit = 2048
	if len(stderr) > limit {
		stderr = stderr[len(stderr)-limit:]
	}

	return string(stderr)
}
