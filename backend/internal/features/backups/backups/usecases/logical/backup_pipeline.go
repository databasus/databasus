package usecases_logical

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"sync/atomic"

	"github.com/klauspost/compress/zstd"

	"databasus-backend/internal/config"
	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/modules/storages"
	io_utils "databasus-backend/internal/util/io"
)

const progressReportIntervalMb = 1.0

type backupStream struct {
	backup           *backups_core_logical.LogicalBackup
	backupConfig     *backups_config_logical.LogicalBackupConfig
	dumpSource       logicalDumpSource
	fileStore        backups_core_logical.BackupFileStore
	progressListener func(completedMBs float64)
	logger           *slog.Logger
}

// streamDumpToStorage runs the dump into the chain [zstd ->] counting -> encryption -> pipe, whose
// other end the storage's background write uploads. A slow storage blocks the pipe and through it
// the tool, so nothing lands on the local disk.
func (uc *CreateBackupUsecase) streamDumpToStorage(
	parentCtx context.Context,
	stream backupStream,
) (*backups_core_logical.BackupArtifacts, error) {
	backupCtx, cancel := backups_core_logical.NewBackupContext(parentCtx, config.GetShutdownSignal())
	defer cancel(nil)

	pipeReader, pipeWriter := io.Pipe()
	storagePipe := &storagePipeWriter{pipe: pipeWriter, hasWriteFailed: atomic.Bool{}}

	// An ended backup closes the pipe, so a write blocked on the storage returns at once and the
	// tool, whose output can no longer go anywhere, is stopped with its context.
	stopClosingPipe := context.AfterFunc(backupCtx, func() {
		_ = pipeWriter.CloseWithError(context.Cause(backupCtx))
	})
	defer stopClosingPipe()

	chain, metadata, err := uc.buildWriterChain(storagePipe, stream)
	if err != nil {
		return nil, err
	}

	storageWrite := &backgroundWriteResult{
		write: stream.fileStore.StartBackgroundWrite(backupCtx, storages.BackgroundWriteSpec{
			Reference: storages.StoredFileReference{
				StorageID: stream.backup.StorageID,
				FileName:  stream.backup.FileName,
			},
			Source:    pipeReader,
			OnFailure: cancel,
		}),
		err:    nil,
		isDone: false,
	}

	stream.logger.InfoContext(parentCtx, "streaming backup to storage")

	dumpErr := stream.dumpSource.Dump(backupCtx, chain.input)

	err = finishBackupStream(backupRun{
		callerCtx:    parentCtx,
		backupCtx:    backupCtx,
		chain:        chain,
		storagePipe:  storagePipe,
		storageWrite: storageWrite,
	}, dumpErr)
	if err != nil {
		return nil, err
	}

	if stream.progressListener != nil {
		stream.progressListener(chain.getStoredSizeMb())
	}

	return &backups_core_logical.BackupArtifacts{
		Metadata: &metadata,
		Receipts: []storages.WriteReceipt{<-storageWrite.write.Receipts},
	}, nil
}

type backupRun struct {
	callerCtx    context.Context
	backupCtx    context.Context
	chain        *backupWriterChain
	storagePipe  *storagePipeWriter
	storageWrite *backgroundWriteResult
}

// finishBackupStream closes the chain and waits for the storage, then picks the error to report.
// An ended backup context comes first: a failed upload cancels it with the upload's error, which
// DescribeBackupInterruption reports as "save to storage". A failed close of the writers comes
// next, then the tool's error, then the storage's.
func finishBackupStream(run backupRun, dumpErr error) error {
	if run.storagePipe.hasWriteFailed.Load() {
		// Only the storage or an ended backup closes the pipe. Waiting for the storage lets a
		// failed upload cancel the backup context before it is read below.
		_ = run.storageWrite.wait()
	}

	if run.backupCtx.Err() != nil {
		run.chain.abandon()
		_ = run.storageWrite.wait()

		return backups_core_logical.DescribeBackupInterruption(run.callerCtx, run.backupCtx)
	}

	closeErr := run.chain.close()
	if closeErr != nil {
		_ = run.storagePipe.pipe.CloseWithError(closeErr)
		_ = run.storageWrite.wait()

		return pickInterruptionOr(run, closeErr)
	}

	saveErr := run.storageWrite.wait()

	switch {
	case dumpErr != nil:
		return pickInterruptionOr(run, dumpErr)
	case saveErr != nil:
		return pickInterruptionOr(run, fmt.Errorf("save to storage: %w", saveErr))
	default:
		return nil
	}
}

func pickInterruptionOr(run backupRun, err error) error {
	if run.backupCtx.Err() != nil {
		return backups_core_logical.DescribeBackupInterruption(run.callerCtx, run.backupCtx)
	}

	return err
}

