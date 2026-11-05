package mariadb

import (
	"context"
	"fmt"
	"log/slog"

	"databasus-backend/internal/engines/mysqlfamily"
)

//nolint:gochecknoglobals // Shared immutable write-privilege lookup is reused for each grant during read-only user checks; it is never mutated.
var writePrivileges = map[string]bool{
	"INSERT":            true,
	"UPDATE":            true,
	"DELETE":            true,
	"CREATE":            true,
	"DROP":              true,
	"ALTER":             true,
	"INDEX":             true,
	"GRANT OPTION":      true,
	"ALL PRIVILEGES":    true,
	"SUPER":             true,
	"EXECUTE":           true,
	"FILE":              true,
	"RELOAD":            true,
	"SHUTDOWN":          true,
	"CREATE ROUTINE":    true,
	"ALTER ROUTINE":     true,
	"CREATE USER":       true,
	"CREATE TABLESPACE": true,
	"DELETE HISTORY":    true,
	"REFERENCES":        true,
}

// ShouldSuggestReadOnlyUser lists what the connecting account may do beyond reading, and suggests a
// read-only user when it may do anything more.
func (s *Source) ShouldSuggestReadOnlyUser(ctx context.Context, logger *slog.Logger) (bool, []string, error) {
	db, err := s.openDB()
	if err != nil {
		return false, nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer closeDB(ctx, db, logger)

	grantLines, err := mysqlfamily.QueryGrantLines(ctx, db, "SHOW GRANTS FOR CURRENT_USER()")
	if err != nil {
		return false, nil, err
	}

	privileges := mysqlfamily.ListGrantedWritePrivileges(grantLines, writePrivileges)

	return len(privileges) > 0, privileges, nil
}

// CreateReadOnlyUser creates an account that can read and dump the database and returns its name
// and password.
func (s *Source) CreateReadOnlyUser(ctx context.Context, logger *slog.Logger) (string, string, error) {
	db, err := s.openDB()
	if err != nil {
		return "", "", fmt.Errorf("failed to connect to database: %w", err)
	}
	defer closeDB(ctx, db, logger)

	username, password, err := mysqlfamily.CreateReadOnlyUser(ctx, mysqlfamily.ReadOnlyUserSpec{
		DB:               db,
		DatabaseName:     s.settings.Database,
		GenerateUsername: s.generateUsername,
		Logger:           logger,
	})
	if err != nil {
		return "", "", err
	}

	logger.InfoContext(ctx, "read-only MariaDB user created successfully", slog.String("username", username))

	return username, password, nil
}
