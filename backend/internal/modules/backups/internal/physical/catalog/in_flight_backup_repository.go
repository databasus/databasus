package catalog

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	"databasus-backend/internal/lib/database"
)

type InFlightBackupRepository struct{ db *gorm.DB }

func NewInFlightBackupRepository(db *gorm.DB) *InFlightBackupRepository {
	return &InFlightBackupRepository{db: db}
}

type ClaimSpec struct {
	DatabaseID uuid.UUID
	BackupType BackupType
	BackupID   uuid.UUID
}

// The claim and typed backup row must commit in the same transaction, carried by ctx.
func (r *InFlightBackupRepository) Claim(ctx context.Context, spec ClaimSpec, now time.Time) (bool, error) {
	if spec.DatabaseID == uuid.Nil || spec.BackupID == uuid.Nil {
		return false, errors.New("database ID and backup ID are required")
	}

	result := database.Conn(ctx, r.db).Exec(
		`INSERT INTO backups_physical_in_flight (database_id, backup_type, backup_id, claimed_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT (database_id) DO NOTHING`,
		spec.DatabaseID, spec.BackupType, spec.BackupID, now,
	)
	if result.Error != nil {
		return false, result.Error
	}

	return result.RowsAffected > 0, nil
}

// FindAll returns every in-flight claim. Restart recovery reads them to fail and
// release each orphaned IN_PROGRESS backup whose goroutine died with the process.
func (r *InFlightBackupRepository) FindAll(ctx context.Context) ([]*InFlightBackup, error) {
	var claims []*InFlightBackup

	if err := database.Conn(ctx, r.db).Find(&claims).Error; err != nil {
		return nil, err
	}

	return claims, nil
}

// Release deletes whatever in-flight claim a database currently holds,
// regardless of which backup owns it. Use it only for teardown — config
// disable, database removal, test cleanup — where the database is being stood
// down and any claim must go. A backuper finishing its own backup must use
// ReleaseOwned so it cannot delete a claim that a newer backup already took.
func (r *InFlightBackupRepository) Release(ctx context.Context, databaseID uuid.UUID) error {
	return database.Conn(ctx, r.db).
		Delete(&InFlightBackup{}, "database_id = ?", databaseID).Error
}

// ReleaseOwned deletes the in-flight claim only while backupID still owns it.
// A backuper that finishes after the scheduler restarted and re-scheduled the
// database would otherwise delete the newer backup's claim (the claim is keyed
// by database_id), silently breaking the single-in-flight invariant; scoping
// the delete to backup_id turns that stale release into a no-op.
func (r *InFlightBackupRepository) ReleaseOwned(ctx context.Context, databaseID, backupID uuid.UUID) error {
	return database.Conn(ctx, r.db).
		Delete(
			&InFlightBackup{},
			"database_id = ? AND backup_id = ?",
			databaseID,
			backupID,
		).Error
}

func (r *InFlightBackupRepository) FindByDatabaseID(ctx context.Context, databaseID uuid.UUID,
) (*InFlightBackup, bool, error) {
	var row InFlightBackup

	if err := database.Conn(ctx, r.db).Where("database_id = ?", databaseID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}

		return nil, false, err
	}

	return &row, true, nil
}

func (r *InFlightBackupRepository) AcquireBackupAndOrphanCleanupLock(ctx context.Context, databaseID uuid.UUID) error {
	return database.Conn(ctx, r.db).Exec(
		`SELECT pg_advisory_xact_lock(hashtextextended('physical-backup:' || ?::text, 0))`,
		databaseID,
	).Error
}

func (r *InFlightBackupRepository) HasInFlightFullBackup(ctx context.Context, databaseID uuid.UUID) (bool, error) {
	var claim InFlightBackup

	err := database.Conn(ctx, r.db).
		Where("database_id = ? AND backup_type = ?", databaseID, BackupTypeFull).
		First(&claim).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (r *InFlightBackupRepository) HasCompletedFullBackupCoveringWal(
	ctx context.Context,
	databaseID uuid.UUID,
	timelineID int,
	walEndLSN walmath.LSN,
) (bool, error) {
	var fullCount int64

	err := database.Conn(ctx, r.db).Model(&FullBackup{}).
		Where(
			"database_id = ? AND timeline_id = ? AND status = ? AND start_lsn IS NOT NULL AND start_lsn < ?::pg_lsn",
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
