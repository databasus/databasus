package files

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

type DeletionBatchRepository struct{ db *gorm.DB }

func NewDeletionBatchRepository(db *gorm.DB) *DeletionBatchRepository {
	return &DeletionBatchRepository{db: db}
}

func (r *DeletionBatchRepository) Insert(ctx context.Context, batch DeletionBatch) (bool, error) {
	inserted := database.Conn(ctx, r.db).Exec(`INSERT INTO storages_deletion_batches
		(id,status,created_at,cleanup_deadline,completed_at,total,pending,deleted,remaining)
		VALUES (?,?,?,?,?,?,?,?,?) ON CONFLICT (id) DO NOTHING`, batch.ID, batch.Status, batch.CreatedAt, batch.CleanupDeadline, batch.CompletedAt, batch.Total, batch.Pending, batch.Deleted, batch.Remaining)
	if inserted.Error != nil {
		return false, fmt.Errorf("insert deletion batch: %w", inserted.Error)
	}
	return inserted.RowsAffected == 1, nil
}

func (r *DeletionBatchRepository) InsertItems(ctx context.Context, request DeletionBatchRequest, now time.Time) error {
	for _, reference := range request.References {
		if err := database.Conn(ctx, r.db).Exec(`INSERT INTO storages_pending_deletions
			(storage_id,file_name,not_before,is_ordinary,created_at,updated_at) VALUES (?,?,?,false,?,?)
			ON CONFLICT (storage_id,file_name) DO UPDATE SET generation=storages_pending_deletions.generation+1,
			not_before=excluded.not_before,updated_at=excluded.updated_at`, reference.StorageID, reference.FileName, now, now, now).Error; err != nil {
			return fmt.Errorf("request batch file deletion: %w", err)
		}
		if err := database.Conn(ctx, r.db).
			Exec(`INSERT INTO storages_deletion_batch_items (batch_id,storage_id,file_name) VALUES (?,?,?)`, request.ID, reference.StorageID, reference.FileName).
			Error; err != nil {
			return fmt.Errorf("insert deletion batch item: %w", err)
		}
	}
	return nil
}

func (r *DeletionBatchRepository) GetBatches(ctx context.Context, ids []uuid.UUID) ([]DeletionBatch, error) {
	batches := []DeletionBatch{}
	if len(ids) == 0 {
		return batches, nil
	}
	if err := database.Conn(ctx, r.db).Where("id IN ?", ids).Find(&batches).Error; err != nil {
		return nil, fmt.Errorf("read deletion batches: %w", err)
	}
	for index := range batches {
		batches[index].CreatedAt = batches[index].CreatedAt.UTC()
		batches[index].CleanupDeadline = batches[index].CleanupDeadline.UTC()
		if batches[index].CompletedAt != nil {
			completedAt := batches[index].CompletedAt.UTC()
			batches[index].CompletedAt = &completedAt
		}
	}
	return batches, nil
}

func (r *DeletionBatchRepository) RefreshOutcomes(ctx context.Context, now time.Time) error {
	if err := r.lockOutcomes(ctx); err != nil {
		return err
	}
	err := database.Conn(ctx, r.db).Exec(`UPDATE storages_deletion_batches AS batch
		SET pending=counts.pending,deleted=counts.deleted,remaining=counts.remaining,
		completed_at=CASE WHEN counts.pending=0 THEN ?::timestamptz ELSE NULL END,
		status=CASE WHEN counts.pending>0 THEN 'CLEANUP_PENDING' WHEN counts.remaining>0 THEN 'COMPLETED_WITH_REMAINDERS' ELSE 'COMPLETED' END
		FROM (SELECT batch_id,count(*) FILTER (WHERE status='PENDING') AS pending,
		count(*) FILTER (WHERE status='DELETED') AS deleted,count(*) FILTER (WHERE status='REMAINING') AS remaining
		FROM storages_deletion_batch_items WHERE batch_id IN (SELECT id FROM storages_deletion_batches WHERE completed_at IS NULL) GROUP BY batch_id) AS counts
		WHERE batch.id=counts.batch_id AND batch.completed_at IS NULL`, now).Error
	if err != nil {
		return fmt.Errorf("refresh deletion batch outcomes: %w", err)
	}
	return nil
}

