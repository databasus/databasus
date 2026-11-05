package pipeline

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"

	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/crypto"
	io_utils "databasus-backend/internal/lib/iox"
	"databasus-backend/internal/lib/shutdown"
	"databasus-backend/internal/modules/backups/internal/runcontext"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/storages"
)

const targetVersionDetectionTimeout = 15 * time.Second

type OpenRestoreTargetSpec struct {
	SourceDatabaseID   uuid.UUID
	SourceDatabaseType databases.Type
	Target             RestoreTarget
	IsTimescaleDB      bool
	Logger             *slog.Logger
}

// StoredBackup names the file of a backup and how it was encrypted.
type StoredBackup struct {
	BackupID  uuid.UUID
	StorageID uuid.UUID
	FileName  string
	// EncryptionHeader is nil for a backup stored without encryption.
	EncryptionHeader *EncryptionHeader
}

// OpenedRestoreTarget is a restore target reached through its tunnel, whose version can take the
// backup. One tunnel serves the version detection and the restore, and Close releases it.
type OpenedRestoreTarget struct {
	sink        logicalRestoreSink
	compression storedCompression
	pipeline    *Pipeline
	logger      *slog.Logger
}

// OpenRestoreTarget refuses a target of another engine with ErrTargetOfAnotherEngine, a target it
// cannot reach with a TargetConnectionError and a target too old for the backup with
// ErrTargetVersionTooOld.
func (p *Pipeline) OpenRestoreTarget(ctx context.Context, spec OpenRestoreTargetSpec) (*OpenedRestoreTarget, error) {
	engine, err := p.getEngine(spec.SourceDatabaseType)
	if err != nil {
		return nil, err
	}

	sink, err := engine.OpenRestoreTarget(ctx, restoreTargetSpec{
		newCleanupContext: func(ctx context.Context) (context.Context, context.CancelFunc) {
			return p.shutdownBudget.NewContext(
				ctx,
				shutdown.ContextSpec{Timeout: 30 * time.Second, Reserve: time.Second},
			)
		},
		target:           spec.Target,
		sourceDatabaseID: spec.SourceDatabaseID,
		isTimescaleDB:    spec.IsTimescaleDB,
		logger:           spec.Logger,
	})
	if err != nil {
		return nil, err
	}

	return &OpenedRestoreTarget{
		sink:        sink,
		compression: engine.GetStoredCompression(),
		pipeline:    p,
		logger:      spec.Logger,
	}, nil
}

// IsArchiveFileNeeded reports whether the restore downloads the archive into a file before the tool
// reads it, so the caller checks the disk first.
func (t *OpenedRestoreTarget) IsArchiveFileNeeded() bool {
	return t.sink.IsArchiveFileNeeded()
}

// Restore streams the backup from its storage into the target under runCtx, which
// runcontext.New derives. An ended runCtx returns the error DescribeRestoreInterruption gives,
// and a panic returns a *background.PanicError, so the caller always records an outcome.
func (t *OpenedRestoreTarget) Restore(runCtx context.Context, backup StoredBackup) error {
	return background.RunRecovering(runCtx, func(runCtx context.Context) error {
		return t.restore(runCtx, backup)
	})
}

func (t *OpenedRestoreTarget) Close() {
	t.sink.Close()
}

func (t *OpenedRestoreTarget) restore(runCtx context.Context, backup StoredBackup) error {
	storageReader, err := t.pipeline.storages.GetStorageReader(runCtx, backup.StorageID)
	if err != nil {
		if runCtx.Err() != nil {
			return runcontext.DescribeRestoreInterruption(runCtx)
		}

		return fmt.Errorf("open the backup's storage: %w", err)
	}

	return t.pipeline.restoreFromStorage(runCtx, restoreRun{
		backup:        backup,
		storageReader: storageReader,
		sink:          t.sink,
		compression:   t.compression,
		logger:        t.logger,
	})
}

type restoreRun struct {
	backup        StoredBackup
	storageReader storages.StorageReader
	sink          logicalRestoreSink
	compression   storedCompression
	logger        *slog.Logger
}

// restoreFromStorage streams the backup file from the storage, decrypts and decompresses it, and
// hands it to the engine. Two trackers record the first failed read of the storage and of the
// decoded stream, because the restore tool fails on the half statement a broken stream fed it, and
// that error would otherwise be reported as the cause.
func (p *Pipeline) restoreFromStorage(runCtx context.Context, run restoreRun) error {
	backupFile, err := run.storageReader.OpenFile(runCtx, run.backup.FileName)
	if err != nil {
		if runCtx.Err() != nil {
			return runcontext.DescribeRestoreInterruption(runCtx)
		}

		return fmt.Errorf("failed to get backup file from storage: %w", err)
	}
	defer func() {
		closeErr := backupFile.Close()
		if closeErr != nil {
			run.logger.ErrorContext(runCtx, "failed to close backup reader", slog.Any("error", closeErr))
		}
	}()

	storageRead := io_utils.NewFailureTrackingReader(backupFile)

	decryptedStream, err := p.decryptIfEncrypted(storageRead, run.backup)
	if err != nil {
		return err
	}

	decodedStream := decryptedStream

	if run.compression.isZstd {
		zstdDecoder, err := zstd.NewReader(decryptedStream)
		if err != nil {
			return fmt.Errorf("failed to create zstd reader: %w", err)
		}
		defer zstdDecoder.Close()

		decodedStream = zstdDecoder
	}

	decodedStreamRead := io_utils.NewFailureTrackingReader(decodedStream)

	restoreErr := run.sink.Restore(runCtx, decodedStreamRead)

	_, hasArchiveFailure := errors.AsType[*archiveRestoreError](restoreErr)
	if runCtx.Err() != nil && !hasArchiveFailure {
		return runcontext.DescribeRestoreInterruption(runCtx)
	}

	streamErr := getBackupStreamFailure(backupStreamTrackers{
		storageRead:               storageRead,
		decodedStream:             decodedStreamRead,
		isDownloadedBeforeRestore: run.sink.IsArchiveFileNeeded(),
	})
	if streamErr != nil {
		return streamErr
	}

	return restoreErr
}

// decryptIfEncrypted reads the header of an encrypted file at once, so a file swapped for another
// backup's fails before the restore tool receives any data.
func (p *Pipeline) decryptIfEncrypted(storedStream io.Reader, backup StoredBackup) (io.Reader, error) {
	if backup.EncryptionHeader == nil {
		return storedStream, nil
	}

	salt, err := base64.StdEncoding.DecodeString(backup.EncryptionHeader.Salt)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encryption salt: %w", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(backup.EncryptionHeader.Nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encryption IV: %w", err)
	}

	decryptingReader, err := crypto.NewDecryptingReader(
		storedStream, p.masterKey, backup.BackupID, crypto.StreamHeader{Salt: salt, Nonce: nonce},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create decryption reader: %w", err)
	}

	return decryptingReader, nil
}
