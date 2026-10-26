package usecases_physical_postgresql

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/klauspost/compress/zstd"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	chain_view "databasus-backend/internal/features/backups/backups/core/physical/chain_view"
	physical_dto "databasus-backend/internal/features/backups/backups/core/physical/dto"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	backup_encryption "databasus-backend/internal/features/backups/backups/encryption"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/storages"
	db "databasus-backend/internal/storage"
	"databasus-backend/internal/util/walmath"
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
	conn *pgx.Conn,
	db databases.PostgresqlPhysicalConnectionSettings,
	fullRepo *physical_repositories.PhysicalFullBackupRepository,
	historyRepo *physical_repositories.PhysicalWalHistoryRepository,
) (*TimelineDecision, error) {
	liveTimelineID, liveSystemIdentifier, err := readClusterIdentity(ctx, conn)
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

	knownTimelineID, err := newestKnownTimeline(db.DatabaseID, fullRepo, historyRepo)
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
	conn *pgx.Conn,
	db databases.PostgresqlPhysicalConnectionSettings,
	rootFullTimelineID int,
) (*TimelineDecision, error) {
	liveTimelineID, liveSystemIdentifier, err := readClusterIdentity(ctx, conn)
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
	dbID uuid.UUID,
	timelineID int,
	startLSN walmath.LSN,
	historyRepo *physical_repositories.PhysicalWalHistoryRepository,
) (chain_view.ValidationResult, error) {
	historyRows, err := historyRepo.FindAllByDatabase(dbID)
	if err != nil {
		return chain_view.ValidationResult{}, fmt.Errorf("load wal history rows: %w", err)
	}

	if len(historyRows) == 0 {
		return chain_view.ValidationResult{Status: chain_view.ValidationStatusOK}, nil
	}

	if !historyHasTimeline(historyRows, timelineID) && timelineID > 1 {
		if historyIsGapped(historyRows, timelineID) {
			return chain_view.ValidationResult{
				Status: chain_view.ValidationStatusOKWithWarning,
				Message: fmt.Sprintf(
					"timeline_history_gap: TL %d has no history row but higher TLs do; older history likely retention-deleted",
					timelineID,
				),
			}, nil
		}

		return chain_view.ValidationResult{
			Status: chain_view.ValidationStatusChainBroken,
			Message: fmt.Sprintf(
				"start_lsn %s outside known timeline range: no history record for TL %d",
				startLSN.String(),
				timelineID,
			),
		}, nil
	}

	return chain_view.ValidationResult{Status: chain_view.ValidationStatusOK}, nil
}

type HistoryUploadSpec struct {
	Conn        *pgx.Conn
	TimelineID  int
	FileStore   FileStore
	SourceDB    databases.PostgresqlPhysicalConnectionSettings
	StorageID   uuid.UUID
	HistoryRepo *physical_repositories.PhysicalWalHistoryRepository
	Encryption  backups_core_enums.BackupEncryption
	MasterKey   string
	Logger      *slog.Logger
}

