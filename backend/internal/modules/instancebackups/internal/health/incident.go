package health

import (
	"time"

	"github.com/google/uuid"
)

// IncidentKind is the protection problem an incident reports.
type IncidentKind string

const (
	IncidentKindFailed  IncidentKind = "FAILED"
	IncidentKindPartial IncidentKind = "PARTIAL"
	IncidentKindOverdue IncidentKind = "OVERDUE"
	IncidentKindStopped IncidentKind = "STOPPED"
)

// MailStatus is how the mail about an incident stands.
type MailStatus string

const (
	MailStatusNotEligible MailStatus = "NOT_ELIGIBLE"
	MailStatusPending     MailStatus = "PENDING"
	MailStatusSent        MailStatus = "SENT"
	MailStatusRetrying    MailStatus = "RETRYING"
)

// Incident is one unresolved protection problem. Its kind follows the symptom while it lasts, and
// its mail state belongs to the whole incident, so a changed symptom sends no new email.
type Incident struct {
	ID                uuid.UUID    `gorm:"column:id"`
	Kind              IncidentKind `gorm:"column:kind"`
	StartedAt         time.Time    `gorm:"column:started_at"`
	ResolvedAt        *time.Time   `gorm:"column:resolved_at"`
	LastMailedAt      *time.Time   `gorm:"column:last_mailed_at"`
	MailAttemptCount  int          `gorm:"column:mail_attempt_count"`
	NextMailAttemptAt *time.Time   `gorm:"column:next_mail_attempt_at"`
	LastMailError     *string      `gorm:"column:last_mail_error"`
}

func (Incident) TableName() string {
	return "instancebackups_incidents"
}

// IncidentView is the open incident as root admin sees it.
type IncidentView struct {
	Incident   Incident
	MailStatus MailStatus
}

// isMailDue is true for the first email of an incident and, a day after the last accepted one, for
// a reminder, unless a failed attempt waits for its retry.
func (i Incident) isMailDue(now time.Time) bool {
	if i.NextMailAttemptAt != nil && now.Before(*i.NextMailAttemptAt) {
		return false
	}

	return i.LastMailedAt == nil || !now.Before(i.LastMailedAt.Add(reminderInterval))
}

func (i Incident) getMailStatus(isMailEligible bool) MailStatus {
	switch {
	case !isMailEligible:
		return MailStatusNotEligible
	case i.LastMailError != nil:
		return MailStatusRetrying
	case i.LastMailedAt != nil:
		return MailStatusSent
	default:
		return MailStatusPending
	}
}

// getIncidentKind is the incident a health calls for; resuming keeps the stopped incident open until
// a fresh copy succeeds, so it calls for none of its own.
func getIncidentKind(health Health) (IncidentKind, bool) {
	switch health {
	case HealthFailed:
		return IncidentKindFailed, true
	case HealthPartial:
		return IncidentKindPartial, true
	case HealthOverdue:
		return IncidentKindOverdue, true
	case HealthStopped:
		return IncidentKindStopped, true
	case HealthNotConfigured, HealthInitializing, HealthHealthy, HealthResuming:
		return "", false
	}

	return "", false
}
