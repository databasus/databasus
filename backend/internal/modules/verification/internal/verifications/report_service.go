package verifications

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/apierror"
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

	if !isFound || verification.Status != VerificationStatusRunning || verification.AgentID == nil ||
		*verification.AgentID != agent.ID {
		return nil, ErrVerificationGone
	}

	stream, err := s.backups.OpenLogicalBackupFile(ctx, verification.BackupID)
	if err != nil {
		var problem *apierror.Problem
		if errors.As(err, &problem) && problem.Code == apierror.CodeNotFound {
			return nil, ErrVerificationGone
		}
		//nolint:errorlint // The agent contract must not expose another module's Problem code.
		return nil, fmt.Errorf("open verification backup: %v", err)
	}
	return stream, nil
}

func (s *Service) SubmitReport(ctx context.Context, agent *agents.Agent, id uuid.UUID, report *ReportRequest) error {
	if report.Status != VerificationStatusCompleted && report.Status != VerificationStatusFailed {
		return apierror.NewInvalidFieldsProblem(
			[]apierror.FieldError{{Field: "status", Detail: "must be COMPLETED or FAILED"}},
		)
	}
	for _, table := range report.TableStats {
		if table.SchemaName == "" || table.Name == "" {
			return apierror.NewInvalidFieldsProblem(
				[]apierror.FieldError{{Field: "tableStats", Detail: "schemaName and name are required"}},
			)
		}
	}
	err := s.transactor.Transaction(
		ctx,
		func(ctx context.Context) error { return s.submitReport(ctx, agent, id, report) },
	)
	if err != nil && !errors.Is(err, ErrVerificationGone) {
		//nolint:errorlint // The agent contract must not expose another module's Problem code.
		return fmt.Errorf("submit verification report: %v", err)
	}
	return err
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

	if !isFound || verification.Status != VerificationStatusRunning || verification.AgentID == nil ||
		*verification.AgentID != agent.ID {
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
