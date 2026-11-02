// Package limiteduser checks whether the user a database connects as may do more than a backup needs,
// and creates a read-only user, or for physical PostgreSQL a replication-only user, on the server.
package limiteduser

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"databasus-backend/internal/lib/logging"
	"databasus-backend/internal/modules/databases/internal/providers"
	"databasus-backend/internal/modules/databases/internal/userdatabase"
	"databasus-backend/internal/modules/users"
)

const (
	checkTimeout = 15 * time.Second
	// Creating a user grants privilege after privilege, which takes longer than a check.
	creationTimeout = 30 * time.Second
)

type Deps struct {
	Databases *userdatabase.Service
	Registry  *providers.Registry
	Logger    *slog.Logger
}

// Service needs the right to change the workspace's content for both operations, because each of them
// connects to the host in the request, with the stored secrets of a named database while its resource
// address stays the stored one.
type Service struct {
	databases *userdatabase.Service
	registry  *providers.Registry
	logger    *slog.Logger
}

func NewService(deps Deps) *Service {
	return &Service{databases: deps.Databases, registry: deps.Registry, logger: deps.Logger}
}

func (s *Service) CheckLimitedUser(
	ctx context.Context,
	actor users.Actor,
	target userdatabase.SettingsTarget,
) (providers.LimitedUserCheck, error) {
	resolvedTarget, err := s.databases.ResolveSettingsTarget(ctx, actor, target)
	if err != nil {
		return providers.LimitedUserCheck{}, err
	}

	ctx, cancel := context.WithTimeout(
		logging.With(ctx, slog.String("workspace_id", target.WorkspaceID.String())),
		checkTimeout,
	)
	defer cancel()

	connectedDatabase, err := s.connect(ctx, resolvedTarget.Settings)
	if err != nil {
		return providers.LimitedUserCheck{}, err
	}

	defer connectedDatabase.Close()

	limitedUserCheck, err := connectedDatabase.CheckLimitedUser(ctx, s.logger)
	if err != nil {
		return providers.LimitedUserCheck{}, userdatabase.ToConnectionError(err)
	}

	return limitedUserCheck, nil
}

// CreateLimitedUser is not cancelled with the request: MySQL and MariaDB commit CREATE USER and each
// GRANT on their own, so a client that disconnects in the middle would leave a half-granted user.
func (s *Service) CreateLimitedUser(
	ctx context.Context,
	actor users.Actor,
	target userdatabase.SettingsTarget,
) (providers.LimitedUser, error) {
	resolvedTarget, err := s.databases.ResolveSettingsTarget(ctx, actor, target)
	if err != nil {
		return providers.LimitedUser{}, err
	}

	creationCtx, cancel := context.WithTimeout(
		logging.With(context.WithoutCancel(ctx), slog.String("workspace_id", target.WorkspaceID.String())),
		creationTimeout,
	)
	defer cancel()

	connectedDatabase, err := s.connect(creationCtx, resolvedTarget.Settings)
	if err != nil {
		return providers.LimitedUser{}, err
	}

	defer connectedDatabase.Close()

	createdUser, err := connectedDatabase.CreateLimitedUser(creationCtx, s.logger)
	if err != nil {
		return providers.LimitedUser{}, userdatabase.ToConnectionError(err)
	}

	message := "Limited user created for a new database (username: " + createdUser.Username + ")"
	if resolvedTarget.Database != nil {
		message = fmt.Sprintf("Limited user created for database: %s (username: %s)",
			resolvedTarget.Database.Name, createdUser.Username)
	}

	err = s.databases.RecordAudit(creationCtx, actor, target.WorkspaceID, message)
	if err != nil {
		return providers.LimitedUser{}, err
	}

	return createdUser, nil
}

func (s *Service) connect(ctx context.Context, settings providers.Settings) (providers.ConnectedDatabase, error) {
	provider, err := s.registry.GetProvider(settings.GetType())
	if err != nil {
		return nil, err
	}

	connectedDatabase, err := provider.Connect(ctx, settings)
	if err != nil {
		return nil, userdatabase.ToConnectionError(err)
	}

	return connectedDatabase, nil
}
