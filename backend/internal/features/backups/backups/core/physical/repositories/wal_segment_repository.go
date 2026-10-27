package physical_repositories

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	"databasus-backend/internal/storage"
)

type PhysicalWalSegmentRepository struct{}

func (r *PhysicalWalSegmentRepository) Insert(segment *physical_models.PhysicalWalSegment) error {
	if segment.DatabaseID == uuid.Nil || segment.StorageID == uuid.Nil {
		return errors.New("database ID and storage ID are required")
	}

	if segment.ID == uuid.Nil {
		segment.ID = uuid.New()
	}

	now := time.Now().UTC()
	if segment.ReceivedAt.IsZero() {
		segment.ReceivedAt = now
	}
	if segment.ClaimedAt.IsZero() {
		segment.ClaimedAt = now
	}

	return storage.GetDb().Create(segment).Error
}

func (r *PhysicalWalSegmentRepository) FindByID(id uuid.UUID) (*physical_models.PhysicalWalSegment, error) {
	var segment physical_models.PhysicalWalSegment

	if err := storage.GetDb().Where("id = ?", id).First(&segment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &segment, nil
}

// FindByChainSpan returns the timeline's WAL segments overlapping [startLSN,
// endLSN), ordered by start_lsn. Overlap (end_lsn > startLSN) — not start_lsn >=
// startLSN — is deliberate: a FULL's start_lsn is usually mid-segment, so the
// segment that physically holds the bytes from the chain's start onward has a
// file-boundary start_lsn BELOW startLSN. Excluding it would drop the very
// segment that bridges the FULL's stop_lsn into the next segment, manufacturing
// a false WAL gap at the start of every restore replay window.
func (r *PhysicalWalSegmentRepository) FindByChainSpan(
	databaseID uuid.UUID,
	timelineID int,
	startLSN, endLSN walmath.LSN,
) ([]*physical_models.PhysicalWalSegment, error) {
	var segments []*physical_models.PhysicalWalSegment

	if err := whereInChainSpan(storage.GetDb(), databaseID, timelineID, startLSN, endLSN).
		Order("start_lsn ASC").
		Find(&segments).Error; err != nil {
		return nil, err
	}

	return segments, nil
}

// GetLatestReceivedAtInChainSpan returns nil when no segment overlaps the span.
func (r *PhysicalWalSegmentRepository) GetLatestReceivedAtInChainSpan(
	databaseID uuid.UUID,
	timelineID int,
	startLSN, endLSN walmath.LSN,
) (*time.Time, error) {
	var latestReceivedAt sql.NullTime

	if err := whereInChainSpan(storage.GetDb(), databaseID, timelineID, startLSN, endLSN).
		Model(&physical_models.PhysicalWalSegment{}).
		Select("MAX(received_at)").
		Row().
		Scan(&latestReceivedAt); err != nil {
		return nil, err
	}

	if !latestReceivedAt.Valid {
		return nil, nil
	}

	return &latestReceivedAt.Time, nil
}

// whereInChainSpan selects the segments overlapping [startLSN, endLSN). The
// end_lsn filter alone leaves the start_lsn index bounded only from above, so
// every lookup would read the timeline's WAL from its first segment. Segments of
// one timeline never overlap, so any segment ending after startLSN starts at or
// after the segment holding startLSN, and that segment's start_lsn is a lower
// index bound that keeps the boundary segment in.
func whereInChainSpan(
	db *gorm.DB,
	databaseID uuid.UUID,
	timelineID int,
	startLSN, endLSN walmath.LSN,
) *gorm.DB {
	return db.Where(`
		database_id = ? AND timeline_id = ?
		AND start_lsn >= COALESCE(
			(SELECT boundary.start_lsn
			 FROM physical_wal_segments boundary
			 WHERE boundary.database_id = ? AND boundary.timeline_id = ? AND boundary.start_lsn <= ?::pg_lsn
			 ORDER BY boundary.start_lsn DESC
			 LIMIT 1),
			'0/0'::pg_lsn)
		AND start_lsn < ?::pg_lsn
		AND end_lsn > ?::pg_lsn`,
		databaseID, timelineID,
		databaseID, timelineID, startLSN.String(),
		endLSN.String(),
		startLSN.String(),
	)
}

// Anti-join not covered by idx_physical_wal_segments_database_id_received_at;
// expect a seq scan on physical_full_backups when invoked in bulk.
//
// The comparison anchors on the segment's end_lsn because a FULL's start_lsn is
// a real recovery position a short way inside the segment that holds it, while a
// segment's own bounds are derived from its filename and are file-aligned.
// Comparing against start_lsn would classify that boundary segment as an orphan
// and delete the WAL the FULL itself starts and stops in.
func (r *PhysicalWalSegmentRepository) FindOrphans(
	databaseID uuid.UUID,
) ([]*physical_models.PhysicalWalSegment, error) {
	var orphans []*physical_models.PhysicalWalSegment

	if err := storage.
		GetDb().
		Raw(`
			SELECT *
			FROM physical_wal_segments w
			WHERE w.database_id = ?
			  AND NOT EXISTS (
			    SELECT 1
			    FROM physical_full_backups f
			    WHERE f.database_id = w.database_id
			      AND f.timeline_id = w.timeline_id
			      AND f.start_lsn IS NOT NULL
			      AND f.start_lsn < w.end_lsn
			      AND f.status = ?
			  )
			ORDER BY w.start_lsn ASC
		`, databaseID, physical_enums.PhysicalBackupStatusCompleted).
		Scan(&orphans).Error; err != nil {
		return nil, err
	}

	return orphans, nil
}

func (r *PhysicalWalSegmentRepository) DeleteByID(id uuid.UUID) error {
	return storage.GetDb().Delete(&physical_models.PhysicalWalSegment{}, "id = ?", id).Error
}

// A NULL file_name does not prove the upload wrote nothing: the object is written
// before the row names it. The bytes such a claim may have left are owned by their
// own pending deletion, so this only removes the aged-out claim row.
func (r *PhysicalWalSegmentRepository) DeleteAbandonedClaims(
	databaseID uuid.UUID,
	olderThan time.Time,
) (int64, error) {
	result := storage.
		GetDb().
		Where("database_id = ? AND file_name IS NULL AND claimed_at < ?", databaseID, olderThan).
		Delete(&physical_models.PhysicalWalSegment{})

	return result.RowsAffected, result.Error
}

// ClaimInsert inserts an upload-claim row (file_name still NULL) for a freshly
// rotated WAL segment, deduplicated on (database_id, timeline_id, start_lsn) via
// ON CONFLICT DO NOTHING. inserted=true means this caller won the claim and must
// proceed to upload; inserted=false means another uploader already holds (or
// completed) the slot — the caller probes FindByChainKey to decide what to do.
func (r *PhysicalWalSegmentRepository) ClaimInsert(
	segment *physical_models.PhysicalWalSegment,
) (inserted bool, err error) {
	if segment.DatabaseID == uuid.Nil || segment.StorageID == uuid.Nil {
		return false, errors.New("database ID and storage ID are required")
	}

	if segment.ID == uuid.Nil {
		segment.ID = uuid.New()
	}

	now := time.Now().UTC()
	if segment.ReceivedAt.IsZero() {
		segment.ReceivedAt = now
	}
	if segment.ClaimedAt.IsZero() {
		segment.ClaimedAt = now
	}

	result := storage.
		GetDb().
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "database_id"}, {Name: "timeline_id"}, {Name: "start_lsn"}},
			DoNothing: true,
		}).
		Create(segment)
	if result.Error != nil {
		return false, result.Error
	}

	return result.RowsAffected == 1, nil
}

