package health

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/backoff"
	"databasus-backend/internal/lib/logging"
)

const (
	IncidentJobName     = "instancebackups_incidents"
	IncidentJobInterval = time.Minute
	reminderInterval    = 24 * time.Hour
	firstMailRetryDelay = time.Minute
	maxMailRetryDelay   = time.Hour
	maxMailErrorRunes   = 1000
)

// RunIncidentPass records the occurrences the destinations owe, so a schedule that never ran still
// becomes overdue, opens, updates or resolves the incident the health calls for, and mails root
// admin about an open one when mail is due. It needs no failure event, so a job that never started is
// noticed as well. The assessment and the incident change share one transaction under the incident
// lock, so a settings update that stops protection meanwhile is never resolved by a stale assessment.
// The mail goes out after the commit, so a slow mail server never holds that lock.
func (s *Service) RunIncidentPass(ctx context.Context, now time.Time) error {
	if err := s.configs.RecordOutstanding(ctx, now); err != nil {
		return err
	}

	var (
		protection assessment
		incident   Incident
		isOpen     bool
	)

	err := s.transactor.Transaction(ctx, func(txCtx context.Context) error {
		if err := s.incidents.lockChanges(txCtx); err != nil {
			return err
		}

		var err error

		protection, err = s.assess(txCtx, now)
		if err != nil {
			return err
		}

		incident, isOpen, err = s.updateIncident(txCtx, protection.Health, now)

		return err
	})
	if err != nil || !isOpen {
		return err
	}

	return s.mailIncident(ctx, incident, protection, now)
}

// OpenStoppedIncident runs in the transaction that removed the last destination, so stopped
// protection shows at once, without the overdue grace and whatever the setup reminder says. It
// holds the incident lock until that transaction commits, so a running pass finishes first and the
// next one sees the stop.
func (s *Service) OpenStoppedIncident(ctx context.Context) error {
	if err := s.incidents.lockChanges(ctx); err != nil {
		return err
	}

	return s.openIncident(ctx, IncidentKindStopped, s.clock.Now())
}

func (s *Service) updateIncident(ctx context.Context, health Health, now time.Time) (Incident, bool, error) {
	openIncident, isOpen, err := s.incidents.getOpen(ctx)
	if err != nil {
		return Incident{}, false, err
	}

	kind, isProblem := getIncidentKind(health)

	switch {
	case isProblem:
		if err := s.openIncident(ctx, kind, now); err != nil {
			return Incident{}, false, err
		}

		return s.incidents.getOpen(ctx)
	case isOpen && health != HealthResuming:
		s.logger.InfoContext(
			ctx,
			"databasus backup incident resolved",
			slog.String("incident_id", openIncident.ID.String()),
		)

		return Incident{}, false, s.incidents.resolve(ctx, openIncident.ID, now)
	default:
		return openIncident, isOpen, nil
	}
}

func (s *Service) openIncident(ctx context.Context, kind IncidentKind, now time.Time) error {
	openIncident, isOpen, err := s.incidents.getOpen(ctx)
	if err != nil {
		return err
	}

	if isOpen {
		if openIncident.Kind == kind {
			return nil
		}

		return s.incidents.updateKind(ctx, openIncident.ID, kind)
	}

	s.logger.WarnContext(ctx, "databasus backup incident opened", slog.String("kind", string(kind)))

	return s.incidents.insert(ctx, Incident{
		ID:                uuid.New(),
		Kind:              kind,
		StartedAt:         now,
		ResolvedAt:        nil,
		LastMailedAt:      nil,
		MailAttemptCount:  0,
		NextMailAttemptAt: nil,
		LastMailError:     nil,
	})
}

