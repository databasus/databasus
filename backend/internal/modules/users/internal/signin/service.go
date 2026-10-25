package signin

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/mailer"
	"databasus-backend/internal/lib/ratelimit"
	"databasus-backend/internal/modules/users/internal/accounts"
	"databasus-backend/internal/modules/users/internal/oauth"
	"databasus-backend/internal/modules/users/internal/settings"
	"databasus-backend/internal/modules/users/internal/turnstile"
)

type Deps struct {
	Repositories Repositories
	Transactor   *database.Transactor
	Clock        clock.Source
	MasterKey    *crypto.MasterKey
	Accounts     *accounts.Service
	Settings     *settings.Service
	AuditLog     *audit.Log
	Mailer       mailer.Sender
	RateLimiter  ratelimit.Counter
	Turnstile    *turnstile.Client
	GitHub       *oauth.GitHub
	Google       *oauth.Google
	Logger       *slog.Logger
}

type Service struct {
	pendingSignIns *pendingSignInRepository
	transactor     *database.Transactor
	clock          clock.Source
	sessions       sessionTokens
	accounts       *accounts.Service
	settings       *settings.Service
	auditLog       *audit.Log
	mailer         mailer.Sender
	rateLimiter    ratelimit.Counter
	turnstile      *turnstile.Client
	gitHub         *oauth.GitHub
	google         *oauth.Google
	logger         *slog.Logger
}

func NewService(deps Deps) *Service {
	return &Service{
		pendingSignIns: deps.Repositories.pendingSignIns,
		transactor:     deps.Transactor,
		clock:          deps.Clock,
		sessions:       sessionTokens{masterKey: deps.MasterKey, clock: deps.Clock},
		accounts:       deps.Accounts,
		settings:       deps.Settings,
		auditLog:       deps.AuditLog,
		mailer:         deps.Mailer,
		rateLimiter:    deps.RateLimiter,
		turnstile:      deps.Turnstile,
		gitHub:         deps.GitHub,
		google:         deps.Google,
		logger:         deps.Logger,
	}
}

// Session is a signed-in user with the token the operation returns once.
type Session struct {
	UserID uuid.UUID
	Email  string
	Token  string
}

// PendingSignInView names the pending sign-in that the emailed code completes and the address the
// code went to.
type PendingSignInView struct {
	PendingSignInID uuid.UUID
	Email           string
}

// SignInOutcome holds exactly one of a session and a pending sign-in.
type SignInOutcome struct {
	Session       *Session
	PendingSignIn *PendingSignInView
}

type SignInOptions struct {
	HasAnyUser bool
}

func (s *Service) GetSignInOptions(ctx context.Context) (SignInOptions, error) {
	hasAnyUser, err := s.accounts.HasAnyUser(ctx)
	if err != nil {
		return SignInOptions{}, err
	}

	return SignInOptions{HasAnyUser: hasAnyUser}, nil
}

// completeSignIn records the sign-in where access is granted: the password step with one factor,
// the code step with two.
func (s *Service) completeSignIn(ctx context.Context, user accounts.UserRecord) (Session, error) {
	session, err := s.issueSession(user)
	if err != nil {
		return Session{}, err
	}

	err = s.recordAuditEvent(ctx, user, "User signed in with email: "+user.Email)
	if err != nil {
		return Session{}, err
	}

	s.logger.InfoContext(ctx, "user signed in", slog.String("user_id", user.ID.String()))

	return session, nil
}

func (s *Service) issueSession(user accounts.UserRecord) (Session, error) {
	token, err := s.sessions.issue(user)
	if err != nil {
		return Session{}, err
	}

	return Session{UserID: user.ID, Email: user.Email, Token: token}, nil
}

func (s *Service) checkRegistrationOpen(ctx context.Context) error {
	instanceSettings, err := s.settings.GetSettings(ctx)
	if err != nil {
		return err
	}

	if !instanceSettings.IsAllowExternalRegistrations {
		return ErrRegistrationClosed
	}

	return nil
}

// checkRateLimit refuses the attempt when the limiter refuses it or cannot evaluate it, and logs
// a limiter failure without the identifier.
func (s *Service) checkRateLimit(ctx context.Context, attempt ratelimit.Attempt) error {
	isAllowed, err := s.rateLimiter.RecordAttemptAndCheckIsAllowed(ctx, attempt)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to evaluate a rate limit",
			slog.String("scope", attempt.Scope), slog.Any("error", err))

		return ErrRateLimited
	}

	if !isAllowed {
		return ErrRateLimited
	}

	return nil
}

func (s *Service) recordAuditEvent(ctx context.Context, author accounts.UserRecord, message string) error {
	return s.auditLog.Record(ctx, audit.Event{
		Author:      author.ToUser().GetAuditAuthor(),
		WorkspaceID: nil,
		Message:     message,
	})
}
