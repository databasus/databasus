package usecases_physical_postgresql

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	physical_dto "databasus-backend/internal/features/backups/backups/core/physical/dto"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/storages"
	db "databasus-backend/internal/storage"
	"databasus-backend/internal/util/walmath"
)

// walSegmentUploadTimeout bounds a single WAL-segment SaveFile. A segment is one
// wal_segment_size (16 MB default), so this is a short bounded upload, not a
// multi-hour stream like a FULL.
const walSegmentUploadTimeout = 2 * time.Minute

// WalUploadDeps carries everything the WAL uploader needs. WalSegmentSizeBytes is
// the source cluster's wal_segment_size (captured on the DB row), used to derive
// per-segment LSN bounds WITHOUT the walmath.WalSegmentSize package global — so a
// cluster with a non-default segsize is parsed correctly even when several DBs
// stream concurrently in one process.
type WalUploadDeps struct {
	DatabaseID          uuid.UUID
	StorageID           uuid.UUID
	FileStore           FileStore
	BackupStatus        BackupStatusRecorder
	Encryption          backups_core_enums.BackupEncryption
	MasterKey           *crypto.MasterKey
	WalSegmentRepo      *physical_repositories.PhysicalWalSegmentRepository
	WalSegmentSizeBytes int64
	Logger              *slog.Logger

	// The local queue: segments waiting here, or in its pending-upload/, are still coming, so a hole
	// in front of a committed segment is not a gap while its predecessor sits here.
	WatchDir string

	// OnGapDetected fires once per WAL gap: only the first segment after a hole reports it, and that
	// segment commits once. nil disables notification.
	OnGapDetected func(gapStart, gapEnd walmath.LSN)
}

// WalUploader runs the insert-first claim -> upload -> commit -> gap-probe flow
// for fully-rotated WAL segments. It is the integrity core of the streamer and is
// unit-testable independent of pg_receivewal: hand it a deps bundle and a path to
// a finalized segment file.
type WalUploader struct {
	deps WalUploadDeps

	mu sync.Mutex

	// The uploader loop retries every second, so a torn segment that never gets
	// re-streamed would otherwise warn once per second forever.
	warnedIncompleteSegments map[string]struct{}

	stats archivingStats
}

func NewWalUploader(deps WalUploadDeps) *WalUploader {
	return &WalUploader{deps: deps, warnedIncompleteSegments: make(map[string]struct{})}
}

// ProcessSegment runs the full insert-first claim flow for one finalized WAL
// segment file. localPath must point at a finalized segment (the caller filters
// out *.partial via walmath.IsWalFilename). It is safe to call repeatedly for the
// same segment: the claim/commit guards make re-processing idempotent.
func (u *WalUploader) ProcessSegment(ctx context.Context, localPath, walFilename string) error {
	timelineID, startLSN, endLSN, err := segmentBounds(walFilename, u.deps.WalSegmentSizeBytes)
	if err != nil {
		return err
	}

	if !isCompleteSegmentFile(localPath, u.deps.WalSegmentSizeBytes) {
		u.warnIncompleteSegmentOnce(walFilename)

		return nil
	}

	u.forgetIncompleteSegment(walFilename)

	claim := &physical_models.PhysicalWalSegment{
		ID:          uuid.New(),
		DatabaseID:  u.deps.DatabaseID,
		StorageID:   u.deps.StorageID,
		TimelineID:  timelineID,
		WalFilename: walFilename,
		StartLSN:    startLSN,
		EndLSN:      endLSN,
		Encryption:  u.deps.Encryption,
		ReceivedAt:  time.Now().UTC(),
		ClaimedAt:   time.Now().UTC(),
	}

	claimStartedAt := time.Now()

	inserted, err := u.deps.WalSegmentRepo.ClaimInsert(claim)
	if err != nil {
		return fmt.Errorf("claim wal segment: %w", err)
	}

	if !inserted {
		return u.handleClaimConflict(localPath, claim)
	}

	return u.uploadAndCommit(ctx, localPath, claim, time.Since(claimStartedAt))
}