func (r *DeletionBatchRepository) CompleteFile(ctx context.Context, pending PendingDeletion, now time.Time) error {
	if err := r.lockOutcomes(ctx); err != nil {
		return err
	}
	err := database.Conn(ctx, r.db).
		Exec(`UPDATE storages_deletion_batch_items AS item SET status='DELETED',attempt_generation=NULL,attempt_expires_at=NULL
		FROM storages_deletion_batches AS batch WHERE item.batch_id=batch.id AND item.storage_id=? AND item.file_name=? AND item.status='PENDING'
		AND batch.cleanup_deadline>? AND item.attempt_generation=? AND item.attempt_expires_at>?`, pending.StorageID, pending.FileName, now, pending.Generation, now).
		Error
	if err != nil {
		return fmt.Errorf("complete deletion batch file: %w", err)
	}
	return r.RefreshOutcomes(ctx, now)
}

func (r *DeletionBatchRepository) ClaimFile(
	ctx context.Context,
	pending PendingDeletion,
	now time.Time,
) (time.Duration, error) {
	if err := r.Expire(ctx, now); err != nil {
		return 0, err
	}

	var hasActiveAttempt bool
	if err := database.Conn(ctx, r.db).Raw(`SELECT EXISTS (
		SELECT 1 FROM storages_deletion_batch_items WHERE storage_id=? AND file_name=?
		AND status='PENDING' AND attempt_generation IS NOT NULL)`, pending.StorageID, pending.FileName).
		Scan(&hasActiveAttempt).Error; err != nil {
		return 0, fmt.Errorf("check active batch attempts: %w", err)
	}
	if hasActiveAttempt {
		return 0, nil
	}

	var deadlines []time.Time
	err := database.Conn(ctx, r.db).Raw(`UPDATE storages_deletion_batch_items AS item
		SET attempt_generation=?,attempt_expires_at=least(?,batch.cleanup_deadline)
		FROM storages_deletion_batches AS batch WHERE item.batch_id=batch.id AND item.storage_id=? AND item.file_name=?
		AND item.status='PENDING' AND batch.cleanup_deadline>?
		RETURNING item.attempt_expires_at`, pending.Generation, pending.NotBefore, pending.StorageID, pending.FileName, now).Scan(&deadlines).Error
	if err != nil {
		return 0, fmt.Errorf("claim deletion batch file: %w", err)
	}

	if len(deadlines) == 0 {
		var isOrdinary bool
		if err := database.Conn(ctx, r.db).
			Raw("SELECT is_ordinary FROM storages_pending_deletions WHERE id=?", pending.ID).
			Scan(&isOrdinary).
			Error; err != nil {
			return 0, fmt.Errorf("read deletion ownership: %w", err)
		}
		if !isOrdinary {
			return 0, nil
		}
	}

	remaining := pending.NotBefore.Sub(now)
	for _, deadline := range deadlines {
		remaining = min(remaining, deadline.Sub(now))
	}
	return remaining, nil
}

func (r *DeletionBatchRepository) FailFile(ctx context.Context, pending PendingDeletion, now time.Time) error {
	if err := r.lockOutcomes(ctx); err != nil {
		return err
	}
	err := database.Conn(ctx, r.db).Exec(`UPDATE storages_deletion_batch_items SET failed_attempts=failed_attempts+1,
		status=CASE WHEN failed_attempts+1>=5 THEN 'REMAINING' ELSE 'PENDING' END,attempt_generation=NULL,attempt_expires_at=NULL
		WHERE storage_id=? AND file_name=? AND status='PENDING' AND attempt_generation=? AND attempt_expires_at>?`, pending.StorageID, pending.FileName, pending.Generation, now).Error
	if err != nil {
		return fmt.Errorf("record deletion batch failure: %w", err)
	}
	return r.RefreshOutcomes(ctx, now)
}

