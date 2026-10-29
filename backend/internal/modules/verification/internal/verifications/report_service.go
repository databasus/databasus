package verifications

import (
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/verification/internal/agents"
)

func (s *Service) GetBackupFile(
	ctx context.Context,
	agent *agents.Agent,
	verificationID uuid.UUID,
) (io.ReadCloser, error) {
	verification, isFound, err := s.verificationRepository.GetVerification(ctx, verificationID)
	if err != nil {
		return nil, err
	}

	if !isFound {
		return nil, ErrVerificationGone
	}

	if verification.Status != VerificationStatusRunning {
		return nil, ErrVerificationGone
	}

	if verification.AgentID == nil || *verification.AgentID != agent.ID {
		return nil, ErrVerificationGone
	}

	return s.backups.OpenLogicalBackupFile(ctx, verification.BackupID)
}

func (s *Service) SubmitReport(ctx context.Context, agent *agents.Agent, id uuid.UUID, report *ReportRequest) error {
	return s.transactor.Transaction(
		ctx,
		func(ctx context.Context) error { return s.submitReport(ctx, agent, id, report) },
	)
}

func (s *Service) submitReport(
	ctx context.Context,
	agent *agents.Agent,
	verificationID uuid.UUID,
	req *ReportRequest,
) error {
	verification, isFound, err := s.verificationRepository.GetVerification(ctx, verificationID)
	if err != nil {
		return err
	}

	if !isFound {
		return ErrVerificationGone
	}

	if req.Status == VerificationStatusFailed {
		return s.RequeueOrFail(ctx, verification, classifyAgentReport(req), req.FailMessage)
	}

	if restoredDbTooSmall, sizeMessage := s.isRestoredTooSmall(ctx, verification, req); restoredDbTooSmall {
		return s.RequeueOrFail(ctx, verification, FailureReasonRestoredTooSmall, &sizeMessage)
	}

	if err := s.verificationRepository.WriteSuccessReport(
		ctx,
		SuccessReport{VerificationID: verificationID, AgentID: agent.ID, Report: req, FinishedAt: s.clock.Now()},
	); err != nil {
		return err
	}

	return database.AfterCommit(ctx, func(ctx context.Context) {
		s.syncBackupVerificationStatus(ctx, verification, VerificationStatusCompleted)
		s.notifyTerminal(ctx, verification, VerificationStatusCompleted, nil)
	})
}

func (s *Service) isRestoredTooSmall(
	ctx context.Context,
	verification *Verification,
	req *ReportRequest,
) (bool, string) {
	if req.RestoredDatabaseSizeBytes == nil {
		return false, ""
	}

	backup, isFound, err := s.getLogicalBackup(ctx, verification.BackupID)
	if err != nil || !isFound || backup.BackupRawDbSizeMb <= 0 {
		return false, ""
	}

	rawDbSizeBytes := int64(backup.BackupRawDbSizeMb * 1024 * 1024)
	minAcceptableBytes := int64(float64(rawDbSizeBytes) * minRestoredSizeRatio)

	if *req.RestoredDatabaseSizeBytes >= minAcceptableBytes {
		return false, ""
	}

	message := fmt.Sprintf(
		"restored database is %d bytes, which is less than %.0f%% of the original size (%d bytes)",
		*req.RestoredDatabaseSizeBytes, minRestoredSizeRatio*100, rawDbSizeBytes,
	)

	return true, message
}
