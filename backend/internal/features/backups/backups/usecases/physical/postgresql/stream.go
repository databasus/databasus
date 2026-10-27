package usecases_physical_postgresql

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"time"

	"github.com/google/uuid"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/modules/storages"
)

// Bounds for the post-success PUT of the reconstructed `.manifest` sidecar.
// serializeManifest emits ~210 bytes per source file, so a 500k-file cluster
// produces a ~100 MB sidecar that a slow S3-compatible endpoint cannot absorb
// under a flat deadline — and losing the sidecar marks an already-stored backup
// CHAIN_BROKEN. The floor matches walSegmentUploadTimeout and covers endpoint
// latency on the common few-MB manifest; the ceiling keeps a wedged endpoint
// from pinning a backup worker indefinitely.
const (
	manifestSaveMinTimeout            = 2 * time.Minute
	manifestSaveMaxTimeout            = 15 * time.Minute
	manifestSaveThroughputBytesPerSec = 256 * 1024
)

// runStreamParams carries everything runStream needs that does not come from the
// *exec.Cmd itself. Grouped into a struct so FULL and INCR pass the same shape.
type runStreamParams struct {
	FileStore FileStore
	StorageID uuid.UUID
	Logger    *slog.Logger

	FileName   string
	Encryption backups_core_enums.BackupEncryption
	MasterKey  *crypto.MasterKey
	BackupID   uuid.UUID

	SystemID uint64
	Codec    physical_enums.PhysicalBackupCompression

	ProgressListener func(completedMb float64, elapsedMs int64)
}

// streamErrorClassifier maps pg_basebackup's OWN failure (a non-zero exit with a
// nil cancel cause) into a terminal streamOutcome. runStream resolves stalls,
// codec rejection, manifest and storage failures from the cancel cause / stderr
// BEFORE calling this, so the classifier only distinguishes generic pg failure
// from the INCR-only SummariesExpired case.
type streamErrorClassifier func(streamErr error, stderr []byte) streamOutcome

// manifestWalkResult is the manifest goroutine's return, delivered over a
// buffered channel so the goroutine never blocks on send during teardown.
type manifestWalkResult struct {
	entries []manifestFileEntry
	err     error
}

