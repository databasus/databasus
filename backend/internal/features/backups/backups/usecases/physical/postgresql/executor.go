package usecases_physical_postgresql

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/google/uuid"

	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	postgresql_shared "databasus-backend/internal/features/databases/databases/postgresql/shared"
	storage_files "databasus-backend/internal/features/storages/files"
	db "databasus-backend/internal/storage"
	files_utils "databasus-backend/internal/util/files"
	"databasus-backend/internal/util/tools"
)

// runStreamParams builds the codec-independent parameters runStream needs from
// the shared spec. backupID and systemID come from the variant-specific row and
// the live timeline check, so they are passed explicitly rather than read off the spec.
func (s CommonBackupSpec) runStreamParams(
	fileName string,
	backupID uuid.UUID,
	systemID uint64,
	codec physical_enums.PhysicalBackupCompression,
) runStreamParams {
	return runStreamParams{
		FileStore:        s.FileStore,
		StorageID:        s.StorageID,
		FieldEncryptor:   s.FieldEncryptor,
		Logger:           s.Logger,
		FileName:         fileName,
		Encryption:       s.Encryption,
		MasterKey:        s.MasterKey,
		BackupID:         backupID,
		SystemID:         systemID,
		Codec:            codec,
		ProgressListener: s.ProgressListener,
	}
}

type streamAttemptSpec struct {
	Common   CommonBackupSpec
	BackupID uuid.UUID
	Creds    *postgresql_shared.CredentialTempFiles
	Label    string
	SystemID uint64
	// "" for a FULL; a non-empty path is the downloaded parent-manifest temp file
	// that makes this an INCR (--incremental=<path>).
	IncrementalManifestPath string
	Classify                streamErrorClassifier

	// MintAndSaveAttemptName produces the object key for the next attempt and
	// persists it on the backup row. The loop owns neither the naming inputs nor
	// the repository, and the two effects cannot be separated: a name the row does
	// not carry is a name nothing can clean up.
	MintAndSaveAttemptName func() (string, error)
}

// The ZSTD -> GZIP -> NONE fallback runs INSIDE the per-backup replication slot,
// so the slot is held across every attempt; only the --compress flag, the recorded
// codec and the object key differ between them. A rejected attempt leaves an object
// of its own for cleanup rather than being overwritten.
func streamWithCodecFallback(
	ctx context.Context,
	spec streamAttemptSpec,
) (PhysicalBackupResult, error) {
	pgBin := tools.GetPostgresqlExecutable(spec.Common.SourceDB.Version, tools.PostgresqlExecutablePgBasebackup)

	var (
		settled         streamOutcome
		settledFileName string
	)

	for i, codec := range codecFallbackOrder {
		fileName, err := spec.MintAndSaveAttemptName()
		if err != nil {
			return PhysicalBackupResult{}, err
		}

		settledFileName = fileName

		buildCmd := func(streamCtx context.Context) (*exec.Cmd, error) {
			return newPgBasebackupCommand(
				streamCtx,
				pgBin,
				spec.Common.SourceDB,
				spec.Creds,
				spec.Label,
				codec,
				spec.IncrementalManifestPath,
			)
		}

		outcome, err := runStream(
			ctx,
			spec.Common.runStreamParams(fileName, spec.BackupID, spec.SystemID, codec),
			buildCmd,
			spec.Classify,
		)
		if err != nil {
			return PhysicalBackupResult{}, err
		}

		if outcome.isCompressionUnsupported {
			if i+1 < len(codecFallbackOrder) {
				spec.Common.Logger.Warn(
					fmt.Sprintf("compression downgraded: %s -> %s", codec, codecFallbackOrder[i+1]),
					"backup_id", spec.BackupID)

				// The next attempt writes a different name, so this one's bytes are
				// owned by nobody. Handing them back now rather than letting the
				// commit window expire keeps the rejected attempt from sitting in
				// the user's storage for as long as a healthy upload would.
				discardAttemptFiles(ctx, spec.Common, fileName)

				continue
			}

			settled = compressionExhaustedOutcome(outcome.Stderr)

			break
		}

		settled = outcome

		break
	}

	return resultFromOutcome(settled, settledFileName), nil
}

