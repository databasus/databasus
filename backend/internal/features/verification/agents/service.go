package verification_agents

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	audit_logs "databasus-backend/internal/features/audit_logs"
	audit_logs_models "databasus-backend/internal/features/audit_logs/models"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/util/ratelimiter"
)

type AgentService struct {
	agentRepository *AgentRepository
	auditLogService *audit_logs.AuditLogService
	rateLimiter     ratelimiter.Counter
	logger          *slog.Logger

	heartbeatListeners []AgentHeartbeatedListener
}

func (s *AgentService) AddAgentHeartbeatedListener(listener AgentHeartbeatedListener) {
	s.heartbeatListeners = append(s.heartbeatListeners, listener)
}

func (s *AgentService) CreateAgent(
	ctx context.Context,
	actor users.Actor,
	req *CreateAgentRequest,
) (*CreatedAgentResponse, error) {
	plainToken := newPlainToken()
	tokenHash := hashAgentToken(plainToken)

	agent := &Agent{
		ID:        uuid.New(),
		Name:      req.Name,
		TokenHash: tokenHash,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.agentRepository.Create(agent); err != nil {
		return nil, err
	}

	s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     fmt.Sprintf("verification agent created: %s", agent.Name),
		UserID:      &actor.UserID,
		WorkspaceID: nil,
	})

	return &CreatedAgentResponse{
		Agent: agent,
		Token: plainToken,
	}, nil
}

func (s *AgentService) ListAgents(ctx context.Context) ([]*Agent, error) {
	return s.agentRepository.FindAll(ctx)
}

func (s *AgentService) CountLiveAgents(ctx context.Context) (int64, error) {
	return s.agentRepository.CountLive(ctx)
}

func (s *AgentService) GetStaleAgents(threshold time.Duration) ([]*Agent, error) {
	return s.agentRepository.FindStale(time.Now().UTC().Add(-threshold))
}

func (s *AgentService) RotateToken(
	ctx context.Context,
	actor users.Actor,
	agentID uuid.UUID,
) (string, error) {
	agent, err := s.agentRepository.FindByID(agentID)
	if err != nil {
		return "", err
	}
	if agent == nil || agent.IsDeleted() {
		return "", ErrAgentNotFound
	}

	plainToken := newPlainToken()
	tokenHash := hashAgentToken(plainToken)

	if err := s.agentRepository.UpdateTokenHash(agentID, tokenHash); err != nil {
		return "", err
	}

	s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     fmt.Sprintf("verification agent token rotated: %s", agent.Name),
		UserID:      &actor.UserID,
		WorkspaceID: nil,
	})

	return plainToken, nil
}

func (s *AgentService) DeleteAgent(ctx context.Context, actor users.Actor, agentID uuid.UUID) error {
	agent, err := s.agentRepository.FindByID(agentID)
	if err != nil {
		return err
	}
	if agent == nil || agent.IsDeleted() {
		return ErrAgentNotFound
	}

	if err := s.agentRepository.SoftDelete(agentID, time.Now().UTC()); err != nil {
		return err
	}

	s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     fmt.Sprintf("verification agent deleted: %s", agent.Name),
		UserID:      &actor.UserID,
		WorkspaceID: nil,
	})

	return nil
}

func (s *AgentService) VerifyAgentCredentials(
	agentID uuid.UUID,
	rawToken string,
) (*Agent, error) {
	if rawToken == "" {
		return nil, ErrInvalidAgentToken
	}

	agent, err := s.agentRepository.FindByID(agentID)
	if err != nil {
		return nil, err
	}
	if agent == nil || agent.IsDeleted() {
		return nil, ErrInvalidAgentToken
	}

	incomingHash := hashAgentToken(rawToken)
	if subtle.ConstantTimeCompare([]byte(agent.TokenHash), []byte(incomingHash)) != 1 {
		return nil, ErrInvalidAgentToken
	}

	return agent, nil
}

func (s *AgentService) GetAgentByID(id uuid.UUID) (*Agent, error) {
	return s.agentRepository.FindByID(id)
}

func (s *AgentService) Heartbeat(
	ctx context.Context,
	agent *Agent,
	req *HeartbeatRequest,
) (seenAt time.Time, abortVerificationIDs []uuid.UUID, err error) {
	now := time.Now().UTC()

	err = s.agentRepository.UpdateCapacityAndLastSeen(
		agent.ID,
		AgentCapacity{
			MaxCPU:            req.MaxCPU,
			MaxRAMGb:          req.MaxRAMGb,
			MaxDiskGb:         req.MaxDiskGb,
			MaxConcurrentJobs: req.MaxConcurrentJobs,
		},
		now,
	)
	if err != nil {
		return time.Time{}, nil, err
	}

	abortSet := make(map[uuid.UUID]struct{})
	for _, listener := range s.heartbeatListeners {
		listenerAborts, listenerErr := listener.OnAgentHeartbeated(ctx, agent, req.CurrentVerificationIDs)
		if listenerErr != nil {
			s.logger.ErrorContext(
				ctx,
				"agent heartbeated listener failed",
				"error", listenerErr,
				"agent_id", agent.ID,
			)

			continue
		}

		for _, id := range listenerAborts {
			abortSet[id] = struct{}{}
		}
	}

	abortVerificationIDs = make([]uuid.UUID, 0, len(abortSet))
	for id := range abortSet {
		abortVerificationIDs = append(abortVerificationIDs, id)
	}

	return now, abortVerificationIDs, nil
}

func newPlainToken() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}

func hashAgentToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", hash)
}