// RecoverSegment runs the insert-first flow for a leftover finalized segment
// found in watch_dir at supervisor startup (crash recovery). It differs from
// ProcessSegment in exactly one branch: a pre-existing claim with file_name NULL
// is taken over and uploaded rather than left in place. Under the
// single-supervisor-per-DB invariant such a claim is always THIS database's own
// pre-crash claim (no concurrent uploader can exist), so finishing it is safe and
// avoids leaving the segment stuck NULL until the cleaner's 1h grace sweep.
func (u *WalUploader) RecoverSegment(ctx context.Context, localPath, walFilename string) error {
	timelineID, startLSN, endLSN, err := segmentBounds(walFilename, u.deps.WalSegmentSizeBytes)
	if err != nil {
		return err
	}

	if !isCompleteSegmentFile(localPath, u.deps.WalSegmentSizeBytes) {
		u.warnIncompleteSegmentOnce(walFilename)

		return nil
	}

	u.forgetIncompleteSegment(walFilename)

	claim := &physical_models.PhysicalWalSegment{
		ID:          uuid.New(),
		DatabaseID:  u.deps.DatabaseID,
		StorageID:   u.deps.StorageID,
		TimelineID:  timelineID,
		WalFilename: walFilename,
		StartLSN:    startLSN,
		EndLSN:      endLSN,
		Encryption:  u.deps.Encryption,
		ReceivedAt:  time.Now().UTC(),
		ClaimedAt:   time.Now().UTC(),
	}

	claimStartedAt := time.Now()

	inserted, err := u.deps.WalSegmentRepo.ClaimInsert(claim)
	if err != nil {
		return fmt.Errorf("claim wal segment: %w", err)
	}

	if inserted {
		return u.uploadAndCommit(ctx, localPath, claim, time.Since(claimStartedAt))
	}

	existing, err := u.deps.WalSegmentRepo.FindByChainKey(claim.DatabaseID, claim.TimelineID, claim.StartLSN)
	if err != nil {
		return fmt.Errorf("probe existing wal claim: %w", err)
	}

	if existing == nil {
		return nil
	}

	if existing.FileName != nil {
		u.removeLocal(localPath, claim.WalFilename)

		return nil
	}

	// Own pre-crash NULL claim: take it over by uploading against the existing row.
	return u.uploadAndCommit(ctx, localPath, existing, time.Since(claimStartedAt))
}

// One rollup line per interval, so the snapshot is consumed rather than read.
func (u *WalUploader) TakeArchivingSnapshot() ArchivingSnapshot {
	return u.stats.takeSnapshot()
}

// The dispatcher knows how many uploads run, the uploader knows what they cost; reporting the count
// here keeps both in the one snapshot the rollup takes.
func (u *WalUploader) RecordConcurrentUploads(count int) {
	u.stats.recordConcurrentUploads(count)
}

// handleClaimConflict resolves a lost ClaimInsert race by probing the existing
// row: a durably-completed segment (file_name NOT NULL) means our local copy is
// redundant; an in-flight claim (file_name NULL) means a live or
// not-yet-reaped owner holds it, so we leave the local file untouched.
func (u *WalUploader) handleClaimConflict(localPath string, claim *physical_models.PhysicalWalSegment) error {
	logger := u.deps.Logger.With("wal_filename", claim.WalFilename, "wal_segment_id", claim.ID)

	existing, err := u.deps.WalSegmentRepo.FindByChainKey(claim.DatabaseID, claim.TimelineID, claim.StartLSN)
	if err != nil {
		return fmt.Errorf("probe existing wal claim: %w", err)
	}

	if existing == nil {
		// Row vanished between the conflict and the probe — leave the local file;
		// the next uploader tick re-claims.
		logger.Debug("wal claim row vanished before the probe, re-claiming on the next tick")

		return nil
	}

	if existing.FileName != nil {
		logger.Debug("wal segment already durable, dropping the local copy")

		u.removeLocal(localPath, claim.WalFilename)

		return nil
	}

	// In-flight claim: do not touch storage, leave the local file for a later tick
	// (the cleaner reaps the stale NULL claim after its grace period). Until that happens this
	// segment cannot be archived by anyone, so it is worth more than a debug line.
	logger.Warn("wal segment is held by another in-flight claim, leaving it queued locally")

	return nil
}

