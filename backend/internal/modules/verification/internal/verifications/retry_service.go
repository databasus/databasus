package verifications

import (
	"context"

	"databasus-backend/internal/engines/postgresql/logical/pgrestore"
	"databasus-backend/internal/lib/database"
)

type VerificationFailure struct {
	Verification *Verification
	Reason       FailureReason
	Message      *string
	Diagnostic   *pgrestore.FailureDiagnostic
}

func (s *Service) RequeueOrFail(ctx context.Context, failure VerificationFailure) error {
	return s.transactor.Transaction(ctx, func(ctx context.Context) error {
		current, isFound, err := s.verificationRepository.LockVerification(ctx, failure.Verification.ID)
		if err != nil {
			return err
		}

		if !isFound || current.Status != failure.Verification.Status ||
			current.AttemptCount != failure.Verification.AttemptCount ||
			(current.AgentID == nil) != (failure.Verification.AgentID == nil) ||
			(current.AgentID != nil && *current.AgentID != *failure.Verification.AgentID) {
			return ErrVerificationGone
		}

		failure.Verification = current

		return s.requeueOrFail(ctx, failure)
	})
}

func (s *Service) requeueOrFail(ctx context.Context, failure VerificationFailure) error {
	row, reason, agentMessage := failure.Verification, failure.Reason, failure.Message
	if shouldRequeueAgentSide(reason, row.AttemptCount) {
		return s.verificationRepository.Requeue(ctx, row.ID, s.clock.Now())
	}

	failMessage := failureMessageFor(reason, agentMessage)
	var output *string
	isTruncated := false
	if failure.Diagnostic != nil {
		output = &failure.Diagnostic.Output
		isTruncated = failure.Diagnostic.IsTruncated
	}

	if err := s.verificationRepository.MarkTerminal(
		ctx,
		TerminalUpdate{VerificationID: row.ID, Status: VerificationStatusFailed, Fields: map[string]any{
			"fail_message":                    failMessage,
			"failure_diagnostic_output":       output,
			"is_failure_diagnostic_truncated": isTruncated,
		}, FinishedAt: s.clock.Now()},
	); err != nil {
		return err
	}

	// Re-read so the notification body reflects the row's freshly stamped finished_at.
	updated, isFound, err := s.verificationRepository.GetVerification(ctx, row.ID)
	if err == nil && isFound {
		row = updated
	}

	return database.AfterCommit(ctx, func(ctx context.Context) {
		s.syncBackupVerificationStatus(ctx, row, VerificationStatusFailed)
		s.notifyTerminal(ctx, row, VerificationStatusFailed, &failMessage)
	})
}
