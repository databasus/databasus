package verification

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/ratelimit"
	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/notifiers"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/verification/internal/agentapi"
	"databasus-backend/internal/modules/verification/internal/agents"
	"databasus-backend/internal/modules/verification/internal/configs"
	verificationhttpapi "databasus-backend/internal/modules/verification/internal/httpapi"
	"databasus-backend/internal/modules/verification/internal/module"
	"databasus-backend/internal/modules/verification/internal/verifications"
	"databasus-backend/internal/modules/workspaces"
)

type (
	Agent              = agents.Agent
	VerificationConfig = configs.Config
)

const (
	ScheduleInterval    = configs.VerificationScheduleInterval
	ScheduleAfterBackup = configs.VerificationScheduleAfterBackup
)

type Module interface {
	RegisterRoutes(gin.IRouter, *apierror.Hooks)
	GetJobs() []background.Job
	OnLogicalBackupCompleted(context.Context, backups.LogicalBackupCompleted) error
	OnLogicalBackupsDeleting(context.Context, backups.LogicalBackupsDeleting) error
	OnDatabaseCopied(context.Context, databases.DatabaseCopied) error
	OnDatabaseDeleting(context.Context, databases.DatabaseDeleting) error
	ListStaleAgents(context.Context) ([]Agent, error)
	ListAgents(context.Context) ([]Agent, error)
	ListEnabledConfigs(context.Context) ([]VerificationConfig, error)
	RegisterAgentRoutes(gin.IRouter, *apierror.Hooks)
}
type Deps struct {
	DB          *gorm.DB
	Transactor  *database.Transactor
	Logger      *slog.Logger
	Clock       clock.Source
	AuditLog    *audit.Log
	Users       users.Module
	Workspaces  workspaces.Module
	Databases   databases.Module
	Backups     backups.Module
	Notifiers   notifiers.Module
	RateLimiter ratelimit.Counter
}

func New(deps Deps) Module {
	agentService := agents.NewService(
		agents.Deps{
			Repositories: agents.NewRepositories(deps.DB),
			Transactor:   deps.Transactor,
			Clock:        deps.Clock,
			AuditLog:     deps.AuditLog,
			Users:        deps.Users,
			RateLimiter:  deps.RateLimiter,
			Logger:       deps.Logger,
		},
	)
	configService := configs.NewService(
		configs.Deps{
			Clock:        deps.Clock,
			Repositories: configs.NewRepositories(deps.DB),
			Transactor:   deps.Transactor,
			AuditLog:     deps.AuditLog,
			Users:        deps.Users,
			Workspaces:   deps.Workspaces,
			Databases:    deps.Databases,
		},
	)
	verificationService := verifications.NewService(
		verifications.Deps{
			Repositories: verifications.NewRepositories(deps.DB),
			Agents:       agentService,
			Configs:      configService,
			Transactor:   deps.Transactor,
			Clock:        deps.Clock,
			AuditLog:     deps.AuditLog,
			Users:        deps.Users,
			Workspaces:   deps.Workspaces,
			Databases:    deps.Databases,
			Backups:      deps.Backups,
			Notifiers:    deps.Notifiers,
			Logger:       deps.Logger,
		},
	)
	return module.New(
		module.Deps{
			Agents:        agentService,
			Configs:       configService,
			Verifications: verificationService,
			AgentHandler:  agentapi.NewHandler(agentapi.Deps{Agents: agentService, Verifications: verificationService}),
			Handler: verificationhttpapi.NewHandler(
				verificationhttpapi.Deps{
					Agents:        agentService,
					Configs:       configService,
					Verifications: verificationService,
				},
			),
		},
	)
}
