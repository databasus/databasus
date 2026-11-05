package walstreaming

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/engines/postgresql/physical/walmath"
	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/storages"
)

const walSegmentUploadTimeout = 2 * time.Minute

type WalUploadDeps struct {
	Publication         config.PublicationFence
	Clock               clock.Source
	Transactor          *database.Transactor
	DatabaseID          uuid.UUID
	StorageID           uuid.UUID
	FileStore           FileStore
	BackupStatus        BackupStatusRecorder
	Encryption          catalog.BackupEncryption
	MasterKey           *crypto.MasterKey
	WalSegmentRepo      *catalog.WalSegmentRepository
	WalSegmentSizeBytes int64
	Logger              *slog.Logger

	WatchDir string

	OnGapDetected func(gapStart, gapEnd walmath.LSN)
}

type WalUploader struct {
	deps WalUploadDeps

	mu sync.Mutex

	warnedIncompleteSegments map[string]struct{}

	stats archivingStats
}

func NewWalUploader(deps WalUploadDeps) *WalUploader {
	return &WalUploader{deps: deps, warnedIncompleteSegments: make(map[string]struct{})}
}

func (u *WalUploader) ProcessSegment(ctx context.Context, localPath, walFilename string) error {
	timelineID, startLSN, endLSN, err := segmentBounds(walFilename, u.deps.WalSegmentSizeBytes)
	if err != nil {
		return err
	}

	if !pgphysical.IsCompleteSegmentFile(localPath, u.deps.WalSegmentSizeBytes) {
		u.warnIncompleteSegmentOnce(ctx, walFilename)

		return nil
	}

	u.forgetIncompleteSegment(walFilename)

	claim := &catalog.WalSegment{
		ID:          uuid.New(),
		DatabaseID:  u.deps.DatabaseID,
		StorageID:   u.deps.StorageID,
		TimelineID:  timelineID,
		WalFilename: walFilename,
		StartLSN:    startLSN,
		EndLSN:      endLSN,
		Encryption:  u.deps.Encryption,
		ReceivedAt:  u.deps.Clock.Now(),
		ClaimedAt:   u.deps.Clock.Now(),
	}

	claimStartedAt := u.deps.Clock.Now()

	inserted, err := u.claimSegment(ctx, claim)
	if err != nil {
		return fmt.Errorf("claim wal segment: %w", err)
	}

	if !inserted {
		return u.handleClaimConflict(ctx, localPath, claim)
	}

	return u.uploadAndCommit(ctx, localPath, claim, u.deps.Clock.Now().Sub(claimStartedAt))
}

func (u *WalUploader) RecoverSegment(ctx context.Context, localPath, walFilename string) error {
	timelineID, startLSN, endLSN, err := segmentBounds(walFilename, u.deps.WalSegmentSizeBytes)
	if err != nil {
		return err
	}

	if !pgphysical.IsCompleteSegmentFile(localPath, u.deps.WalSegmentSizeBytes) {
		u.warnIncompleteSegmentOnce(ctx, walFilename)

		return nil
	}

	u.forgetIncompleteSegment(walFilename)

	claim := &catalog.WalSegment{
		ID:          uuid.New(),
		DatabaseID:  u.deps.DatabaseID,
		StorageID:   u.deps.StorageID,
		TimelineID:  timelineID,
		WalFilename: walFilename,
		StartLSN:    startLSN,
		EndLSN:      endLSN,
		Encryption:  u.deps.Encryption,
		ReceivedAt:  u.deps.Clock.Now(),
		ClaimedAt:   u.deps.Clock.Now(),
	}

	claimStartedAt := u.deps.Clock.Now()

	inserted, err := u.claimSegment(ctx, claim)
	if err != nil {
		return fmt.Errorf("claim wal segment: %w", err)
	}

	if inserted {
		return u.uploadAndCommit(ctx, localPath, claim, u.deps.Clock.Now().Sub(claimStartedAt))
	}

	existing, _, err := u.deps.WalSegmentRepo.FindByChainKey(ctx, claim.DatabaseID, claim.TimelineID, claim.StartLSN)
	if err != nil {
		return fmt.Errorf("probe existing wal claim: %w", err)
	}

	action := decideExistingWalClaim(existingWalClaim{Segment: existing, IsRecovering: true})
	if action == walClaimWait {
		return nil
	}
	if action == walClaimRemoveLocal {
		u.removeLocal(ctx, localPath, claim.WalFilename)

		return nil
	}

	return u.uploadAndCommit(ctx, localPath, existing, u.deps.Clock.Now().Sub(claimStartedAt))
}

