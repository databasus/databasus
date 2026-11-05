package files

import (
	"context"
	"fmt"
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
		INSERT INTO storages_pending_file_deletions (id, storage_id, file_name, not_before, created_at, updated_at)
		SELECT ?, id, ?, ?, ?, ? FROM storages WHERE id = ? FOR KEY SHARE
		ON CONFLICT (storage_id, file_name) DO NOTHING
		RETURNING *`,
		registration.ID, registration.Reference.FileName,
		registration.NotBefore, registration.Now, registration.Now, registration.Reference.StorageID,
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
	request FileDeletionRequest,
	now time.Time,
) error {
	unique := deduplicateReferences(request.References)
	if len(unique) == 0 {
		return nil
	}

	for _, reference := range unique {
		if reference.FileName == "" {
			continue
		}

		var databaseID, workspaceID *uuid.UUID
		if request.Scope != nil {
			databaseID, workspaceID = &request.Scope.DatabaseID, &request.Scope.WorkspaceID
		}

		if err := database.Conn(ctx, r.db).Exec(`
            INSERT INTO storages_pending_file_deletions
                (storage_id, file_name, not_before, created_at, updated_at, cleanup_database_id, cleanup_workspace_id)
            SELECT id, ?, ?, ?, ?, ?, ? FROM storages WHERE id = ? FOR KEY SHARE
            ON CONFLICT (storage_id, file_name) DO UPDATE
            SET generation = storages_pending_file_deletions.generation + 1,
                not_before = CASE WHEN storages_pending_file_deletions.attempt_count = 0 THEN least(storages_pending_file_deletions.not_before, EXCLUDED.not_before) ELSE storages_pending_file_deletions.not_before END,
                cleanup_database_id = coalesce(storages_pending_file_deletions.cleanup_database_id, EXCLUDED.cleanup_database_id),
                cleanup_workspace_id = coalesce(storages_pending_file_deletions.cleanup_workspace_id, EXCLUDED.cleanup_workspace_id),
                updated_at = EXCLUDED.updated_at`,
			reference.FileName, now, now, now, databaseID, workspaceID, reference.StorageID).Error; err != nil {
			return fmt.Errorf("record pending deletion: %w", err)
		}
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
		UPDATE storages_pending_file_deletions
		SET generation = generation + 1,
		    not_before = ?,
		    updated_at = ?
		WHERE id IN (
			SELECT id
			FROM storages_pending_file_deletions
			WHERE not_before <= ? %s
			AND pg_try_advisory_xact_lock(hashtextextended('storages-file-write:' || storage_id::text || ':' || file_name,0))
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
		FROM storages_pending_file_deletions`, map[string]any{"now": now}).Scan(&summary).Error
	if err != nil {
		return PendingDeletionSummary{}, fmt.Errorf("summarize pending deletions: %w", err)
	}

	return summary, nil
}

func (r *PendingDeletionRepository) ListByStorage(ctx context.Context, storageID uuid.UUID) ([]PendingDeletion, error) {
	var pending []PendingDeletion

	err := database.Conn(ctx, r.db).Where("storage_id = ?", storageID).Order("created_at, id").Find(&pending).Error
	if err != nil {
		return nil, fmt.Errorf("list pending deletions of a storage: %w", err)
	}

	return pending, nil
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

type AttemptLock struct {
	Pending            PendingDeletion
	Now                time.Time
	ShouldAllowExpired bool
}

func (r *PendingDeletionRepository) StartAttempt(
	ctx context.Context,
	pending PendingDeletion,
	now time.Time,
) (bool, error) {
	result := database.Conn(ctx, r.db).Model(&PendingDeletion{}).
		Where("id = ? AND generation = ? AND not_before > ?", pending.ID, pending.Generation, now).
		Updates(map[string]any{
			"attempt_count": gorm.Expr("attempt_count + 1"),
			"updated_at":    now,
		})
	if result.Error != nil {
		return false, fmt.Errorf("start deletion attempt: %w", result.Error)
	}

	return result.RowsAffected == 1, nil
}

func (r *PendingDeletionRepository) LockAttempt(
	ctx context.Context,
	lock AttemptLock,
) (bool, error) {
	var ids []uuid.UUID
	err := database.Conn(ctx, r.db).
		Raw(`SELECT id FROM storages_pending_file_deletions WHERE id=? AND generation=? AND (not_before>? OR ?) FOR UPDATE`, lock.Pending.ID, lock.Pending.Generation, lock.Now, lock.ShouldAllowExpired).
		Scan(&ids).
		Error
	if err != nil {
		return false, fmt.Errorf("lock deletion attempt: %w", err)
	}
	return len(ids) > 0, nil
}

func (r *PendingDeletionRepository) ListReferencesByPrefix(
	ctx context.Context,
	prefix string,
) ([]StoredFileReference, error) {
	references := []StoredFileReference{}
	if err := database.Conn(ctx, r.db).
		Raw("SELECT storage_id,file_name FROM storages_pending_file_deletions WHERE left(file_name,length(?))=?", prefix, prefix).
		Scan(&references).
		Error; err != nil {
		return nil, fmt.Errorf("list unfinished files: %w", err)
	}
	return references, nil
}