// runStream drives one pg_basebackup attempt: it tees the (codec-compressed)
// stdout into two independent branches — the storage upload and an in-flight
// manifest reconstruction — runs the process to completion, then builds and
// saves the reconstructed backup_manifest sidecar.
//
// Concurrency invariants (the riskiest part of the whole feature; each has a
// matching teardown below):
//
//   - The tee is io.MultiWriter(counter, manifestWriter). The counter sits on the
//     STORAGE branch only, so BackupSizeMb and the byte-stall watcher both measure
//     bytes reaching storage (a stuck upload is exactly the stall we want to
//     catch). The manifest branch taps PRE-encryption stdout because it must parse
//     the plaintext tar.
//   - The two branches are independent: storage stores opaque compressed bytes and
//     knows nothing of tar boundaries; only the manifest branch parses tar.
//   - context.WithCancelCause lets post-stream classification read WHY the stream
//     aborted: the stall watcher cancels with errByteStall, the manifest goroutine
//     with errManifestWalk, a storage failure with the real save error. cmd is
//     built bound to streamCtx so cancelling actually SIGINTs pg_basebackup.
//   - io.MultiWriter is fail-fast and sequential, so a failure in either branch
//     starves the other's reader. The manifest goroutine ALWAYS CloseWithError on
//     exit so a dead walk unblocks the copy instantly; teardown then symmetrically
//     closes both pipe writers and drains both result channels — including on the
//     cmd.Start early-return path — so no goroutine and no pipe leaks.
func runStream(
	ctx context.Context,
	p runStreamParams,
	buildCmd func(context.Context) (*exec.Cmd, error),
	classify streamErrorClassifier,
) (streamOutcome, error) {
	streamCtx, cancelStream := context.WithCancelCause(ctx)
	defer cancelStream(nil)

	cmd, err := buildCmd(streamCtx)
	if err != nil {
		return streamOutcome{}, err
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return streamOutcome{}, fmt.Errorf("stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return streamOutcome{}, fmt.Errorf("stderr pipe: %w", err)
	}

	storageReader, storageWriter := io.Pipe()

	finalWriter, encryptionWriter, encSalt, encNonce, err := setupEncryption(
		storageWriter, p.Encryption, p.MasterKey, p.BackupID,
	)
	if err != nil {
		_ = storageWriter.Close()

		return streamOutcome{}, err
	}

	counter := NewByteCounter(finalWriter)

	stopWatcher := WithByteStallWatcher(streamCtx, counter, ByteStallThresholds{
		FirstByteTimeout: FirstByteTimeout,
		StallTimeout:     ByteStallTimeout,
	}, func() {
		p.Logger.Warn("byte-stall timeout tripped; terminating pg_basebackup",
			"backup_id", p.BackupID,
			"file_name", p.FileName)

		cancelStream(errByteStall)
	})
	defer stopWatcher()

	if p.ProgressListener != nil {
		stopProgress := WithProgressReporter(streamCtx, counter, progressReportInterval, p.ProgressListener)
		defer stopProgress()
	}

	fileWrite := p.FileStore.StartBackgroundWrite(streamCtx, storages.BackgroundWriteSpec{
		Reference: storages.StoredFileReference{StorageID: p.StorageID, FileName: p.FileName},
		Source:    storageReader,
		OnFailure: cancelStream,
	})

	manifestReader, manifestWriter := io.Pipe()
	manifestResultCh := make(chan manifestWalkResult, 1)

	go func() {
		entries, walkErr := walkTarForManifest(manifestReader, p.Codec)

		// Unblock the MultiWriter's manifestWriter.Write instantly if the walk died
		// early — without this the copy would hang on a full pipe nobody reads.
		_ = manifestReader.CloseWithError(walkErr)

		// Cancel only for a genuine reconstruction failure (a malformed tar on a
		// stream pg_basebackup finished cleanly). A truncated stream is pg's own
		// failure surfacing here; leaving the cause unset lets the precedence below
		// attribute it to pg_basebackup, not to the manifest.
		if walkErr != nil && !errors.Is(walkErr, io.ErrUnexpectedEOF) && !errors.Is(walkErr, io.EOF) {
			cancelStream(fmt.Errorf("%w: %w", errManifestWalk, walkErr))
		}

		manifestResultCh <- manifestWalkResult{entries: entries, err: walkErr}
	}()

	stderr := newStderrCapture(stderrPipe)

	if startErr := cmd.Start(); startErr != nil {
		// Symmetric teardown for the early-return path: close BOTH writers and
		// drain BOTH goroutines, or the manifest goroutine leaks on manifestReader.
		_ = storageWriter.Close()
		_ = manifestWriter.Close()
		saveErr := <-fileWrite.Errors
		<-manifestResultCh
		stderr.stop()

		// A storage failure cancels streamCtx, and cmd is bound to streamCtx, so when
		// the upload fails before Start runs the failure surfaces here as "context
		// canceled". The real cause is the upload, not the spawn — classify it as
		// STORAGE_UPLOAD_FAILED so a down backend is never mislabelled a start error.
		if saveErr != nil && errors.Is(startErr, context.Canceled) {
			return outcomeStorageUploadFailed(saveErr), nil
		}

		return streamOutcome{}, fmt.Errorf("start pg_basebackup: %w", startErr)
	}

	copyErrCh := make(chan error, 1)

	go func() {
		_, copyErr := io.Copy(io.MultiWriter(counter, manifestWriter), stdoutPipe)
		copyErrCh <- copyErr
	}()

	copyErr := <-copyErrCh

	if streamCtx.Err() != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}

	// Drain stderr to EOF before cmd.Wait: the child has closed stdout (the copy
	// returned) and, on the abort path, been killed, so its stderr is at or near
	// EOF. Reading it here keeps cmd.Wait — which closes the stderr pipe on process
	// exit — from truncating a fast child's stderr (e.g. the codec-rejection line)
	// out from under the capture goroutine under load.
	stderr.drain(PgBasebackupWaitTimeout)

	waitErr := waitCmdWithTimeout(cmd, PgBasebackupWaitTimeout, p.Logger, p.BackupID.String())

	if encryptionWriter != nil {
		_ = encryptionWriter.Close()
	}

	_ = storageWriter.Close()
	_ = manifestWriter.Close()

	saveErr := <-fileWrite.Errors
	manifestRes := <-manifestResultCh

	stderr.stop()
	stderrBytes := stderr.contents()

	// Error precedence. The cmd is bound to streamCtx, so ANY cancel (stall,
	// storage, manifest) also kills pg_basebackup and yields a non-nil waitErr —
	// the cancel CAUSE, not waitErr, is therefore authoritative for an aborted
	// stream. Read it once and branch deterministically.
	cause := context.Cause(streamCtx)

	// (1) Stall wins over everything.
	if errors.Is(cause, errByteStall) {
		return outcomeNetworkStall(stderrBytes), nil
	}

	// (2) Codec rejection is a pre-stream, stderr-only signal; check it before
	// storage so a rejected attempt's empty-upload error cannot mask the downgrade.
	if isCompressionUnsupportedError(stderrBytes) {
		return streamOutcome{isCompressionUnsupported: true, Stderr: stderrBytes}, nil
	}

	// (3) The manifest goroutine flagged a genuine reconstruction failure.
	if errors.Is(cause, errManifestWalk) {
		return outcomeManifestCorrupted(fmt.Errorf("walk tar for manifest: %w", manifestRes.err), stderrBytes), nil
	}

	// (4) Storage upload failed (its goroutine set the cause, or a non-cancel path
	// left saveErr).
	if saveErr != nil {
		return outcomeStorageUploadFailed(saveErr), nil
	}

	// (5) pg_basebackup's own failure — non-zero exit with no cancel cause.
	if waitErr != nil {
		return classify(waitErr, stderrBytes), nil
	}

	if copyErr != nil && !errors.Is(copyErr, io.EOF) {
		return classify(copyErr, stderrBytes), nil
	}

	// (6) Defensive: a walk error that did not record a cause (e.g. truncation on a
	// stream that nonetheless exited clean).
	if manifestRes.err != nil {
		return outcomeManifestCorrupted(fmt.Errorf("walk tar for manifest: %w", manifestRes.err), stderrBytes), nil
	}

	return buildSuccessOutcome(ctx, successOutcomeSpec{
		Params:           p,
		Files:            manifestRes.entries,
		Counter:          counter,
		EncryptionWriter: encryptionWriter,
		EncryptionSalt:   encSalt,
		EncryptionIV:     encNonce,
		Stderr:           stderrBytes,
		BaseReceipt:      <-fileWrite.Receipts,
	})
}

