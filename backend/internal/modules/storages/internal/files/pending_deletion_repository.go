package files

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

// Registration is the obligation a write records before its first byte leaves.
type Registration struct {
	ID        uuid.UUID
	Reference StoredFileReference
	NotBefore time.Time
	Now       time.Time
}

// Rescheduling moves the deadline of a pending deletion, unless something raised its generation.
type Rescheduling struct {
	PendingDeletionID uuid.UUID
	Generation        int
	NotBefore         time.Time
	Now               time.Time
}

type ClaimRequest struct {
	Limit        int
	AttemptLease time.Duration
	// ExcludedIDs are the writes running in this process, which have an obligation but no owner
	// willing to let it go yet.
	ExcludedIDs []uuid.UUID
	Now         time.Time
}

type PendingDeletionSummary struct {
	Pending     int64
	Overdue     int64
	OldestAgeMs int64
}

// PendingDeletionRepository takes every deadline from the caller, which reads it from the clock of
// the module, so every path compares against one clock and no skew can shorten a commit window on
// one path and lengthen it on another.
type PendingDeletionRepository struct {
	db *gorm.DB
}

func NewPendingDeletionRepository(db *gorm.DB) *PendingDeletionRepository {
	return &PendingDeletionRepository{db: db}
}

// IsInTransaction tells whether ctx carries a transaction. Both batch methods of the store are
// all-or-nothing: on the shared handle a batch that fails partway has already released the receipts
// it got through, stranding those files forever.
func (r *PendingDeletionRepository) IsInTransaction(ctx context.Context) bool {
	_, isInTransaction := database.Conn(ctx, r.db).Statement.ConnPool.(gorm.TxCommitter)

	return isInTransaction
}

// InsertIfAbsent refuses on a conflict, because the pair already carries an obligation and the write
// must not start: either a cascade reached the file first, or a second writer picked a name
// per-attempt naming should have made unique.
func (r *PendingDeletionRepository) InsertIfAbsent(
	ctx context.Context,
	registration Registration,
) (*PendingDeletion, error) {
	var inserted []PendingDeletion

	result := database.Conn(ctx, r.db).Raw(`
		INSERT INTO storages_pending_deletions (id, storage_id, file_name, not_before, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (storage_id, file_name) DO NOTHING
		RETURNING *`,
		registration.ID, registration.Reference.StorageID, registration.Reference.FileName,
		registration.NotBefore, registration.Now, registration.Now,
	).Scan(&inserted)
	if result.Error != nil {
		return nil, fmt.Errorf("insert pending deletion: %w", result.Error)
	}

	if len(inserted) == 0 {
		return nil, ErrFileNameAlreadyRegistered
	}

	return &inserted[0], nil
}

// InsertOrTake also increments the generation of an existing row, which is what takes the file away
// from a writer that is still running and stops it publishing.
func (r *PendingDeletionRepository) InsertOrTake(
	ctx context.Context,
	references []StoredFileReference,
	now time.Time,
) error {
	unique := deduplicateReferences(references)
	if len(unique) == 0 {
		return nil
	}

	placeholders := make([]string, 0, len(unique))
	args := make([]any, 0, len(unique)*5+2)

	for _, reference := range unique {
		placeholders = append(placeholders, "(?, ?, ?, ?, ?)")
		args = append(args, reference.StorageID, reference.FileName, now, now, now)
	}

	args = append(args, now, now)

	statement := fmt.Sprintf(`
		INSERT INTO storages_pending_deletions (storage_id, file_name, not_before, created_at, updated_at)
		VALUES %s
		ON CONFLICT (storage_id, file_name) DO UPDATE
		SET not_before = ?,
		    generation = storages_pending_deletions.generation + 1,
		    updated_at = ?`, strings.Join(placeholders, ", "))

	if err := database.Conn(ctx, r.db).Exec(statement, args...).Error; err != nil {
		return fmt.Errorf("record pending deletions: %w", err)
	}

	return nil
}

// CompleteIfGeneration is how a publishing transaction keeps a file and how the deletion job records
// a finished deletion, so it refuses when the generation moved and someone else owns the outcome.
func (r *PendingDeletionRepository) CompleteIfGeneration(
	ctx context.Context,
	pendingDeletionID uuid.UUID,
	generation int,
) (bool, error) {
	result := database.Conn(ctx, r.db).
		Where("id = ? AND generation = ?", pendingDeletionID, generation).
		Delete(&PendingDeletion{})
	if result.Error != nil {
		return false, fmt.Errorf("complete pending deletion: %w", result.Error)
	}

	return result.RowsAffected == 1, nil
}

func (r *PendingDeletionRepository) SetNotBeforeIfGeneration(
	ctx context.Context,
	rescheduling Rescheduling,
) (bool, error) {
	result := database.Conn(ctx, r.db).Model(&PendingDeletion{}).
		Where("id = ? AND generation = ?", rescheduling.PendingDeletionID, rescheduling.Generation).
		Updates(map[string]any{
			"not_before": rescheduling.NotBefore,
			"updated_at": rescheduling.Now,
		})
	if result.Error != nil {
		return false, fmt.Errorf("reschedule pending deletion: %w", result.Error)
	}

	return result.RowsAffected == 1, nil
}

