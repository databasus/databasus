package timelines

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/klauspost/compress/zstd"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/engines/postgresql/physical/walmath"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/storages"
)

type TimelineDecisionKind int

const (
	TimelineContinue TimelineDecisionKind = iota
	TimelineFailoverDetected
	TimelineRegression
	TimelineDifferentCluster
)

type TimelineDecision struct {
	Kind                     TimelineDecisionKind
	ExpectedTimelineID       int
	LiveTimelineID           int
	ExpectedSystemIdentifier string
	LiveSystemIdentifier     string
}

type timelineComparison struct {
	ExpectedSystemIdentifier *string
	LiveTimelineID           int
	LiveSystemIdentifier     string
	ExpectedTimelineID       int
}

func CheckFullTimelineCompatibility(
	ctx context.Context,
	conn *pgphysical.Inspection,
	db databases.PostgresqlPhysicalConnectionSettings,
	backupRepo *catalog.BackupRepository,
	historyRepo *catalog.WalHistoryRepository,
) (*TimelineDecision, error) {
	liveTimelineID, liveSystemIdentifier, err := conn.ReadLiveTimeline(ctx)
	if err != nil {
		return nil, err
	}

	decision := decideTimelineCompatibility(timelineComparison{
		ExpectedSystemIdentifier: db.SystemIdentifier,
		LiveTimelineID:           liveTimelineID,
		LiveSystemIdentifier:     liveSystemIdentifier,
	})
	if decision.Kind == TimelineDifferentCluster {
		return decision, nil
	}

	knownTimelineID, err := newestKnownTimeline(ctx, db.DatabaseID, backupRepo, historyRepo)
	if err != nil {
		return nil, err
	}

	return decideTimelineCompatibility(timelineComparison{
		ExpectedSystemIdentifier: db.SystemIdentifier,
		LiveTimelineID:           liveTimelineID,
		LiveSystemIdentifier:     liveSystemIdentifier,
		ExpectedTimelineID:       knownTimelineID,
	}), nil
}

func CheckIncrementalTimelineCompatibility(
	ctx context.Context,
	conn *pgphysical.Inspection,
	db databases.PostgresqlPhysicalConnectionSettings,
	rootFullTimelineID int,
) (*TimelineDecision, error) {
	liveTimelineID, liveSystemIdentifier, err := conn.ReadLiveTimeline(ctx)
	if err != nil {
		return nil, err
	}

	return decideTimelineCompatibility(timelineComparison{
		ExpectedSystemIdentifier: db.SystemIdentifier,
		LiveTimelineID:           liveTimelineID,
		LiveSystemIdentifier:     liveSystemIdentifier,
		ExpectedTimelineID:       rootFullTimelineID,
	}), nil
}

func decideTimelineCompatibility(comparison timelineComparison) *TimelineDecision {
	decision := &TimelineDecision{
		Kind:                     TimelineContinue,
		ExpectedTimelineID:       comparison.ExpectedTimelineID,
		LiveTimelineID:           comparison.LiveTimelineID,
		LiveSystemIdentifier:     comparison.LiveSystemIdentifier,
		ExpectedSystemIdentifier: "",
	}

	if comparison.ExpectedSystemIdentifier != nil {
		decision.ExpectedSystemIdentifier = *comparison.ExpectedSystemIdentifier
		if *comparison.ExpectedSystemIdentifier != comparison.LiveSystemIdentifier {
			decision.Kind = TimelineDifferentCluster

			return decision
		}
	}

	if comparison.ExpectedTimelineID == 0 {
		return decision
	}

	if comparison.LiveTimelineID < comparison.ExpectedTimelineID {
		decision.Kind = TimelineRegression

		return decision
	}

	if comparison.LiveTimelineID > comparison.ExpectedTimelineID {
		decision.Kind = TimelineFailoverDetected
	}

	return decision
}

// ValidateStartLsnAgainstHistory checks that startLSN falls inside the LSN
// range covered by timelineID per the catalog's .history files. Skipped
// when no history rows exist (first FULL on a fresh DB). OK_WITH_WARNING
// when history is gapped (TL N+1 present but N missing); the caller logs
// timeline_history_gap. CHAIN_BROKEN means the FULL is on a stale or
// rewound timeline whose history moved on — refuse the artifact.
func ValidateStartLsnAgainstHistory(
	ctx context.Context,
	dbID uuid.UUID,
	timelineID int,
	startLSN walmath.LSN,
	historyRepo *catalog.WalHistoryRepository,
) (ValidationResult, error) {
	historyRows, err := historyRepo.FindAllByDatabase(ctx, dbID)
	if err != nil {
		return ValidationResult{}, fmt.Errorf("load wal history rows: %w", err)
	}

	if len(historyRows) == 0 {
		return ValidationResult{Status: ValidationStatusOK}, nil
	}

	if !historyHasTimeline(historyRows, timelineID) && timelineID > 1 {
		if historyIsGapped(historyRows, timelineID) {
			return ValidationResult{
				Status: ValidationStatusOKWithWarning,
				Message: fmt.Sprintf(
					"timeline_history_gap: TL %d has no history row but higher TLs do; older history likely retention-deleted",
					timelineID,
				),
			}, nil
		}

		return ValidationResult{
			Status: ValidationStatusChainBroken,
			Message: fmt.Sprintf(
				"start_lsn %s outside known timeline range: no history record for TL %d",
				startLSN.String(),
				timelineID,
			),
		}, nil
	}

	return ValidationResult{Status: ValidationStatusOK}, nil
}