// FindByChainKey returns the row for a specific (database_id, timeline_id,
// start_lsn) — the uploader's conflict probe after ClaimInsert lost the race,
// to distinguish a durably-completed segment (file_name NOT NULL) from an
// in-flight claim (file_name NULL). Returns nil when absent.
func (r *PhysicalWalSegmentRepository) FindByChainKey(
	databaseID uuid.UUID,
	timelineID int,
	startLSN walmath.LSN,
) (*physical_models.PhysicalWalSegment, error) {
	var segment physical_models.PhysicalWalSegment

	err := storage.
		GetDb().
		Where("database_id = ? AND timeline_id = ? AND start_lsn = ?::pg_lsn",
			databaseID, timelineID, startLSN.String()).
		First(&segment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &segment, nil
}

type WalSegmentUpload struct {
	SegmentID        uuid.UUID
	FileName         string
	CompressedSizeMb float64
	EncryptionSalt   *string
	EncryptionIV     *string
}

// The file_name IS NULL guard makes this a no-op when a DeleteFull cascade removed
// the claim mid-upload. An updated=false result therefore means nothing published
// the object, and the caller hands it back through the file store.
func (r *PhysicalWalSegmentRepository) MarkUploadedInTransaction(
	tx *gorm.DB,
	upload WalSegmentUpload,
) (updated bool, err error) {
	result := tx.
		Model(&physical_models.PhysicalWalSegment{}).
		Where("id = ? AND file_name IS NULL", upload.SegmentID).
		Updates(map[string]any{
			"file_name":          upload.FileName,
			"compressed_size_mb": upload.CompressedSizeMb,
			"encryption_salt":    upload.EncryptionSalt,
			"encryption_iv":      upload.EncryptionIV,
		})
	if result.Error != nil {
		return false, result.Error
	}

	return result.RowsAffected == 1, nil
}

// DeleteClaim removes an upload-claim row, guarded by file_name IS NULL so it can
// never delete a durably-committed segment. Idempotent. Used on SaveFile failure
// to release the claim so the next uploader tick can re-claim the same segment.
func (r *PhysicalWalSegmentRepository) DeleteClaim(id uuid.UUID) error {
	return storage.
		GetDb().
		Where("id = ? AND file_name IS NULL", id).
		Delete(&physical_models.PhysicalWalSegment{}).Error
}

// FindLatestCommittedBefore returns the highest end_lsn among committed
// (file_name NOT NULL) segments on (database_id, timeline_id) whose start_lsn is
// strictly before startLSN — the predecessor the post-upload gap probe compares
// against. Returns nil when there is no committed predecessor (the just-uploaded
// segment is the first on its timeline). In-flight claims are excluded because
// their end_lsn is predicted, not durable.
func (r *PhysicalWalSegmentRepository) FindLatestCommittedBefore(
	databaseID uuid.UUID,
	timelineID int,
	startLSN walmath.LSN,
) (*walmath.LSN, error) {
	var segment physical_models.PhysicalWalSegment

	err := storage.
		GetDb().
		Where("database_id = ? AND timeline_id = ? AND start_lsn < ?::pg_lsn AND file_name IS NOT NULL",
			databaseID, timelineID, startLSN.String()).
		Order("start_lsn DESC").
		First(&segment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	endLSN := segment.EndLSN

	return &endLSN, nil
}

// The newest segment that is durable in storage — the database's actual recovery
// point. In-flight claims are excluded: a claim row means an upload was started,
// not that the WAL can be restored from.
func (r *PhysicalWalSegmentRepository) FindLatestCommittedSegment(
	databaseID uuid.UUID,
) (*physical_models.PhysicalWalSegment, error) {
	var segment physical_models.PhysicalWalSegment

	err := storage.
		GetDb().
		Where("database_id = ? AND file_name IS NOT NULL", databaseID).
		Order("timeline_id DESC, start_lsn DESC").
		First(&segment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &segment, nil
}
