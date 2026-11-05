// Package connection tests whether a saved database or unsaved settings can be connected to, and runs
// the liveness probe of scheduled health checks.
package connection

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/logging"
	"databasus-backend/internal/modules/databases/internal/providers"
	"databasus-backend/internal/modules/databases/internal/userdatabase"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

type Deps struct {
	Databases *userdatabase.Service
	Registry  *providers.Registry
	Logger    *slog.Logger
}

// Service writes nothing: a test of a saved database leaves its settings, notifiers, last backup error
// and health status as they were.
type Service struct {
	databases *userdatabase.Service
	registry  *providers.Registry
	logger    *slog.Logger
}

func NewService(deps Deps) *Service {
	return &Service{databases: deps.Databases, registry: deps.Registry, logger: deps.Logger}
}

// TestDatabase tests the stored settings. A viewer may test, as the settings never leave the backend
// for anywhere but the database's own server.
func (s *Service) TestDatabase(ctx context.Context, actor users.Actor, databaseID uuid.UUID) (TestResult, error) {
	testedDatabase, err := s.databases.GetAuthorizedWithSettings(ctx, actor, databaseID, workspaces.ActionView)
	if err != nil {
		return TestResult{}, err
	}

	ctx = logging.With(ctx, slog.String("database_id", databaseID.String()))

	return s.testConnection(ctx, testedDatabase.Settings)
}

// TestSettings tests settings that are not saved; userdatabase.ResolveSettingsTarget refuses
// everything a test may not do before anything connects.
func (s *Service) TestSettings(
	ctx context.Context,
	actor users.Actor,
	target userdatabase.SettingsTarget,
) (TestResult, error) {
	resolvedTarget, err := s.databases.ResolveSettingsTarget(ctx, actor, target)
	if err != nil {
		return TestResult{}, err
	}

	ctx = logging.With(ctx, slog.String("workspace_id", target.WorkspaceID.String()))

	return s.testConnection(ctx, resolvedTarget.Settings)
}

// ProbeLiveness is the trusted path of scheduled health checks.
func (s *Service) ProbeLiveness(ctx context.Context, databaseID uuid.UUID) error {
	probedDatabase, err := s.databases.GetWithSettings(ctx, databaseID)
	if err != nil {
		return err
	}

	provider, err := s.registry.GetProvider(probedDatabase.Type)
	if err != nil {
		return err
	}

	ctx = logging.With(ctx, slog.String("database_id", databaseID.String()))

	connectedDatabase, err := provider.Connect(ctx, probedDatabase.Settings)
	if err != nil {
		return err
	}

	defer connectedDatabase.Close()

	return connectedDatabase.ProbeLiveness(ctx, s.logger)
}

// testConnection answers a refusal of the server as the outcome of the test, and keeps an internal
// failure, such as a stored secret that does not decrypt, an error.
func (s *Service) testConnection(ctx context.Context, settings providers.Settings) (TestResult, error) {
	provider, err := s.registry.GetProvider(settings.GetType())
	if err != nil {
		return TestResult{}, err
	}

	connectedDatabase, err := provider.Connect(ctx, settings)
	if errors.Is(err, providers.ErrSecretUnreadable) {
		return TestResult{}, err
	}

	if err != nil {
		return newTestResult(err), nil
	}

	defer connectedDatabase.Close()

	return newTestResult(connectedDatabase.TestConnection(ctx, s.logger)), nil
}
