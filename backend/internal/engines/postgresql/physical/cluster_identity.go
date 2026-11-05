package physical

import (
	"context"
	"fmt"
	"log/slog"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
)

// ClusterIdentity is what tells one cluster from another and what WAL archiving depends on.
type ClusterIdentity struct {
	Version             pgshared.Version
	SystemIdentifier    string
	WalSegmentSizeBytes int64
}

func (s *Source) InspectCluster(ctx context.Context, logger *slog.Logger) (ClusterIdentity, error) {
	inspection, err := s.OpenInspection(ctx)
	if err != nil {
		return ClusterIdentity{}, fmt.Errorf("failed to connect to cluster: %w", err)
	}
	defer closeInspection(ctx, inspection, logger)
	return inspection.InspectCluster(ctx)
}

func (i *Inspection) InspectCluster(ctx context.Context) (ClusterIdentity, error) {
	sources, err := i.readClusterIdentitySources(ctx)
	if err != nil {
		return ClusterIdentity{}, err
	}
	version, err := parseVersion(sources.RawVersion)
	if err != nil {
		return ClusterIdentity{}, err
	}

	return ClusterIdentity{
		Version:             version,
		SystemIdentifier:    sources.SystemIdentifier,
		WalSegmentSizeBytes: sources.WalSegmentSizeBytes,
	}, nil
}

type clusterIdentitySources struct {
	// NULL during recovery, where pg_current_wal_lsn() cannot be called.
	CurrentWalFilename    *string
	ControlFileTimelineID int
	SystemIdentifier      string
	RawVersion            string
	WalSegmentSizeBytes   int64
}

func (i *Inspection) readClusterIdentitySources(ctx context.Context) (clusterIdentitySources, error) {
	var sources clusterIdentitySources

	err := i.conn.QueryRow(ctx, `
		SELECT
			version(),
			(SELECT setting::bigint FROM pg_settings WHERE name = 'wal_segment_size'),
			CASE WHEN pg_is_in_recovery() THEN NULL ELSE pg_walfile_name(pg_current_wal_lsn()) END,
			(SELECT timeline_id FROM pg_control_checkpoint()),
			(SELECT system_identifier::text FROM pg_control_system())
	`).Scan(
		&sources.RawVersion,
		&sources.WalSegmentSizeBytes,
		&sources.CurrentWalFilename,
		&sources.ControlFileTimelineID,
		&sources.SystemIdentifier,
	)
	if err != nil {
		return clusterIdentitySources{}, fmt.Errorf("read cluster identity: %w", err)
	}

	return sources, nil
}

func (i *Inspection) ReadLiveTimeline(ctx context.Context) (int, string, error) {
	sources, err := i.readClusterIdentitySources(ctx)
	if err != nil {
		return 0, "", err
	}

	timelineID, err := resolveLiveTimelineID(sources)
	if err != nil {
		return 0, "", err
	}

	return timelineID, sources.SystemIdentifier, nil
}

// Every cluster answers the control file, but it lags a promotion: PostgreSQL stamps it
// at checkpoints, and the checkpoint a promotion requests is spread across
// checkpoint_completion_target, so a promoted primary keeps naming the pre-failover
// timeline for minutes. A stale reading lets an incremental extend a chain anchored to a
// timeline the cluster has already left.
//
// A source still in recovery has no fresher reading to offer: pg_walfile_name() refuses
// to run there, and pg_stat_wal_receiver hides received_tli from a role without
// pg_read_all_stats, which the replication user this product provisions does not have.
func resolveLiveTimelineID(sources clusterIdentitySources) (int, error) {
	if sources.CurrentWalFilename != nil {
		timelineID, err := walmath.ParseWALFilenameTimeline(*sources.CurrentWalFilename)
		if err != nil {
			return 0, fmt.Errorf("read timeline from current WAL file name: %w", err)
		}

		return int(timelineID), nil
	}

	return sources.ControlFileTimelineID, nil
}
