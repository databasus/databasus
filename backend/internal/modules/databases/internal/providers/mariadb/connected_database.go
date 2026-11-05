package mariadb

import (
	"context"
	"log/slog"
	"time"

	mariadbengine "databasus-backend/internal/engines/mariadb"
	"databasus-backend/internal/modules/databases/internal/providers"
)

const connectionTimeout = 15 * time.Second

// connectedDatabase is a MariaDB database reached through its tunnel, when it has one.
type connectedDatabase struct {
	settings Settings
	tunnel   *mariadbengine.Tunnel
}

func (d *connectedDatabase) TestConnection(ctx context.Context, logger *slog.Logger) error {
	ctx, cancel := context.WithTimeout(ctx, connectionTimeout)
	defer cancel()

	_, err := d.tunnel.GetSource().TestConnection(ctx, logger, d.settings.ExcludeTables)

	return err
}

// Detect leaves the version and the privileges as they were while no database is picked, because
// there is nothing to inspect yet.
func (d *connectedDatabase) Detect(ctx context.Context, logger *slog.Logger) (providers.Settings, error) {
	if d.settings.Database == "" {
		return d.settings, nil
	}

	ctx, cancel := context.WithTimeout(ctx, connectionTimeout)
	defer cancel()

	inspection, err := d.tunnel.GetSource().Inspect(ctx, logger, d.settings.ExcludeTables)
	if err != nil {
		return nil, err
	}

	detectedSettings := d.settings
	detectedSettings.Version = inspection.Version
	detectedSettings.Privileges = inspection.Privileges.EffectivePrivileges

	return detectedSettings, nil
}

func (d *connectedDatabase) ProbeLiveness(ctx context.Context, logger *slog.Logger) error {
	return d.TestConnection(ctx, logger)
}

func (d *connectedDatabase) CheckLimitedUser(
	ctx context.Context,
	logger *slog.Logger,
) (providers.LimitedUserCheck, error) {
	isAdvisable, privileges, err := d.tunnel.GetSource().ShouldSuggestReadOnlyUser(ctx, logger)
	if err != nil {
		return providers.LimitedUserCheck{}, err
	}

	return providers.LimitedUserCheck{IsAdvisable: isAdvisable, Privileges: privileges}, nil
}

func (d *connectedDatabase) CreateLimitedUser(ctx context.Context, logger *slog.Logger) (providers.LimitedUser, error) {
	username, password, err := d.tunnel.GetSource().CreateReadOnlyUser(ctx, logger)
	if err != nil {
		return providers.LimitedUser{}, err
	}

	return providers.LimitedUser{Username: username, Password: password, IsForcedWalRotationAvailable: false}, nil
}

func (d *connectedDatabase) Close() {
	d.tunnel.Close()
}