func (r *PendingDeletionRepository) RescheduleIfGeneration(
	ctx context.Context,
	rescheduling Rescheduling,
	lastError string,
) (bool, error) {
	result := database.Conn(ctx, r.db).Model(&PendingDeletion{}).
		Where("id = ? AND generation = ?", rescheduling.PendingDeletionID, rescheduling.Generation).
		Updates(map[string]any{
			"not_before": rescheduling.NotBefore,
			"last_error": lastError,
			"updated_at": rescheduling.Now,
		})
	if result.Error != nil {
		return false, fmt.Errorf("reschedule pending deletion: %w", result.Error)
	}

	return result.RowsAffected == 1, nil
}

// ClaimDue raises the generation, which invalidates any receipt still outstanding for these files,
// and pushes not_before by the lease, which keeps a stalled provider call from holding a row forever.
// SKIP LOCKED keeps an open deletion request from stalling the whole batch.
func (r *PendingDeletionRepository) ClaimDue(ctx context.Context, request ClaimRequest) ([]PendingDeletion, error) {
	exclusion := ""
	args := []any{request.Now.Add(request.AttemptLease), request.Now, request.Now}

	if len(request.ExcludedIDs) > 0 {
		exclusion = "AND id NOT IN (?)"

		args = append(args, request.ExcludedIDs)
	}

	args = append(args, request.Limit)

	statement := fmt.Sprintf(`
		UPDATE storages_pending_deletions
		SET generation = generation + 1,
		    attempt_count = attempt_count + 1,
		    not_before = ?,
		    updated_at = ?
		WHERE id IN (
			SELECT id
			FROM storages_pending_deletions
			WHERE not_before <= ? %s
			ORDER BY not_before
			LIMIT ?
			FOR UPDATE SKIP LOCKED
		)
		RETURNING *`, exclusion)

	var claimed []PendingDeletion

	if err := database.Conn(ctx, r.db).Raw(statement, args...).Scan(&claimed).Error; err != nil {
		return nil, fmt.Errorf("claim pending deletions: %w", err)
	}

	return claimed, nil
}

func (r *PendingDeletionRepository) GetPendingDeletionSummary(
	ctx context.Context,
	now time.Time,
) (PendingDeletionSummary, error) {
	var summary PendingDeletionSummary

	err := database.Conn(ctx, r.db).Raw(`
		SELECT count(*)                                                                  AS pending,
		       count(*) FILTER (WHERE not_before <= @now)                                AS overdue,
		       coalesce(extract(epoch FROM @now - min(created_at)) * 1000, 0)::bigint     AS oldest_age_ms
		FROM storages_pending_deletions`, map[string]any{"now": now}).Scan(&summary).Error
	if err != nil {
		return PendingDeletionSummary{}, fmt.Errorf("summarize pending deletions: %w", err)
	}

	return summary, nil
}

func (r *PendingDeletionRepository) CountByReferences(
	ctx context.Context,
	references []StoredFileReference,
) (int64, error) {
	unique := deduplicateReferences(references)
	if len(unique) == 0 {
		return 0, nil
	}

	pairs := make([][]any, 0, len(unique))
	for _, reference := range unique {
		pairs = append(pairs, []any{reference.StorageID, reference.FileName})
	}

	var count int64

	err := database.Conn(ctx, r.db).Model(&PendingDeletion{}).
		Where("(storage_id, file_name) IN ?", pairs).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("count pending deletions: %w", err)
	}

	return count, nil
}

func (r *PendingDeletionRepository) ListByStorage(ctx context.Context, storageID uuid.UUID) ([]PendingDeletion, error) {
	var pending []PendingDeletion

	err := database.Conn(ctx, r.db).Where("storage_id = ?", storageID).Order("created_at, id").Find(&pending).Error
	if err != nil {
		return nil, fmt.Errorf("list pending deletions of a storage: %w", err)
	}

	return pending, nil
}

func (r *PendingDeletionRepository) FindByReference(
	ctx context.Context,
	reference StoredFileReference,
) (PendingDeletion, bool, error) {
	var pending PendingDeletion

	err := database.Conn(ctx, r.db).
		Where("storage_id = ? AND file_name = ?", reference.StorageID, reference.FileName).
		First(&pending).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return PendingDeletion{}, false, nil
	}

	if err != nil {
		return PendingDeletion{}, false, fmt.Errorf("find pending deletion: %w", err)
	}

	return pending, true, nil
}

func deduplicateReferences(references []StoredFileReference) []StoredFileReference {
	seen := make(map[StoredFileReference]struct{}, len(references))
	unique := make([]StoredFileReference, 0, len(references))

	for _, reference := range references {
		if _, isDuplicate := seen[reference]; isDuplicate {
			continue
		}

		seen[reference] = struct{}{}
		unique = append(unique, reference)
	}

	return unique
}