type HistoryUploadSpec struct {
	Publication config.PublicationFence
	Inspection  *pgphysical.Inspection
	TimelineID  int
	FileStore   storages.Module
	Clock       clock.Source
	Transactor  *database.Transactor
	SourceDB    databases.PostgresqlPhysicalConnectionSettings
	StorageID   uuid.UUID
	HistoryRepo *catalog.WalHistoryRepository
	Encryption  catalog.BackupEncryption
	MasterKey   *crypto.MasterKey
	Logger      *slog.Logger
}

// Idempotent on (database_id, timeline_id) through the UNIQUE constraint: a
// duplicate insert returns the existing row instead of an error.
func UploadHistoryFile(
	ctx context.Context,
	spec HistoryUploadSpec,
) (*catalog.WalHistoryFile, error) {
	conn := spec.Inspection
	timelineID := spec.TimelineID
	historyRepo := spec.HistoryRepo
	encryption := spec.Encryption
	masterKey := spec.MasterKey
	logger := spec.Logger
	storageID := spec.StorageID

	databaseID := spec.SourceDB.DatabaseID

	existing, _, err := historyRepo.FindByDatabaseTimeline(ctx, databaseID, timelineID)
	if err != nil {
		return nil, fmt.Errorf("query existing history row: %w", err)
	}

	if existing != nil {
		logger.DebugContext(
			ctx,
			"history file already in catalog",
			slog.Any("database_id", databaseID),
			slog.Any("timeline_id", timelineID),
		)

		return existing, nil
	}

	historyFilename := walmath.FormatHistoryFilename(uint32(timelineID))

	body, err := conn.GetTimelineHistory(ctx, historyFilename)
	if err != nil {
		return nil, err
	}

	historyFileID := uuid.New()
	// The row's own identity is the per-attempt component: a retry for the same
	// timeline writes a different object than the one a pending cleanup owns.
	storageObjectName := fmt.Sprintf("%s-HIST-tl%d-%s.history.zst", databaseID, timelineID, historyFileID)

	artifactReader, encryptionSalt, encryptionIV, err := buildHistoryArtifactReader(
		body, encryption, masterKey, historyFileID,
	)
	if err != nil {
		return nil, err
	}

	artifactReference := storages.StoredFileReference{StorageID: storageID, FileName: storageObjectName}

	artifactReceipt, err := spec.FileStore.WriteFile(ctx, artifactReference, artifactReader)
	if err != nil {
		return nil, fmt.Errorf("upload history artifact: %w", err)
	}

	compressedSizeBytes := int64(artifactReader.Len())

	sidecarFilename := storageObjectName + catalog.MetadataSuffix

	sidecar := catalog.WalHistoryMetadata{
		WalHistoryFileID:    historyFileID,
		DatabaseID:          databaseID,
		TimelineID:          timelineID,
		HistoryFilename:     historyFilename,
		CompressedSizeBytes: compressedSizeBytes,
		Encryption:          encryption,
		EncryptionSalt:      encryptionSalt,
		EncryptionIV:        encryptionIV,
		CreatedAt:           spec.Clock.Now(),
	}

	sidecarBytes, err := json.Marshal(sidecar)
	if err != nil {
		return nil, fmt.Errorf("marshal history sidecar: %w", err)
	}

	sidecarReference := storages.StoredFileReference{StorageID: storageID, FileName: sidecarFilename}

	sidecarReceipt, err := spec.FileStore.WriteFile(ctx, sidecarReference, bytes.NewReader(sidecarBytes))
	if err != nil {
		// An artifact with no sidecar cannot be restored from, so the attempt gives
		// the artifact back rather than leaving half a history file behind.
		if discardErr := spec.Transactor.Transaction(ctx, func(ctx context.Context) error {
			return spec.FileStore.RequestFileDeletions(
				ctx, []storages.StoredFileReference{artifactReference})
		}); discardErr != nil {
			logger.WarnContext(
				ctx,
				"failed to discard a history artifact after its sidecar failed",
				slog.Any("file_name", storageObjectName),
				slog.Any("error", discardErr),
			)
		}

		return nil, fmt.Errorf("upload history sidecar: %w", err)
	}

	row := &catalog.WalHistoryFile{
		ID:               historyFileID,
		DatabaseID:       databaseID,
		StorageID:        storageID,
		TimelineID:       timelineID,
		FileName:         storageObjectName,
		HistoryFilename:  historyFilename,
		CompressedSizeMb: float64(compressedSizeBytes) / (1024 * 1024),
		CreatedAt:        sidecar.CreatedAt,
	}

	err = spec.Transactor.Transaction(ctx, func(ctx context.Context) error {
		if err := spec.Publication.Validate(ctx); err != nil {
			return err
		}
		if insertErr := historyRepo.Insert(ctx, row, spec.Clock.Now()); insertErr != nil {
			return insertErr
		}

		return spec.FileStore.ConfirmFileWrites(ctx, []storages.WriteReceipt{
			artifactReceipt, sidecarReceipt,
		})
	})
	if err != nil {
		if isUniqueViolation(err) {
			logger.DebugContext(
				ctx,
				"history row inserted by concurrent caller",
				slog.Any("database_id", databaseID),
				slog.Any("timeline_id", timelineID),
			)

			existing, _, lookupErr := historyRepo.FindByDatabaseTimeline(ctx, databaseID, timelineID)
			return existing, lookupErr
		}

		return nil, fmt.Errorf("insert history row: %w", err)
	}

	return row, nil
}

