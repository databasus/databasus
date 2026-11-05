package accounts

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/users/internal/identity"
	"databasus-backend/internal/modules/users/internal/oauth"
	"databasus-backend/internal/modules/users/internal/settings"
)

type Deps struct {
	Repositories Repositories
	Transactor   *database.Transactor
	Clock        clock.Source
	AuditLog     *audit.Log
	Settings     *settings.Service
	Logger       *slog.Logger
}

type Service struct {
	repository *userRepository
	transactor *database.Transactor
	clock      clock.Source
	auditLog   *audit.Log
	settings   *settings.Service
	logger     *slog.Logger
}

func NewService(deps Deps) *Service {
	return &Service{
		repository: deps.Repositories.users,
		transactor: deps.Transactor,
		clock:      deps.Clock,
		auditLog:   deps.AuditLog,
		settings:   deps.Settings,
		logger:     deps.Logger,
	}
}

// GetUser serves a user's own profile and an administrator's view of any user. Anyone else gets
// permission_denied before the lookup, so the answer does not tell which IDs exist.
func (s *Service) GetUser(ctx context.Context, actor identity.Actor, userID uuid.UUID) (identity.User, error) {
	if actor.UserID != userID && !actor.IsAdmin() {
		return identity.User{}, ErrAnotherUser
	}

	record, isFound, err := s.repository.getByID(ctx, userID)
	if err != nil {
		return identity.User{}, err
	}

	if !isFound {
		return identity.User{}, ErrUserNotFound
	}

	return record.ToUser(), nil
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
