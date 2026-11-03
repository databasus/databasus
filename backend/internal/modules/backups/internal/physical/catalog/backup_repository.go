package catalog

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	"databasus-backend/internal/lib/database"
)

type BackupRepository struct{ db *gorm.DB }

func NewBackupRepository(db *gorm.DB) *BackupRepository { return &BackupRepository{db: db} }

func (r *BackupRepository) FindByID(ctx context.Context, id uuid.UUID) (*Backup, bool, error) {
	var backup Backup

	if err := database.Conn(ctx, r.db).Where("id = ?", id).First(&backup).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}

		return nil, false, err
	}

	return &backup, true, nil
}

func (r *BackupRepository) FindCompletedFullNewestFirstByDatabase(ctx context.Context, databaseID uuid.UUID,
) ([]*Backup, error) {
	var backups []*Backup

	if err := database.Conn(ctx, r.db).
		Where("kind = 'FULL' AND database_id = ? AND status = ?", databaseID, BackupStatusCompleted).
		Order("created_at DESC").
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *BackupRepository) UpdateProgress(
	ctx context.Context,
	id uuid.UUID,
	sizeMb float64,
	durationMs int64,
) error {
	return database.Conn(ctx, r.db).
		Model(&Backup{}).
		Where("id = ? AND status = ?", id, BackupStatusInProgress).
		Updates(map[string]any{"backup_size_mb": sizeMb, "backup_duration_ms": durationMs}).Error
}

func (r *BackupRepository) DeleteByID(ctx context.Context, id uuid.UUID) error {
	return database.Conn(ctx, r.db).Delete(&Backup{}, "id = ?", id).Error
}

func (r *BackupRepository) FindNewestFullAnyStatusByDatabase(ctx context.Context, databaseID uuid.UUID,
	limit int,
) ([]*Backup, error) {
	var backups []*Backup

	if err := database.Conn(ctx, r.db).
		Where("kind = 'FULL' AND database_id = ?", databaseID).
		Order("created_at DESC").
		Limit(limit).
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *BackupRepository) FindAllFullByDatabaseOutsideStorage(ctx context.Context, databaseID uuid.UUID,
	storageID uuid.UUID,
) ([]*Backup, error) {
	var backups []*Backup

	if err := database.Conn(ctx, r.db).
		Where("kind = 'FULL' AND database_id = ? AND storage_id <> ?", databaseID, storageID).
		Order("created_at ASC").
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *BackupRepository) FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*Backup, bool, error) {
	var backup Backup
	err := database.Conn(ctx, r.db).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&backup).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return &backup, true, nil
}