func discardAttemptFiles(ctx context.Context, common CommonBackupSpec, fileName string) {
	err := db.GetTransactor().Transaction(ctx, func(ctx context.Context) error {
		return common.FileStore.RequestFileDeletions(ctx, []storage_files.StoredFileReference{
			{StorageID: common.StorageID, FileName: fileName},
		})
	})
	if err != nil {
		common.Logger.ErrorContext(ctx, "failed to discard a rejected codec attempt",
			"file_name", fileName, "error", err)
	}
}

func resultFromOutcome(outcome streamOutcome, fileName string) PhysicalBackupResult {
	return PhysicalBackupResult{
		Status:                 outcome.Status,
		ErrorReason:            outcome.ErrorReason,
		ErrorMessage:           outcome.ErrorMessage,
		Receipts:               outcome.Receipts,
		FileName:               fileName,
		TimelineID:             outcome.TimelineID,
		StartLSN:               outcome.StartLSN,
		StopLSN:                outcome.StopLSN,
		BackupSizeMb:           outcome.BackupSizeMb,
		EncryptionAlgo:         outcome.EncryptionAlgo,
		EncryptionSalt:         outcome.EncryptionSalt,
		EncryptionIV:           outcome.EncryptionIV,
		Compression:            outcome.Compression,
		ManifestFileName:       outcome.ManifestFileName,
		ManifestEncryptionSalt: outcome.ManifestEncryptionSalt,
		ManifestEncryptionIV:   outcome.ManifestEncryptionIV,
	}
}

// errorResult is the terminal result for a pre-stream failure (credential setup,
// timeline-check plumbing, slot lifecycle). stage names the step for the persisted
// message.
func errorResult(
	reason physical_enums.PhysicalBackupErrorReason,
	stage string,
	err error,
) PhysicalBackupResult {
	r := reason

	return PhysicalBackupResult{
		Status:       physical_enums.PhysicalBackupStatusError,
		ErrorReason:  &r,
		ErrorMessage: fmt.Sprintf("%s: %v", stage, err),
		CompletedAt:  time.Now().UTC(),
	}
}

// buildBackupLabel names the backup, not the file it produces:
// "<dbName>-<kind>-<timestamp>-<backupID>", kind being FULL or INCR, sanitized for
// storage portability the same way logical backups are. It stays the same across
// codec attempts, so pg_basebackup reports one label for the whole backup.
func buildBackupLabel(
	databaseName string,
	backupID uuid.UUID,
	now time.Time,
	kind string,
) string {
	return fmt.Sprintf("%s-%s-%s-%s",
		files_utils.SanitizeFilename(databaseName),
		kind,
		now.Format("20060102-150405"),
		backupID.String(),
	)
}

// Each codec attempt writes its own object, because a name whose cleanup is
// pending cannot be written again. The codec is recorded on the row, never in the
// key, so the name stays extension-less.
func buildObjectName(label string, attemptID uuid.UUID) string {
	return label + "-" + attemptID.String()
}

func verifyFullTimelineCompatibility(
	ctx context.Context,
	common CommonBackupSpec,
) (refusal PhysicalBackupResult, liveTimelineID int, canProceed bool) {
	conn, err := common.SourceDB.OpenInspectionConn(ctx, common.FieldEncryptor)
	if err != nil {
		return errorResult(physical_enums.PhysicalBackupErrorNetworkFailure,
			"open inspection connection", err), 0, false
	}
	defer func() { _ = conn.Close(ctx) }()

	decision, err := CheckFullTimelineCompatibility(ctx, conn, common.SourceDB, common.FullRepo, common.HistoryRepo)
	if err != nil {
		return errorResult(physical_enums.PhysicalBackupErrorNetworkFailure,
			"timeline compatibility check", err), 0, false
	}

	refusal, canProceed = timelineRefusalResult(decision, false)

	return refusal, decision.LiveTimelineID, canProceed
}