func (u *WalUploader) TakeArchivingSnapshot() ArchivingSnapshot {
	return u.stats.takeSnapshot()
}

func (u *WalUploader) RecordConcurrentUploads(count int) {
	u.stats.recordConcurrentUploads(count)
}

func (u *WalUploader) handleClaimConflict(ctx context.Context, localPath string, claim *catalog.WalSegment) error {
	logger := u.deps.Logger.With(slog.Any("wal_filename", claim.WalFilename), slog.Any("wal_segment_id", claim.ID))

	existing, _, err := u.deps.WalSegmentRepo.FindByChainKey(ctx, claim.DatabaseID, claim.TimelineID, claim.StartLSN)
	if err != nil {
		return fmt.Errorf("probe existing wal claim: %w", err)
	}

	action := decideExistingWalClaim(existingWalClaim{Segment: existing, IsRecovering: false})
	if action == walClaimWait && existing == nil {
		logger.DebugContext(ctx, "wal claim row vanished before the probe, re-claiming on the next tick")
		return nil
	}
	if action == walClaimRemoveLocal {
		logger.DebugContext(ctx, "wal segment already durable, dropping the local copy")
		u.removeLocal(ctx, localPath, claim.WalFilename)
		return nil
	}

	logger.WarnContext(ctx, "wal segment is held by another active WAL upload, leaving it queued locally")

	return nil
}

func (u *WalUploader) uploadAndCommit(
	ctx context.Context,
	localPath string,
	claim *catalog.WalSegment,
	claimDuration time.Duration,
) error {
	timings := segmentUploadTimings{Claim: claimDuration}

	objectName := walSegmentObjectName(claim.DatabaseID, claim.TimelineID, claim.WalFilename, uuid.New())

	artifact, salt, iv, err := buildWalSegmentArtifactReader(
		ctx,
		u.deps.Logger,
		localPath,
		u.deps.Encryption,
		u.deps.MasterKey,
		claim.ID,
	)
	if err != nil {
		u.releaseClaim(ctx, claim.ID)

		return fmt.Errorf("build wal segment artifact: %w", err)
	}

	saveCtx, cancel := context.WithTimeout(ctx, walSegmentUploadTimeout)
	defer cancel()

	segmentWriteStartedAt := u.deps.Clock.Now()

	receipt, err := u.deps.FileStore.WriteFile(saveCtx, u.reference(objectName), artifact)
	if err != nil {
		artifact.abort(err)
		_, _ = artifact.wait()
		u.discardAttempt(ctx, objectName)
		u.releaseClaim(ctx, claim.ID)

		return fmt.Errorf("upload wal segment: %w", err)
	}

	compressedSizeBytes, err := artifact.wait()
	if err != nil {
		u.discardAttempt(ctx, objectName)
		u.releaseClaim(ctx, claim.ID)

		return fmt.Errorf("stream wal segment artifact: %w", err)
	}

	timings.SegmentWrite = u.deps.Clock.Now().Sub(segmentWriteStartedAt)
	sidecarWriteStartedAt := u.deps.Clock.Now()

	sidecarReceipt, err := u.uploadSidecar(saveCtx, claim, objectName, salt, iv, compressedSizeBytes)
	if err != nil {
		u.discardAttempt(ctx, objectName)
		u.releaseClaim(ctx, claim.ID)

		return fmt.Errorf("upload wal segment sidecar: %w", err)
	}

	timings.SidecarWrite = u.deps.Clock.Now().Sub(sidecarWriteStartedAt)
	commitStartedAt := u.deps.Clock.Now()

	committed, err := u.commitSegment(ctx, commitSegmentSpec{
		Claim:            claim,
		ObjectName:       objectName,
		CompressedSizeMb: float64(compressedSizeBytes) / (1024 * 1024),
		Salt:             salt,
		IV:               iv,
		Receipts:         []storages.WriteReceipt{receipt, sidecarReceipt},
	})
	if err != nil {
		return fmt.Errorf("commit wal segment: %w", err)
	}

	if !committed {

		u.deps.Logger.DebugContext(
			ctx,
			"discarding an uploaded wal segment, its chain or database was deleted mid-upload",
			slog.Any("wal_filename", claim.WalFilename),
			slog.Any("wal_segment_id", claim.ID),
		)

		u.discardAttempt(ctx, objectName)
		u.removeLocal(ctx, localPath, claim.WalFilename)

		return nil
	}

	u.deps.Logger.DebugContext(
		ctx,
		"archived wal segment",
		slog.Any("wal_filename", claim.WalFilename),
		slog.Any("size_mb", float64(compressedSizeBytes)/(1024*1024)),
		slog.Any("wal_segment_id", claim.ID),
		slog.Any("timeline_id", claim.TimelineID),
	)

	u.probeChainGap(ctx, claim)

	timings.Commit = u.deps.Clock.Now().Sub(commitStartedAt)
	u.stats.recordArchived(compressedSizeBytes, timings)

	u.removeLocal(ctx, localPath, claim.WalFilename)

	return nil
}

