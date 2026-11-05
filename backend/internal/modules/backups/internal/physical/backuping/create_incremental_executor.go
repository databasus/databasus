package backuping

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/backups/internal/physical/timelines"
)

type IncrementalExecutor struct{}

func NewIncrementalExecutor() *IncrementalExecutor {
	return &IncrementalExecutor{}
}

func (uc *IncrementalExecutor) Execute(
	ctx context.Context,
	spec IncrementalBackupSpec,
) (executionResult Result, executionErr error) {
	defer func() {
		if executionResult.CompletedAt.IsZero() {
			executionResult.CompletedAt = spec.Clock.Now()
		}
	}()
	start := spec.Clock.Now()
	executionStartedAt := spec.ExecutionClock.Now()

	password := spec.SourceDB.Engine.Password

	creds, err := pgshared.WriteCredentialFilesToTempDir(spec.SourceDB.Engine.GetCredentialSpec(), password)
	if err != nil {
		return errorResult(catalog.BackupErrorPgBasebackupFailed, "write credentials", err), nil
	}
	defer creds.Remove()

	refusalResult, rootFullTimelineID, canProceed := verifyIncrTimelineCompatibility(ctx, spec)
	if !canProceed {
		return refusalResult, nil
	}

	// Gate on WAL-summarizer readiness BEFORE any work: summarizer off and expired
	// summaries are conditions no retry can fix, so they close the chain here rather
	// than after a doomed pg_basebackup --incremental. Runs before the
	// parent-manifest download so a bail costs nothing.
	preCheckResult, canProceed := runSummarizerPreCheck(ctx, spec)
	if !canProceed {
		return preCheckResult, nil
	}

	// Manifest System-Identifier from the stored row (see create_full_executor.go Execute).
	systemID := spec.SourceDB.GetSystemIdentifierUint64()

	manifestPath, manifestCleanup, err := downloadParentManifest(ctx, spec)
	if err != nil {
		reason := catalog.BackupErrorParentManifestMissing

		return Result{
			Status:       catalog.BackupStatusChainBroken,
			ErrorReason:  &reason,
			ErrorMessage: fmt.Sprintf("download parent manifest: %v", err),
		}, nil
	}
	defer manifestCleanup()

	label := buildBackupLabel(spec.DatabaseName, spec.Backup.ID, start, "INCR")

	// Every codec attempt gets its own object key, and the row has to carry it
	// before the bytes leave, or a failed attempt leaves a file nothing names.
	mintAndSaveAttemptName := func() (string, error) {
		attemptName := buildObjectName(
			backupObjectName{DatabaseID: spec.SourceDB.DatabaseID, Label: label, AttemptID: uuid.New()},
		)

		spec.Backup.FileName = &attemptName
		if err := spec.Transactor.Transaction(context.WithoutCancel(ctx), func(ctx context.Context) error {
			if err := spec.Publication.Validate(ctx); err != nil {
				return err
			}
			return spec.BackupRepo.UpdateAttemptFileName(ctx, spec.Backup)
		}); err != nil {
			return "", fmt.Errorf("persist file_name at upload-start: %w", err)
		}

		return attemptName, nil
	}

	var result Result

	slotErr := WithBackupSlot(ctx, spec.SourceDB, spec.Logger, func() error {
		defer func() {
			if result.CompletedAt.IsZero() {
				result.CompletedAt = spec.Clock.Now()
			}
		}()

		streamResult, err := streamWithCodecFallback(ctx, streamAttemptSpec{
			Common:                  spec.CommonBackupSpec,
			BackupID:                spec.Backup.ID,
			Creds:                   creds,
			Label:                   label,
			SystemID:                systemID,
			IncrementalManifestPath: manifestPath,
			Classify:                classifyIncrStreamError,
			MintAndSaveAttemptName:  mintAndSaveAttemptName,
		})
		if err != nil {
			result = errorResult(catalog.BackupErrorPgBasebackupFailed,
				"pg_basebackup --incremental stream", err)
			return nil
		}

		if streamResult.Status != catalog.BackupStatusCompleted {
			result = recheckIncrementalStreamFailure(
				ctx,
				spec.CommonBackupSpec,
				rootFullTimelineID,
				streamResult,
			)
			return nil
		}

		streamResult.BackupDurationMs = spec.ExecutionClock.Now().Sub(executionStartedAt).Milliseconds()
		streamResult.CompletedAt = spec.Clock.Now()

		metadataReceipt, err := uploadIncrMetadata(
			ctx,
			spec.Logger, spec.FileStore, spec.StorageID, spec.SourceDB, spec.Backup, streamResult,
		)
		if err != nil {
			result = errorResult(catalog.BackupErrorStorageUploadFailed, "upload metadata", err)
			return nil
		}

		streamResult.Receipts = append(streamResult.Receipts, metadataReceipt)
		result = streamResult

		return nil
	})

	if slotErr != nil {
		return errorResult(catalog.BackupErrorNetworkFailure,
			"per-backup slot lifecycle", slotErr), nil
	}

	return result, nil
}