// buildSuccessOutcome runs in the linear section after Wait: it parses the LSNs
// (only complete once the process exited), serializes the manifest, and saves the
// sidecar. Any failure here is fatal to the backup — an INCR chain without a
// valid parent manifest is unusable — so it maps to MANIFEST_CORRUPTED.
type successOutcomeSpec struct {
	Params           runStreamParams
	Files            []manifestFileEntry
	Counter          *ByteCounter
	EncryptionWriter *crypto.EncryptingWriter
	EncryptionSalt   string
	EncryptionIV     string
	Stderr           []byte
	BaseReceipt      storages.WriteReceipt
}

func buildSuccessOutcome(ctx context.Context, spec successOutcomeSpec) (streamOutcome, error) {
	p := spec.Params
	files := spec.Files
	counter := spec.Counter
	encryptionWriter := spec.EncryptionWriter
	encSalt, encNonce := spec.EncryptionSalt, spec.EncryptionIV
	stderrBytes := spec.Stderr

	startLSN, stopLSN, timelineID, err := parseLsnsFromStderr(stderrBytes)
	if err != nil {
		return outcomeManifestCorrupted(fmt.Errorf("parse pg_basebackup LSNs: %w", err), stderrBytes), nil
	}

	if p.SystemID == 0 {
		return outcomeManifestCorrupted(
			errors.New("source system identifier is 0; cannot build a manifest pg_combinebackup would accept"),
			stderrBytes,
		), nil
	}

	manifestBytes, err := serializeManifest(serializeManifestInput{
		Files:      files,
		SystemID:   p.SystemID,
		TimelineID: timelineID,
		StartLSN:   startLSN,
		StopLSN:    stopLSN,
	})
	if err != nil {
		return outcomeManifestCorrupted(fmt.Errorf("serialize manifest: %w", err), stderrBytes), nil
	}

	manifestFileName := p.FileName + manifestSuffix

	manifest, err := saveManifestSidecar(ctx, p, manifestFileName, manifestBytes)
	if err != nil {
		return outcomeManifestCorrupted(fmt.Errorf("save manifest sidecar: %w", err), stderrBytes), nil
	}

	encAlgo := backups_core_enums.BackupEncryptionNone
	if encryptionWriter != nil {
		encAlgo = backups_core_enums.BackupEncryptionEncrypted
	}

	return streamOutcome{
		Status:                 physical_enums.PhysicalBackupStatusCompleted,
		TimelineID:             timelineID,
		StartLSN:               startLSN,
		StopLSN:                stopLSN,
		BackupSizeMb:           float64(counter.BytesWritten()) / (1024 * 1024),
		EncryptionAlgo:         encAlgo,
		EncryptionSalt:         encSalt,
		EncryptionIV:           encNonce,
		Compression:            p.Codec,
		ManifestFileName:       manifestFileName,
		ManifestEncryptionSalt: manifest.SaltBase64,
		ManifestEncryptionIV:   manifest.NonceBase64,
		Receipts:               []storages.WriteReceipt{spec.BaseReceipt, manifest.Receipt},
		Stderr:                 stderrBytes,
	}, nil
}