// Idempotent on (database_id, timeline_id) through the UNIQUE constraint: a
// duplicate insert returns the existing row instead of an error.
func UploadHistoryFile(
	ctx context.Context,
	spec HistoryUploadSpec,
) (*physical_models.PhysicalWalHistoryFile, error) {
	conn := spec.Conn
	timelineID := spec.TimelineID
	historyRepo := spec.HistoryRepo
	encryption := spec.Encryption
	masterKey := spec.MasterKey
	logger := spec.Logger
	storageID := spec.StorageID

	databaseID := spec.SourceDB.DatabaseID

	existing, err := historyRepo.FindByDatabaseTimeline(databaseID, timelineID)
	if err != nil {
		return nil, fmt.Errorf("query existing history row: %w", err)
	}

	if existing != nil {
		logger.DebugContext(ctx, "history file already in catalog",
			"database_id", databaseID,
			"timeline_id", timelineID)

		return existing, nil
	}

	historyFilename := walmath.FormatHistoryFilename(uint32(timelineID))

	body, err := readHistoryFromCluster(ctx, conn, historyFilename)
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

	sidecarFilename := storageObjectName + metadataSuffix

	sidecar := physical_dto.PhysicalWalHistoryMetadata{
		WalHistoryFileID:    historyFileID,
		DatabaseID:          databaseID,
		TimelineID:          timelineID,
		HistoryFilename:     historyFilename,
		CompressedSizeBytes: compressedSizeBytes,
		Encryption:          encryption,
		EncryptionSalt:      encryptionSalt,
		EncryptionIV:        encryptionIV,
		CreatedAt:           time.Now().UTC(),
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
		if discardErr := db.GetTransactor().Transaction(ctx, func(ctx context.Context) error {
			return spec.FileStore.RequestFileDeletions(
				ctx, []storages.StoredFileReference{artifactReference})
		}); discardErr != nil {
			logger.WarnContext(ctx, "failed to discard a history artifact after its sidecar failed",
				"file_name", storageObjectName,
				"error", discardErr)
		}

		return nil, fmt.Errorf("upload history sidecar: %w", err)
	}

	row := &physical_models.PhysicalWalHistoryFile{
		ID:               historyFileID,
		DatabaseID:       databaseID,
		StorageID:        storageID,
		TimelineID:       timelineID,
		FileName:         storageObjectName,
		HistoryFilename:  historyFilename,
		CompressedSizeMb: float64(compressedSizeBytes) / (1024 * 1024),
		CreatedAt:        sidecar.CreatedAt,
	}

	err = db.GetTransactor().Transaction(ctx, func(ctx context.Context) error {
		tx := database.Conn(ctx, db.GetDb())

		if insertErr := historyRepo.InsertInTransaction(tx, row); insertErr != nil {
			return insertErr
		}

		return spec.FileStore.ConfirmFileWrites(ctx, []storages.WriteReceipt{
			artifactReceipt, sidecarReceipt,
		})
	})
	if err != nil {
		if isUniqueViolation(err) {
			logger.DebugContext(ctx, "history row inserted by concurrent caller",
				"database_id", databaseID,
				"timeline_id", timelineID)

			return historyRepo.FindByDatabaseTimeline(databaseID, timelineID)
		}

		return nil, fmt.Errorf("insert history row: %w", err)
	}

	return row, nil
}

type clusterIdentitySources struct {
	// NULL during recovery, where pg_current_wal_lsn() cannot be called.
	CurrentWalFilename    *string
	ControlFileTimelineID int
	SystemIdentifier      string
}

func readClusterIdentitySources(ctx context.Context, conn *pgx.Conn) (clusterIdentitySources, error) {
	var sources clusterIdentitySources

	err := conn.QueryRow(ctx, `
		SELECT
			CASE WHEN pg_is_in_recovery() THEN NULL ELSE pg_walfile_name(pg_current_wal_lsn()) END,
			(SELECT timeline_id FROM pg_control_checkpoint()),
			(SELECT system_identifier::text FROM pg_control_system())
	`).Scan(
		&sources.CurrentWalFilename,
		&sources.ControlFileTimelineID,
		&sources.SystemIdentifier,
	)
	if err != nil {
		return clusterIdentitySources{}, fmt.Errorf("read cluster identity: %w", err)
	}

	return sources, nil
}

func readClusterIdentity(ctx context.Context, conn *pgx.Conn) (int, string, error) {
	sources, err := readClusterIdentitySources(ctx, conn)
	if err != nil {
		return 0, "", err
	}

	timelineID, err := resolveLiveTimelineID(sources)
	if err != nil {
		return 0, "", err
	}

	return timelineID, sources.SystemIdentifier, nil
}

