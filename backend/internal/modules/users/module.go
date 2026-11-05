// Package users owns users, sign-in, password reset, instance settings and the global roles. Other
// modules reach it only through this file.
package users

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/mailer"
	"databasus-backend/internal/lib/ratelimit"
	"databasus-backend/internal/modules/users/internal/accounts"
	usershttpapi "databasus-backend/internal/modules/users/internal/httpapi"
	"databasus-backend/internal/modules/users/internal/identity"
	"databasus-backend/internal/modules/users/internal/module"
	"databasus-backend/internal/modules/users/internal/oauth"
	"databasus-backend/internal/modules/users/internal/passwordreset"
	"databasus-backend/internal/modules/users/internal/settings"
	"databasus-backend/internal/modules/users/internal/signin"
	"databasus-backend/internal/modules/users/internal/turnstile"
)

type (
	// Role is the global role of a user.
	Role = identity.Role
	// Status is the state of a user: invited and not signed up yet, active, or deactivated.
	Status = identity.Status
	// Actor is the user a request runs for: the user ID and the global role.
	Actor = identity.Actor
	// User is a user as other modules see it, without the password or the identity provider IDs.
	User = identity.User
	// Invitation names the address to invite and the workspace the invitation is for.
	Invitation = accounts.Invitation
	// ConsoleUserRequest, ConsoleUserOutcome and ConsolePasswordChange describe the console
	// commands that create a user and change a password.
	ConsoleUserRequest    = accounts.ConsoleUserRequest
	ConsoleUserOutcome    = accounts.ConsoleUserOutcome
	ConsolePasswordChange = accounts.ConsolePasswordChange
)

const (
	RoleAdmin  = identity.RoleAdmin
	RoleMember = identity.RoleMember

	StatusInvited  = identity.StatusInvited
	StatusActive   = identity.StatusActive
	StatusInactive = identity.StatusInactive
)

func WithActor(ctx context.Context, actor Actor) context.Context {
	return identity.WithActor(ctx, actor)
}

func GetActor(ctx context.Context) (Actor, bool) {
	return identity.GetActor(ctx)
}

type Module interface {
	// The contract guard must run first to set each operation's access mark.
	GetAccessMiddleware() gin.HandlerFunc
	RegisterRoutes(routes gin.IRouter, hooks *apierror.Hooks)
	GetJobs() []background.Job

	// GetUser and GetUserByEmail report false when no user matches.
	GetUser(ctx context.Context, userID uuid.UUID) (User, bool, error)
	// GetUsers leaves out the IDs that match no user.
	GetUsers(ctx context.Context, userIDs []uuid.UUID) ([]User, error)
	GetUserByEmail(ctx context.Context, email string) (User, bool, error)
	// InviteUser creates a user who finishes sign-up later. It answers permission_denied when the
	// instance settings forbid the actor to invite and email_taken when the address has a user.
	InviteUser(ctx context.Context, actor Actor, invitation Invitation) (User, error)
	// CanCreateWorkspaces answers the instance settings for a user who is not an instance admin; an
	// instance admin always may.
	CanCreateWorkspaces(ctx context.Context, actor Actor) (bool, error)
	GetUserCount(ctx context.Context) (int64, error)

	// The console methods serve the commands an operator runs inside the instance. They take no
	// actor: whoever runs them can already run anything there.
	CreateUserFromConsole(ctx context.Context, request ConsoleUserRequest) (ConsoleUserOutcome, error)
	ChangePasswordFromConsole(ctx context.Context, change ConsolePasswordChange) error
	// DisableTwoFactorAuth reports false when the second factor was already off.
	DisableTwoFactorAuth(ctx context.Context) (bool, error)
	RenderAdminListing(ctx context.Context) (string, error)
}

type Deps struct {
	DB          *gorm.DB
	Transactor  *database.Transactor
	Clock       clock.Source
	Logger      *slog.Logger
	MasterKey   *crypto.MasterKey
	Mailer      mailer.Sender
	AuditLog    *audit.Log
	RateLimiter ratelimit.Counter
	Turnstile   TurnstileSettings
	GitHub      GitHubSettings
	Google      GoogleSettings
}

// TurnstileSettings configure the human-verification challenge of sign-up, sign-in and the second
// factor. An empty SecretKey turns the challenge off.
type TurnstileSettings struct {
	SecretKey string
	VerifyURL string
}

type (
	// GitHubSettings and GoogleSettings configure sign-in with the provider. An empty client ID or
	// secret turns it off.
	GitHubSettings = oauth.GitHubSettings
	GoogleSettings = oauth.GoogleSettings
)

func New(deps Deps) Module {
	accountRepositories := accounts.NewRepositories(deps.DB)
	directory := accounts.NewDirectory(accountRepositories)

	settingsService := settings.NewService(settings.Deps{
		Repositories: settings.NewRepositories(deps.DB),
		Transactor:   deps.Transactor,
		Users:        directory,
		AuditLog:     deps.AuditLog,
		Mailer:       deps.Mailer,
		RateLimiter:  deps.RateLimiter,
		Logger:       deps.Logger,
	})

	accountsService := accounts.NewService(accounts.Deps{
		Repositories: accountRepositories,
		Transactor:   deps.Transactor,
		Clock:        deps.Clock,
		AuditLog:     deps.AuditLog,
		Settings:     settingsService,
		Logger:       deps.Logger,
	})

	turnstileClient := turnstile.New(turnstile.Deps{
		SecretKey: deps.Turnstile.SecretKey,
		VerifyURL: deps.Turnstile.VerifyURL,
		Logger:    deps.Logger,
	})

	signInService := signin.NewService(signin.Deps{
		Repositories: signin.NewRepositories(deps.DB),
		Transactor:   deps.Transactor,
		Clock:        deps.Clock,
		MasterKey:    deps.MasterKey,
		Accounts:     accountsService,
		Settings:     settingsService,
		AuditLog:     deps.AuditLog,
		Mailer:       deps.Mailer,
		RateLimiter:  deps.RateLimiter,
		Turnstile:    turnstileClient,
		GitHub:       oauth.NewGitHub(deps.GitHub),
		Google:       oauth.NewGoogle(deps.Google),
		Logger:       deps.Logger,
	})

	passwordResetService := passwordreset.NewService(passwordreset.Deps{
		Repositories: passwordreset.NewRepositories(deps.DB),
		Transactor:   deps.Transactor,
		Clock:        deps.Clock,
		Accounts:     accountsService,
		AuditLog:     deps.AuditLog,
		Mailer:       deps.Mailer,
		RateLimiter:  deps.RateLimiter,
		Turnstile:    turnstileClient,
		Logger:       deps.Logger,
	})

	return module.New(module.Deps{
		Directory:     directory,
		Accounts:      accountsService,
		PasswordReset: passwordResetService,
		Settings:      settingsService,
		SignIn:        signInService,
		Access:        usershttpapi.NewAccess(signInService, deps.Logger),
		Handler: usershttpapi.NewHandler(usershttpapi.Deps{
			SignIn:        signInService,
			Accounts:      accountsService,
			Settings:      settingsService,
			PasswordReset: passwordResetService,
		}),
		Logger: deps.Logger,
	})
}
