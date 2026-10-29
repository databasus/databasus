package module

import (
	"context"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/verification/internal/agents"
	"databasus-backend/internal/modules/verification/internal/configs"
	verificationhttpapi "databasus-backend/internal/modules/verification/internal/httpapi"
	"databasus-backend/internal/modules/verification/internal/verifications"
)

type Deps struct {
	Agents        *agents.Service
	Configs       *configs.Service
	Verifications *verifications.Service
	Handler       *verificationhttpapi.Handler
}
type Module struct {
	agents        *agents.Service
	configs       *configs.Service
	verifications *verifications.Service
	handler       *verificationhttpapi.Handler
}

func New(deps Deps) *Module {
	return &Module{agents: deps.Agents, configs: deps.Configs, verifications: deps.Verifications, handler: deps.Handler}
}

func (m *Module) RegisterRoutes(routes gin.IRouter, hooks *apierror.Hooks) {
	verificationhttpapi.RegisterHandlersWithOptions(
		routes,
		verificationhttpapi.NewStrictHandlerWithOptions(
			m.handler,
			nil,
			verificationhttpapi.StrictGinServerOptions{
				RequestErrorHandlerFunc:  hooks.HandleRequestError,
				HandlerErrorFunc:         hooks.HandleHandlerError,
				ResponseErrorHandlerFunc: hooks.HandleResponseError,
			},
		),
		verificationhttpapi.GinServerOptions{BaseURL: "", Middlewares: nil, ErrorHandler: hooks.HandleParameterError},
	)
}
func (m *Module) GetJobs() []background.Job { return m.verifications.GetJobs() }
func (m *Module) OnLogicalBackupCompleted(ctx context.Context, event backups.LogicalBackupCompleted) error {
	return m.verifications.OnLogicalBackupCompleted(ctx, event)
}

func (m *Module) OnLogicalBackupsDeleting(ctx context.Context, event backups.LogicalBackupsDeleting) error {
	return m.verifications.OnLogicalBackupsDeleting(ctx, event)
}

func (m *Module) OnDatabaseCopied(ctx context.Context, event databases.DatabaseCopied) error {
	return m.configs.OnDatabaseCopied(ctx, event)
}

func (m *Module) OnDatabaseDeleting(ctx context.Context, event databases.DatabaseDeleting) error {
	return m.configs.OnDatabaseDeleting(ctx, event)
}

func (m *Module) ListStaleAgents(ctx context.Context) ([]agents.Agent, error) {
	return m.agents.ListStaleAgents(ctx)
}

func (m *Module) ListAgents(ctx context.Context) ([]agents.Agent, error) {
	return m.agents.ListAgents(ctx)
}

func (m *Module) ListEnabledConfigs(ctx context.Context) ([]configs.Config, error) {
	return m.configs.ListEnabledConfigs(ctx)
}

func (m *Module) AuthenticateAgent(ctx context.Context, id uuid.UUID, token string) (agents.Agent, error) {
	return m.agents.AuthenticateAgent(ctx, id, token)
}

type AgentHeartbeat struct {
	Capacity               agents.AgentCapacity
	CurrentVerificationIDs []uuid.UUID
}

func (m *Module) RecordAgentHeartbeat(
	ctx context.Context,
	agent agents.Agent,
	heartbeat AgentHeartbeat,
) ([]uuid.UUID, time.Time, error) {
	seenAt, err := m.agents.RecordHeartbeat(ctx, agent.ID, heartbeat.Capacity)
	if err != nil {
		return nil, time.Time{}, err
	}
	abortIDs, err := m.verifications.OnAgentHeartbeated(ctx, &agent, heartbeat.CurrentVerificationIDs)
	return abortIDs, seenAt, err
}

func (m *Module) ClaimVerification(
	ctx context.Context,
	agent agents.Agent,
	capacity verifications.AgentCapacity,
) (verifications.JobAssignment, bool, error) {
	assignment, err := m.verifications.ClaimVerification(ctx, &agent, capacity)
	if err != nil || assignment == nil {
		return verifications.JobAssignment{}, false, err
	}
	return *assignment, true, nil
}

func (m *Module) OpenVerificationBackup(ctx context.Context, agent agents.Agent, id uuid.UUID) (io.ReadCloser, error) {
	return m.verifications.GetBackupFile(ctx, &agent, id)
}

func (m *Module) SubmitVerificationReport(
	ctx context.Context,
	agent agents.Agent,
	id uuid.UUID,
	report verifications.ReportRequest,
) error {
	return m.verifications.SubmitReport(ctx, &agent, id, &report)
}