func (u *WalUploader) probeChainGap(ctx context.Context, claim *catalog.WalSegment) {
	prevEnd, _, err := u.deps.WalSegmentRepo.FindLatestCommittedBefore(
		ctx,
		claim.DatabaseID,
		claim.TimelineID,
		claim.StartLSN,
	)
	if err != nil {
		u.deps.Logger.ErrorContext(
			ctx,
			"wal gap probe query failed",
			slog.Any("wal_filename", claim.WalFilename),
			slog.Any("error", err),
		)

		return
	}

	if prevEnd == nil || *prevEnd == claim.StartLSN {
		return
	}

	queuedSegments := listQueuedWalSegments(u.deps.WatchDir, u.deps.WalSegmentSizeBytes)
	if isPredecessorQueued(queuedSegments, claim) {
		return
	}

	predecessorStartLSN := claim.StartLSN - walmath.LSN(u.deps.WalSegmentSizeBytes)

	predecessor, _, err := u.deps.WalSegmentRepo.FindByChainKey(
		ctx,
		claim.DatabaseID,
		claim.TimelineID,
		predecessorStartLSN,
	)
	if err != nil {
		u.deps.Logger.ErrorContext(
			ctx,
			"wal gap probe query failed",
			slog.Any("wal_filename", claim.WalFilename),
			slog.Any("error", err),
		)

		return
	}

	if predecessor != nil {
		return
	}

	gap := decideWalChainGap(walGapState{
		PreviousEnd: prevEnd, Claim: claim, QueuedSegments: queuedSegments, HasPredecessor: predecessor != nil,
	})
	if gap == nil {
		return
	}
	gapStart := gap.Start

	u.deps.Logger.WarnContext(
		ctx,
		"wal_gap_detected_post_upload",
		slog.Any("gap_start_lsn", gapStart.String()),
		slog.Any("gap_end_lsn", claim.StartLSN.String()),
		slog.Any("database_id", claim.DatabaseID),
		slog.Any("timeline_id", claim.TimelineID),
	)

	if u.deps.OnGapDetected == nil {
		return
	}

	u.deps.OnGapDetected(gapStart, claim.StartLSN)
}

func isPredecessorQueued(queuedSegments []queuedWalSegment, claim *catalog.WalSegment) bool {
	return slices.ContainsFunc(queuedSegments, func(queued queuedWalSegment) bool {
		return queued.isComplete && queued.timelineID == claim.TimelineID && queued.endLSN == claim.StartLSN
	})
}

func getHighestQueuedEndBelow(
	queuedSegments []queuedWalSegment,
	claim *catalog.WalSegment,
) walmath.LSN {
	var highestEnd walmath.LSN

	for _, queued := range queuedSegments {
		if queued.isComplete && queued.timelineID == claim.TimelineID && queued.endLSN <= claim.StartLSN {
			highestEnd = max(highestEnd, queued.endLSN)
		}
	}

	return highestEnd
}

func (u *WalUploader) uploadSidecar(
	ctx context.Context,
	claim *catalog.WalSegment,
	objectName string,
	salt, iv string,
	compressedSizeBytes int64,
) (storages.WriteReceipt, error) {
	sidecar := catalog.WalSegmentMetadata{
		WalSegmentID:        claim.ID,
		DatabaseID:          claim.DatabaseID,
		TimelineID:          claim.TimelineID,
		WalFilename:         claim.WalFilename,
		StartLSN:            claim.StartLSN.String(),
		EndLSN:              claim.EndLSN.String(),
		CompressedSizeBytes: compressedSizeBytes,
		Encryption:          u.deps.Encryption,
		EncryptionSalt:      salt,
		EncryptionIV:        iv,
		ReceivedAt:          claim.ReceivedAt,
	}

	body, err := json.Marshal(sidecar)
	if err != nil {
		return storages.WriteReceipt{}, fmt.Errorf("marshal wal segment sidecar: %w", err)
	}

	sidecarName := objectName + catalog.MetadataSuffix

	return u.deps.FileStore.WriteFile(ctx, u.reference(sidecarName), bytes.NewReader(body))
}