// mailIncident sends the first email of an incident, or a reminder a day after the last accepted
// one, to root admin's current address. Without a mail server or a usable address the incident
// stays in-app only and counts no attempt, so it is mailed once both exist. A crash between the mail
// server's acceptance and recording it can send the email again.
func (s *Service) mailIncident(ctx context.Context, incident Incident, protection assessment, now time.Time) error {
	if !incident.isMailDue(now) {
		return nil
	}

	rootAdmin, isFound, err := s.users.GetRootAdmin(ctx)
	if err != nil || !isFound || !s.isMailEligible(rootAdmin) {
		return err
	}

	subject, body := renderIncidentMail(incident.Kind, protection)

	if err := s.mailer.SendEmail(ctx, rootAdmin.Email, subject, body); err != nil {
		return s.recordMailFailure(ctx, incident, now, err)
	}

	s.logger.InfoContext(ctx, "databasus backup incident mailed", slog.String("incident_id", incident.ID.String()))

	return s.incidents.recordMailSent(ctx, incident.ID, now)
}

func (s *Service) recordMailFailure(ctx context.Context, incident Incident, now time.Time, mailErr error) error {
	s.logger.WarnContext(ctx, "databasus backup incident mail failed",
		slog.String("incident_id", incident.ID.String()), slog.Any("error", mailErr))

	lastMailError := logging.RedactError(mailErr, maxMailErrorRunes)
	nextMailAttemptAt := now.Add(
		backoff.Exponential{First: firstMailRetryDelay, Max: maxMailRetryDelay}.GetDelay(incident.MailAttemptCount + 1),
	)
	incident.MailAttemptCount++
	incident.NextMailAttemptAt = &nextMailAttemptAt
	incident.LastMailError = &lastMailError

	return s.incidents.recordMailFailure(ctx, incident)
}

// renderIncidentMail names the condition, the last successful snapshot and the destinations by their
// names, and nothing secret: no credentials, key, links or provider output.
func renderIncidentMail(kind IncidentKind, protection assessment) (string, string) {
	condition := describeIncident(kind)

	var lastSuccessfulSnapshotAt *time.Time

	var destinationLines strings.Builder

	for _, destination := range protection.Destinations {
		if destination.LastSuccessfulSnapshotAt != nil &&
			(lastSuccessfulSnapshotAt == nil || destination.LastSuccessfulSnapshotAt.After(*lastSuccessfulSnapshotAt)) {
			lastSuccessfulSnapshotAt = destination.LastSuccessfulSnapshotAt
		}

		fmt.Fprintf(&destinationLines, "<li>%s in %s: %s</li>",
			html.EscapeString(destination.StorageName), html.EscapeString(destination.WorkspaceName),
			describeDestinationHealth(destination.Health))
	}

	lastSnapshot := "none yet"
	if lastSuccessfulSnapshotAt != nil {
		lastSnapshot = lastSuccessfulSnapshotAt.UTC().Format("2006-01-02 15:04 UTC")
	}

	body := fmt.Sprintf("<p>%s.</p><p>Last successful snapshot: %s.</p>", condition, lastSnapshot)
	if destinationLines.Len() > 0 {
		body += "<p>Destinations:</p><ul>" + destinationLines.String() + "</ul>"
	}

	body += "<p>Open the Databasus backups section of the instance settings for details.</p>"

	return "Databasus backups need attention: " + condition, body
}

func describeIncident(kind IncidentKind) string {
	switch kind {
	case IncidentKindFailed:
		return "no destination received the latest Databasus backup"
	case IncidentKindPartial:
		return "some destinations did not receive the latest Databasus backup"
	case IncidentKindOverdue:
		return "a scheduled Databasus backup is more than 15 minutes late"
	case IncidentKindStopped:
		return "Databasus backups stopped because the last destination was removed"
	}

	return "Databasus backups have a problem"
}

func describeDestinationHealth(health Health) string {
	switch health {
	case HealthHealthy:
		return "protected"
	case HealthOverdue:
		return "overdue"
	case HealthFailed:
		return "failing"
	case HealthNotConfigured, HealthInitializing, HealthPartial, HealthStopped, HealthResuming:
		return "waiting for its first copy"
	}

	return "unknown"
}
