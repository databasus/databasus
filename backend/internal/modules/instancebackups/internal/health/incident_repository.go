package health

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

// Repositories are the repositories of the package; module.go builds them from the pool.
type Repositories struct {
	incidents *incidentRepository
}

func NewRepositories(db *gorm.DB) Repositories {
	return Repositories{incidents: &incidentRepository{db: db}}
}

type incidentRepository struct {
	db *gorm.DB
}

// lockChanges holds the incident lock until the transaction in ctx ends, so an incident pass and the
// settings update that stops protection never act on each other's uncommitted state.
func (r *incidentRepository) lockChanges(ctx context.Context) error {
	if err := database.RequireTransaction(ctx); err != nil {
		return err
	}

	err := database.Conn(ctx, r.db).
		Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", "instancebackups_incidents").Error
	if err != nil {
		return fmt.Errorf("lock the Databasus backup incidents: %w", err)
	}

	return nil
}

func (r *incidentRepository) getOpen(ctx context.Context) (Incident, bool, error) {
	var incidents []Incident

	err := database.Conn(ctx, r.db).Where("resolved_at IS NULL").Limit(1).Find(&incidents).Error
	if err != nil {
		return Incident{}, false, fmt.Errorf("read the open Databasus backup incident: %w", err)
	}

	if len(incidents) == 0 {
		return Incident{}, false, nil
	}

	return incidents[0], true, nil
}

func (r *incidentRepository) insert(ctx context.Context, incident Incident) error {
	if err := database.Conn(ctx, r.db).Create(&incident).Error; err != nil {
		return fmt.Errorf("open the Databasus backup incident: %w", err)
	}

	return nil
}

func (r *incidentRepository) updateKind(ctx context.Context, incidentID uuid.UUID, kind IncidentKind) error {
	err := database.Conn(ctx, r.db).Exec(
		"UPDATE instancebackups_incidents SET kind = ? WHERE id = ?", kind, incidentID,
	).Error
	if err != nil {
		return fmt.Errorf("update the Databasus backup incident: %w", err)
	}

	return nil
}

// resolve also ends any mail still waiting, so a resolved problem sends nothing.
func (r *incidentRepository) resolve(ctx context.Context, incidentID uuid.UUID, now time.Time) error {
	err := database.Conn(ctx, r.db).Exec(
		"UPDATE instancebackups_incidents SET resolved_at = ?, next_mail_attempt_at = NULL WHERE id = ?",
		now, incidentID,
	).Error
	if err != nil {
		return fmt.Errorf("resolve the Databasus backup incident: %w", err)
	}

	return nil
}

func (r *incidentRepository) recordMailSent(ctx context.Context, incidentID uuid.UUID, now time.Time) error {
	err := database.Conn(ctx, r.db).Exec(`UPDATE instancebackups_incidents
		SET last_mailed_at = ?, mail_attempt_count = 0, next_mail_attempt_at = NULL, last_mail_error = NULL
		WHERE id = ?`,
		now, incidentID,
	).Error
	if err != nil {
		return fmt.Errorf("record the sent Databasus backup incident mail: %w", err)
	}

	return nil
}

func (r *incidentRepository) recordMailFailure(ctx context.Context, failedIncident Incident) error {
	err := database.Conn(ctx, r.db).Exec(`UPDATE instancebackups_incidents
		SET mail_attempt_count = ?, next_mail_attempt_at = ?, last_mail_error = ?
		WHERE id = ?`,
		failedIncident.MailAttemptCount, failedIncident.NextMailAttemptAt, failedIncident.LastMailError,
		failedIncident.ID,
	).Error
	if err != nil {
		return fmt.Errorf("record the failed Databasus backup incident mail: %w", err)
	}

	return nil
}
