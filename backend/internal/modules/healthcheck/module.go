package healthcheck

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
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/healthcheck/internal/attempts"
	"databasus-backend/internal/modules/healthcheck/internal/configs"
	healthcheckhttpapi "databasus-backend/internal/modules/healthcheck/internal/httpapi"
	"databasus-backend/internal/modules/healthcheck/internal/module"
	"databasus-backend/internal/modules/notifiers"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

type Module interface {
	RegisterRoutes(gin.IRouter, *apierror.Hooks)
	GetJobs() []background.Job
	OnDatabaseCreated(context.Context, databases.DatabaseCreated) error
	OnDatabaseCopied(context.Context, databases.DatabaseCopied) error
	OnDatabaseDeleting(context.Context, databases.DatabaseDeleting) error
}
type Deps struct {
	DB         *gorm.DB
	Transactor *database.Transactor
	Logger     *slog.Logger
	Clock      clock.Source
	AuditLog   *audit.Log
	Users      users.Module
	Workspaces workspaces.Module
	Databases  databases.Module
	Notifiers  notifiers.Module
}

func New(deps Deps) Module {
	configService := configs.NewService(
		configs.Deps{
			Repositories: configs.NewRepositories(deps.DB),
			Transactor:   deps.Transactor,
			AuditLog:     deps.AuditLog,
			Users:        deps.Users,
			Workspaces:   deps.Workspaces,
			Databases:    deps.Databases,
		},
	)
	attemptService := attempts.NewService(
		attempts.Deps{
			Repositories: attempts.NewRepositories(deps.DB),
			Transactor:   deps.Transactor,
			Clock:        deps.Clock,
			Logger:       deps.Logger,
			Configs:      configService,
			Databases:    deps.Databases,
			Notifiers:    deps.Notifiers,
		},
	)
	return module.New(
		module.Deps{
			Configs:  configService,
			Attempts: attemptService,
			Handler: healthcheckhttpapi.NewHandler(
				healthcheckhttpapi.Deps{Configs: configService, Attempts: attemptService},
			),
		},
	)
}