func newestKnownTimeline(
	ctx context.Context,
	dbID uuid.UUID,
	backupRepo *catalog.BackupRepository,
	historyRepo *catalog.WalHistoryRepository,
) (int, error) {
	fulls, err := backupRepo.FindCompletedFullNewestFirstByDatabase(ctx, dbID)
	if err != nil {
		return 0, fmt.Errorf("load full backups: %w", err)
	}

	historyRows, err := historyRepo.FindAllByDatabase(ctx, dbID)
	if err != nil {
		return 0, fmt.Errorf("load history rows: %w", err)
	}

	maxTL := 0
	for _, full := range fulls {
		if full.TimelineID > maxTL {
			maxTL = full.TimelineID
		}
	}

	for _, row := range historyRows {
		if row.TimelineID > maxTL {
			maxTL = row.TimelineID
		}
	}

	return maxTL, nil
}

func historyHasTimeline(rows []*catalog.WalHistoryFile, timelineID int) bool {
	for _, row := range rows {
		if row.TimelineID == timelineID {
			return true
		}
	}

	return false
}

func historyIsGapped(rows []*catalog.WalHistoryFile, timelineID int) bool {
	for _, row := range rows {
		if row.TimelineID > timelineID {
			return true
		}
	}

	return false
}

// limitedBuffer is bytes.Buffer plus a Len() accessor for size reporting.
// We compress the entire history file in memory because .history files are
// small (~1 KB even for clusters with many promotions) — the streaming
// pipeline used for FULL artifacts would be overkill here.
type limitedBuffer struct {
	*bytes.Reader
	length int
}

func (b *limitedBuffer) Len() int {
	return b.length
}

func buildHistoryArtifactReader(
	body []byte,
	encryption catalog.BackupEncryption,
	masterKey *crypto.MasterKey,
	historyFileID uuid.UUID,
) (*limitedBuffer, string, string, error) {
	compressed, err := compressZstd(body)
	if err != nil {
		return nil, "", "", err
	}

	if encryption != catalog.BackupEncryptionEncrypted {
		return &limitedBuffer{
			Reader: bytes.NewReader(compressed),
			length: len(compressed),
		}, "", "", nil
	}

	var encBuf bytes.Buffer

	encryptingWriter, header, err := crypto.NewEncryptingWriter(&encBuf, masterKey, historyFileID)
	if err != nil {
		return nil, "", "", fmt.Errorf("setup history encryption: %w", err)
	}

	if _, err := encryptingWriter.Write(compressed); err != nil {
		return nil, "", "", fmt.Errorf("encrypt history body: %w", err)
	}

	if err := encryptingWriter.Close(); err != nil {
		return nil, "", "", fmt.Errorf("close history encryption writer: %w", err)
	}

	encryptedBytes := encBuf.Bytes()

	return &limitedBuffer{
		Reader: bytes.NewReader(encryptedBytes),
		length: len(encryptedBytes),
	}, base64.StdEncoding.EncodeToString(header.Salt), base64.StdEncoding.EncodeToString(header.Nonce), nil
}

func compressZstd(input []byte) ([]byte, error) {
	var buf bytes.Buffer

	encoder, err := zstd.NewWriter(&buf, zstd.WithEncoderLevel(zstd.SpeedDefault))
	if err != nil {
		return nil, fmt.Errorf("init zstd writer: %w", err)
	}

	if _, err := io.Copy(encoder, bytes.NewReader(input)); err != nil {
		_ = encoder.Close()

		return nil, fmt.Errorf("zstd encode: %w", err)
	}

	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("close zstd writer: %w", err)
	}

	return buf.Bytes(), nil
}

func isUniqueViolation(err error) bool {
	var postgresError *pgconn.PgError
	return errors.As(err, &postgresError) && postgresError.Code == "23505"
}

type ValidationStatus string

const (
	ValidationStatusOK            ValidationStatus = "OK"
	ValidationStatusOKWithWarning ValidationStatus = "OK_WITH_WARNING"
	ValidationStatusChainBroken   ValidationStatus = "CHAIN_BROKEN"
)

type ValidationResult struct {
	Status  ValidationStatus
	Message string
}
