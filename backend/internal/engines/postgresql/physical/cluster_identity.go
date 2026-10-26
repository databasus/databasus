package physical

import (
	"context"
	"fmt"
	"log/slog"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
)

// ClusterIdentity is what tells one cluster from another and what WAL archiving depends on.
type ClusterIdentity struct {
	Version             pgshared.Version
	SystemIdentifier    string
	WalSegmentSizeBytes int64
}

func (s *Source) InspectCluster(ctx context.Context, logger *slog.Logger) (ClusterIdentity, error) {
	conn, err := s.connect(ctx)
	if err != nil {
		return ClusterIdentity{}, fmt.Errorf("failed to connect to cluster: %w", err)
	}
	defer closeConn(ctx, conn, logger)

	version, err := detectVersion(ctx, conn)
	if err != nil {
		return ClusterIdentity{}, err
	}

	var systemIdentifier string

	if err := conn.QueryRow(ctx, "SELECT system_identifier::text FROM pg_control_system()").
		Scan(&systemIdentifier); err != nil {
		return ClusterIdentity{}, fmt.Errorf("failed to read system_identifier: %w", err)
	}

	var walSegmentSizeBytes int64

	if err := conn.QueryRow(ctx, "SELECT setting::bigint FROM pg_settings WHERE name = 'wal_segment_size'").
		Scan(&walSegmentSizeBytes); err != nil {
		return ClusterIdentity{}, fmt.Errorf("failed to read wal_segment_size: %w", err)
	}

	return ClusterIdentity{
		Version:             version,
		SystemIdentifier:    systemIdentifier,
		WalSegmentSizeBytes: walSegmentSizeBytes,
	}, nil
}