func (r *BackupRepository) FindNextCompletedFull(ctx context.Context, full *Backup) (*Backup, bool, error) {
	var successor Backup
	err := database.Conn(ctx, r.db).
		Where("kind = 'FULL' AND database_id = ? AND timeline_id = ? AND status = ? AND start_lsn > ?::pg_lsn", full.DatabaseID, full.TimelineID, BackupStatusCompleted, full.StartLSN.String()).
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

func (r *BackupRepository) HasCompletedFullBefore(ctx context.Context, full *Backup) (bool, error) {
	var count int64
	err := database.Conn(ctx, r.db).
		Model(&Backup{}).
		Where("kind = 'FULL' AND database_id = ? AND timeline_id = ? AND status = ? AND start_lsn IS NOT NULL AND start_lsn < ?::pg_lsn", full.DatabaseID, full.TimelineID, BackupStatusCompleted, full.StartLSN.String()).
		Count(&count).
		Error
	return count > 0, err
}

func (r *BackupRepository) HasSurvivingCompletedFullOnTimeline(ctx context.Context, full *Backup) (bool, error) {
	var count int64
	err := database.Conn(ctx, r.db).
		Model(&Backup{}).
		Where("kind = 'FULL' AND database_id = ? AND timeline_id = ? AND status = ? AND id != ?", full.DatabaseID, full.TimelineID, BackupStatusCompleted, full.ID).
		Count(&count).
		Error
	return count > 0, err
}

func (r *BackupRepository) GetOwnershipSpan(ctx context.Context, full *Backup) (LSNRange, error) {
	if full.StartLSN == nil {
		return LSNRange{}, errors.New("cannot determine WAL ownership without FULL start_lsn")
	}

	successor, isFound, err := r.FindNextCompletedFull(ctx, full)
	if err != nil {
		return LSNRange{}, err
	}

	end := LSNMax
	if isFound && successor.StartLSN != nil {
		end = *successor.StartLSN
	}
	return LSNRange{Start: *full.StartLSN, End: end}, nil
}

func (r *BackupRepository) UpdateAttemptFileName(ctx context.Context, backup *Backup) error {
	return database.Conn(ctx, r.db).
		Model(&Backup{}).
		Where("id = ?", backup.ID).
		Update("file_name", backup.FileName).
		Error
}

func (r *BackupRepository) FindLatestCompletedByRootFull(ctx context.Context, rootFullBackupID uuid.UUID,
) (*Backup, bool, error) {
	var backup Backup

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
// because failed attempts have no start LSN.
func (r *BackupRepository) FindNewestAnyStatusByRootFull(ctx context.Context, rootFullBackupID uuid.UUID,
	limit int,
) ([]*Backup, error) {
	var backups []*Backup

	if err := database.Conn(ctx, r.db).
		Where("root_full_backup_id = ?", rootFullBackupID).
		Order("created_at DESC").
		Limit(limit).
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *BackupRepository) FindAllByRootFull(ctx context.Context, rootFullBackupID uuid.UUID,
) ([]*Backup, error) {
	var backups []*Backup

	if err := database.Conn(ctx, r.db).
		Where("root_full_backup_id = ?", rootFullBackupID).
		Order("start_lsn ASC").
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *BackupRepository) AcquireBackupAndOrphanCleanupLock(ctx context.Context, databaseID uuid.UUID) error {
	return database.Conn(ctx, r.db).Exec(
		`SELECT pg_advisory_xact_lock(hashtextextended('physical-backup:' || ?::text, 0))`,
		databaseID,
	).Error
}

func (r *BackupRepository) HasCompletedFullBackupCoveringWal(
	ctx context.Context,
	databaseID uuid.UUID,
	timelineID int,
	walEndLSN walmath.LSN,
) (bool, error) {
	var fullCount int64

	err := database.Conn(ctx, r.db).Model(&Backup{}).
		Where(
			"kind = 'FULL' AND database_id = ? AND timeline_id = ? AND status = ? AND start_lsn IS NOT NULL AND start_lsn < ?::pg_lsn",
			databaseID,
			timelineID,
			BackupStatusCompleted,
			walEndLSN.String(),
		).
		Count(&fullCount).Error
	if err != nil {
		return false, err
	}

	return fullCount > 0, nil
}

func (r *BackupRepository) InsertRunning(ctx context.Context, backup *Backup) (bool, error) {
	insertion := database.Conn(ctx, r.db).Clauses(clause.OnConflict{
		Columns:     []clause.Column{{Name: "database_id"}},
		TargetWhere: clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "status = 'IN_PROGRESS'"}}},
		DoNothing:   true,
	}).Create(backup)
	if insertion.Error != nil {
		return false, fmt.Errorf("insert running physical backup: %w", insertion.Error)
	}
	return insertion.RowsAffected == 1, nil
}

