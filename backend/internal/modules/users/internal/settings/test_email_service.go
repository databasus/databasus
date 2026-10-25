package settings

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/ratelimit"
	"databasus-backend/internal/modules/users/internal/identity"
)

const (
	testEmailSubject = "Databasus test email"
	testEmailBody    = `<p>This message confirms that Databasus can deliver email through its mail server.</p>`

	testEmailRateLimitScope  = "test-email"
	testEmailRateLimit       = 5
	testEmailRateLimitWindow = 10 * time.Minute
)

// TestEmailResult is a test email the instance tried to deliver. A refusal of the mail server is
// a result, not an error: the administrator reads the server's response in FailureDetail.
type TestEmailResult struct {
	RecipientEmail string
	IsDelivered    bool
	FailureDetail  string
}

// SendTestEmail sends a message to the administrator who asked, so they see delivery work end to
// end. Every attempt that reaches the mail server is recorded in the audit log.
func (s *Service) SendTestEmail(ctx context.Context, actor identity.Actor) (TestEmailResult, error) {
	if !actor.IsAdmin() {
		return TestEmailResult{}, ErrNotAdmin
	}

	isAllowed, err := s.rateLimiter.RecordAttemptAndCheckIsAllowed(ctx, ratelimit.Attempt{
		Scope:      testEmailRateLimitScope,
		Identifier: actor.UserID.String(),
		Limit:      testEmailRateLimit,
		Window:     testEmailRateLimitWindow,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to evaluate a rate limit",
			slog.String("scope", testEmailRateLimitScope), slog.Any("error", err))

		return TestEmailResult{}, ErrTestEmailRateLimited
	}

	if !isAllowed {
		return TestEmailResult{}, ErrTestEmailRateLimited
	}

	if !s.mailer.IsConfigured() {
		return TestEmailResult{}, ErrEmailNotConfigured
	}

	admin, err := s.getActorUser(ctx, actor)
	if err != nil {
		return TestEmailResult{}, err
	}

	if !identity.IsEmailAddress(admin.Email) {
		return TestEmailResult{}, ErrActorEmailMissing
	}

	result := TestEmailResult{RecipientEmail: admin.Email, IsDelivered: true, FailureDetail: ""}
	auditMessage := "Test email sent to " + admin.Email

	sendError := s.mailer.SendEmail(ctx, admin.Email, testEmailSubject, testEmailBody)
	if sendError != nil {
		s.logger.WarnContext(ctx, "test email was not delivered", slog.Any("error", sendError))

		result.IsDelivered = false
		result.FailureDetail = sendError.Error()
		auditMessage = fmt.Sprintf("Test email to %s failed", admin.Email)
	}

	err = s.auditLog.Record(ctx, audit.Event{Author: admin.GetAuditAuthor(), WorkspaceID: nil, Message: auditMessage})
	if err != nil {
		return TestEmailResult{}, err
	}

	return result, nil
}
