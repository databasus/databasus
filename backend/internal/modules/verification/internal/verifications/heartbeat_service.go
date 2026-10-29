package verifications

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/verification/internal/agents"
)

func (s *Service) OnAgentHeartbeated(
	ctx context.Context,
	agent *agents.Agent,
	currentVerificationIDs []uuid.UUID,
) ([]uuid.UUID, error) {
	ownedRunning, err := s.verificationRepository.GetRunningByAgent(ctx, agent.ID)
	if err != nil {
		s.logger.ErrorContext(
			ctx,
			"failed to reconcile verification heartbeat",
			slog.Any("agent_id", agent.ID),
			slog.Any("error", err),
		)
		return []uuid.UUID{}, nil
	}

	reportedVerifications := make(map[uuid.UUID]struct{}, len(currentVerificationIDs))
	for _, id := range currentVerificationIDs {
		reportedVerifications[id] = struct{}{}
	}

	ownedVerifications := make(map[uuid.UUID]struct{}, len(ownedRunning))
	for _, row := range ownedRunning {
		ownedVerifications[row.ID] = struct{}{}
	}

	verificationsToAbort := make([]uuid.UUID, 0)
	for _, id := range currentVerificationIDs {
		if _, stillOwned := ownedVerifications[id]; !stillOwned {
			verificationsToAbort = append(verificationsToAbort, id)
		}
	}

	s.reclaimDroppedRunningJobs(ctx, agent, ownedRunning, reportedVerifications)

	return verificationsToAbort, nil
}

func (s *Service) reclaimDroppedRunningJobs(
	ctx context.Context,
	agent *agents.Agent,
	ownedRunning []*Verification,
	reported map[uuid.UUID]struct{},
) {
	reclaimEligibleBefore := s.clock.Now().Add(-agentJobReclaimGrace)

	for _, row := range ownedRunning {
		if _, stillReported := reported[row.ID]; stillReported {
			continue
		}

		if row.StartedAt == nil || !row.StartedAt.Before(reclaimEligibleBefore) {
			continue
		}

		logger := s.logger.With(slog.Any("verification_id", row.ID), slog.Any("agent_id", agent.ID))
		logger.InfoContext(ctx, "reclaiming dropped verification")

		if err := s.RequeueOrFail(ctx, row, FailureReasonAgentDroppedJob, nil); err != nil {
			logger.ErrorContext(ctx, "failed to reclaim dropped verification", slog.Any("error", err))
		}
	}
}