func verifyIncrTimelineCompatibility(
	ctx context.Context,
	spec IncrementalBackupSpec,
) (Result, int, bool) {
	rootFull, _, err := spec.BackupRepo.FindByID(ctx, *spec.Backup.RootFullBackupID)
	if err != nil {
		return errorResult(catalog.BackupErrorNetworkFailure, "load root FULL", err), 0, false
	}

	if rootFull == nil || rootFull.Status != catalog.BackupStatusCompleted {
		reason := catalog.BackupErrorParentManifestMissing

		return Result{
			Status:       catalog.BackupStatusChainBroken,
			ErrorReason:  &reason,
			ErrorMessage: "root FULL is missing or incomplete",
		}, 0, false
	}

	conn, err := pgphysical.NewSource(spec.SourceDB.Engine).OpenInspection(ctx)
	if err != nil {
		return errorResult(catalog.BackupErrorNetworkFailure,
			"open inspection connection", err), 0, false
	}
	defer func() { _ = conn.Close(ctx) }()

	decision, err := timelines.CheckIncrementalTimelineCompatibility(ctx, conn, spec.SourceDB, rootFull.TimelineID)
	if err != nil {
		return errorResult(catalog.BackupErrorNetworkFailure,
			"timeline compatibility check", err), 0, false
	}

	refusalResult, canProceed := timelineRefusalResult(decision, true)

	return refusalResult, rootFull.TimelineID, canProceed
}

// runSummarizerPreCheck opens an inspection connection and resolves the
// WAL-summarizer decision (including the bounded wait for a summarizer that has
// published no summary yet). It returns proceed=true to run the incremental, or
// a terminal result the caller must return verbatim:
//   - DecisionGoIncremental    → proceed
//   - DecisionRetryNextCadence → ERROR / SUMMARIZER_FALLING_BEHIND
//   - DecisionFullNewChain     → CHAIN_BROKEN / SUMMARIZER_OFF | SUMMARIES_EXPIRED
//   - ctx cancelled mid-wait   → CANCELED / CANCELED_BY_USER
func runSummarizerPreCheck(ctx context.Context, spec IncrementalBackupSpec) (Result, bool) {
	conn, err := pgphysical.NewSource(spec.SourceDB.Engine).OpenInspection(ctx)
	if err != nil {
		return errorResult(catalog.BackupErrorNetworkFailure,
			"open inspection conn for summarizer pre-check", err), false
	}
	defer func() { _ = conn.Close(context.WithoutCancel(ctx)) }()

	decision, err := resolveSummarizerDecision(
		ctx,
		conn,
		spec.ParentManifest.StopLSN,
		spec.IncrementalCadence,
		spec.ExecutionClock,
	)
	if err != nil {
		if ctx.Err() != nil {
			return canceledResult(catalog.BackupErrorCanceledByUser,
				"incremental cancelled during summarizer wait"), false
		}

		return errorResult(catalog.BackupErrorNetworkFailure, "summarizer pre-check", err), false
	}

	switch decision.Decision {
	case DecisionGoIncremental:
		return Result{}, true

	case DecisionRetryNextCadence, DecisionWait:
		return summarizerRefusedResult(
			catalog.BackupStatusError,
			catalog.BackupErrorSummarizerFallingBehind,
			"WAL summarizer is not publishing summaries (none yet, or its process is gone); retrying on the next cadence",
		), false

	case DecisionFullNewChain:
		return summarizerRefusedResult(
			catalog.BackupStatusChainBroken,
			*decision.Reason,
			"summarizer pre-check refused incremental; new FULL required",
		), false
	}

	panic("unknown summarizer decision")
}

func summarizerRefusedResult(
	status catalog.BackupStatus,
	reason catalog.BackupErrorReason,
	message string,
) Result {
	return Result{
		Status:       status,
		ErrorReason:  &reason,
		ErrorMessage: message,
	}
}

func canceledResult(
	reason catalog.BackupErrorReason,
	message string,
) Result {
	return Result{
		Status:       catalog.BackupStatusCanceled,
		ErrorReason:  &reason,
		ErrorMessage: message,
	}
}

