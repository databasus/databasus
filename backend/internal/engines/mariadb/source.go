package mariadb

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"

	"databasus-backend/internal/engines/mysqlfamily"
)

const (
	connectionTimeout   = 15 * time.Second
	skipVerifyTLSConfig = "mariadb-skip-verify"
)

// Source is the engine's handle on one MariaDB database.
type Source struct {
	settings    Settings
	clientTools ClientTools
	// generateUsername is a seam: a test makes the generated name collide with an existing account.
	generateUsername func() string
}

// NewSource takes the client tools because a server version counts as supported only when a client
// for it is shipped.
func NewSource(settings Settings, clientTools ClientTools) *Source {
	return &Source{
		settings:         settings,
		clientTools:      clientTools,
		generateUsername: generateReadOnlyUsername,
	}
}

// GetSettings returns the settings the source connects with. Through a tunnel they carry the local
// end of the tunnel, so a client tool the caller starts reaches the same server.
func (s *Source) GetSettings() Settings {
	return s.settings
}

// Inspection is what the server reports about the database and the connecting account.
type Inspection struct {
	Version    Version
	Privileges mysqlfamily.BackupPrivileges
}

// TestConnection pings the database, inspects it, and fails when the account cannot read
// everything the dump would include apart from excludeTables.
func (s *Source) TestConnection(
	ctx context.Context,
	logger *slog.Logger,
	excludeTables []string,
) (Inspection, error) {
	if s.settings.Database == "" {
		return Inspection{}, ErrDatabaseNameRequired
	}

	db, err := s.openDB()
	if err != nil {
		return Inspection{}, fmt.Errorf("failed to connect to MariaDB database '%s': %w", s.settings.Database, err)
	}
	defer closeDB(ctx, db, logger)

	if err := db.PingContext(ctx); err != nil {
		logger.ErrorContext(ctx, "failed to ping the mariadb database",
			slog.String("database_name", s.settings.Database),
			slog.Any("error", err),
		)

		return Inspection{}, fmt.Errorf("failed to ping MariaDB database '%s': %w", s.settings.Database, err)
	}

	inspection, err := s.inspect(ctx, db, logger, excludeTables)
	if err != nil {
		return Inspection{}, err
	}

	if !inspection.Privileges.IsSufficientForDump() {
		return Inspection{}, inspection.Privileges.NewInsufficiencyError()
	}

	return inspection, nil
}

// Inspect detects the version and reads what the account may dump apart from excludeTables,
// without judging whether that is enough.
func (s *Source) Inspect(ctx context.Context, logger *slog.Logger, excludeTables []string) (Inspection, error) {
	db, err := s.openDB()
	if err != nil {
		return Inspection{}, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer closeDB(ctx, db, logger)

	return s.inspect(ctx, db, logger, excludeTables)
}

func (s *Source) DetectVersion(ctx context.Context, logger *slog.Logger) (Version, error) {
	db, err := s.openDB()
	if err != nil {
		return "", fmt.Errorf("failed to connect to database: %w", err)
	}
	defer closeDB(ctx, db, logger)

	return s.detectVersion(ctx, db)
}

func (s *Source) inspect(
	ctx context.Context,
	db *sql.DB,
	logger *slog.Logger,
	excludeTables []string,
) (Inspection, error) {
	version, err := s.detectVersion(ctx, db)
	if err != nil {
		return Inspection{}, err
	}

	backupPrivileges, err := readBackupPrivileges(ctx, backupPrivilegesQuery{
		DB:            db,
		Logger:        logger,
		Version:       version,
		SchemaName:    s.settings.Database,
		ExcludeTables: excludeTables,
	})
	if err != nil {
		return Inspection{}, err
	}

	return Inspection{Version: version, Privileges: backupPrivileges}, nil
}

// openDB opens a pool of one connection, so every query runs in the same session.
func (s *Source) openDB() (*sql.DB, error) {
	db, err := sql.Open("mysql", s.buildDSN())
	if err != nil {
		return nil, err
	}

	db.SetConnMaxLifetime(connectionTimeout)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	return db, nil
}

func (s *Source) buildDSN() string {
	tlsConfig := "false"

	if s.settings.IsHTTPS {
		// The HTTPS option encrypts the connection without verifying the server certificate. A
		// second registration under the same name replaces the first, so the error is ignored.
		// nosemgrep: problem-based-packs.insecure-transport.go-stdlib.bypass-tls-verification.bypass-tls-verification
		_ = mysqldriver.RegisterTLSConfig(skipVerifyTLSConfig, &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: true,
		})

		tlsConfig = skipVerifyTLSConfig
	}

	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?parseTime=true&timeout=15s&tls=%s&charset=utf8mb4",
		s.settings.Username,
		s.settings.Password,
		s.settings.Host,
		s.settings.Port,
		s.settings.Database,
		tlsConfig,
	)
}

func closeDB(ctx context.Context, db *sql.DB, logger *slog.Logger) {
	if err := db.Close(); err != nil {
		logger.ErrorContext(ctx, "failed to close the mariadb connection", slog.Any("error", err))
	}
}

// MariaDB 5.5 limits a user name to 16 characters, so the prefix is shorter than MySQL's.
func generateReadOnlyUsername() string {
	return "pgs-" + uuid.New().String()[:8]
}
