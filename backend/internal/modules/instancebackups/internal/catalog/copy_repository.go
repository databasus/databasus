package catalog

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

type copyRepository struct {
	db *gorm.DB
}

func (r *copyRepository) insert(ctx context.Context, backupCopy Copy) error {
	if err := database.Conn(ctx, r.db).Create(&backupCopy).Error; err != nil {
		return fmt.Errorf("insert the Databasus backup copy: %w", err)
	}

	return nil
}

func (r *copyRepository) get(ctx context.Context, copyID uuid.UUID) (Copy, bool, error) {
	var copies []Copy

	err := database.Conn(ctx, r.db).Where("id = ?", copyID).Limit(1).Find(&copies).Error
	if err != nil {
		return Copy{}, false, fmt.Errorf("read the Databasus backup copy: %w", err)
	}

	if len(copies) == 0 {
		return Copy{}, false, nil
	}

	return copies[0], true, nil
}

func (r *copyRepository) listByBackupIDs(ctx context.Context, backupIDs []uuid.UUID) ([]Copy, error) {
	copies := []Copy{}
	if len(backupIDs) == 0 {
		return copies, nil
	}

	err := database.Conn(ctx, r.db).Where("backup_id IN ?", backupIDs).
		Order("workspace_name, storage_name, id").Find(&copies).Error
	if err != nil {
		return nil, fmt.Errorf("list Databasus backup copies: %w", err)
	}

	return copies, nil
}

// listDue returns the copies waiting for an attempt at now, the oldest backup first.
func (r *copyRepository) listDue(ctx context.Context, now time.Time) ([]Copy, error) {
	var copies []Copy

	err := database.Conn(ctx, r.db).
		Where("status IN ? AND next_attempt_at <= ?", []CopyStatus{CopyStatusPending, CopyStatusRetrying}, now).
		Order("next_attempt_at, id").Find(&copies).Error
	if err != nil {
		return nil, fmt.Errorf("list due Databasus backup copies: %w", err)
	}

	return copies, nil
}

// startAttempt claims a waiting copy for one upload under a new file name and reports false when
// the copy no longer waits.
func (r *copyRepository) startAttempt(ctx context.Context, copyID uuid.UUID, fileName string) (bool, error) {
	result := database.Conn(ctx, r.db).Exec(`UPDATE instancebackups_copies
		SET status = ?, attempt_count = attempt_count + 1, file_name = ?, next_attempt_at = NULL
		WHERE id = ? AND status IN ?`,
		CopyStatusUploading, fileName, copyID, []CopyStatus{CopyStatusPending, CopyStatusRetrying},
	)
	if result.Error != nil {
		return false, fmt.Errorf("start the Databasus backup copy attempt: %w", result.Error)
	}

	return result.RowsAffected == 1, nil
}

func (r *copyRepository) recordStored(ctx context.Context, copyID uuid.UUID, storedAt time.Time) error {
	err := database.Conn(ctx, r.db).Exec(
		"UPDATE instancebackups_copies SET status = ?, stored_at = ?, last_error = NULL WHERE id = ? AND status = ?",
		CopyStatusStored, storedAt, copyID, CopyStatusUploading,
	).Error
	if err != nil {
		return fmt.Errorf("record the stored Databasus backup copy: %w", err)
	}

	return nil
}

func (r *copyRepository) scheduleRetry(
	ctx context.Context,
	copyID uuid.UUID,
	nextAttemptAt time.Time,
	lastError string,
) error {
	err := database.Conn(ctx, r.db).Exec(`UPDATE instancebackups_copies
		SET status = ?, next_attempt_at = ?, last_error = ?
		WHERE id = ? AND status = ?`,
		CopyStatusRetrying, nextAttemptAt, lastError, copyID, CopyStatusUploading,
	).Error
	if err != nil {
		return fmt.Errorf("schedule the Databasus backup copy retry: %w", err)
	}

	return nil
}

