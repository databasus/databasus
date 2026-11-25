package authenticator

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/users/internal/identity"
	"databasus-backend/internal/modules/users/internal/settings"
)

// UserReader reads the users whose apps the package manages. The accounts package implements it;
// this package cannot import accounts, because accounts resets apps through it.
type UserReader interface {
	GetUser(ctx context.Context, userID uuid.UUID) (identity.User, bool, error)
	GetUserByEmail(ctx context.Context, email string) (identity.User, bool, error)
}

type Deps struct {
	Repositories Repositories
	Transactor   *database.Transactor
	Clock        clock.Source
	MasterKey    *crypto.MasterKey
	Users        UserReader
	Settings     *settings.Service
	AuditLog     *audit.Log
	Logger       *slog.Logger
}

type Service struct {
	apps          *appRepository
	setups        *setupRepository
	recoveryCodes *recoveryCodeRepository
	failures      *failureRepository
	transactor    *database.Transactor
	clock         clock.Source
	encryptor     *crypto.FieldEncryptor
	users         UserReader
	settings      *settings.Service
	auditLog      *audit.Log
	logger        *slog.Logger
}

func NewService(deps Deps) *Service {
	return &Service{
		apps:          deps.Repositories.apps,
		setups:        deps.Repositories.setups,
		recoveryCodes: deps.Repositories.recoveryCodes,
		failures:      deps.Repositories.failures,
		transactor:    deps.Transactor,
		clock:         deps.Clock,
		encryptor:     crypto.NewFieldEncryptor(deps.MasterKey),
		users:         deps.Users,
		settings:      deps.Settings,
		auditLog:      deps.AuditLog,
		logger:        deps.Logger,
	}
}

func (s *Service) GetConnectedApp(ctx context.Context, userID uuid.UUID) (AuthenticatorApp, bool, error) {
	return s.apps.getByUserID(ctx, userID)
}

func (s *Service) GetAppStates(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]AuthenticatorAppState, error) {
	if len(userIDs) == 0 {
		return map[uuid.UUID]AuthenticatorAppState{}, nil
	}

	apps, err := s.apps.listByUserIDs(ctx, userIDs)
	if err != nil || len(apps) == 0 {
		return map[uuid.UUID]AuthenticatorAppState{}, err
	}

	appIDs := make([]uuid.UUID, 0, len(apps))
	for _, app := range apps {
		appIDs = append(appIDs, app.ID)
	}

	unusedCounts, err := s.recoveryCodes.countUnusedByAppIDs(ctx, appIDs)
	if err != nil {
		return nil, err
	}

	states := make(map[uuid.UUID]AuthenticatorAppState, len(apps))
	for _, app := range apps {
		states[app.UserID] = AuthenticatorAppState{
			ConnectedAt:             app.ConnectedAt,
			UnusedRecoveryCodeCount: unusedCounts[app.ID],
		}
	}

	return states, nil
}

func (s *Service) DeleteExpired(ctx context.Context, now time.Time) (int64, int64, error) {
	deletedFailureCount, err := s.failures.deleteBefore(ctx, now.Add(-wrongCodeWindow))
	if err != nil {
		return 0, 0, err
	}

	deletedSetupCount, err := s.setups.deleteExpiredBefore(ctx, now)
	if err != nil {
		return 0, 0, err
	}

	return deletedFailureCount, deletedSetupCount, nil
}

func (s *Service) recordAuditEvent(ctx context.Context, author identity.User, message string) error {
	return s.auditLog.Record(ctx, audit.Event{Author: author.GetAuditAuthor(), WorkspaceID: nil, Message: message})
}
