package postgresqllogical

import (
	"context"
	"log/slog"
	"time"

	pglogical "databasus-backend/internal/engines/postgresql/logical"
	"databasus-backend/internal/modules/databases/internal/providers"
)

const connectionTimeout = 15 * time.Second

// connectedDatabase is a logical PostgreSQL database reached through its tunnel, when it has one.
type connectedDatabase struct {
	settings Settings
	tunnel   *pglogical.Tunnel
}

// connection pairs the settings with their decrypted engine settings.
type connection struct {
	settings       Settings
	engineSettings pglogical.Settings
}

func openConnectedDatabase(ctx context.Context, logger *slog.Logger, toOpen connection) (*connectedDatabase, error) {
	tunnel, err := pglogical.NewSource(toOpen.engineSettings).OpenTunnel(ctx, logger)
	if err != nil {
		return nil, err
	}

	return &connectedDatabase{settings: toOpen.settings, tunnel: tunnel}, nil
}

func (d *connectedDatabase) TestConnection(ctx context.Context, logger *slog.Logger) error {
	ctx, cancel := context.WithTimeout(ctx, connectionTimeout)
	defer cancel()

	_, err := d.tunnel.GetSource().TestConnection(ctx, logger, pglogical.DumpCheck{
		Filter: pglogical.DumpFilter{
			IncludeSchemas:           d.settings.IncludeSchemas,
			ExcludeTablePatterns:     d.settings.ExcludeTables,
			ExcludeTableDataPatterns: d.settings.ExcludeTableData,
		},
		IsSkipUserMappings: d.settings.IsSkipUserMappings,
	})

	return err
}

// Detect leaves the version as it was while no database is picked, because there is nothing to
// connect to yet.
func (d *connectedDatabase) Detect(ctx context.Context, logger *slog.Logger) (providers.Settings, error) {
	if d.settings.Database == "" {
		return d.settings, nil
	}

	ctx, cancel := context.WithTimeout(ctx, connectionTimeout)
	defer cancel()

	version, err := d.tunnel.GetSource().DetectVersion(ctx, logger)
	if err != nil {
		return nil, err
	}

	detectedSettings := d.settings
	detectedSettings.Version = version

	return detectedSettings, nil
}

func (d *connectedDatabase) ProbeLiveness(ctx context.Context, logger *slog.Logger) error {
	return d.TestConnection(ctx, logger)
}

func (d *connectedDatabase) CheckLimitedUser(
	ctx context.Context,
	logger *slog.Logger,
) (providers.LimitedUserCheck, error) {
	isAdvisable, privileges, err := d.tunnel.GetSource().
		ShouldSuggestReadOnlyUser(ctx, logger, d.settings.IncludeSchemas)
	if err != nil {
		return providers.LimitedUserCheck{}, err
	}

	return providers.LimitedUserCheck{IsAdvisable: isAdvisable, Privileges: privileges}, nil
}

func (d *connectedDatabase) CreateLimitedUser(ctx context.Context, logger *slog.Logger) (providers.LimitedUser, error) {
	username, password, err := d.tunnel.GetSource().CreateReadOnlyUser(ctx, logger, d.settings.IncludeSchemas)
	if err != nil {
		return providers.LimitedUser{}, err
	}

	return providers.LimitedUser{Username: username, Password: password, IsForcedWalRotationAvailable: false}, nil
}

func (d *connectedDatabase) Close() {
	d.tunnel.Close()
}