// end gives a copy that did not reach its storage its final status.
func (r *copyRepository) end(ctx context.Context, copyID uuid.UUID, status CopyStatus, lastError string) error {
	err := database.Conn(ctx, r.db).Exec(`UPDATE instancebackups_copies
		SET status = ?, next_attempt_at = NULL, last_error = ?
		WHERE id = ? AND status IN ?`,
		status, lastError, copyID, getUnfinishedCopyStatuses(),
	).Error
	if err != nil {
		return fmt.Errorf("end the Databasus backup copy: %w", err)
	}

	return nil
}

// endUnfinished ends every copy of a backup that has not ended yet.
func (r *copyRepository) endUnfinished(
	ctx context.Context,
	backupID uuid.UUID,
	status CopyStatus,
	lastError string,
) error {
	err := database.Conn(ctx, r.db).Exec(`UPDATE instancebackups_copies
		SET status = ?, next_attempt_at = NULL, last_error = ?
		WHERE backup_id = ? AND status IN ?`,
		status, lastError, backupID, getUnfinishedCopyStatuses(),
	).Error
	if err != nil {
		return fmt.Errorf("end the unfinished Databasus backup copies: %w", err)
	}

	return nil
}

// retryUploading returns the uploads a stopped process left running to the queue. Their files
// were never published, so the file store removes them.
func (r *copyRepository) retryUploading(
	ctx context.Context,
	backupID uuid.UUID,
	now time.Time,
	lastError string,
) error {
	err := database.Conn(ctx, r.db).Exec(`UPDATE instancebackups_copies
		SET status = ?, next_attempt_at = ?, last_error = ?
		WHERE backup_id = ? AND status = ?`,
		CopyStatusRetrying, now, lastError, backupID, CopyStatusUploading,
	).Error
	if err != nil {
		return fmt.Errorf("return interrupted Databasus backup uploads to the queue: %w", err)
	}

	return nil
}

// detachStorages keeps the history of copies in deleted storages, whose files went with them.
func (r *copyRepository) detachStorages(ctx context.Context, storageIDs []uuid.UUID) error {
	err := database.Conn(ctx, r.db).Exec(
		"UPDATE instancebackups_copies SET storage_id = NULL WHERE storage_id IN ?", storageIDs,
	).Error
	if err != nil {
		return fmt.Errorf("detach Databasus backup copies from deleted storages: %w", err)
	}

	return nil
}

// listExpired returns the stored copies in a storage older than storedBefore, always leaving out the
// storage's newest stored copy.
func (r *copyRepository) listExpired(ctx context.Context, storageID uuid.UUID, storedBefore time.Time) ([]Copy, error) {
	var copies []Copy

	err := database.Conn(ctx, r.db).Raw(`SELECT * FROM instancebackups_copies
		WHERE storage_id = ? AND status = ? AND stored_at < ?
		AND id <> (
			SELECT id FROM instancebackups_copies
			WHERE storage_id = ? AND status = ?
			ORDER BY stored_at DESC, id DESC
			LIMIT 1
		)
		ORDER BY stored_at, id`,
		storageID, CopyStatusStored, storedBefore, storageID, CopyStatusStored,
	).Scan(&copies).Error
	if err != nil {
		return nil, fmt.Errorf("list expired Databasus backup copies: %w", err)
	}

	return copies, nil
}

func (r *copyRepository) markDeleted(ctx context.Context, copyIDs []uuid.UUID) error {
	err := database.Conn(ctx, r.db).Exec(
		"UPDATE instancebackups_copies SET status = ? WHERE id IN ? AND status = ?",
		CopyStatusDeleted, copyIDs, CopyStatusStored,
	).Error
	if err != nil {
		return fmt.Errorf("mark Databasus backup copies deleted: %w", err)
	}

	return nil
}

func getUnfinishedCopyStatuses() []CopyStatus {
	return []CopyStatus{CopyStatusPending, CopyStatusUploading, CopyStatusRetrying}
}
