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

	artifactReader, encryptionSalt, encryptionIV, err := buildHistoryArtifactReader(historyArtifactSpec{
		Body: body, Encryption: encryption, MasterKey: masterKey, HistoryFileID: historyFileID,
	})
	if err != nil {
		return nil, err
	}

	artifactReference := storages.StoredFileReference{StorageID: storageID, FileName: storageObjectName}
	sidecarFilename := storageObjectName + catalog.MetadataSuffix
	sidecarReference := storages.StoredFileReference{StorageID: storageID, FileName: sidecarFilename}
	hasPublishedHistory := false
	defer func() {
		if hasPublishedHistory {
			return
		}

		discardCtx := context.WithoutCancel(ctx)
		if discardErr := spec.Transactor.Transaction(discardCtx, func(ctx context.Context) error {
			return spec.FileStore.RequestDiscardedFileDeletions(ctx, storages.DiscardedFiles{
				DatabaseID: databaseID,
				References: []storages.StoredFileReference{artifactReference, sidecarReference},
			})
		}); discardErr != nil {
			logger.WarnContext(discardCtx, "failed to discard a history upload",
				slog.Any("file_name", storageObjectName), slog.Any("error", discardErr))
		}
	}()

	artifactReceipt, err := spec.FileStore.WriteFile(ctx, artifactReference, artifactReader)
	if err != nil {
		return nil, fmt.Errorf("upload history artifact: %w", err)
	}

	compressedSizeBytes := artifactReader.Size()

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

	sidecarReceipt, err := spec.FileStore.WriteFile(ctx, sidecarReference, bytes.NewReader(sidecarBytes))
	if err != nil {
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

	hasPublishedHistory = true

	return row, nil
}

type historyArtifactSpec struct {
	Body          []byte
	Encryption    catalog.BackupEncryption
	MasterKey     *crypto.MasterKey
	HistoryFileID uuid.UUID
}

// Timeline history is small enough to compress in memory before uploading.
func buildHistoryArtifactReader(spec historyArtifactSpec) (*bytes.Reader, string, string, error) {
	compressed, err := compressZstd(spec.Body)
	if err != nil {
		return nil, "", "", err
	}

	if spec.Encryption != catalog.BackupEncryptionEncrypted {
		return bytes.NewReader(compressed), "", "", nil
	}

	var encBuf bytes.Buffer

	encryptingWriter, header, err := crypto.NewEncryptingWriter(&encBuf, spec.MasterKey, spec.HistoryFileID)
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

	salt := base64.StdEncoding.EncodeToString(header.Salt)
	iv := base64.StdEncoding.EncodeToString(header.Nonce)

	return bytes.NewReader(encryptedBytes), salt, iv, nil
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