func (r *BackupRepository) MarkCompleted(ctx context.Context, backup *Backup) (bool, error) {
	return r.markTerminal(ctx, backup, BackupStatusCompleted, map[string]any{
		"file_name":                backup.FileName,
		"timeline_id":              backup.TimelineID,
		"start_lsn":                backup.StartLSN,
		"stop_lsn":                 backup.StopLSN,
		"backup_size_mb":           backup.BackupSizeMb,
		"raw_size_mb":              backup.RawSizeMb,
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

func (r *BackupRepository) MarkFailed(ctx context.Context, backup *Backup) (bool, error) {
	return r.markTerminal(ctx, backup, BackupStatusError, map[string]any{})
}

func (r *BackupRepository) MarkChainBroken(ctx context.Context, backup *Backup) (bool, error) {
	return r.markTerminal(ctx, backup, BackupStatusChainBroken, map[string]any{})
}

func (r *BackupRepository) MarkCanceled(ctx context.Context, backup *Backup) (bool, error) {
	return r.markTerminal(ctx, backup, BackupStatusCanceled, map[string]any{})
}

func (r *BackupRepository) FindAllRunning(ctx context.Context) ([]*Backup, error) {
	var backups []*Backup
	if err := database.Conn(ctx, r.db).Where("status = ?", BackupStatusInProgress).Find(&backups).Error; err != nil {
		return nil, fmt.Errorf("list running physical backups: %w", err)
	}
	return backups, nil
}

func (r *BackupRepository) FindRunningByDatabaseID(ctx context.Context, databaseID uuid.UUID) (*Backup, bool, error) {
	var backup Backup
	err := database.Conn(ctx, r.db).
		Where("database_id = ? AND status = ?", databaseID, BackupStatusInProgress).
		First(&backup).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("find running physical backup: %w", err)
	}
	return &backup, true, nil
}

func (r *BackupRepository) HasRunningFullBackup(ctx context.Context, databaseID uuid.UUID) (bool, error) {
	var count int64
	err := database.Conn(ctx, r.db).Model(&Backup{}).
		Where("database_id = ? AND kind = ? AND status = ?", databaseID, BackupKindFull, BackupStatusInProgress).
		Count(&count).Error
	return count > 0, err
}

// NewestBackups holds the newest backup of each kind of one database, whatever their status.
type NewestBackups struct {
	Full        *Backup
	Incremental *Backup
}

// GetNewestBackupsByDatabases looks up each database and kind through the index on database, kind and
// creation time, so the read does not walk the history of a database.
func (r *BackupRepository) GetNewestBackupsByDatabases(
	ctx context.Context,
	databaseIDs []uuid.UUID,
) (map[uuid.UUID]NewestBackups, error) {
	newestBackups := make(map[uuid.UUID]NewestBackups, len(databaseIDs))
	if len(databaseIDs) == 0 {
		return newestBackups, nil
	}

	var backups []*Backup
	err := database.Conn(ctx, r.db).Raw(`
		SELECT newest.*
		FROM unnest(?::uuid[]) AS requested (database_id)
		CROSS JOIN (VALUES ('FULL'), ('INCREMENTAL')) AS kinds (kind)
		CROSS JOIN LATERAL (
			SELECT * FROM backups_physical
			WHERE database_id = requested.database_id AND kind = kinds.kind
			ORDER BY created_at DESC
			LIMIT 1
		) AS newest`, pq.Array(databaseIDs)).Scan(&backups).Error
	if err != nil {
		return nil, err
	}

	for _, backup := range backups {
		databaseBackups := newestBackups[backup.DatabaseID]
		if backup.Kind == BackupKindFull {
			databaseBackups.Full = backup
		} else {
			databaseBackups.Incremental = backup
		}

		newestBackups[backup.DatabaseID] = databaseBackups
	}

	return newestBackups, nil
}

func (r *BackupRepository) GetLatestCompletedFullBackups(
	ctx context.Context,
	databaseIDs []uuid.UUID,
) (map[uuid.UUID]Backup, error) {
	indexed := make(map[uuid.UUID]Backup, len(databaseIDs))
	if len(databaseIDs) == 0 {
		return indexed, nil
	}

	var backups []Backup
	err := database.Conn(ctx, r.db).Raw(`SELECT DISTINCT ON (database_id) * FROM backups_physical
 WHERE database_id IN ? AND kind = 'FULL' AND status = ?
 ORDER BY database_id, created_at DESC, id DESC`, databaseIDs, BackupStatusCompleted).Scan(&backups).Error
	if err != nil {
		return nil, err
	}

	for _, backup := range backups {
		indexed[backup.DatabaseID] = backup
	}

	return indexed, nil
}

func (r *BackupRepository) markTerminal(
	ctx context.Context,
	backup *Backup,
	status BackupStatus,
	columns map[string]any,
) (bool, error) {
	columns["status"] = status
	columns["error_reason"] = backup.ErrorReason
	columns["fail_message"] = backup.FailMessage
	columns["completed_at"] = backup.CompletedAt

	transition := database.Conn(ctx, r.db).Model(&Backup{}).
		Where("id = ? AND status = ?", backup.ID, BackupStatusInProgress).Updates(columns)
	if transition.Error != nil {
		return false, fmt.Errorf("mark physical backup %s: %w", status, transition.Error)
	}
	return transition.RowsAffected == 1, nil
}