func (u *WalUploader) releaseClaim(ctx context.Context, id uuid.UUID) {
	if err := u.deps.WalSegmentRepo.DeleteClaim(ctx, id); err != nil {
		u.deps.Logger.ErrorContext(
			ctx,
			"failed to release wal segment claim",
			slog.Any("wal_segment_id", id),
			slog.Any("error", err),
		)
	}
}

func (u *WalUploader) reference(objectName string) storages.StoredFileReference {
	return storages.StoredFileReference{StorageID: u.deps.StorageID, FileName: objectName}
}

func (u *WalUploader) discardAttempt(ctx context.Context, objectName string) {
	references := []storages.StoredFileReference{
		u.reference(objectName),
		u.reference(objectName + catalog.MetadataSuffix),
	}

	ctx = context.WithoutCancel(ctx)

	if err := u.deps.Transactor.Transaction(ctx, func(ctx context.Context) error {
		return u.deps.FileStore.RequestDiscardedFileDeletions(
			ctx,
			storages.DiscardedFiles{DatabaseID: u.deps.DatabaseID, References: references},
		)
	}); err != nil {
		u.deps.Logger.WarnContext(
			ctx,
			"failed to discard a wal segment attempt",
			slog.Any("file_name", objectName),
			slog.Any("error", err),
		)
	}
}

type commitSegmentSpec struct {
	Claim            *catalog.WalSegment
	ObjectName       string
	CompressedSizeMb float64
	Salt             string
	IV               string
	Receipts         []storages.WriteReceipt
}

