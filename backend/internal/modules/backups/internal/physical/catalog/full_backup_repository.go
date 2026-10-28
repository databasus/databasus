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

type FullBackupRepository struct{ db *gorm.DB }

func NewFullBackupRepository(db *gorm.DB) *FullBackupRepository { return &FullBackupRepository{db: db} }

func (r *FullBackupRepository) Create(ctx context.Context, backup *FullBackup) error {
	return database.Conn(ctx, r.db).Create(backup).Error
}

func (r *FullBackupRepository) FindByID(ctx context.Context, id uuid.UUID) (*FullBackup, bool, error) {
	var backup FullBackup

	if err := database.Conn(ctx, r.db).Where("id = ?", id).First(&backup).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}

		return nil, false, err
	}

	return &backup, true, nil
}

func (r *FullBackupRepository) FindCompletedNewestFirstByDatabase(ctx context.Context, databaseID uuid.UUID,
) ([]*FullBackup, error) {
	var backups []*FullBackup

	if err := database.Conn(ctx, r.db).
		Where("database_id = ? AND status = ?", databaseID, BackupStatusCompleted).
		Order("created_at DESC").
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *FullBackupRepository) FindAllForGFS(ctx context.Context, databaseID uuid.UUID,
) ([]*FullBackup, error) {
	var backups []*FullBackup

	if err := database.Conn(ctx, r.db).
		Where("database_id = ? AND status = ?", databaseID, BackupStatusCompleted).
		Order("created_at ASC").
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *FullBackupRepository) UpdateStatus(ctx context.Context, id uuid.UUID,
	status BackupStatus,
	errorReason *BackupErrorReason, now time.Time,
) error {
	updates := map[string]any{"status": status, "error_reason": errorReason}

	if status == BackupStatusCompleted {
		updates["completed_at"] = new(now)
	}

	return database.Conn(ctx, r.db).
		Model(&FullBackup{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// UpdateProgress writes the in-flight size and duration for a still-running
// backup. It touches only those two columns and is gated on IN_PROGRESS — the
// same supersession guard as saveTerminalResultIfInProgress — so a late progress
// write can never resurrect a row a restart-recovery sweep already finalized.
func (r *FullBackupRepository) UpdateProgress(
	ctx context.Context,
	id uuid.UUID,
	sizeMb float64,
	durationMs int64,
) error {
	return database.Conn(ctx, r.db).
		Model(&FullBackup{}).
		Where("id = ? AND status = ?", id, BackupStatusInProgress).
		Updates(map[string]any{"backup_size_mb": sizeMb, "backup_duration_ms": durationMs}).Error
}

func (r *FullBackupRepository) DeleteByID(ctx context.Context, id uuid.UUID) error {
	return database.Conn(ctx, r.db).Delete(&FullBackup{}, "id = ?", id).Error
}

func (r *FullBackupRepository) FindNewestAnyStatusByDatabase(ctx context.Context, databaseID uuid.UUID,
	limit int,
) ([]*FullBackup, error) {
	var backups []*FullBackup

	if err := database.Conn(ctx, r.db).
		Where("database_id = ?", databaseID).
		Order("created_at DESC").
		Limit(limit).
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *FullBackupRepository) FindLastFullAnyStatusByDatabase(ctx context.Context, databaseID uuid.UUID,
) (*FullBackup, bool, error) {
	backups, err := r.FindNewestAnyStatusByDatabase(ctx, databaseID, 1)
	if err != nil {
		return nil, false, err
	}

	if len(backups) == 0 {
		return nil, false, nil
	}

	return backups[0], true, nil
}

func (r *FullBackupRepository) FindAllByDatabaseOutsideStorage(ctx context.Context, databaseID uuid.UUID,
	storageID uuid.UUID,
) ([]*FullBackup, error) {
	var backups []*FullBackup

	if err := database.Conn(ctx, r.db).
		Where("database_id = ? AND storage_id <> ?", databaseID, storageID).
		Order("created_at ASC").
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *FullBackupRepository) FindAllInProgress(ctx context.Context) ([]*FullBackup, error) {
	var backups []*FullBackup

	if err := database.Conn(ctx, r.db).
		Where("status = ?", BackupStatusInProgress).
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *FullBackupRepository) FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*FullBackup, bool, error) {
	var backup FullBackup
	err := database.Conn(ctx, r.db).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&backup).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return &backup, true, nil
}

func (r *FullBackupRepository) FindNextCompleted(ctx context.Context, full *FullBackup) (*FullBackup, bool, error) {
	var successor FullBackup
	err := database.Conn(ctx, r.db).
		Where("database_id = ? AND timeline_id = ? AND status = ? AND start_lsn > ?::pg_lsn", full.DatabaseID, full.TimelineID, BackupStatusCompleted, full.StartLSN.String()).
		Order("start_lsn ASC").
		First(&successor).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return &successor, true, nil
}

func (r *FullBackupRepository) HasCompletedBefore(ctx context.Context, full *FullBackup) (bool, error) {
	var count int64
	err := database.Conn(ctx, r.db).
		Model(&FullBackup{}).
		Where("database_id = ? AND timeline_id = ? AND status = ? AND start_lsn IS NOT NULL AND start_lsn < ?::pg_lsn", full.DatabaseID, full.TimelineID, BackupStatusCompleted, full.StartLSN.String()).
		Count(&count).
		Error
	return count > 0, err
}

func (r *FullBackupRepository) HasSurvivingCompletedOnTimeline(ctx context.Context, full *FullBackup) (bool, error) {
	var count int64
	err := database.Conn(ctx, r.db).
		Model(&FullBackup{}).
		Where("database_id = ? AND timeline_id = ? AND status = ? AND id != ?", full.DatabaseID, full.TimelineID, BackupStatusCompleted, full.ID).
		Count(&count).
		Error
	return count > 0, err
}

func (r *FullBackupRepository) GetOwnershipSpan(ctx context.Context, full *FullBackup) (LSNRange, error) {
	if full.StartLSN == nil {
		return LSNRange{}, errors.New("cannot determine WAL ownership without FULL start_lsn")
	}
	successor, isFound, err := r.FindNextCompleted(ctx, full)
	if err != nil {
		return LSNRange{}, err
	}
	end := LSNMax
	if isFound && successor.StartLSN != nil {
		end = *successor.StartLSN
	}
	return LSNRange{Start: *full.StartLSN, End: end}, nil
}

func (r *FullBackupRepository) UpdateAttemptFileName(ctx context.Context, backup *FullBackup) error {
	return database.Conn(ctx, r.db).
		Model(&FullBackup{}).
		Where("id = ?", backup.ID).
		Update("file_name", backup.FileName).
		Error
}

func (r *FullBackupRepository) UpdateTerminalResult(ctx context.Context, backup *FullBackup) error {
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
			"raw_size_mb":              backup.RawSizeMb,
		})
	}
	return database.Conn(ctx, r.db).Model(&FullBackup{}).Where("id = ?", backup.ID).Updates(updates).Error
}

func (r *FullBackupRepository) UpdateFailure(ctx context.Context, backup *FullBackup) error {
	updates := map[string]any{
		"status":       backup.Status,
		"error_reason": backup.ErrorReason,
		"fail_message": backup.FailMessage,
	}

	return database.Conn(ctx, r.db).Model(&FullBackup{}).Where("id = ?", backup.ID).Updates(updates).Error
}

func (r *FullBackupRepository) GetStatusForUpdate(ctx context.Context, id uuid.UUID) (BackupStatus, bool, error) {
	var backup FullBackup
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
