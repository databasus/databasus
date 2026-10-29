package verifications

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"databasus-backend/internal/lib/database"
)

type Repositories struct{ verifications *verificationRepository }

func NewRepositories(db *gorm.DB) Repositories {
	return Repositories{verifications: &verificationRepository{db: db}}
}

type verificationRepository struct{ db *gorm.DB }

func (r *verificationRepository) Create(ctx context.Context, verification *Verification) error {
	if verification.ID == uuid.Nil {
		verification.ID = uuid.New()
	}

	return database.Conn(ctx, r.db).Create(verification).Error
}

func (r *verificationRepository) GetVerification(ctx context.Context, id uuid.UUID) (*Verification, bool, error) {
	var verification Verification

	err := database.Conn(ctx, r.db).
		Where("id = ?", id).
		First(&verification).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}

		return nil, false, err
	}

	if err := database.Conn(ctx, r.db).
		Where("run_id = ?", id).
		Order("schema_name, name, id").
		Find(&verification.TableStats).
		Error; err != nil {
		return nil, false, fmt.Errorf("get verification table stats: %w", err)
	}
	return &verification, true, nil
}

func (r *verificationRepository) ListRunningBackupIDsByAgentID(
	ctx context.Context,
	agentID uuid.UUID,
) ([]uuid.UUID, error) {
	backupIDs := make([]uuid.UUID, 0)

	err := database.Conn(ctx, r.db).
		Model(&Verification{}).
		Where("agent_id = ? AND status = ?", agentID, VerificationStatusRunning).
		Pluck("backup_id", &backupIDs).Error

	return backupIDs, wrapQueryError(err)
}

func (r *verificationRepository) LockOldestPendingTop100(ctx context.Context) ([]*Verification, error) {
	verifications := make([]*Verification, 0)

	err := database.Conn(ctx, r.db).
		Where("status = ?", VerificationStatusPending).
		Order("created_at").
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Limit(100).
		Find(&verifications).Error

	return verifications, wrapQueryError(err)
}

type ClaimUpdate struct {
	VerificationID uuid.UUID
	AgentID        uuid.UUID
	StartedAt      time.Time
}

func (r *verificationRepository) UpdateClaim(ctx context.Context, claim ClaimUpdate) error {
	result := database.Conn(ctx, r.db).Model(&Verification{}).
		Where("id = ? AND status = ? AND agent_id IS NULL", claim.VerificationID, VerificationStatusPending).
		Updates(map[string]any{
			"status":     VerificationStatusRunning,
			"agent_id":   claim.AgentID,
			"started_at": claim.StartedAt,
		})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errClaimConflict
	}

	return nil
}

type SuccessReport struct {
	VerificationID uuid.UUID
	AgentID        uuid.UUID
	Report         *ReportRequest
	FinishedAt     time.Time
}

func (r *verificationRepository) WriteSuccessReport(ctx context.Context, success SuccessReport) error {
	fields := map[string]any{
		"status":                      VerificationStatusCompleted,
		"finished_at":                 success.FinishedAt,
		"pg_restore_exit_code":        success.Report.PgRestoreExitCode,
		"restore_duration_ms":         success.Report.RestoreDurationMs,
		"verify_duration_ms":          success.Report.VerifyDurationMs,
		"db_size_bytes_after_restore": success.Report.RestoredDatabaseSizeBytes,
		"table_count":                 success.Report.TableCount,
		"schema_count":                success.Report.SchemaCount,
		"fail_message":                nil,
	}

	result := database.Conn(ctx, r.db).Model(&Verification{}).
		Where(
			"id = ? AND agent_id = ? AND status = ?",
			success.VerificationID, success.AgentID, VerificationStatusRunning,
		).
		Updates(fields)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrVerificationGone
	}

	return r.insertTableStats(ctx, success.VerificationID, success.Report.TableStats)
}

func (r *verificationRepository) GetActiveForDatabase(
	ctx context.Context, databaseID uuid.UUID,
) ([]*Verification, error) {
	// Row locks cannot serialize creation when the database has no verification yet.
	if err := database.Conn(ctx, r.db).Exec(
		"SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", "verification_runs:"+databaseID.String(),
	).Error; err != nil {
		return nil, fmt.Errorf("lock database verifications: %w", err)
	}
	verifications := make([]*Verification, 0)

	err := database.Conn(ctx, r.db).
		Where(
			"database_id = ? AND status IN ?",
			databaseID,
			[]VerificationStatus{VerificationStatusPending, VerificationStatusRunning},
		).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Find(&verifications).Error

	return verifications, wrapQueryError(err)
}

func (r *verificationRepository) GetStalePending(
	ctx context.Context, threshold time.Time,
) ([]*Verification, error) {
	verifications := make([]*Verification, 0)

	err := database.Conn(ctx, r.db).
		Where("status = ? AND created_at < ?", VerificationStatusPending, threshold).
		Find(&verifications).Error

	return verifications, wrapQueryError(err)
}

