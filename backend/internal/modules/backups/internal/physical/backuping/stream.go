package backuping

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/google/uuid"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/storages"
)

type runStreamParams struct {
	FileStore  storages.Module
	StorageID  uuid.UUID
	Logger     *slog.Logger
	Source     *pgphysical.Source
	FileName   string
	Encryption catalog.BackupEncryption
	MasterKey  *crypto.MasterKey
	BackupID   uuid.UUID
	Codec      catalog.BackupCompression
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

	// A stalled storage blocks the write and may never return from its upload, so both are ended here.
	spec.Backup.ReleaseOutput = func(cause error) {
		_ = storageWriter.CloseWithError(cause)
		cancelStream(cause)
	}

	if encryptionWriter != nil {
		spec.Backup.FlushOutput = encryptionWriter.Close
	}

	backup, streamErr := p.Source.StreamBaseBackup(streamCtx, spec.Backup, finalWriter)
	_ = storageWriter.Close()
	saveErr := <-fileWrite.Errors

	if ctx.Err() != nil {
		reason := catalog.BackupErrorCanceledByUser

		return streamOutcome{
			Status:       catalog.BackupStatusCanceled,
			ErrorReason:  &reason,
			ErrorMessage: ctx.Err().Error(),
		}, nil
	}

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

	manifestFileName := p.FileName + catalog.ManifestSuffix
	manifest, err := saveManifestSidecar(
		ctx,
		manifestSidecarSpec{Params: p, FileName: manifestFileName, Manifest: backup.Manifest},
	)
	if err != nil {
		return outcomeManifestCorrupted(fmt.Errorf("save manifest sidecar: %w", err), nil), nil
	}

	encAlgo := catalog.BackupEncryptionNone
	if encryptionWriter != nil {
		encAlgo = catalog.BackupEncryptionEncrypted
	}

	return streamOutcome{
		Status:                 catalog.BackupStatusCompleted,
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
	reason := catalog.BackupErrorNetworkStallTimeout

	return streamOutcome{
		Status:       catalog.BackupStatusError,
		ErrorReason:  &reason,
		ErrorMessage: "byte-stall watcher cancelled pg_basebackup",
		Stderr:       stderr,
	}
}

func outcomeStorageUploadFailed(saveErr error) streamOutcome {
	reason := catalog.BackupErrorStorageUploadFailed

	return streamOutcome{
		Status:       catalog.BackupStatusError,
		ErrorReason:  &reason,
		ErrorMessage: fmt.Sprintf("save to storage: %v", saveErr),
	}
}

func outcomeManifestCorrupted(cause error, stderr []byte) streamOutcome {
	reason := catalog.BackupErrorManifestCorrupted

	return streamOutcome{
		Status:       catalog.BackupStatusChainBroken,
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
	reason := catalog.BackupErrorPgBasebackupFailed

	return streamOutcome{
		Status:      catalog.BackupStatusError,
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
