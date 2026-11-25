package accounts

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/users/internal/authenticator"
	"databasus-backend/internal/modules/users/internal/identity"
	"databasus-backend/internal/modules/users/internal/oauth"
	"databasus-backend/internal/modules/users/internal/settings"
)

type Deps struct {
	Repositories  Repositories
	Transactor    *database.Transactor
	Clock         clock.Source
	AuditLog      *audit.Log
	Settings      *settings.Service
	Authenticator *authenticator.Service
	Logger        *slog.Logger
}

type Service struct {
	repository    *userRepository
	transactor    *database.Transactor
	clock         clock.Source
	auditLog      *audit.Log
	settings      *settings.Service
	authenticator *authenticator.Service
	logger        *slog.Logger
}

func NewService(deps Deps) *Service {
	return &Service{
		repository:    deps.Repositories.users,
		transactor:    deps.Transactor,
		clock:         deps.Clock,
		auditLog:      deps.AuditLog,
		settings:      deps.Settings,
		authenticator: deps.Authenticator,
		logger:        deps.Logger,
	}
}

// GetUser serves a user's own profile and an administrator's view of any user. Anyone else gets
// permission_denied before the lookup, so the answer does not tell which IDs exist.
func (s *Service) GetUser(
	ctx context.Context,
	actor identity.Actor,
	userID uuid.UUID,
) (UserWithAuthenticatorApp, error) {
	if actor.UserID != userID && !actor.IsAdmin() {
		return UserWithAuthenticatorApp{}, ErrAnotherUser
	}

	record, isFound, err := s.repository.getByID(ctx, userID)
	if err != nil {
		return UserWithAuthenticatorApp{}, err
	}

	if !isFound {
		return UserWithAuthenticatorApp{}, ErrUserNotFound
	}

	return s.withAuthenticatorApp(ctx, record.ToUser())
}

func (s *Service) DisconnectAuthenticatorApp(
	ctx context.Context,
	actor identity.Actor,
	userID uuid.UUID,
	code string,
) (UserWithAuthenticatorApp, error) {
	err := s.authenticator.Disconnect(ctx, actor, userID, code)
	if err != nil {
		return UserWithAuthenticatorApp{}, err
	}

	return s.GetUser(ctx, actor, userID)
}

func (s *Service) GetRecord(ctx context.Context, userID uuid.UUID) (UserRecord, bool, error) {
	return s.repository.getByID(ctx, userID)
}

func (s *Service) GetRecordByEmail(ctx context.Context, email string) (UserRecord, bool, error) {
	return s.repository.getByEmail(ctx, email)
}

func (s *Service) GetRecordByOAuthID(
	ctx context.Context,
	provider oauth.Provider,
	oauthID string,
) (UserRecord, bool, error) {
	return s.repository.getByOAuthID(ctx, provider, oauthID)
}

// LinkOAuthID makes the provider's account sign the user in from now on.
func (s *Service) LinkOAuthID(ctx context.Context, userID uuid.UUID, provider oauth.Provider, oauthID string) error {
	return s.repository.linkOAuthID(ctx, userID, provider, oauthID)
}

// HasAnyUser tells the sign-in page whether to offer the registration that claims the instance.
func (s *Service) HasAnyUser(ctx context.Context) (bool, error) {
	userCount, err := s.repository.countAll(ctx)
	if err != nil {
		return false, err
	}

	return userCount > 0, nil
}