func (r *DeletionBatchRepository) Expire(ctx context.Context, now time.Time) error {
	if err := r.lockOutcomes(ctx); err != nil {
		return err
	}

	if err := database.Conn(ctx, r.db).Exec(`UPDATE storages_deletion_batch_items SET failed_attempts=failed_attempts+1,
 status=CASE WHEN failed_attempts+1>=5 THEN 'REMAINING' ELSE 'PENDING' END,
 attempt_generation=NULL,attempt_expires_at=NULL WHERE status='PENDING' AND attempt_expires_at<=?`, now).Error; err != nil {
		return fmt.Errorf("settle expired batch attempts: %w", err)
	}

	err := database.Conn(ctx, r.db).Exec(`UPDATE storages_deletion_batch_items AS item
		SET status='REMAINING',attempt_generation=NULL,attempt_expires_at=NULL
		FROM storages_deletion_batches AS batch WHERE item.batch_id=batch.id AND item.status='PENDING' AND batch.cleanup_deadline<=?`, now).Error
	if err != nil {
		return fmt.Errorf("expire deletion batch items: %w", err)
	}
	return r.RefreshOutcomes(ctx, now)
}

func (r *DeletionBatchRepository) RemoveUnownedPending(ctx context.Context) error {
	err := database.Conn(ctx, r.db).
		Exec(`DELETE FROM storages_pending_deletions AS pending WHERE pending.id IN (SELECT id FROM storages_pending_deletions ORDER BY id FOR UPDATE SKIP LOCKED) AND NOT pending.is_ordinary
 AND NOT EXISTS (SELECT 1 FROM storages_deletion_batch_items AS item WHERE item.storage_id=pending.storage_id AND item.file_name=pending.file_name AND item.status='PENDING')`).
		Error
	if err != nil {
		return fmt.Errorf("release exhausted batch obligations: %w", err)
	}
	return nil
}

func (r *DeletionBatchRepository) ListRemainingFiles(ctx context.Context, storageID uuid.UUID) ([]string, error) {
	var fileNames []string
	if err := database.Conn(ctx, r.db).
		Raw("SELECT DISTINCT file_name FROM storages_deletion_batch_items WHERE storage_id=? AND status='REMAINING' ORDER BY file_name", storageID).
		Scan(&fileNames).Error; err != nil {
		return nil, fmt.Errorf("list remaining batch artifacts: %w", err)
	}
	return fileNames, nil
}

func (r *DeletionBatchRepository) SettleStorage(
	ctx context.Context,
	settlement StorageDeletionSettlement,
) error {
	storageID, confirmedFiles, now := settlement.StorageID, settlement.ConfirmedFiles, settlement.Now
	if err := r.lockOutcomes(ctx); err != nil {
		return err
	}

	if len(confirmedFiles) > 0 {
		err := database.Conn(ctx, r.db).
			Exec(`UPDATE storages_deletion_batch_items AS item SET status='DELETED',attempt_generation=NULL,attempt_expires_at=NULL
		FROM storages_deletion_batches AS batch WHERE item.batch_id=batch.id AND item.storage_id=? AND item.file_name IN ?
		AND item.status='PENDING' AND batch.cleanup_deadline>?`, storageID, confirmedFiles, now).
			Error
		if err != nil {
			return fmt.Errorf("confirm drained batch files: %w", err)
		}
	}

	if err := database.Conn(ctx, r.db).
		Exec(`UPDATE storages_deletion_batch_items SET status='REMAINING',attempt_generation=NULL,attempt_expires_at=NULL WHERE storage_id=? AND status='PENDING'`, storageID).
		Error; err != nil {
		return fmt.Errorf("settle deleted storage batches: %w", err)
	}
	return r.RefreshOutcomes(ctx, now)
}

func (r *DeletionBatchRepository) lockOutcomes(ctx context.Context) error {
	// Provider calls run outside this lock; result accounting and deadline settlement share one snapshot.
	if err := database.Conn(ctx, r.db).
		Exec("SELECT pg_advisory_xact_lock(hashtextextended(?,0))", "storages_deletion_batches").
		Error; err != nil {
		return fmt.Errorf("lock deletion batch outcomes: %w", err)
	}
	return nil
}
