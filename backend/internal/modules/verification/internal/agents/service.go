package agents

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/httpserver"
	"databasus-backend/internal/lib/ratelimit"
	"databasus-backend/internal/modules/users"
)

type Deps struct {
	Repositories Repositories
	Transactor   *database.Transactor
	Clock        clock.Source
	AuditLog     *audit.Log
	Users        users.Module
	RateLimiter  ratelimit.Counter
	Logger       *slog.Logger
}
type Service struct {
	repository  *agentRepository
	transactor  *database.Transactor
	clock       clock.Source
	auditLog    *audit.Log
	users       users.Module
	rateLimiter ratelimit.Counter
	logger      *slog.Logger
}

func NewService(deps Deps) *Service {
	return &Service{
		repository:  deps.Repositories.agents,
		transactor:  deps.Transactor,
		clock:       deps.Clock,
		auditLog:    deps.AuditLog,
		users:       deps.Users,
		rateLimiter: deps.RateLimiter,
		logger:      deps.Logger,
	}
}

type CreatedAgent struct {
	Agent Agent
	Token string
}
type ListQuery struct {
	Limit *int
	After *string
}
type Page struct {
	Agents     []Agent
	NextCursor *string
}

func (s *Service) CreateAgent(ctx context.Context, actor users.Actor, name string) (CreatedAgent, error) {
	if utf8.RuneCountInString(strings.TrimSpace(name)) < 1 || utf8.RuneCountInString(name) > 255 {
		return CreatedAgent{}, apierror.NewInvalidFieldsProblem(
			[]apierror.FieldError{{Field: "name", Detail: "must contain 1 to 255 characters"}},
		)
	}

	token := newPlainToken()
	agent := Agent{
		ID:                uuid.New(),
		Name:              name,
		TokenHash:         hashAgentToken(token),
		MaxCPU:            0,
		MaxRAMMb:          0,
		MaxDiskGb:         0,
		MaxConcurrentJobs: 0,
		LastSeenAt:        nil,
		CreatedAt:         s.clock.Now(),
		DeletedAt:         nil,
	}

	err := s.transactor.Transaction(ctx, func(ctx context.Context) error {
		if err := s.repository.insert(ctx, agent); err != nil {
			return err
		}
		return s.record(ctx, actor, "verification agent created: "+agent.Name)
	})

	return CreatedAgent{Agent: agent, Token: token}, err
}

func (s *Service) ListAgents(ctx context.Context) ([]Agent, error) { return s.repository.list(ctx) }

func (s *Service) CountAgents(ctx context.Context) (int64, error) { return s.repository.count(ctx) }

func (s *Service) ListStaleAgents(ctx context.Context) ([]Agent, error) {
	return s.repository.stale(ctx, s.clock.Now().Add(-5*time.Minute))
}

func (s *Service) GetAgent(ctx context.Context, id uuid.UUID) (Agent, bool, error) {
	return s.repository.get(ctx, id)
}

func (s *Service) ListPage(ctx context.Context, query ListQuery) (Page, error) {
	limit := 20
	if query.Limit != nil {
		limit = *query.Limit
	}

	if limit < 1 || limit > 100 {
		return Page{}, apierror.NewInvalidFieldsProblem(
			[]apierror.FieldError{{Field: "limit", Detail: "must be between 1 and 100"}},
		)
	}

	var after *database.TimeCursor
	if query.After != nil {
		cursor, err := database.DecodeTimeCursor(*query.After)
		if err != nil {
			return Page{}, err
		}
		after = &cursor
	}

	agents, err := s.repository.page(ctx, limit+1, after)
	if err != nil {
		return Page{}, err
	}

	page := Page{Agents: agents[:min(len(agents), limit)], NextCursor: nil}
	if len(agents) > limit {
		lastAgent := agents[limit-1]
		cursor := database.TimeCursor{Time: lastAgent.CreatedAt, ID: lastAgent.ID}.Encode()
		page.NextCursor = &cursor
	}

	return page, nil
}

func (s *Service) RotateToken(ctx context.Context, actor users.Actor, id uuid.UUID) (string, error) {
	token := newPlainToken()
	err := s.updateAgentAndRecordAudit(ctx, agentChange{
		Actor: actor, AgentID: id,
		Fields:        map[string]any{"token_hash": hashAgentToken(token)},
		MessagePrefix: "verification agent token rotated: ",
	})
	return token, err
}

