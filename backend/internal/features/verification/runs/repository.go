package verification_runs

import (
	"context"
	"errors"
	"maps"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"databasus-backend/internal/storage"
)

type VerificationRepository struct{}

func (r *VerificationRepository) Create(verification *RestoreVerification) error {
	if verification.ID == uuid.Nil {
		verification.ID = uuid.New()
	}

	return storage.GetDb().Create(verification).Error
}

func (r *VerificationRepository) FindByID(ctx context.Context, id uuid.UUID) (*RestoreVerification, error) {
	var verification RestoreVerification

	err := storage.GetDb().WithContext(ctx).
		Preload("TableStats").
		Where("id = ?", id).
		First(&verification).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &verification, nil
}

func (r *VerificationRepository) FindByDatabaseID(
	databaseID uuid.UUID,
) ([]*RestoreVerification, error) {
	verifications := make([]*RestoreVerification, 0)

	err := storage.GetDb().
		Where("database_id = ?", databaseID).
		Order("created_at DESC").
		Find(&verifications).Error

	return verifications, err
}

func (r *VerificationRepository) FindByDatabaseIDWithPagination(
	databaseID uuid.UUID,
	limit, offset int,
) ([]*RestoreVerification, error) {
	verifications := make([]*RestoreVerification, 0)

	err := storage.GetDb().
		Where("database_id = ?", databaseID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&verifications).Error

	return verifications, err
}

func (r *VerificationRepository) CountByDatabaseID(databaseID uuid.UUID) (int64, error) {
	var count int64

	err := storage.GetDb().
		Model(&RestoreVerification{}).
		Where("database_id = ?", databaseID).
		Count(&count).Error

	return count, err
}

func (r *VerificationRepository) ListRunningBackupIDsByAgentID(agentID uuid.UUID) ([]uuid.UUID, error) {
	backupIDs := make([]uuid.UUID, 0)

	err := storage.GetDb().
		Model(&RestoreVerification{}).
		Where("agent_id = ? AND status = ?", agentID, VerificationStatusRunning).
		Pluck("backup_id", &backupIDs).Error

	return backupIDs, err
}

// LockOldestPendingTop100 locks up to 100 oldest PENDING verification rows with SKIP LOCKED.
// The service walks them in order and claims the first that fits the agent's budget. Locks
// held for the rest are released when the caller's tx ends; SKIP LOCKED keeps concurrent
// agents from queueing on them.
func (r *VerificationRepository) LockOldestPendingTop100(tx *gorm.DB) ([]*RestoreVerification, error) {
	verifications := make([]*RestoreVerification, 0)

	err := tx.
		Where("status = ?", VerificationStatusPending).
		Order("created_at").
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Limit(100).
		Find(&verifications).Error

	return verifications, err
}

func (r *VerificationRepository) UpdateClaim(
	tx *gorm.DB,
	verificationID, agentID uuid.UUID,
	startedAt time.Time,
) error {
	result := tx.Model(&RestoreVerification{}).
		Where("id = ? AND status = ? AND agent_id IS NULL", verificationID, VerificationStatusPending).
		Updates(map[string]any{
			"status":     VerificationStatusRunning,
			"agent_id":   agentID,
			"started_at": startedAt,
		})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("verification claim failed: row no longer PENDING")
	}

	return nil
}

// WriteSuccessReport atomically applies a COMPLETED agent report: CAS-updates the
// verification row (still RUNNING and owned by this agent) and inserts its table
// stats in the same transaction. Returns errVerificationGone when the CAS sees zero
// rows updated — the controller maps that to 410.
func (r *VerificationRepository) WriteSuccessReport(
	verificationID, agentID uuid.UUID,
	report *ReportRequest,
) error {
	return storage.GetDb().Transaction(func(tx *gorm.DB) error {
		fields := map[string]any{
			"status":                      VerificationStatusCompleted,
			"finished_at":                 time.Now().UTC(),
			"pg_restore_exit_code":        report.PgRestoreExitCode,
			"restore_duration_ms":         report.RestoreDurationMs,
			"verify_duration_ms":          report.VerifyDurationMs,
			"db_size_bytes_after_restore": report.DBSizeBytesAfterRestore,
			"table_count":                 report.TableCount,
			"schema_count":                report.SchemaCount,
			"fail_message":                nil,
		}

		result := tx.Model(&RestoreVerification{}).
			Where(
				"id = ? AND agent_id = ? AND status = ?",
				verificationID, agentID, VerificationStatusRunning,
			).
			Updates(fields)
		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return errors.New("verification gone")
		}

		return r.insertTableStats(tx, verificationID, report.TableStats)
	})
}