// Every cluster answers the control file, but it lags a promotion: PostgreSQL stamps it
// at checkpoints, and the checkpoint a promotion requests is spread across
// checkpoint_completion_target, so a promoted primary keeps naming the pre-failover
// timeline for minutes. A stale reading lets an incremental extend a chain anchored to a
// timeline the cluster has already left.
//
// A source still in recovery has no fresher reading to offer: pg_walfile_name() refuses
// to run there, and pg_stat_wal_receiver hides received_tli from a role without
// pg_read_all_stats, which the replication user this product provisions does not have.
func resolveLiveTimelineID(sources clusterIdentitySources) (int, error) {
	if sources.CurrentWalFilename != nil {
		timelineID, err := walmath.ParseWALFilenameTimeline(*sources.CurrentWalFilename)
		if err != nil {
			return 0, fmt.Errorf("read timeline from current WAL file name: %w", err)
		}

		return int(timelineID), nil
	}

	return sources.ControlFileTimelineID, nil
}

func newestKnownTimeline(
	dbID uuid.UUID,
	fullRepo *physical_repositories.PhysicalFullBackupRepository,
	historyRepo *physical_repositories.PhysicalWalHistoryRepository,
) (int, error) {
	fulls, err := fullRepo.FindCompletedNewestFirstByDatabase(dbID)
	if err != nil {
		return 0, fmt.Errorf("load full backups: %w", err)
	}

	historyRows, err := historyRepo.FindAllByDatabase(dbID)
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

func historyHasTimeline(rows []*physical_models.PhysicalWalHistoryFile, timelineID int) bool {
	for _, row := range rows {
		if row.TimelineID == timelineID {
			return true
		}
	}

	return false
}

func historyIsGapped(rows []*physical_models.PhysicalWalHistoryFile, timelineID int) bool {
	for _, row := range rows {
		if row.TimelineID > timelineID {
			return true
		}
	}

	return false
}

func readHistoryFromCluster(ctx context.Context, conn *pgx.Conn, historyFilename string) ([]byte, error) {
	var body []byte

	err := conn.QueryRow(
		ctx,
		`SELECT pg_read_binary_file('pg_wal/' || $1)`,
		historyFilename,
	).Scan(&body)
	if err != nil {
		return nil, fmt.Errorf("read history file %q from cluster: %w", historyFilename, err)
	}

	return body, nil
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
	encryption backups_core_enums.BackupEncryption,
	masterKey string,
	historyFileID uuid.UUID,
) (*limitedBuffer, string, string, error) {
	compressed, err := compressZstd(body)
	if err != nil {
		return nil, "", "", err
	}

	if encryption != backups_core_enums.BackupEncryptionEncrypted {
		return &limitedBuffer{
			Reader: bytes.NewReader(compressed),
			length: len(compressed),
		}, "", "", nil
	}

	var encBuf bytes.Buffer

	encSetup, err := backup_encryption.SetupEncryptionWriter(&encBuf, masterKey, historyFileID)
	if err != nil {
		return nil, "", "", fmt.Errorf("setup history encryption: %w", err)
	}

	if _, err := encSetup.Writer.Write(compressed); err != nil {
		return nil, "", "", fmt.Errorf("encrypt history body: %w", err)
	}

	if err := encSetup.Writer.Close(); err != nil {
		return nil, "", "", fmt.Errorf("close history encryption writer: %w", err)
	}

	encryptedBytes := encBuf.Bytes()

	return &limitedBuffer{
		Reader: bytes.NewReader(encryptedBytes),
		length: len(encryptedBytes),
	}, encSetup.SaltBase64, encSetup.NonceBase64, nil
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

// isUniqueViolation classifies a PostgreSQL unique-constraint failure
// without binding to a specific driver error code constant — we only need
// "is this the dup-key race we just lost" vs "real failure".
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}

	msg := err.Error()

	return errors.Is(err, errUniqueViolationSentinel) ||
		// pgx surfaces "ERROR: duplicate key value violates unique constraint"
		bytes.Contains([]byte(msg), []byte("duplicate key value")) ||
		bytes.Contains([]byte(msg), []byte("SQLSTATE 23505"))
}

var errUniqueViolationSentinel = errors.New("unique constraint violation")

// _ keeps the base64 import live for sidecar-shape changes; encryption
// salt/IV currently passed as base64 strings from SetupEncryptionWriter.
var _ = base64.StdEncoding
