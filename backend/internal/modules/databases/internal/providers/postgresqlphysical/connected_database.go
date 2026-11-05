package postgresqlphysical

import (
	"context"
	"log/slog"
	"time"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/modules/databases/internal/providers"
)

const (
	detectionTimeout = 15 * time.Second
	// The replication test opens a replication connection and may force a WAL switch, which takes
	// longer than an ordinary connection.
	replicationTestTimeout = 30 * time.Second
)

// connectedDatabase is a physical PostgreSQL cluster reached through its tunnel, when it has one.
type connectedDatabase struct {
	settings Settings
	tunnel   *pgphysical.Tunnel
}

// connection pairs the settings with their decrypted engine settings.
type connection struct {
	settings       Settings
	engineSettings pgphysical.Settings
}

func openConnectedDatabase(ctx context.Context, logger *slog.Logger, toOpen connection) (*connectedDatabase, error) {
	tunnel, err := pgphysical.NewSource(toOpen.engineSettings).OpenTunnel(ctx, logger)
	if err != nil {
		return nil, err
	}

	return &connectedDatabase{settings: toOpen.settings, tunnel: tunnel}, nil
}

// TestConnection checks the WAL summarization and the forced WAL switch that the backup type needs,
// and that the cluster is still the one detected at the first save. The physical-backups specs
// require both gates.
func (d *connectedDatabase) TestConnection(ctx context.Context, logger *slog.Logger) error {
	ctx, cancel := context.WithTimeout(ctx, replicationTestTimeout)
	defer cancel()

	expectedSystemIdentifier := ""
	if d.settings.SystemIdentifier != nil {
		expectedSystemIdentifier = *d.settings.SystemIdentifier
	}

	return d.tunnel.GetSource().TestReplicationConnection(ctx, logger, pgphysical.ReplicationCheck{
		IsWalSummaryRequired:     d.settings.BackupType.IsRequiringWalSummary(),
		IsWalRotationRequired:    d.settings.BackupType.IsWalStreaming(),
		ExpectedSystemIdentifier: expectedSystemIdentifier,
	})
}

// Detect reads the version every time, and the identity of the cluster only while it is unknown.
func (d *connectedDatabase) Detect(ctx context.Context, logger *slog.Logger) (providers.Settings, error) {
	ctx, cancel := context.WithTimeout(ctx, detectionTimeout)
	defer cancel()

	clusterIdentity, err := d.tunnel.GetSource().InspectCluster(ctx, logger)
	if err != nil {
		return nil, err
	}

	detectedSettings := d.settings
	detectedSettings.Version = clusterIdentity.Version

	if detectedSettings.SystemIdentifier == nil {
		detectedSettings.SystemIdentifier = &clusterIdentity.SystemIdentifier
	}

	if detectedSettings.WalSegmentSizeBytes == nil {
		detectedSettings.WalSegmentSizeBytes = &clusterIdentity.WalSegmentSizeBytes
	}

	return detectedSettings, nil
}

// ProbeLiveness opens no replication connection, so a scheduled health check never competes with the
// backups for a WAL sender slot.
func (d *connectedDatabase) ProbeLiveness(ctx context.Context, logger *slog.Logger) error {
	return d.tunnel.GetSource().ProbeLiveness(ctx, logger)
}

func (d *connectedDatabase) CheckLimitedUser(
	ctx context.Context,
	logger *slog.Logger,
) (providers.LimitedUserCheck, error) {
	isAdvisable, privileges, err := d.tunnel.GetSource().ShouldSuggestReplicationOnlyUser(ctx, logger)
	if err != nil {
		return providers.LimitedUserCheck{}, err
	}

	return providers.LimitedUserCheck{IsAdvisable: isAdvisable, Privileges: privileges}, nil
}

func (d *connectedDatabase) CreateLimitedUser(ctx context.Context, logger *slog.Logger) (providers.LimitedUser, error) {
	replicationUser, err := d.tunnel.GetSource().CreateReplicationOnlyUser(ctx, logger)
	if err != nil {
		return providers.LimitedUser{}, err
	}

	return providers.LimitedUser{
		Username:                     replicationUser.Username,
		Password:                     replicationUser.Password,
		IsForcedWalRotationAvailable: replicationUser.IsForcedWalRotationAvailable,
	}, nil
}

func (d *connectedDatabase) Close() {
	d.tunnel.Close()
}