// uploadAndCommit uploads the artifact + sidecar, then flips file_name to non-NULL.
// The commit is guarded by file_name IS NULL, so a DeleteFull cascade that removed
// the claim mid-upload yields committed=false and the orphaned ciphertext is
// deleted. On any pre-commit failure the claim row is released so the next tick
// re-claims from the retained local file.
func (u *WalUploader) uploadAndCommit(
	ctx context.Context,
	localPath string,
	claim *physical_models.PhysicalWalSegment,
	claimDuration time.Duration,
) error {
	timings := segmentUploadTimings{Claim: claimDuration}

	// One name per upload attempt, minted here and passed down: a retry under the
	// same segment identity must not address the object a pending cleanup owns.
	objectName := walSegmentObjectName(claim.DatabaseID, claim.TimelineID, claim.WalFilename, uuid.New())

	artifact, salt, iv, err := buildWalSegmentArtifactReader(localPath, u.deps.Encryption, u.deps.MasterKey, claim.ID)
	if err != nil {
		u.releaseClaim(claim.ID)

		return fmt.Errorf("build wal segment artifact: %w", err)
	}

	saveCtx, cancel := context.WithTimeout(ctx, walSegmentUploadTimeout)
	defer cancel()

	segmentWriteStartedAt := time.Now()

	receipt, err := u.deps.FileStore.WriteFile(saveCtx, u.reference(objectName), artifact)
	if err != nil {
		artifact.abort(err)
		_, _ = artifact.wait()
		u.discardAttempt(ctx, objectName)
		u.releaseClaim(claim.ID)

		return fmt.Errorf("upload wal segment: %w", err)
	}

	compressedSizeBytes, err := artifact.wait()
	if err != nil {
		u.discardAttempt(ctx, objectName)
		u.releaseClaim(claim.ID)

		return fmt.Errorf("stream wal segment artifact: %w", err)
	}

	timings.SegmentWrite = time.Since(segmentWriteStartedAt)
	sidecarWriteStartedAt := time.Now()

	sidecarReceipt, err := u.uploadSidecar(saveCtx, claim, objectName, salt, iv, compressedSizeBytes)
	if err != nil {
		u.discardAttempt(ctx, objectName)
		u.releaseClaim(claim.ID)

		return fmt.Errorf("upload wal segment sidecar: %w", err)
	}

	timings.SidecarWrite = time.Since(sidecarWriteStartedAt)
	commitStartedAt := time.Now()

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
		// DeleteFull cascade caught the claim mid-upload, so the row that would
		// name these bytes is gone and nothing published them.
		u.deps.Logger.DebugContext(ctx,
			"discarding an uploaded wal segment, its chain was deleted mid-upload",
			"wal_filename", claim.WalFilename, "wal_segment_id", claim.ID)

		u.discardAttempt(ctx, objectName)
		u.removeLocal(localPath, claim.WalFilename)

		return nil
	}

	// Per segment rather than per archive: a busy cluster finalizes one every few seconds, which
	// would swamp INFO. The periodic rollup carries the same information at a readable rate.
	u.deps.Logger.DebugContext(ctx, fmt.Sprintf("archived wal segment %s (%.2f MB)",
		claim.WalFilename, float64(compressedSizeBytes)/(1024*1024)),
		"wal_segment_id", claim.ID, "timeline_id", claim.TimelineID)

	u.probeChainGap(claim)

	timings.Commit = time.Since(commitStartedAt)
	u.stats.recordArchived(compressedSizeBytes, timings)

	u.removeLocal(localPath, claim.WalFilename)

	return nil
}

