package catalog

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

const oneRunningDumpIndex = "uq_instancebackups_backups_one_running_dump"

// historyOrder orders backups newest snapshot first; a backup without a snapshot sorts by its start.
const historyOrder = "COALESCE(snapshot_at, started_at) DESC, id DESC"

// Repositories are the repositories of the package; module.go builds them from the pool.
type Repositories struct {
	backups *backupRepository
	copies  *copyRepository
}

func NewRepositories(db *gorm.DB) Repositories {
	return Repositories{backups: &backupRepository{db: db}, copies: &copyRepository{db: db}}
}

type backupRepository struct {
	db *gorm.DB
}

// insertRunning reports false when another dump is already running.
func (r *backupRepository) insertRunning(ctx context.Context, backup Backup) (bool, error) {
	err := database.Conn(ctx, r.db).Create(&backup).Error
	if database.IsUniqueViolation(err, oneRunningDumpIndex) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("insert the Databasus backup: %w", err)
	}

	return true, nil
}

func (r *backupRepository) get(ctx context.Context, backupID uuid.UUID) (Backup, bool, error) {
	var backups []Backup

	err := database.Conn(ctx, r.db).Where("id = ?", backupID).Limit(1).Find(&backups).Error
	if err != nil {
		return Backup{}, false, fmt.Errorf("read the Databasus backup: %w", err)
	}

	if len(backups) == 0 {
		return Backup{}, false, nil
	}

	return backups[0], true, nil
}

func (r *backupRepository) getRunning(ctx context.Context) (Backup, bool, error) {
	var backups []Backup

	err := database.Conn(ctx, r.db).Where("dump_status = ?", DumpStatusRunning).Limit(1).Find(&backups).Error
	if err != nil {
		return Backup{}, false, fmt.Errorf("read the running Databasus backup: %w", err)
	}

	if len(backups) == 0 {
		return Backup{}, false, nil
	}

	return backups[0], true, nil
}

func (r *backupRepository) getLatestStartedAt(ctx context.Context) (*time.Time, error) {
	var startedAts []time.Time

	err := database.Conn(ctx, r.db).Raw("SELECT MAX(started_at) FROM instancebackups_backups HAVING COUNT(*) > 0").
		Scan(&startedAts).Error
	if err != nil {
		return nil, fmt.Errorf("read the latest Databasus backup start: %w", err)
	}

	if len(startedAts) == 0 {
		return nil, nil
	}

	return &startedAts[0], nil
}

func (r *backupRepository) listPage(ctx context.Context, after *database.TimeCursor, limit int) ([]Backup, error) {
	var backups []Backup

	query := database.Conn(ctx, r.db).Order(historyOrder).Limit(limit)
	if after != nil {
		query = query.Where("(COALESCE(snapshot_at, started_at), id) < (?, ?)", after.Time, after.ID)
	}

	if err := query.Find(&backups).Error; err != nil {
		return nil, fmt.Errorf("list Databasus backups: %w", err)
	}

	return backups, nil
}

// listSpooled returns the backups whose spool file copies may still need.
func (r *backupRepository) listSpooled(ctx context.Context) ([]Backup, error) {
	var backups []Backup

	err := database.Conn(ctx, r.db).Where("spool_path IS NOT NULL").Order("started_at, id").Find(&backups).Error
	if err != nil {
		return nil, fmt.Errorf("list spooled Databasus backups: %w", err)
	}

	return backups, nil
}

func (r *backupRepository) recordDumped(ctx context.Context, dumped Backup) error {
	err := database.Conn(ctx, r.db).Exec(`UPDATE instancebackups_backups
		SET dump_status = ?, snapshot_at = ?, size_bytes = ?, spool_path = ?, spool_expires_at = ?
		WHERE id = ?`,
		DumpStatusSucceeded, dumped.SnapshotAt, dumped.SizeBytes, dumped.SpoolPath, dumped.SpoolExpiresAt, dumped.ID,
	).Error
	if err != nil {
		return fmt.Errorf("record the Databasus backup dump: %w", err)
	}

	return nil
}

// endDump ends a running dump whose backup ends with it, because no copy can follow.
func (r *backupRepository) endDump(ctx context.Context, backupID uuid.UUID, ended dumpEnd) error {
	err := database.Conn(ctx, r.db).Exec(`UPDATE instancebackups_backups
		SET dump_status = ?, status = ?, failure_reason = ?, completed_at = ?
		WHERE id = ? AND dump_status = ?`,
		ended.DumpStatus, ended.Status, ended.FailureReason, ended.CompletedAt, backupID, DumpStatusRunning,
	).Error
	if err != nil {
		return fmt.Errorf("end the Databasus backup dump: %w", err)
	}

	return nil
}

// interruptRunningDumps ends the dumps a stopped process left running.
func (r *backupRepository) interruptRunningDumps(ctx context.Context, now time.Time) error {
	err := database.Conn(ctx, r.db).Exec(`UPDATE instancebackups_backups
		SET dump_status = ?, status = ?, completed_at = ?
		WHERE dump_status = ?`,
		DumpStatusInterrupted, StatusInterrupted, now, DumpStatusRunning,
	).Error
	if err != nil {
		return fmt.Errorf("interrupt running Databasus backup dumps: %w", err)
	}

	return nil
}

// finish records the outcome of a backup whose copies all ended and forgets its spool file.
func (r *backupRepository) finish(ctx context.Context, backupID uuid.UUID, status Status, now time.Time) error {
	err := database.Conn(ctx, r.db).Exec(`UPDATE instancebackups_backups
		SET status = ?, completed_at = ?, spool_path = NULL, spool_expires_at = NULL
		WHERE id = ? AND status = ?`,
		status, now, backupID, StatusInProgress,
	).Error
	if err != nil {
		return fmt.Errorf("finish the Databasus backup: %w", err)
	}

	return nil
}
