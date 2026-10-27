package physical

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
)

const livenessTimeout = 30 * time.Second

var versionPattern = regexp.MustCompile(`PostgreSQL (\d+)`)

// Source is the engine's handle on one PostgreSQL cluster backed up physically.
type Source struct {
	settings Settings
	// generateUsername is a seam: a test makes the generated name collide with an existing role.
	generateUsername func() string
}

func NewSource(settings Settings) *Source {
	return &Source{settings: settings, generateUsername: generateReplicationUsername}
}

// GetSettings returns the settings the source connects with. Through a tunnel they carry the local
// end of the tunnel, so a client tool the caller starts reaches the same server.
func (s *Source) GetSettings() Settings {
	return s.settings
}

// ProbeLiveness answers only whether the source accepts the stored credentials. It deliberately
// avoids a replication connection: that would take a WAL sender slot, which on a source sized
// tightly for its replicas makes the probe fail during every backup and can starve the backup
// itself. Replication readiness stays the job of TestReplicationConnection.
func (s *Source) ProbeLiveness(ctx context.Context, logger *slog.Logger) error {
	ctx, cancel := context.WithTimeout(ctx, livenessTimeout)
	defer cancel()

	conn, err := s.connect(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to the source: %w", err)
	}
	defer closeConn(ctx, conn, logger)

	var result int
	if err := conn.QueryRow(ctx, "SELECT 1").Scan(&result); err != nil {
		return fmt.Errorf("failed to query the source: %w", err)
	}

	return nil
}

func (s *Source) GetClusterSizeMb(ctx context.Context, logger *slog.Logger) (float64, error) {
	conn, err := s.connect(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to connect to cluster: %w", err)
	}
	defer closeConn(ctx, conn, logger)

	var sizeBytes int64
	if err := conn.QueryRow(ctx, `
		SELECT COALESCE(SUM(pg_database_size(datname)), 0)::bigint
		FROM pg_database
		WHERE datistemplate = false
	`).Scan(&sizeBytes); err != nil {
		return 0, fmt.Errorf("failed to query cluster size: %w", err)
	}

	return float64(sizeBytes) / (1024 * 1024), nil
}

// The `postgres` database always exists, and a physical source has no per-database selection.
func (s *Source) connect(ctx context.Context) (*pgx.Conn, error) {
	credentialSpec := s.settings.GetCredentialSpec()

	files, err := pgshared.WriteCredentialFilesToTempDir(credentialSpec, s.settings.Password)
	if err != nil {
		return nil, err
	}
	defer files.Remove()

	connConfig, err := pgshared.BuildConnConfig(credentialSpec, s.settings.Password, "postgres", files)
	if err != nil {
		return nil, err
	}

	return pgx.ConnectConfig(ctx, connConfig)
}

// connectReplication exercises the "host replication" pg_hba path and the REPLICATION privilege at
// connect time; an ordinary "host all" rule does not cover it, so a logical (replication=database)
// probe would wrongly accept a cluster that real backups cannot stream. It uses the low-level
// pgconn because no ordinary SQL is allowed on a physical replication connection.
func (s *Source) connectReplication(ctx context.Context) (*pgconn.PgConn, error) {
	credentialSpec := s.settings.GetCredentialSpec()

	files, err := pgshared.WriteCredentialFilesToTempDir(credentialSpec, s.settings.Password)
	if err != nil {
		return nil, err
	}
	defer files.Remove()

	connConfig, err := pgshared.BuildReplicationConnConfig(credentialSpec, s.settings.Password, "postgres", files)
	if err != nil {
		return nil, err
	}

	return pgconn.ConnectConfig(ctx, connConfig)
}

func parseVersion(rawServerVersion string) (pgshared.Version, error) {
	matches := versionPattern.FindStringSubmatch(rawServerVersion)
	if len(matches) < 2 {
		return "", fmt.Errorf("%w from: %s", ErrUnreadableVersion, rawServerVersion)
	}

	switch matches[1] {
	case "17":
		return pgshared.Version17, nil
	case "18":
		return pgshared.Version18, nil
	default:
		return "", fmt.Errorf("%w, detected %s", ErrUnsupportedVersion, matches[1])
	}
}

func closeConn(ctx context.Context, conn *pgx.Conn, logger *slog.Logger) {
	if err := conn.Close(ctx); err != nil {
		logger.WarnContext(ctx, "failed to close connection", slog.Any("error", err))
	}
}

func generateReplicationUsername() string {
	return "databasus-" + uuid.New().String()[:8]
}