// probeChainGap reports a gap in front of the just-committed segment only when the segment directly
// before it will never arrive. Under concurrent uploads a later segment often commits first, so a
// missing predecessor that still waits in the local queue, or that commits meanwhile, is not a gap.
// Only the first segment after a real hole reports it, and that segment commits once, which keeps
// each gap to one notification without any bookkeeping.
func (u *WalUploader) probeChainGap(claim *physical_models.PhysicalWalSegment) {
	prevEnd, err := u.deps.WalSegmentRepo.FindLatestCommittedBefore(claim.DatabaseID, claim.TimelineID, claim.StartLSN)
	if err != nil {
		u.deps.Logger.Error("wal gap probe query failed", "wal_filename", claim.WalFilename, "error", err)

		return
	}

	if prevEnd == nil || *prevEnd == claim.StartLSN {
		return
	}

	// The local check runs before the catalog check: a predecessor that commits concurrently writes
	// its row before it removes its file, so at least one of the two sees it.
	queuedSegments := listQueuedWalSegments(u.deps.WatchDir, u.deps.WalSegmentSizeBytes)
	if isPredecessorQueued(queuedSegments, claim) {
		return
	}

	predecessorStartLSN := claim.StartLSN - walmath.LSN(u.deps.WalSegmentSizeBytes)

	predecessor, err := u.deps.WalSegmentRepo.FindByChainKey(claim.DatabaseID, claim.TimelineID, predecessorStartLSN)
	if err != nil {
		u.deps.Logger.Error("wal gap probe query failed", "wal_filename", claim.WalFilename, "error", err)

		return
	}

	if predecessor != nil {
		return
	}

	gapStart := max(*prevEnd, getHighestQueuedEndBelow(queuedSegments, claim))

	u.deps.Logger.Warn(
		fmt.Sprintf("wal_gap_detected_post_upload: gap [%s, %s)", gapStart.String(), claim.StartLSN.String()),
		"database_id", claim.DatabaseID,
		"timeline_id", claim.TimelineID,
	)

	if u.deps.OnGapDetected == nil {
		return
	}

	u.deps.OnGapDetected(gapStart, claim.StartLSN)
}

// A short file never counts: pg_receivewal resumes after the highest complete segment and never
// streams a segment below that point again, so counting a torn file would hide its hole forever.
// Such a file surfaces through the archive staleness alert instead.
func isPredecessorQueued(queuedSegments []queuedWalSegment, claim *physical_models.PhysicalWalSegment) bool {
	return slices.ContainsFunc(queuedSegments, func(queued queuedWalSegment) bool {
		return queued.isComplete && queued.timelineID == claim.TimelineID && queued.endLSN == claim.StartLSN
	})
}

// The newest committed predecessor can lag behind a segment that is still uploading, which would make
// the reported hole wider than the real one.
func getHighestQueuedEndBelow(
	queuedSegments []queuedWalSegment,
	claim *physical_models.PhysicalWalSegment,
) walmath.LSN {
	var highestEnd walmath.LSN

	for _, queued := range queuedSegments {
		if queued.isComplete && queued.timelineID == claim.TimelineID && queued.endLSN <= claim.StartLSN {
			highestEnd = max(highestEnd, queued.endLSN)
		}
	}

	return highestEnd
}