func downloadParentManifest(
	ctx context.Context,
	spec IncrementalBackupSpec,
) (manifestPath string, cleanup func(), err error) {
	tmpDir, err := os.MkdirTemp(os.TempDir(), "pgincr_"+uuid.New().String())
	if err != nil {
		return "", func() {}, fmt.Errorf("create temp dir: %w", err)
	}

	cleanupAll := func() { _ = os.RemoveAll(tmpDir) }

	reader, err := spec.StorageReader.OpenFile(ctx, spec.ParentManifest.FileName)
	if err != nil {
		cleanupAll()

		return "", func() {}, fmt.Errorf("fetch parent manifest: %w", err)
	}
	defer func() { _ = reader.Close() }()

	decoded := io.Reader(reader)

	if spec.ParentManifest.Encryption == catalog.BackupEncryptionEncrypted {
		decoded, err = decryptParentManifest(reader, spec)
		if err != nil {
			cleanupAll()

			return "", func() {}, err
		}
	}

	manifestPath = filepath.Join(tmpDir, "backup_manifest")

	out, err := os.Create(manifestPath)
	if err != nil {
		cleanupAll()

		return "", func() {}, fmt.Errorf("create manifest temp file: %w", err)
	}

	if _, err := io.Copy(out, decoded); err != nil {
		_ = out.Close()
		cleanupAll()

		return "", func() {}, fmt.Errorf("copy manifest into temp file: %w", err)
	}

	if err := out.Close(); err != nil {
		cleanupAll()

		return "", func() {}, fmt.Errorf("close manifest temp file: %w", err)
	}

	return manifestPath, cleanupAll, nil
}

func decryptParentManifest(reader io.Reader, spec IncrementalBackupSpec) (io.Reader, error) {
	salt, err := base64.StdEncoding.DecodeString(spec.ParentManifest.Salt)
	if err != nil {
		return nil, fmt.Errorf("decode parent manifest salt: %w", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(spec.ParentManifest.IV)
	if err != nil {
		return nil, fmt.Errorf("decode parent manifest nonce: %w", err)
	}

	dec, err := crypto.NewDecryptingReader(
		reader, spec.MasterKey, spec.ParentManifest.BackupID, crypto.StreamHeader{Salt: salt, Nonce: nonce},
	)
	if err != nil {
		return nil, fmt.Errorf("init decryption reader: %w", err)
	}

	return dec, nil
}

// Verbatim messages from PostgreSQL's basebackup_incremental.c and
// walsummarizer.c, matched by substring because the timeline and LSN arguments
// are interpolated per call.
const (
	summariesMissingForRangeStderr = "WAL summaries are required on timeline"
	summarizerStalledStderr        = "WAL summarization is not progressing"
)

func classifyIncrStreamError(streamErr error, stderr []byte) streamOutcome {
	message := fmt.Sprintf("%v; stderr: %s", streamErr, truncateStderr(stderr))
	if strings.Contains(string(stderr), "could not parse backup manifest") {
		return outcomeManifestCorrupted(fmt.Errorf("%s", message), stderr)
	}

	if isSummariesExpiredError(stderr) {
		reason := catalog.BackupErrorSummariesExpired

		return streamOutcome{
			Status:       catalog.BackupStatusChainBroken,
			ErrorReason:  &reason,
			ErrorMessage: message,
		}
	}

	if isSummarizerStalledError(stderr) {
		reason := catalog.BackupErrorSummarizerFallingBehind

		return streamOutcome{
			Status:       catalog.BackupStatusError,
			ErrorReason:  &reason,
			ErrorMessage: message,
		}
	}

	reason := catalog.BackupErrorPgBasebackupFailed

	return streamOutcome{
		Status:       catalog.BackupStatusError,
		ErrorReason:  &reason,
		ErrorMessage: message,
	}
}

// isSummariesExpiredError detects the post-readiness-check race where the source cluster
// pruned WAL summaries between CheckSummarizerReadiness returning OK and pg_basebackup
// --incremental opening the actual range. The same message covers a range that was never
// summarized at all and one summarized with a hole in the middle.
func isSummariesExpiredError(stderr []byte) bool {
	return strings.Contains(string(stderr), summariesMissingForRangeStderr)
}

// isSummarizerStalledError detects a summarizer that stopped absorbing WAL.
// PostgreSQL waits for summarization through the backup start LSN and gives up
// only after a minute without a single absorbed record, so this is a real stall
// rather than a slow cluster.
func isSummarizerStalledError(stderr []byte) bool {
	return strings.Contains(string(stderr), summarizerStalledStderr)
}
