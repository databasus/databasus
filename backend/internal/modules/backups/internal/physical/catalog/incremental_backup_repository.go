package catalog

import (
	"context"
	"errors"
	"maps"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"databasus-backend/internal/lib/database"
)

type IncrementalBackupRepository struct{ db *gorm.DB }

func NewIncrementalBackupRepository(db *gorm.DB) *IncrementalBackupRepository {
	return &IncrementalBackupRepository{db: db}
}

func (r *IncrementalBackupRepository) Create(ctx context.Context, backup *IncrementalBackup) error {
	return database.Conn(ctx, r.db).Create(backup).Error
}

func (r *IncrementalBackupRepository) FindByID(ctx context.Context, id uuid.UUID,
) (*IncrementalBackup, bool, error) {
	var backup IncrementalBackup

	if err := database.Conn(ctx, r.db).Where("id = ?", id).First(&backup).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}

		return nil, false, err
	}

	return &backup, true, nil
}

func (r *IncrementalBackupRepository) FindLatestCompletedByRootFull(ctx context.Context, rootFullBackupID uuid.UUID,
) (*IncrementalBackup, bool, error) {
	var backup IncrementalBackup

	err := database.Conn(ctx, r.db).
		Where("root_full_backup_id = ? AND status = ?",
			rootFullBackupID, BackupStatusCompleted).
		Order("start_lsn DESC").
		First(&backup).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}

		return nil, false, err
	}

	return &backup, true, nil
}

// FindNewestAnyStatusByRootFull orders by created_at rather than start_lsn
// because failed attempts never get one: persistIncrResult writes LSNs only for
// COMPLETED rows.
func (r *IncrementalBackupRepository) FindNewestAnyStatusByRootFull(ctx context.Context, rootFullBackupID uuid.UUID,
	limit int,
) ([]*IncrementalBackup, error) {
	var backups []*IncrementalBackup

	if err := database.Conn(ctx, r.db).
		Where("root_full_backup_id = ?", rootFullBackupID).
		Order("created_at DESC").
		Limit(limit).
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *IncrementalBackupRepository) FindAllByRootFull(ctx context.Context, rootFullBackupID uuid.UUID,
) ([]*IncrementalBackup, error) {
	var backups []*IncrementalBackup

	if err := database.Conn(ctx, r.db).
		Where("root_full_backup_id = ?", rootFullBackupID).
		Order("start_lsn ASC").
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *IncrementalBackupRepository) FindByIncrementalParent(ctx context.Context, parentIncrementalBackupID uuid.UUID,
) ([]*IncrementalBackup, error) {
	var backups []*IncrementalBackup

	if err := database.Conn(ctx, r.db).
		Where("parent_incremental_backup_id = ?", parentIncrementalBackupID).
		Order("start_lsn ASC").
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *IncrementalBackupRepository) UpdateStatus(ctx context.Context, id uuid.UUID,
	status BackupStatus,
	errorReason *BackupErrorReason, now time.Time,
) error {
	updates := map[string]any{"status": status, "error_reason": errorReason}

	if status == BackupStatusCompleted {
		updates["completed_at"] = new(now)
	}

	return database.Conn(ctx, r.db).
		Model(&IncrementalBackup{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// UpdateProgress writes the in-flight size and duration for a still-running
// backup. It touches only those two columns and is gated on IN_PROGRESS — the
// same supersession guard as saveTerminalResultIfInProgress — so a late progress
// write can never resurrect a row a restart-recovery sweep already finalized.
func (r *IncrementalBackupRepository) UpdateProgress(
	ctx context.Context,
	id uuid.UUID,
	sizeMb float64,
	durationMs int64,
) error {
	return database.Conn(ctx, r.db).
		Model(&IncrementalBackup{}).
		Where("id = ? AND status = ?", id, BackupStatusInProgress).
		Updates(map[string]any{"backup_size_mb": sizeMb, "backup_duration_ms": durationMs}).Error
}

func (r *IncrementalBackupRepository) DeleteByID(ctx context.Context, id uuid.UUID) error {
	return database.Conn(ctx, r.db).Delete(&IncrementalBackup{}, "id = ?", id).Error
}

func (r *IncrementalBackupRepository) FindLastByDatabase(ctx context.Context, databaseID uuid.UUID,
) (*IncrementalBackup, bool, error) {
	var backup IncrementalBackup

	err := database.Conn(ctx, r.db).
		Where("database_id = ?", databaseID).
		Order("created_at DESC").
		First(&backup).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}

		return nil, false, err
	}

	return &backup, true, nil
}

// FindAllInProgress returns every INCR still marked IN_PROGRESS across all
// databases — the INCR counterpart to the FULL restart sweep.
func (r *IncrementalBackupRepository) FindAllInProgress(ctx context.Context) ([]*IncrementalBackup, error) {
	var backups []*IncrementalBackup

	if err := database.Conn(ctx, r.db).
		Where("status = ?", BackupStatusInProgress).
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *IncrementalBackupRepository) UpdateAttemptFileName(ctx context.Context, backup *IncrementalBackup) error {
	return database.Conn(ctx, r.db).
		Model(&IncrementalBackup{}).
		Where("id = ?", backup.ID).
		Update("file_name", backup.FileName).
		Error
}

func (r *IncrementalBackupRepository) UpdateTerminalResult(ctx context.Context, backup *IncrementalBackup) error {
	updates := map[string]any{
		"status":       backup.Status,
		"error_reason": backup.ErrorReason,
		"fail_message": backup.FailMessage,
		"completed_at": backup.CompletedAt,
	}
	if backup.Status == BackupStatusCompleted {
		maps.Copy(updates, map[string]any{
			"timeline_id":              backup.TimelineID,
			"start_lsn":                backup.StartLSN,
			"stop_lsn":                 backup.StopLSN,
			"backup_size_mb":           backup.BackupSizeMb,
			"backup_duration_ms":       backup.BackupDurationMs,
			"encryption":               backup.Encryption,
			"encryption_salt":          backup.EncryptionSalt,
			"encryption_iv":            backup.EncryptionIV,
			"compression":              backup.Compression,
			"manifest_file_name":       backup.ManifestFileName,
			"manifest_encryption_salt": backup.ManifestEncryptionSalt,
			"manifest_encryption_iv":   backup.ManifestEncryptionIV,
		})
	}
	return database.Conn(ctx, r.db).Model(&IncrementalBackup{}).Where("id = ?", backup.ID).Updates(updates).Error
}

func (r *IncrementalBackupRepository) UpdateFailure(ctx context.Context, backup *IncrementalBackup) error {
	updates := map[string]any{
		"status":       backup.Status,
		"error_reason": backup.ErrorReason,
		"fail_message": backup.FailMessage,
	}
	updates["completed_at"] = backup.CompletedAt
	return database.Conn(ctx, r.db).Model(&IncrementalBackup{}).Where("id = ?", backup.ID).Updates(updates).Error
}

func (r *IncrementalBackupRepository) GetStatusForUpdate(
	ctx context.Context,
	id uuid.UUID,
) (BackupStatus, bool, error) {
	var backup IncrementalBackup
	err := database.Conn(ctx, r.db).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("status").
		Where("id = ?", id).
		First(&backup).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return backup.Status, true, nil
}
