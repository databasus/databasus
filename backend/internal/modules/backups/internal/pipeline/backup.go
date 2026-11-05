package pipeline

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"

	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/crypto"
	io_utils "databasus-backend/internal/lib/iox"
	"databasus-backend/internal/modules/backups/internal/runcontext"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/storages"
)

type BackupSpec struct {
	BackupID     uuid.UUID
	DatabaseID   uuid.UUID
	DatabaseType databases.Type
	StorageID    uuid.UUID
	FileName     string
	IsEncrypted  bool
	// OnProgress receives the stored size every ProgressInterval while the dump streams, apart from
	// the goroutine that copies it, and once more when the stream ends successfully.
	OnProgress func(storedSizeMb float64)
	// ProgressInterval is how often OnProgress receives the stored size while the dump streams.
	ProgressInterval time.Duration
	Logger           *slog.Logger
}

type BackupResult struct {
	// EncryptionHeader is nil for a backup stored without encryption.
	EncryptionHeader *EncryptionHeader
	// RawDatabaseSizeMb is 0 when the database's size could not be read before the dump.
	RawDatabaseSizeMb  float64
	TimescaledbVersion string
	StoredSizeMb       float64
	// Receipt is what the publishing transaction spends to keep the stored file.
	Receipt storages.WriteReceipt
}

// EncryptionHeader is the salt and the nonce of an encrypted file, base64-encoded.
type EncryptionHeader struct {
	Salt  string
	Nonce string
}

// RunBackup dumps the database into its storage under ctx, which runcontext.New derives from the application context. An
// ended ctx returns the error DescribeBackupInterruption gives, and a panic returns a
// *background.PanicError, so the caller always records an outcome.
func (p *Pipeline) RunBackup(ctx context.Context, spec BackupSpec) (BackupResult, error) {
	var result BackupResult

	err := background.RunRecovering(ctx, func(ctx context.Context) error {
		var err error

		result, err = p.runBackup(ctx, spec)
		if err != nil && ctx.Err() != nil {
			return runcontext.DescribeBackupInterruption(ctx, ctx)
		}

		return err
	})

	return result, err
}

func (p *Pipeline) runBackup(ctx context.Context, spec BackupSpec) (BackupResult, error) {
	engine, err := p.getEngine(spec.DatabaseType)
	if err != nil {
		return BackupResult{}, err
	}

	dumpSource, err := engine.OpenDumpSource(ctx, spec.DatabaseID, spec.Logger)
	if err != nil {
		return BackupResult{}, err
	}
	defer dumpSource.Close()

	rawSizeMb, _ := dumpSource.GetRawSizeMb(ctx)
	timescaledbVersion := dumpSource.GetTimescaleDBVersion(ctx)

	result, err := p.streamDumpToStorage(ctx, backupStream{
		spec:        spec,
		dumpSource:  dumpSource,
		compression: engine.GetStoredCompression(),
	})
	if err != nil {
		return BackupResult{}, err
	}

	result.RawDatabaseSizeMb = rawSizeMb
	result.TimescaledbVersion = timescaledbVersion

	return result, nil
}

type backupStream struct {
	spec        BackupSpec
	dumpSource  logicalDumpSource
	compression storedCompression
}

// streamDumpToStorage runs the dump into the chain [zstd ->] counting -> encryption -> pipe, whose
// other end the storage's background write uploads. A slow storage blocks the pipe and through it
// the tool, so nothing lands on the local disk.
func (p *Pipeline) streamDumpToStorage(runCtx context.Context, stream backupStream) (BackupResult, error) {
	backupCtx, cancel := context.WithCancelCause(runCtx)
	defer cancel(nil)

	pipeReader, pipeWriter := io.Pipe()
	storagePipe := &storagePipeWriter{pipe: pipeWriter, hasWriteFailed: atomic.Bool{}}

	// An ended backup closes the pipe, so a write blocked on the storage returns at once and the
	// tool, whose output can no longer go anywhere, is stopped with its context.
	stopClosingPipe := context.AfterFunc(backupCtx, func() {
		_ = pipeWriter.CloseWithError(context.Cause(backupCtx))
	})
	defer stopClosingPipe()

	chain, encryptionHeader, err := p.buildWriterChain(storagePipe, stream)
	if err != nil {
		return BackupResult{}, err
	}

	storageWrite := &backgroundWriteResult{
		write: p.storages.StartBackgroundWrite(backupCtx, storages.BackgroundWriteSpec{
			Reference: storages.StoredFileReference{
				StorageID: stream.spec.StorageID,
				FileName:  stream.spec.FileName,
			},
			Source:    pipeReader,
			OnFailure: cancel,
		}),
		err:    nil,
		isDone: false,
	}

	stream.spec.Logger.InfoContext(runCtx, "streaming backup to storage")

	stopReportingProgress := startReportingProgress(backupCtx, progressReportSpec{
		interval:        stream.spec.ProgressInterval,
		getStoredSizeMb: chain.getStoredSizeMb,
		listener:        stream.spec.OnProgress,
		logger:          stream.spec.Logger,
	})
	defer stopReportingProgress()

	dumpErr := stream.dumpSource.Dump(backupCtx, chain.input)

	err = finishBackupStream(backupRun{
		runCtx:       runCtx,
		backupCtx:    backupCtx,
		chain:        chain,
		storagePipe:  storagePipe,
		storageWrite: storageWrite,
	}, dumpErr)

	stopReportingProgress()

	if err != nil {
		return BackupResult{}, err
	}

	storedSizeMb := chain.getStoredSizeMb()
	stream.spec.OnProgress(storedSizeMb)

	return BackupResult{
		EncryptionHeader:   encryptionHeader,
		RawDatabaseSizeMb:  0,
		TimescaledbVersion: "",
		StoredSizeMb:       storedSizeMb,
		Receipt:            <-storageWrite.write.Receipts,
	}, nil
}

type backupRun struct {
	runCtx       context.Context
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

		return runcontext.DescribeBackupInterruption(run.runCtx, run.backupCtx)
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
		return runcontext.DescribeBackupInterruption(run.runCtx, run.backupCtx)
	}

	return err
}

func (p *Pipeline) buildWriterChain(
	storagePipe *storagePipeWriter,
	stream backupStream,
) (*backupWriterChain, *EncryptionHeader, error) {
	chain := &backupWriterChain{
		input:            nil,
		zstdEncoder:      nil,
		storedBytes:      nil,
		encryptingWriter: nil,
		storagePipe:      storagePipe,
	}

	var encryptionHeader *EncryptionHeader

	var encryptedOutput io.Writer = storagePipe

	if stream.spec.IsEncrypted {
		encryptingWriter, header, err := crypto.NewEncryptingWriter(storagePipe, p.masterKey, stream.spec.BackupID)
		if err != nil {
			return nil, nil, err
		}

		encryptionHeader = &EncryptionHeader{
			Salt:  base64.StdEncoding.EncodeToString(header.Salt),
			Nonce: base64.StdEncoding.EncodeToString(header.Nonce),
		}
		chain.encryptingWriter = encryptingWriter
		encryptedOutput = encryptingWriter
	}

	chain.storedBytes = io_utils.NewCountingWriter(encryptedOutput)
	chain.input = chain.storedBytes

	if stream.compression.isZstd {
		zstdEncoder, err := zstd.NewWriter(
			chain.storedBytes,
			zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(stream.compression.zstdLevel)),
		)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create zstd writer: %w", err)
		}

		chain.zstdEncoder = zstdEncoder
		chain.input = zstdEncoder
	}

	return chain, encryptionHeader, nil
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
