package logical

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
)

// Some builds omit the minor version: "PostgreSQL 16 maintained by ...".
var versionPattern = regexp.MustCompile(`PostgreSQL (\d+)`)

// Source is the engine's handle on one logical PostgreSQL database.
type Source struct {
	settings Settings
	// generateUsername is a seam: a test makes the generated name collide with an existing role.
	generateUsername func() string
}

func NewSource(settings Settings) *Source {
	return &Source{settings: settings, generateUsername: generateReadOnlyUsername}
}

// GetSettings returns the settings the source connects with. Through a tunnel they carry the local
// end of the tunnel, so a client tool the caller starts reaches the same server.
func (s *Source) GetSettings() Settings {
	return s.settings
}

// DumpCheck is what a connection test checks beyond reaching the database: that the role can read
// everything pg_dump would dump in this scope.
type DumpCheck struct {
	Filter             DumpFilter
	IsSkipUserMappings bool
}

// TestConnection connects, detects the version and checks that the role can read everything the
// dump would include.
func (s *Source) TestConnection(
	ctx context.Context,
	logger *slog.Logger,
	check DumpCheck,
) (pgshared.Version, error) {
	if s.settings.Database == "" {
		return "", ErrDatabaseNameRequired
	}

	conn, err := s.connect(ctx, s.settings.Database)
	if err != nil {
		return "", fmt.Errorf("failed to connect to database '%s': %w", s.settings.Database, err)
	}
	defer closeConn(ctx, conn, logger)

	version, err := detectVersion(ctx, conn)
	if err != nil {
		return "", err
	}

	if err := checkDumpReadPrivileges(ctx, conn, check.Filter); err != nil {
		return "", err
	}

	if !check.IsSkipUserMappings {
		if err := checkUserMappingsReadable(ctx, conn); err != nil {
			return "", err
		}
	}

	return version, nil
}

func (s *Source) DetectVersion(ctx context.Context, logger *slog.Logger) (pgshared.Version, error) {
	conn, err := s.connect(ctx, s.settings.Database)
	if err != nil {
		return "", fmt.Errorf("failed to connect to database: %w", err)
	}
	defer closeConn(ctx, conn, logger)

	return detectVersion(ctx, conn)
}

// connect writes the credential files, opens a pgx connection to databaseName and removes the files
// once the TLS handshake has completed.
func (s *Source) connect(ctx context.Context, databaseName string) (*pgx.Conn, error) {
	credentialSpec := s.settings.GetCredentialSpec()

	files, err := pgshared.WriteCredentialFilesToTempDir(credentialSpec, s.settings.Password)
	if err != nil {
		return nil, err
	}
	defer files.Remove()

	connConfig, err := pgshared.BuildConnConfig(credentialSpec, s.settings.Password, databaseName, files)
	if err != nil {
		return nil, err
	}

	return pgx.ConnectConfig(ctx, connConfig)
}

func detectVersion(ctx context.Context, conn *pgx.Conn) (pgshared.Version, error) {
	var versionText string

	if err := conn.QueryRow(ctx, "SELECT version()").Scan(&versionText); err != nil {
		return "", fmt.Errorf("failed to query database version: %w", err)
	}

	matches := versionPattern.FindStringSubmatch(versionText)
	if len(matches) < 2 {
		return "", fmt.Errorf("%w from: %s", ErrUnreadableVersion, versionText)
	}

	switch majorVersion := matches[1]; majorVersion {
	case "12", "13", "14", "15", "16", "17", "18":
		return pgshared.Version(majorVersion), nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedVersion, majorVersion)
	}
}

func closeConn(ctx context.Context, conn *pgx.Conn, logger *slog.Logger) {
	if err := conn.Close(ctx); err != nil {
		logger.ErrorContext(ctx, "failed to close connection", slog.Any("error", err))
	}
}

func generateReadOnlyUsername() string {
	return "databasus-" + uuid.New().String()[:8]
}