// uploadSidecar writes the `<object>.metadata` JSON describing the segment so a
// fresh Databasus can rebuild the catalog row from storage alone (import-from-DR).
func (u *WalUploader) uploadSidecar(
	ctx context.Context,
	claim *physical_models.PhysicalWalSegment,
	objectName string,
	salt, iv string,
	compressedSizeBytes int64,
) (storages.WriteReceipt, error) {
	sidecar := physical_dto.PhysicalWalSegmentMetadata{
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

	sidecarName := objectName + metadataSuffix

	return u.deps.FileStore.WriteFile(ctx, u.reference(sidecarName), bytes.NewReader(body))
}

func (u *WalUploader) releaseClaim(id uuid.UUID) {
	if err := u.deps.WalSegmentRepo.DeleteClaim(id); err != nil {
		u.deps.Logger.Error("failed to release wal segment claim", "wal_segment_id", id, "error", err)
	}
}

func (u *WalUploader) reference(objectName string) storages.StoredFileReference {
	return storages.StoredFileReference{StorageID: u.deps.StorageID, FileName: objectName}
}

// No catalog row survives to carry these files, so the request gets a transaction
// of its own.
func (u *WalUploader) discardAttempt(ctx context.Context, objectName string) {
	references := []storages.StoredFileReference{
		u.reference(objectName),
		u.reference(objectName + metadataSuffix),
	}

	ctx = context.WithoutCancel(ctx)

	if err := db.GetTransactor().Transaction(ctx, func(ctx context.Context) error {
		return u.deps.FileStore.RequestFileDeletions(ctx, references)
	}); err != nil {
		u.deps.Logger.Warn("failed to discard a wal segment attempt", "file_name", objectName, "error", err)
	}
}

type commitSegmentSpec struct {
	Claim            *physical_models.PhysicalWalSegment
	ObjectName       string
	CompressedSizeMb float64
	Salt             string
	IV               string
	Receipts         []storages.WriteReceipt
}

// commitSegment publishes the segment, spends its receipts and records it as the database's last
// backup together, so the files are kept only if the row that names them commits.
func (u *WalUploader) commitSegment(ctx context.Context, spec commitSegmentSpec) (bool, error) {
	committed := false

	err := db.GetTransactor().Transaction(ctx, func(ctx context.Context) error {
		tx := database.Conn(ctx, db.GetDb())

		updated, err := u.deps.WalSegmentRepo.MarkUploadedInTransaction(tx, physical_repositories.WalSegmentUpload{
			SegmentID:        spec.Claim.ID,
			FileName:         spec.ObjectName,
			CompressedSizeMb: spec.CompressedSizeMb,
			EncryptionSalt:   nilIfEmpty(spec.Salt),
			EncryptionIV:     nilIfEmpty(spec.IV),
		})
		if err != nil {
			return err
		}

		committed = updated
		if !updated {
			return nil
		}

		err = u.deps.FileStore.ConfirmFileWrites(ctx, spec.Receipts)
		if err != nil {
			return err
		}

		return u.deps.BackupStatus.RecordBackupSuccess(ctx, u.deps.DatabaseID, spec.Claim.ReceivedAt)
	})

	return committed, err
}

func (u *WalUploader) removeLocal(localPath, walFilename string) {
	if err := os.Remove(localPath); err != nil && !os.IsNotExist(err) {
		u.deps.Logger.Warn("failed to remove local wal segment", "wal_filename", walFilename, "error", err)
	}
}

func (u *WalUploader) warnIncompleteSegmentOnce(walFilename string) {
	u.mu.Lock()
	_, wasWarned := u.warnedIncompleteSegments[walFilename]
	u.warnedIncompleteSegments[walFilename] = struct{}{}
	u.mu.Unlock()

	if wasWarned {
		return
	}

	u.deps.Logger.Warn("skipping incomplete local wal segment", "wal_filename", walFilename)
}

func (u *WalUploader) forgetIncompleteSegment(walFilename string) {
	u.mu.Lock()
	defer u.mu.Unlock()

	delete(u.warnedIncompleteSegments, walFilename)
}

func segmentBounds(walFilename string, segSizeBytes int64) (timelineID int, startLSN, endLSN walmath.LSN, err error) {
	timeline, segmentNo, err := walmath.ParseWALFilenameWithSize(walFilename, uint64(segSizeBytes))
	if err != nil {
		return 0, 0, 0, err
	}

	start := walmath.LSN(segmentNo * uint64(segSizeBytes))

	return int(timeline), start, start + walmath.LSN(segSizeBytes), nil
}

// The attempt UUID is what keeps a retry from addressing the object a pending
// cleanup already owns; the rest of the key stays readable for an operator
// listing a bucket.
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
	localPath string,
	encryption backups_core_enums.BackupEncryption,
	masterKey *crypto.MasterKey,
	segmentID uuid.UUID,
) (*walSegmentArtifactReader, string, string, error) {
	file, err := os.Open(localPath)
	if err != nil {
		return nil, "", "", fmt.Errorf("open wal segment file: %w", err)
	}

	pipeReader, pipeWriter := io.Pipe()
	output := io.Writer(pipeWriter)
	var encryptionWriter io.Closer
	var salt, iv string

	if encryption == backups_core_enums.BackupEncryptionEncrypted {
		encryptingWriter, header, err := crypto.NewEncryptingWriter(pipeWriter, masterKey, segmentID)
		if err != nil {
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

	go func() {
		defer func() { _ = file.Close() }()

		err := streamWalSegmentArtifact(file, output, encryptionWriter)
		if err != nil {
			_ = pipeWriter.CloseWithError(err)
			artifact.done <- err

			return
		}

		artifact.done <- pipeWriter.Close()
	}()

	return artifact, salt, iv, nil
}

func streamWalSegmentArtifact(source io.Reader, output io.Writer, encryptionWriter io.Closer) error {
	// Up to four segments compress at once per database. One goroutine per encoder and the
	// lower-memory history buffer take each encoder from about 18 MB to about 9.5 MB, and neither
	// changes the compressed format or the window.
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
