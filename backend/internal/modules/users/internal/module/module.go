// Package module implements the users.Module contract on top of the module's services.
package module

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/modules/users/internal/accounts"
	usershttpapi "databasus-backend/internal/modules/users/internal/httpapi"
	"databasus-backend/internal/modules/users/internal/identity"
	"databasus-backend/internal/modules/users/internal/passwordreset"
	"databasus-backend/internal/modules/users/internal/settings"
	"databasus-backend/internal/modules/users/internal/signin"
)

type Deps struct {
	Directory     *accounts.Directory
	Accounts      *accounts.Service
	PasswordReset *passwordreset.Service
	Settings      *settings.Service
	SignIn        *signin.Service
	Access        *usershttpapi.Access
	Handler       *usershttpapi.Handler
	Logger        *slog.Logger
}

type Module struct {
	directory     *accounts.Directory
	accounts      *accounts.Service
	passwordReset *passwordreset.Service
	settings      *settings.Service
	signIn        *signin.Service
	access        *usershttpapi.Access
	handler       *usershttpapi.Handler
	logger        *slog.Logger
}

func New(deps Deps) *Module {
	return &Module{
		directory:     deps.Directory,
		accounts:      deps.Accounts,
		passwordReset: deps.PasswordReset,
		settings:      deps.Settings,
		signIn:        deps.SignIn,
		access:        deps.Access,
		handler:       deps.Handler,
		logger:        deps.Logger,
	}
}

func (m *Module) GetAccessMiddleware() gin.HandlerFunc {
	return m.access.GetAccessMiddleware()
}

func (m *Module) RegisterRoutes(routes gin.IRouter, hooks *apierror.Hooks) {
	usershttpapi.RegisterHandlersWithOptions(
		routes,
		usershttpapi.NewStrictHandlerWithOptions(m.handler, nil, usershttpapi.StrictGinServerOptions{
			RequestErrorHandlerFunc:  hooks.HandleRequestError,
			HandlerErrorFunc:         hooks.HandleHandlerError,
			ResponseErrorHandlerFunc: hooks.HandleResponseError,
		}),
		usershttpapi.GinServerOptions{BaseURL: "", Middlewares: nil, ErrorHandler: hooks.HandleParameterError},
	)
}

func (m *Module) GetJobs() []background.Job {
	return []background.Job{m.newExpiredCodesCleanupJob()}
}

func (m *Module) GetUser(ctx context.Context, userID uuid.UUID) (identity.User, bool, error) {
	return m.directory.GetUser(ctx, userID)
}

func (m *Module) GetUsers(ctx context.Context, userIDs []uuid.UUID) ([]identity.User, error) {
	return m.directory.GetUsers(ctx, userIDs)
}

func (m *Module) GetUserByEmail(ctx context.Context, email string) (identity.User, bool, error) {
	return m.directory.GetUserByEmail(ctx, email)
}

func (m *Module) InviteUser(
	ctx context.Context,
	actor identity.Actor,
	invitation accounts.Invitation,
) (identity.User, error) {
	return m.accounts.InviteUser(ctx, actor, invitation)
}

func (m *Module) CanCreateWorkspaces(ctx context.Context, actor identity.Actor) (bool, error) {
	return m.settings.CanCreateWorkspaces(ctx, actor)
}

func (m *Module) GetUserCount(ctx context.Context) (int64, error) {
	return m.directory.GetUserCount(ctx)
}

func (m *Module) CreateUserFromConsole(
	ctx context.Context,
	request accounts.ConsoleUserRequest,
) (accounts.ConsoleUserOutcome, error) {
	return m.accounts.CreateUserFromConsole(ctx, request)
}

func (m *Module) ChangePasswordFromConsole(ctx context.Context, change accounts.ConsolePasswordChange) error {
	return m.accounts.ChangePasswordFromConsole(ctx, change)
}

func (m *Module) DisableTwoFactorAuth(ctx context.Context) (bool, error) {
	return m.settings.DisableTwoFactorAuthFromConsole(ctx)
}

func (m *Module) RenderAdminListing(ctx context.Context) (string, error) {
	return m.accounts.RenderAdminListing(ctx)
}
