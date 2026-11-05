package backuping

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	files_utils "databasus-backend/internal/lib/filename"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/backups/internal/physical/timelines"
	"databasus-backend/internal/modules/storages"
)

func (s CommonBackupSpec) runStreamParams(
	fileName string,
	backupID uuid.UUID,
	codec catalog.BackupCompression,
) runStreamParams {
	return runStreamParams{
		FileStore:  s.FileStore,
		StorageID:  s.StorageID,
		Logger:     s.Logger,
		FileName:   fileName,
		Encryption: s.Encryption,
		MasterKey:  s.MasterKey,
		BackupID:   backupID,
		Source:     pgphysical.NewSource(s.SourceDB.Engine),
		Codec:      codec,
	}
}

type streamAttemptSpec struct {
	Common   CommonBackupSpec
	BackupID uuid.UUID
	Creds    *pgshared.CredentialTempFiles
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
) (Result, error) {
	var (
		settled         streamOutcome
		settledFileName string
	)

	codecs := pgphysical.GetCodecFallbackOrder()
	for i, codec := range codecs {
		fileName, err := spec.MintAndSaveAttemptName()
		if err != nil {
			return Result{}, err
		}

		settledFileName = fileName

		outcome, err := runStream(
			ctx,
			streamSpec{
				Params: spec.Common.runStreamParams(fileName, spec.BackupID, getStoredCodec(codec)),
				Backup: pgphysical.BaseBackupSpec{
					ClientTools:             spec.Common.ClientTools,
					Version:                 spec.Common.SourceDB.Version,
					Credentials:             spec.Creds,
					Label:                   spec.Label,
					Codec:                   codec,
					IncrementalManifestPath: spec.IncrementalManifestPath,
					SystemID:                spec.SystemID,
					Clock:                   spec.Common.ExecutionClock,
					FirstByteTimeout:        10 * time.Minute,
					StallTimeout:            60 * time.Second,
					StallPollInterval:       10 * time.Second,
					ProgressInterval:        5 * time.Second,
					ProgressListener:        spec.Common.ProgressListener,
					Logger:                  spec.Common.Logger,
				},
				Classify: spec.Classify,
			},
		)
		if err != nil {
			return Result{}, err
		}

		if outcome.isCompressionUnsupported {
			if i+1 < len(codecs) {
				spec.Common.Logger.WarnContext(
					ctx,
					"compression downgraded",
					slog.Any("requested_codec", codec),
					slog.Any("selected_codec", codecs[i+1]),
					slog.Any("backup_id", spec.BackupID),
				)

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
	err := common.Transactor.Transaction(ctx, func(ctx context.Context) error {
		return common.FileStore.RequestDiscardedFileDeletions(
			ctx,
			storages.DiscardedFiles{
				DatabaseID: common.Publication.Publication.DatabaseID,
				References: []storages.StoredFileReference{{StorageID: common.StorageID, FileName: fileName}},
			},
		)
	})
	if err != nil {
		common.Logger.ErrorContext(
			ctx,
			"failed to discard a rejected codec attempt",
			slog.Any("file_name", fileName),
			slog.Any("error", err),
		)
	}
}

func resultFromOutcome(outcome streamOutcome, fileName string) Result {
	return Result{
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
	reason catalog.BackupErrorReason,
	stage string,
	err error,
) Result {
	r := reason

	return Result{
		Status:       catalog.BackupStatusError,
		ErrorReason:  &r,
		ErrorMessage: fmt.Sprintf("%s: %v", stage, err),
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
type backupObjectName struct {
	DatabaseID uuid.UUID
	Label      string
	AttemptID  uuid.UUID
}

func buildObjectName(object backupObjectName) string {
	return object.DatabaseID.String() + "-" + object.Label + "-" + object.AttemptID.String()
}

func verifyFullTimelineCompatibility(
	ctx context.Context,
	common CommonBackupSpec,
) (refusal Result, liveTimelineID int, canProceed bool) {
	conn, err := pgphysical.NewSource(common.SourceDB.Engine).OpenInspection(ctx)
	if err != nil {
		return errorResult(catalog.BackupErrorNetworkFailure,
			"open inspection connection", err), 0, false
	}
	defer func() { _ = conn.Close(ctx) }()

	decision, err := timelines.CheckFullTimelineCompatibility(
		ctx,
		conn,
		common.SourceDB,
		common.BackupRepo,
		common.HistoryRepo,
	)
	if err != nil {
		return errorResult(catalog.BackupErrorNetworkFailure,
			"timeline compatibility check", err), 0, false
	}

	refusal, canProceed = timelineRefusalResult(decision, false)

	return refusal, decision.LiveTimelineID, canProceed
}

func timelineRefusalResult(decision *timelines.TimelineDecision, refuseNewerTimeline bool) (Result, bool) {
	switch decision.Kind {
	case timelines.TimelineContinue:
		return Result{}, true

	case timelines.TimelineFailoverDetected:
		if !refuseNewerTimeline {
			return Result{}, true
		}

		reason := catalog.BackupErrorTimelineSwitchDetected

		return Result{
			Status:      catalog.BackupStatusChainBroken,
			ErrorReason: &reason,
			ErrorMessage: fmt.Sprintf(
				"timeline switch detected: root FULL TL %d, live TL %d",
				decision.ExpectedTimelineID, decision.LiveTimelineID,
			),
		}, false

	case timelines.TimelineRegression:
		reason := catalog.BackupErrorTimelineRegression

		return Result{
			Status:      catalog.BackupStatusChainBroken,
			ErrorReason: &reason,
			ErrorMessage: fmt.Sprintf(
				"timeline regression: expected TL %d, live TL %d",
				decision.ExpectedTimelineID, decision.LiveTimelineID,
			),
		}, false

	case timelines.TimelineDifferentCluster:
		reason := catalog.BackupErrorSystemIdentifierMismatch

		return Result{
			Status:      catalog.BackupStatusChainBroken,
			ErrorReason: &reason,
			ErrorMessage: fmt.Sprintf(
				"system_identifier mismatch: catalog %s, live %s",
				decision.ExpectedSystemIdentifier, decision.LiveSystemIdentifier,
			),
		}, false
	}

	return Result{}, true
}

func recheckFullStreamFailure(
	ctx context.Context,
	common CommonBackupSpec,
	preflightTimelineID int,
	streamResult Result,
) Result {
	if ctx.Err() != nil || streamResult.Status == catalog.BackupStatusCanceled {
		return streamResult
	}

	decision, err := inspectTimelineAgainstExpected(ctx, common, preflightTimelineID)
	if err != nil {
		common.Logger.WarnContext(
			ctx,
			"cluster identity recheck failed after full stream failure",
			slog.Any("error", err),
		)

		return streamResult
	}

	return classifyFullStreamFailureAfterIdentity(streamResult, preflightTimelineID, decision)
}

func classifyFullStreamFailureAfterIdentity(
	streamResult Result,
	preflightTimelineID int,
	decision *timelines.TimelineDecision,
) Result {
	if decision.Kind == timelines.TimelineFailoverDetected {
		reason := catalog.BackupErrorFailoverDuringBackup

		return Result{
			Status:      catalog.BackupStatusError,
			ErrorReason: &reason,
			ErrorMessage: fmt.Sprintf(
				"timeline advanced during FULL: preflight TL %d, live TL %d",
				preflightTimelineID,
				decision.LiveTimelineID,
			),
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
	streamResult Result,
) Result {
	if ctx.Err() != nil || streamResult.Status == catalog.BackupStatusCanceled {
		return streamResult
	}

	decision, err := inspectTimelineAgainstExpected(ctx, common, rootFullTimelineID)
	if err != nil {
		common.Logger.WarnContext(
			ctx,
			"cluster identity recheck failed after incremental stream failure",
			slog.Any("error", err),
		)

		return streamResult
	}

	return classifyIncrementalStreamFailureAfterIdentity(streamResult, decision)
}

func classifyIncrementalStreamFailureAfterIdentity(
	streamResult Result,
	decision *timelines.TimelineDecision,
) Result {
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
) (*timelines.TimelineDecision, error) {
	if common.timelineProbe != nil {
		return common.timelineProbe(ctx, expectedTimelineID)
	}

	conn, err := pgphysical.NewSource(common.SourceDB.Engine).OpenInspection(ctx)
	if err != nil {
		return nil, fmt.Errorf("open inspection connection: %w", err)
	}
	defer func() { _ = conn.Close(context.WithoutCancel(ctx)) }()

	return timelines.CheckIncrementalTimelineCompatibility(ctx, conn, common.SourceDB, expectedTimelineID)
}

func getStoredCodec(codec pgphysical.Codec) catalog.BackupCompression {
	switch codec {
	case pgphysical.CodecZstd:
		return catalog.BackupCompressionZstd
	case pgphysical.CodecGzip:
		return catalog.BackupCompressionGzip
	case pgphysical.CodecNone:
		return catalog.BackupCompressionNone
	}
	panic("unknown physical backup codec")
}