func (u *WalUploader) commitSegment(ctx context.Context, spec commitSegmentSpec) (bool, error) {
	err := u.deps.Transactor.Transaction(ctx, func(ctx context.Context) error {
		if err := u.deps.Publication.Validate(ctx); err != nil {
			return err
		}
		err := u.deps.BackupStatus.RecordBackupSuccess(ctx, u.deps.DatabaseID, spec.Claim.ReceivedAt)
		if err != nil {
			return err
		}

		updated, err := u.deps.WalSegmentRepo.MarkUploaded(ctx, catalog.WalSegmentUpload{
			SegmentID:        spec.Claim.ID,
			FileName:         spec.ObjectName,
			CompressedSizeMb: spec.CompressedSizeMb,
			EncryptionSalt:   nilIfEmpty(spec.Salt),
			EncryptionIV:     nilIfEmpty(spec.IV),
		})
		if err != nil {
			return err
		}

		if !updated {
			return errSegmentClaimLost
		}

		return u.deps.FileStore.ConfirmFileWrites(ctx, spec.Receipts)
	})
	if errors.Is(err, config.ErrStaleStoragePublication) || errors.Is(err, errSegmentClaimLost) ||
		errors.Is(err, databases.ErrDatabaseNotFound) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

func (u *WalUploader) removeLocal(ctx context.Context, localPath, walFilename string) {
	if err := os.Remove(localPath); err != nil && !os.IsNotExist(err) {
		u.deps.Logger.WarnContext(
			ctx,
			"failed to remove local wal segment",
			slog.Any("wal_filename", walFilename),
			slog.Any("error", err),
		)
	}
}

func (u *WalUploader) warnIncompleteSegmentOnce(ctx context.Context, walFilename string) {
	u.mu.Lock()
	_, wasWarned := u.warnedIncompleteSegments[walFilename]
	u.warnedIncompleteSegments[walFilename] = struct{}{}
	u.mu.Unlock()

	if wasWarned {
		return
	}

	u.deps.Logger.WarnContext(ctx, "skipping incomplete local wal segment", slog.Any("wal_filename", walFilename))
}

func (u *WalUploader) forgetIncompleteSegment(walFilename string) {
	u.mu.Lock()
	defer u.mu.Unlock()

	delete(u.warnedIncompleteSegments, walFilename)
}

func segmentBounds(walFilename string, segSizeBytes int64) (timelineID int, startLSN, endLSN walmath.LSN, err error) {
	timeline, segmentNo, err := walmath.ParseWALFilename(walFilename, uint64(segSizeBytes))
	if err != nil {
		return 0, 0, 0, err
	}

	start := walmath.LSN(segmentNo * uint64(segSizeBytes))

	return int(timeline), start, start + walmath.LSN(segSizeBytes), nil
}

func walSegmentObjectName(
	databaseID uuid.UUID,
	timelineID int,
	walFilename string,
	attemptID uuid.UUID,
) string {
	return fmt.Sprintf("%s-WAL-tl%d-%s-%s.zst", databaseID, timelineID, walFilename, attemptID)
}

type walSegmentArtifactReader struct {
	reader     *countingReader
	pipeReader *io.PipeReader
	done       chan error
}

func (r *walSegmentArtifactReader) Read(p []byte) (int, error) {
	return r.reader.Read(p)
}

func (r *walSegmentArtifactReader) abort(err error) {
	_ = r.pipeReader.CloseWithError(err)
}

func (r *walSegmentArtifactReader) wait() (int64, error) {
	err := <-r.done

	return r.reader.bytesRead, err
}

type countingReader struct {
	reader    io.Reader
	bytesRead int64
}

func (r *countingReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.bytesRead += int64(n)

	return n, err
}

func buildWalSegmentArtifactReader(
	ctx context.Context, logger *slog.Logger,
	localPath string,
	encryption catalog.BackupEncryption,
	masterKey *crypto.MasterKey,
	segmentID uuid.UUID,
) (*walSegmentArtifactReader, string, string, error) {
	file, err := os.Open(localPath)
	if err != nil {
		return nil, "", "", fmt.Errorf("open wal segment file: %w", err)
	}

	pipeReader, pipeWriter := io.Pipe()
	stopCancellation := context.AfterFunc(ctx, func() { _ = pipeWriter.CloseWithError(ctx.Err()) })
	output := io.Writer(pipeWriter)
	var encryptionWriter io.Closer
	var salt, iv string

	if encryption == catalog.BackupEncryptionEncrypted {
		encryptingWriter, header, err := crypto.NewEncryptingWriter(pipeWriter, masterKey, segmentID)
		if err != nil {
			stopCancellation()
			_ = file.Close()
			_ = pipeReader.Close()
			_ = pipeWriter.Close()

			return nil, "", "", fmt.Errorf("setup wal segment encryption: %w", err)
		}

		output = encryptingWriter
		encryptionWriter = encryptingWriter
		salt = base64.StdEncoding.EncodeToString(header.Salt)
		iv = base64.StdEncoding.EncodeToString(header.Nonce)
	}

	artifact := &walSegmentArtifactReader{
		reader:     &countingReader{reader: pipeReader},
		pipeReader: pipeReader,
		done:       make(chan error, 1),
	}

	background.New(background.Deps{Logger: logger}).Go(ctx, "wal_segment_artifact", func(_ context.Context) error {
		defer func() { _ = file.Close() }()
		defer stopCancellation()

		err := background.RunRecovering(ctx, func(context.Context) error {
			return streamWalSegmentArtifact(file, output, encryptionWriter)
		})
		if err != nil {
			_ = pipeWriter.CloseWithError(err)
			artifact.done <- err

			return nil
		}

		artifact.done <- pipeWriter.Close()
		return nil
	})

	return artifact, salt, iv, nil
}

func streamWalSegmentArtifact(source io.Reader, output io.Writer, encryptionWriter io.Closer) error {
	encoder, err := zstd.NewWriter(
		output,
		zstd.WithEncoderLevel(zstd.SpeedDefault),
		zstd.WithEncoderConcurrency(1),
		zstd.WithLowerEncoderMem(true),
	)
	if err != nil {
		return fmt.Errorf("init zstd writer: %w", err)
	}

	if _, err := io.Copy(encoder, source); err != nil {
		_ = encoder.Close()

		return fmt.Errorf("zstd encode: %w", err)
	}

	if err := encoder.Close(); err != nil {
		return fmt.Errorf("close zstd writer: %w", err)
	}

	if encryptionWriter != nil {
		if err := encryptionWriter.Close(); err != nil {
			return fmt.Errorf("close wal segment encryption writer: %w", err)
		}
	}

	return nil
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}

	return &s
}

func (u *WalUploader) claimSegment(ctx context.Context, claim *catalog.WalSegment) (bool, error) {
	var isInserted bool
	err := u.deps.Transactor.Transaction(ctx, func(ctx context.Context) error {
		if err := u.deps.Publication.Validate(ctx); err != nil {
			return err
		}
		var err error
		isInserted, err = u.deps.WalSegmentRepo.ClaimInsert(ctx, claim, u.deps.Clock.Now())
		return err
	})
	return isInserted, err
}
