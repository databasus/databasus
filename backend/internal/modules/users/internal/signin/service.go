package signin

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/mailer"
	"databasus-backend/internal/lib/ratelimit"
	"databasus-backend/internal/modules/users/internal/accounts"
	"databasus-backend/internal/modules/users/internal/identity"
	"databasus-backend/internal/modules/users/internal/oauth"
	"databasus-backend/internal/modules/users/internal/settings"
	"databasus-backend/internal/modules/users/internal/turnstile"
)

const (
	signInRateLimitScope  = "signin"
	signInRateLimit       = 10
	signInRateLimitWindow = time.Minute
)

// unmatchableHash is the bcrypt hash of a random secret nobody knows. A sign-in for an address
// without a usable password is compared against it, so it takes as long as a wrong password.
const unmatchableHash = "$2a$10$J1I5VBn.teMq6ueky2hqy.ECHJ6JOS4AcT6kjk62wWxRONkvkj7n6"

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

type SignUpRequest struct {
	Email           string
	Password        string
	Name            string
	TurnstileAnswer *string
}

type SignInRequest struct {
	Email           string
	Password        string
	TurnstileAnswer *string
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

// SignUp creates the user and signs them in. An invited user finishes sign-up here even when
// registration is closed; anyone else needs open registration, except the first user, who claims
// the instance.
func (s *Service) SignUp(ctx context.Context, request SignUpRequest) (Session, error) {
	err := validateSignUp(request)
	if err != nil {
		return Session{}, err
	}

	err = s.turnstile.Verify(ctx, request.TurnstileAnswer)
	if err != nil {
		return Session{}, err
	}

	hashedPassword, err := identity.HashPassword(request.Password)
	if err != nil {
		return Session{}, err
	}

	var signedUpUser accounts.UserRecord

	err = s.transactor.Transaction(ctx, func(ctx context.Context) error {
		existingUser, isFound, err := s.accounts.GetRecordByEmail(ctx, request.Email)
		if err != nil {
			return err
		}

		if isFound && existingUser.Status != identity.StatusInvited {
			return accounts.ErrEmailTaken
		}

		if isFound {
			signedUpUser, err = s.accounts.ActivateInvitedUser(ctx, existingUser, request.Name, &hashedPassword)
			if err != nil {
				return err
			}

			return s.recordAuditEvent(ctx, signedUpUser, "Invited user completed registration: "+signedUpUser.Email)
		}

		signedUpUser, err = s.accounts.CreateUserClaimingEmptyInstance(ctx, accounts.NewUser{
			Email:                    request.Email,
			Name:                     request.Name,
			HashedPassword:           &hashedPassword,
			GitHubOAuthID:            nil,
			GoogleOAuthID:            nil,
			OrdinaryRole:             identity.RoleMember,
			CheckOrdinaryUserAllowed: s.checkRegistrationOpen,
		})
		if err != nil {
			return err
		}

		return s.recordAuditEvent(ctx, signedUpUser, "User registered with email: "+signedUpUser.Email)
	})
	if err != nil {
		return Session{}, err
	}

	return s.issueSession(signedUpUser)
}

// SignIn refuses alike until the password is verified: an unknown address, a user who has not
// finished sign-up, a user without a password and a wrong password all get invalid_credentials
// after the same bcrypt comparison. Only then may the answer name a deactivated user. The log keeps
// each reason, because failed sign-ins are the signal that someone guesses credentials.
func (s *Service) SignIn(ctx context.Context, request SignInRequest) (SignInOutcome, error) {
	err := s.turnstile.Verify(ctx, request.TurnstileAnswer)
	if err != nil {
		return SignInOutcome{}, err
	}

	err = s.checkRateLimit(ctx, ratelimit.Attempt{
		Scope:      signInRateLimitScope,
		Identifier: request.Email,
		Limit:      signInRateLimit,
		Window:     signInRateLimitWindow,
	})
	if err != nil {
		return SignInOutcome{}, err
	}

	user, err := s.verifyPassword(ctx, request)
	if err != nil {
		return SignInOutcome{}, err
	}

	if !user.IsActive() {
		s.logger.WarnContext(ctx, "sign-in refused: the user is deactivated", slog.String("user_id", user.ID.String()))

		return SignInOutcome{}, ErrUserDeactivated
	}

	instanceSettings, err := s.settings.GetSettings(ctx)
	if err != nil {
		return SignInOutcome{}, err
	}

	if instanceSettings.IsTwoFactorAuthRequired {
		pendingSignIn, err := s.startTwoFactorSignIn(ctx, user)
		if err != nil {
			return SignInOutcome{}, err
		}

		return SignInOutcome{Session: nil, PendingSignIn: &pendingSignIn}, nil
	}

	session, err := s.completeSignIn(ctx, user)
	if err != nil {
		return SignInOutcome{}, err
	}

	return SignInOutcome{Session: &session, PendingSignIn: nil}, nil
}

func (s *Service) GetSignInOptions(ctx context.Context) (SignInOptions, error) {
	hasAnyUser, err := s.accounts.HasAnyUser(ctx)
	if err != nil {
		return SignInOptions{}, err
	}

	return SignInOptions{HasAnyUser: hasAnyUser}, nil
}

// Authenticate returns the actor of a session token. A token whose signature, expiry, user, user
// status or password time does not hold answers ErrInvalidSession.
func (s *Service) Authenticate(ctx context.Context, token string) (identity.Actor, error) {
	claims, err := s.sessions.read(token)
	if err != nil {
		return identity.Actor{}, err
	}

	user, isFound, err := s.accounts.GetRecord(ctx, claims.userID)
	if err != nil {
		return identity.Actor{}, err
	}

	if !isFound {
		return identity.Actor{}, fmt.Errorf("%w: the user does not exist", ErrInvalidSession)
	}

	if !user.IsActive() {
		return identity.Actor{}, fmt.Errorf("%w: the user is not active", ErrInvalidSession)
	}

	if !user.PasswordCreationTime.Truncate(time.Second).Equal(claims.passwordCreationTime.Truncate(time.Second)) {
		return identity.Actor{}, fmt.Errorf("%w: the password changed after the token was issued", ErrInvalidSession)
	}

	return user.ToActor(), nil
}

func (s *Service) verifyPassword(ctx context.Context, request SignInRequest) (accounts.UserRecord, error) {
	user, isFound, err := s.accounts.GetRecordByEmail(ctx, request.Email)
	if err != nil {
		return accounts.UserRecord{}, err
	}

	refusal := ""

	switch {
	case !isFound:
		refusal = "no user with this email"
	case user.Status == identity.StatusInvited:
		refusal = "the user has not finished sign-up"
	case !user.HasPassword():
		refusal = "the user has no password"
	}

	if refusal != "" {
		_ = bcrypt.CompareHashAndPassword([]byte(unmatchableHash), []byte(request.Password))

		s.logger.WarnContext(
			ctx,
			"sign-in refused",
			slog.String("reason", refusal),
			slog.String("email", request.Email),
		)

		return accounts.UserRecord{}, ErrInvalidCredentials
	}

	err = bcrypt.CompareHashAndPassword([]byte(*user.HashedPassword), []byte(request.Password))
	if err != nil {
		s.logger.WarnContext(ctx, "sign-in refused: incorrect password", slog.String("user_id", user.ID.String()))

		return accounts.UserRecord{}, ErrInvalidCredentials
	}

	return user, nil
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

func validateSignUp(request SignUpRequest) error {
	var fieldErrors []apierror.FieldError

	if !identity.IsEmailAddress(request.Email) {
		fieldErrors = append(fieldErrors, apierror.FieldError{Field: "email", Detail: "is not an email address"})
	}

	if !identity.IsPasswordLongEnough(request.Password) {
		fieldErrors = append(fieldErrors, apierror.FieldError{
			Field:  "password",
			Detail: fmt.Sprintf("must be at least %d characters long", identity.MinPasswordLength),
		})
	}

	if strings.TrimSpace(request.Name) == "" {
		fieldErrors = append(fieldErrors, apierror.FieldError{Field: "name", Detail: "is required"})
	}

	if len(fieldErrors) > 0 {
		return apierror.NewInvalidFieldsProblem(fieldErrors)
	}

	return nil
}