func (s *Service) DeleteAgent(ctx context.Context, actor users.Actor, id uuid.UUID) error {
	return s.updateAgentAndRecordAudit(ctx, agentChange{
		Actor: actor, AgentID: id,
		Fields:        map[string]any{"deleted_at": s.clock.Now()},
		MessagePrefix: "verification agent deleted: ",
	})
}

type AgentCredentials struct {
	AgentID string
	Token   string
}

func (s *Service) AuthenticateAgent(ctx context.Context, credentials AgentCredentials) (Agent, error) {
	id, err := uuid.Parse(credentials.AgentID)
	if err != nil {
		return Agent{}, s.refuseAgent(ctx, "invalid_uuid")
	}

	isAllowed, err := s.rateLimiter.RecordAttemptAndCheckIsAllowed(ctx, ratelimit.Attempt{
		Scope: "verification-agent-id", Identifier: id.String(), Limit: 10, Window: time.Second,
	})
	if err != nil {
		return Agent{}, fmt.Errorf("check agent rate limit: %w", err)
	}

	if !isAllowed {
		s.logger.WarnContext(
			ctx,
			"verification agent auth failure",
			slog.String("reason", "rate_limited"),
			slog.String("client_ip", httpserver.GetClientIP(ctx)),
		)
		return Agent{}, &apierror.Problem{Code: apierror.CodeRateLimited, Detail: "too many requests"}
	}
	if credentials.Token == "" {
		return Agent{}, s.refuseAgent(ctx, "missing_token")
	}

	agent, isFound, err := s.repository.get(ctx, id)
	if err != nil {
		return Agent{}, fmt.Errorf("authenticate agent: %w", err)
	}
	if !isFound || agent.IsDeleted() ||
		subtle.ConstantTimeCompare([]byte(agent.TokenHash), []byte(hashAgentToken(credentials.Token))) != 1 {
		return Agent{}, s.refuseAgent(ctx, "invalid_credentials")
	}

	return agent, nil
}

func (s *Service) RecordHeartbeat(ctx context.Context, id uuid.UUID, capacity AgentCapacity) (time.Time, error) {
	if capacity.MaxCPU < 0 || capacity.MaxDiskGb < 0 || capacity.MaxConcurrentJobs < 0 || capacity.MaxRAMMb < 0 {
		return time.Time{}, apierror.NewInvalidFieldsProblem(
			[]apierror.FieldError{{Field: "capacity", Detail: "must be nonnegative"}},
		)
	}

	now := s.clock.Now()
	isUpdated, err := s.repository.update(
		ctx,
		id,
		map[string]any{
			"max_cpu":             capacity.MaxCPU,
			"max_ram_mb":          capacity.MaxRAMMb,
			"max_disk_gb":         capacity.MaxDiskGb,
			"max_concurrent_jobs": capacity.MaxConcurrentJobs,
			"last_seen_at":        now,
		},
	)
	if err != nil {
		return time.Time{}, err
	}
	if !isUpdated {
		return time.Time{}, ErrInvalidAgentToken
	}

	return now, nil
}

type agentChange struct {
	Actor         users.Actor
	AgentID       uuid.UUID
	Fields        map[string]any
	MessagePrefix string
}

func (s *Service) updateAgentAndRecordAudit(ctx context.Context, change agentChange) error {
	return s.transactor.Transaction(ctx, func(ctx context.Context) error {
		agent, isFound, err := s.repository.get(ctx, change.AgentID)
		if err != nil {
			return err
		}
		if !isFound || agent.IsDeleted() {
			return ErrAgentNotFound
		}

		isUpdated, err := s.repository.update(ctx, change.AgentID, change.Fields)
		if err != nil {
			return err
		}
		if !isUpdated {
			return ErrAgentNotFound
		}

		return s.record(ctx, change.Actor, change.MessagePrefix+agent.Name)
	})
}

func (s *Service) record(ctx context.Context, actor users.Actor, message string) error {
	author, isFound, err := s.users.GetUser(ctx, actor.UserID)
	if err != nil {
		return err
	}
	if !isFound {
		return ErrInvalidAgentToken
	}
	return s.auditLog.Record(ctx, audit.Event{Author: author.GetAuditAuthor(), WorkspaceID: nil, Message: message})
}

func newPlainToken() string { return strings.ReplaceAll(uuid.NewString(), "-", "") }

func hashAgentToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", hash)
}

func (s *Service) refuseAgent(ctx context.Context, reason string) error {
	s.logger.WarnContext(
		ctx,
		"verification agent auth failure",
		slog.String("reason", reason),
		slog.String("client_ip", httpserver.GetClientIP(ctx)),
	)
	return ErrInvalidAgentToken
}
