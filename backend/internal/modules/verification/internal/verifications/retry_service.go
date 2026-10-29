package verifications

import (
	"context"

	"databasus-backend/internal/lib/database"
)

func (s *Service) RequeueOrFail(
	ctx context.Context,
	verification *Verification,
	reason FailureReason,
	message *string,
) error {
	return s.transactor.Transaction(
		ctx,
		func(ctx context.Context) error { return s.requeueOrFail(ctx, verification, reason, message) },
	)
}

func (s *Service) requeueOrFail(
	ctx context.Context,
	row *Verification,
	reason FailureReason,
	agentMessage *string,
) error {
	if shouldRequeueAgentSide(reason, row.AttemptCount) {
		return s.verificationRepository.Requeue(ctx, row.ID, s.clock.Now())
	}

	failMessage := failureMessageFor(reason, agentMessage)

	if err := s.verificationRepository.MarkTerminal(
		ctx,
		TerminalUpdate{VerificationID: row.ID, Status: VerificationStatusFailed, Fields: map[string]any{
			"fail_message": failMessage,
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