func manifestSaveTimeout(payloadSizeBytes int) time.Duration {
	budget := time.Duration(payloadSizeBytes/manifestSaveThroughputBytesPerSec) * time.Second

	return min(max(budget, manifestSaveMinTimeout), manifestSaveMaxTimeout)
}

// saveManifestSidecar uploads the reconstructed manifest. When encryption is on
// it uses a FRESH salt+nonce (NewEncryptingWriter) — never the tar's — because
// the chunk nonce is baseNonce||chunkIndex, so reusing the tar's (salt, nonce)
// would reuse (key, nonce) on chunk 0 (a catastrophic GCM failure). The fresh
// salt/IV are returned (base64) for persistence so the INCR consumer can decrypt.
func saveManifestSidecar(
	ctx context.Context,
	p runStreamParams,
	manifestFileName string,
	manifestBytes []byte,
) (manifestSidecar, error) {
	var (
		saltBase64  string
		nonceBase64 string
	)

	payload := bytes.NewReader(manifestBytes)
	payloadSizeBytes := len(manifestBytes)

	if p.Encryption == backups_core_enums.BackupEncryptionEncrypted {
		var encrypted bytes.Buffer

		encryptingWriter, header, setupErr := crypto.NewEncryptingWriter(&encrypted, p.MasterKey, p.BackupID)
		if setupErr != nil {
			return manifestSidecar{}, fmt.Errorf("setup manifest encryption: %w", setupErr)
		}

		if _, writeErr := encryptingWriter.Write(manifestBytes); writeErr != nil {
			return manifestSidecar{}, fmt.Errorf("encrypt manifest: %w", writeErr)
		}

		if closeErr := encryptingWriter.Close(); closeErr != nil {
			return manifestSidecar{}, fmt.Errorf("flush manifest encryption: %w", closeErr)
		}

		payload = bytes.NewReader(encrypted.Bytes())
		payloadSizeBytes = encrypted.Len()
		saltBase64 = base64.StdEncoding.EncodeToString(header.Salt)
		nonceBase64 = base64.StdEncoding.EncodeToString(header.Nonce)
	}

	// The deadline starts here, after encryption, so CPU-bound work never eats the
	// upload's budget, and it is sized from the bytes that actually go over the wire.
	saveTimeout := manifestSaveTimeout(payloadSizeBytes)

	p.Logger.Debug(
		fmt.Sprintf("uploading manifest sidecar: %.2f MB, timeout %s",
			float64(payloadSizeBytes)/(1024*1024), saveTimeout),
		"backup_id", p.BackupID,
	)

	saveCtx, cancel := context.WithTimeout(ctx, saveTimeout)
	defer cancel()

	receipt, err := p.FileStore.WriteFile(
		saveCtx,
		storages.StoredFileReference{StorageID: p.StorageID, FileName: manifestFileName},
		payload,
	)
	if err != nil {
		return manifestSidecar{}, err
	}

	return manifestSidecar{Receipt: receipt, SaltBase64: saltBase64, NonceBase64: nonceBase64}, nil
}

// manifestSidecar is what the manifest upload produced: the receipt that keeps the
// file and the encryption material its catalog row records.
type manifestSidecar struct {
	Receipt     storages.WriteReceipt
	SaltBase64  string
	NonceBase64 string
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