func timelineRefusalResult(decision *TimelineDecision, refuseNewerTimeline bool) (PhysicalBackupResult, bool) {
	switch decision.Kind {
	case TimelineContinue:
		return PhysicalBackupResult{}, true

	case TimelineFailoverDetected:
		if !refuseNewerTimeline {
			return PhysicalBackupResult{}, true
		}

		reason := physical_enums.PhysicalBackupErrorTimelineSwitchDetected

		return PhysicalBackupResult{
			Status:      physical_enums.PhysicalBackupStatusChainBroken,
			ErrorReason: &reason,
			ErrorMessage: fmt.Sprintf(
				"timeline switch detected: root FULL TL %d, live TL %d",
				decision.ExpectedTimelineID, decision.LiveTimelineID,
			),
			CompletedAt: time.Now().UTC(),
		}, false

	case TimelineRegression:
		reason := physical_enums.PhysicalBackupErrorTimelineRegression

		return PhysicalBackupResult{
			Status:      physical_enums.PhysicalBackupStatusChainBroken,
			ErrorReason: &reason,
			ErrorMessage: fmt.Sprintf(
				"timeline regression: expected TL %d, live TL %d",
				decision.ExpectedTimelineID, decision.LiveTimelineID,
			),
			CompletedAt: time.Now().UTC(),
		}, false

	case TimelineDifferentCluster:
		reason := physical_enums.PhysicalBackupErrorSystemIdentifierMismatch

		return PhysicalBackupResult{
			Status:      physical_enums.PhysicalBackupStatusChainBroken,
			ErrorReason: &reason,
			ErrorMessage: fmt.Sprintf(
				"system_identifier mismatch: catalog %s, live %s",
				decision.ExpectedSystemIdentifier, decision.LiveSystemIdentifier,
			),
			CompletedAt: time.Now().UTC(),
		}, false
	}

	return PhysicalBackupResult{}, true
}

func recheckFullStreamFailure(
	ctx context.Context,
	common CommonBackupSpec,
	preflightTimelineID int,
	streamResult PhysicalBackupResult,
) PhysicalBackupResult {
	if ctx.Err() != nil || streamResult.Status == physical_enums.PhysicalBackupStatusCanceled {
		return streamResult
	}

	decision, err := inspectTimelineAgainstExpected(ctx, common, preflightTimelineID)
	if err != nil {
		common.Logger.WarnContext(ctx, "cluster identity recheck failed after FULL stream failure", "error", err)

		return streamResult
	}

	return classifyFullStreamFailureAfterIdentity(streamResult, preflightTimelineID, decision)
}

func classifyFullStreamFailureAfterIdentity(
	streamResult PhysicalBackupResult,
	preflightTimelineID int,
	decision *TimelineDecision,
) PhysicalBackupResult {
	if decision.Kind == TimelineFailoverDetected {
		reason := physical_enums.PhysicalBackupErrorFailoverDuringBackup

		return PhysicalBackupResult{
			Status:      physical_enums.PhysicalBackupStatusError,
			ErrorReason: &reason,
			ErrorMessage: fmt.Sprintf(
				"timeline advanced during FULL: preflight TL %d, live TL %d",
				preflightTimelineID,
				decision.LiveTimelineID,
			),
			CompletedAt: time.Now().UTC(),
		}
	}

	refusal, canProceed := timelineRefusalResult(decision, false)
	if !canProceed {
		return refusal
	}

	return streamResult
}

func recheckIncrementalStreamFailure(
	ctx context.Context,
	common CommonBackupSpec,
	rootFullTimelineID int,
	streamResult PhysicalBackupResult,
) PhysicalBackupResult {
	if ctx.Err() != nil || streamResult.Status == physical_enums.PhysicalBackupStatusCanceled {
		return streamResult
	}

	decision, err := inspectTimelineAgainstExpected(ctx, common, rootFullTimelineID)
	if err != nil {
		common.Logger.WarnContext(ctx, "cluster identity recheck failed after incremental stream failure", "error", err)

		return streamResult
	}

	return classifyIncrementalStreamFailureAfterIdentity(streamResult, decision)
}

func classifyIncrementalStreamFailureAfterIdentity(
	streamResult PhysicalBackupResult,
	decision *TimelineDecision,
) PhysicalBackupResult {
	refusal, canProceed := timelineRefusalResult(decision, true)
	if !canProceed {
		return refusal
	}

	return streamResult
}

func inspectTimelineAgainstExpected(
	ctx context.Context,
	common CommonBackupSpec,
	expectedTimelineID int,
) (*TimelineDecision, error) {
	if common.timelineProbe != nil {
		return common.timelineProbe(ctx, expectedTimelineID)
	}

	conn, err := common.SourceDB.OpenInspectionConn(ctx, common.FieldEncryptor)
	if err != nil {
		return nil, fmt.Errorf("open inspection connection: %w", err)
	}
	defer func() { _ = conn.Close(context.Background()) }()

	return CheckIncrementalTimelineCompatibility(ctx, conn, common.SourceDB, expectedTimelineID)
}