func (r *verificationRepository) GetRunning(ctx context.Context) ([]*Verification, error) {
	verifications := make([]*Verification, 0)

	err := database.Conn(ctx, r.db).
		Where("status = ?", VerificationStatusRunning).
		Find(&verifications).Error

	return verifications, wrapQueryError(err)
}

func (r *verificationRepository) GetRunningByAgent(
	ctx context.Context, agentID uuid.UUID,
) ([]*Verification, error) {
	verifications := make([]*Verification, 0)

	err := database.Conn(ctx, r.db).
		Where("status = ? AND agent_id = ?", VerificationStatusRunning, agentID).
		Find(&verifications).Error

	return verifications, wrapQueryError(err)
}

func (r *verificationRepository) GetNonTerminalAutomaticVerificationsForDisabledSchedules(ctx context.Context) (
	[]*Verification,
	error,
) {
	verifications := make([]*Verification, 0)

	err := database.Conn(ctx, r.db).
		Table("verification_runs v").
		Select("v.*").
		Joins("LEFT JOIN verification_configs c ON c.database_id = v.database_id").
		Where(
			"COALESCE(c.is_scheduled_verification_enabled, false) = ? AND v.trigger IN ? AND v.status IN ?",
			false,
			[]VerificationTrigger{VerificationTriggerScheduled, VerificationTriggerAfterBackup},
			[]VerificationStatus{VerificationStatusPending, VerificationStatusRunning},
		).
		Scan(&verifications).Error

	return verifications, wrapQueryError(err)
}

func (r *verificationRepository) GetLatestFinishedAt(
	ctx context.Context, databaseID uuid.UUID,
) (*time.Time, error) {
	var verification Verification

	err := database.Conn(ctx, r.db).
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

func (r *verificationRepository) Requeue(ctx context.Context, verificationID uuid.UUID, now time.Time) error {
	db := database.Conn(ctx, r.db)

	result := db.Model(&Verification{}).
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
		return ErrVerificationGone
	}

	return nil
}

type TerminalUpdate struct {
	VerificationID uuid.UUID
	Status         VerificationStatus
	Fields         map[string]any
	FinishedAt     time.Time
}

func (r *verificationRepository) MarkTerminal(ctx context.Context, terminal TerminalUpdate) error {
	db := database.Conn(ctx, r.db)

	updates := make(map[string]any, len(terminal.Fields)+2)
	maps.Copy(updates, terminal.Fields)
	updates["status"] = terminal.Status
	updates["finished_at"] = terminal.FinishedAt

	statement := db.Model(&Verification{}).Where("id = ?", terminal.VerificationID)
	if terminal.Status == VerificationStatusCanceled {
		statement = statement.Where(
			"status IN ?",
			[]VerificationStatus{VerificationStatusPending, VerificationStatusRunning},
		)
	}
	result := statement.Updates(updates)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		if terminal.Status == VerificationStatusCanceled {
			return ErrNotRunning
		}
		return ErrVerificationGone
	}
	return nil
}

func (r *verificationRepository) DeleteByBackupIDs(ctx context.Context, backupIDs []uuid.UUID) error {
	return database.Conn(ctx, r.db).Where("backup_id IN ?", backupIDs).Delete(&Verification{}).Error
}

type VerificationPageQuery struct {
	DatabaseID uuid.UUID
	Limit      int
	After      *database.TimeCursor
}

func (r *verificationRepository) ListPage(ctx context.Context, query VerificationPageQuery) ([]*Verification, error) {
	verifications := []*Verification{}
	statement := database.Conn(ctx, r.db).Where("database_id = ?", query.DatabaseID)
	if query.After != nil {
		statement = statement.Where("(created_at, id) < (?, ?)", query.After.Time, query.After.ID)
	}
	err := statement.Order("created_at DESC, id DESC").Limit(query.Limit).Find(&verifications).Error
	return verifications, wrapQueryError(err)
}

func (r *verificationRepository) insertTableStats(
	ctx context.Context, verificationID uuid.UUID,
	stats []ReportTableStat,
) error {
	if len(stats) == 0 {
		return nil
	}

	rows := make([]TableStat, len(stats))
	for i, stat := range stats {
		rows[i] = TableStat{
			ID:             uuid.New(),
			VerificationID: verificationID,
			SchemaName:     stat.SchemaName,
			Name:           stat.Name,
			RowCount:       stat.RowCount,
		}
	}

	return database.Conn(ctx, r.db).Create(&rows).Error
}

func wrapQueryError(err error) error {
	if err != nil {
		return fmt.Errorf("query verifications: %w", err)
	}
	return nil
}