func (uc *CreateBackupUsecase) buildWriterChain(
	storagePipe *storagePipeWriter,
	stream backupStream,
) (*backupWriterChain, backups_core_logical.BackupMetadata, error) {
	chain := &backupWriterChain{
		input:            nil,
		zstdEncoder:      nil,
		storedBytes:      nil,
		encryptingWriter: nil,
		storagePipe:      storagePipe,
	}

	metadata := backups_core_logical.BackupMetadata{
		BackupID:       stream.backup.ID,
		EncryptionSalt: nil,
		EncryptionIV:   nil,
		Encryption:     backups_core_enums.BackupEncryptionNone,
	}

	var encryptedOutput io.Writer = storagePipe

	if stream.backupConfig.Encryption == backups_core_enums.BackupEncryptionEncrypted {
		encryptingWriter, header, err := crypto.NewEncryptingWriter(storagePipe, uc.masterKey, stream.backup.ID)
		if err != nil {
			return nil, metadata, err
		}

		saltBase64 := base64.StdEncoding.EncodeToString(header.Salt)
		nonceBase64 := base64.StdEncoding.EncodeToString(header.Nonce)
		metadata.Encryption = backups_core_enums.BackupEncryptionEncrypted
		metadata.EncryptionSalt = &saltBase64
		metadata.EncryptionIV = &nonceBase64

		chain.encryptingWriter = encryptingWriter
		encryptedOutput = encryptingWriter

		stream.logger.Info("encryption enabled for backup")
	} else {
		stream.logger.Info("encryption disabled for backup")
	}

	chain.storedBytes = io_utils.NewCountingWriter(encryptedOutput)

	var dumpOutput io.Writer = chain.storedBytes

	if compression := stream.dumpSource.GetStoredCompression(); compression.isZstd {
		zstdEncoder, err := zstd.NewWriter(
			chain.storedBytes,
			zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(compression.zstdLevel)),
		)
		if err != nil {
			return nil, metadata, fmt.Errorf("failed to create zstd writer: %w", err)
		}

		chain.zstdEncoder = zstdEncoder
		dumpOutput = zstdEncoder
	}

	chain.input = &progressReportingWriter{
		output:         dumpOutput,
		storedBytes:    chain.storedBytes,
		listener:       stream.progressListener,
		lastReportedMb: 0,
	}

	return chain, metadata, nil
}

// backupWriterChain is the writers between the tool's output and the storage pipe. storedBytes
// counts the stored format before encryption, which is what progress reports.
type backupWriterChain struct {
	input            io.Writer
	zstdEncoder      *zstd.Encoder
	storedBytes      *io_utils.CountingWriter
	encryptingWriter *crypto.EncryptingWriter
	storagePipe      *storagePipeWriter
}

// close flushes the zstd frame and the last encrypted chunk, then ends the pipe, so the storage
// sees the end of the file.
func (c *backupWriterChain) close() error {
	if c.zstdEncoder != nil {
		err := c.zstdEncoder.Close()
		if err != nil {
			return fmt.Errorf("failed to close zstd writer: %w", err)
		}
	}

	if c.encryptingWriter != nil {
		err := c.encryptingWriter.Close()
		if err != nil {
			return fmt.Errorf("failed to close encryption writer: %w", err)
		}
	}

	return c.storagePipe.pipe.Close()
}

// abandon releases the zstd encoder of a backup that failed; the pipe is already closed.
func (c *backupWriterChain) abandon() {
	if c.zstdEncoder != nil {
		_ = c.zstdEncoder.Close()
	}
}

func (c *backupWriterChain) getStoredSizeMb() float64 {
	return float64(c.storedBytes.GetBytesWritten()) / (1024 * 1024)
}

// progressReportingWriter reports the stored size every megabyte. It sits where the tool's output
// enters the chain, so the listener is called from the one goroutine that copies that output.
type progressReportingWriter struct {
	output         io.Writer
	storedBytes    *io_utils.CountingWriter
	listener       func(completedMBs float64)
	lastReportedMb float64
}

func (w *progressReportingWriter) Write(p []byte) (int, error) {
	writtenBytes, err := w.output.Write(p)

	if w.listener != nil {
		storedSizeMb := float64(w.storedBytes.GetBytesWritten()) / (1024 * 1024)
		if storedSizeMb >= w.lastReportedMb+progressReportIntervalMb {
			w.listener(storedSizeMb)
			w.lastReportedMb = storedSizeMb
		}
	}

	return writtenBytes, err
}

// storagePipeWriter records whether a write into the pipe failed, which happens only when the
// storage or an ended backup closed the pipe.
type storagePipeWriter struct {
	pipe           *io.PipeWriter
	hasWriteFailed atomic.Bool
}

func (w *storagePipeWriter) Write(p []byte) (int, error) {
	writtenBytes, err := w.pipe.Write(p)
	if err != nil {
		w.hasWriteFailed.Store(true)
	}

	return writtenBytes, err
}

// backgroundWriteResult reads the storage's result once and keeps it.
type backgroundWriteResult struct {
	write  storages.BackgroundWrite
	err    error
	isDone bool
}

func (r *backgroundWriteResult) wait() error {
	if !r.isDone {
		r.err = <-r.write.Errors
		r.isDone = true
	}

	return r.err
}