func (r *VerificationRepository) FindNonTerminalForDatabase(
	tx *gorm.DB,
	databaseID uuid.UUID,
) ([]*RestoreVerification, error) {
	verifications := make([]*RestoreVerification, 0)

	err := tx.
		Where(
			"database_id = ? AND status IN ?",
			databaseID,
			[]VerificationStatus{VerificationStatusPending, VerificationStatusRunning},
		).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Find(&verifications).Error

	return verifications, err
}

func (r *VerificationRepository) FindStalePending(
	threshold time.Time,
) ([]*RestoreVerification, error) {
	verifications := make([]*RestoreVerification, 0)

	err := storage.GetDb().
		Where("status = ? AND created_at < ?", VerificationStatusPending, threshold).
		Find(&verifications).Error

	return verifications, err
}

func (r *VerificationRepository) FindAllRunning() ([]*RestoreVerification, error) {
	verifications := make([]*RestoreVerification, 0)

	err := storage.GetDb().
		Where("status = ?", VerificationStatusRunning).
		Find(&verifications).Error

	return verifications, err
}

func (r *VerificationRepository) FindRunningByAgent(
	agentID uuid.UUID,
) ([]*RestoreVerification, error) {
	verifications := make([]*RestoreVerification, 0)

	err := storage.GetDb().
		Where("status = ? AND agent_id = ?", VerificationStatusRunning, agentID).
		Find(&verifications).Error

	return verifications, err
}

func (r *VerificationRepository) GetNonTerminalAutomaticVerificationsForDisabledSchedules() (
	[]*RestoreVerification,
	error,
) {
	verifications := make([]*RestoreVerification, 0)

	err := storage.GetDb().
		Table("restore_verifications v").
		Select("v.*").
		Joins("JOIN backup_verification_configs c ON c.database_id = v.database_id").
		Where(
			"c.is_scheduled_verification_enabled = ? AND v.trigger IN ? AND v.status IN ?",
			false,
			[]VerificationTrigger{VerificationTriggerScheduled, VerificationTriggerAfterBackup},
			[]VerificationStatus{VerificationStatusPending, VerificationStatusRunning},
		).
		Scan(&verifications).Error

	return verifications, err
}

func (r *VerificationRepository) FindLatestFinishedAt(
	databaseID uuid.UUID,
) (*time.Time, error) {
	var verification RestoreVerification

	err := storage.GetDb().
		Where(
			"database_id = ? AND status IN ? AND finished_at IS NOT NULL",
			databaseID,
			[]VerificationStatus{
				VerificationStatusCompleted,
				VerificationStatusFailed,
				VerificationStatusCanceled,
			},
		).
		Order("finished_at DESC").
		First(&verification).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return verification.FinishedAt, nil
}

// Requeue resets created_at so each requeue gets a fresh PENDING-too-long window.
func (r *VerificationRepository) Requeue(tx *gorm.DB, verificationID uuid.UUID) error {
	db := tx
	if db == nil {
		db = storage.GetDb()
	}

	now := time.Now().UTC()

	result := db.Model(&RestoreVerification{}).
		Where("id = ?", verificationID).
		Updates(map[string]any{
			"status":        VerificationStatusPending,
			"agent_id":      nil,
			"started_at":    nil,
			"finished_at":   nil,
			"created_at":    now,
			"attempt_count": gorm.Expr("attempt_count + 1"),
			"fail_message":  nil,
		})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("requeue failed: verification not found")
	}

	return nil
}

// MarkTerminal runs inside tx when non-nil to avoid deadlock against the caller's
// own row lock (e.g. displacement during EnqueueManualVerification).
func (r *VerificationRepository) MarkTerminal(
	tx *gorm.DB,
	verificationID uuid.UUID,
	status VerificationStatus,
	fields map[string]any,
) error {
	db := tx
	if db == nil {
		db = storage.GetDb()
	}

	updates := make(map[string]any, len(fields)+2)
	maps.Copy(updates, fields)
	updates["status"] = status
	updates["finished_at"] = time.Now().UTC()

	result := db.Model(&RestoreVerification{}).
		Where("id = ?", verificationID).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("mark terminal failed: verification not found")
	}

	return nil
}

func (r *VerificationRepository) insertTableStats(
	tx *gorm.DB,
	verificationID uuid.UUID,
	stats []ReportTableStat,
) error {
	if len(stats) == 0 {
		return nil
	}

	rows := make([]RestoreVerificationTableStat, len(stats))
	for i, stat := range stats {
		rows[i] = RestoreVerificationTableStat{
			ID:                    uuid.New(),
			RestoreVerificationID: verificationID,
			SchemaName:            stat.SchemaName,
			Name:                  stat.Name,
			RowCount:              stat.RowCount,
		}
	}

	return tx.Create(&rows).Error
}
