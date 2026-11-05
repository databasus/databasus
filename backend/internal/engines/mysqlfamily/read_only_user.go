package mysqlfamily

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"slices"

	"databasus-backend/internal/lib/crypto"
)

const maxReadOnlyUserAttempts = 3

// ListGrantedWritePrivileges returns which of writePrivileges the grant lines hold. Each engine
// passes its own set, because MariaDB knows privileges MySQL does not.
func ListGrantedWritePrivileges(grantLines []string, writePrivileges map[string]bool) []string {
	grantedWritePrivileges := map[string]bool{}

	for _, grantLine := range grantLines {
		grant := ParseGrantLine(grantLine)
		if grant == nil {
			continue
		}

		for _, privilege := range grant.Privileges {
			if writePrivileges[privilege] {
				grantedWritePrivileges[privilege] = true
			}
		}
	}

	privileges := make([]string, 0, len(grantedWritePrivileges))
	for privilege := range grantedWritePrivileges {
		privileges = append(privileges, privilege)
	}

	slices.Sort(privileges)

	return privileges
}

type ReadOnlyUserSpec struct {
	DB           *sql.DB
	DatabaseName string
	// GenerateUsername is a seam: a test makes the generated name collide with an existing account.
	GenerateUsername func() string
	Logger           *slog.Logger
}

// CreateReadOnlyUser creates an account that can read and dump one schema and returns its name and
// password. A generated name that is already taken is retried with a new one; any other failure
// ends the creation.
func CreateReadOnlyUser(ctx context.Context, spec ReadOnlyUserSpec) (string, string, error) {
	for range maxReadOnlyUserAttempts {
		username := spec.GenerateUsername()

		password, err := createReadOnlyUser(ctx, spec, username)
		if classifyServerError(err) == serverErrorNameTaken {
			continue
		}

		if err != nil {
			return "", "", err
		}

		return username, password, nil
	}

	return "", "", fmt.Errorf("%w after %d attempts", ErrUserNameCollision, maxReadOnlyUserAttempts)
}

// createReadOnlyUser is one attempt in a transaction of its own, so a failed attempt is rolled back
// before the next one starts.
func createReadOnlyUser(ctx context.Context, spec ReadOnlyUserSpec, username string) (string, error) {
	tx, err := spec.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}

	isCommitted := false

	defer func() {
		if isCommitted {
			return
		}

		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			spec.Logger.ErrorContext(ctx, "failed to rollback transaction", slog.Any("error", rollbackErr))
		}
	}()

	password := crypto.GeneratePassword()

	if _, err := tx.ExecContext(ctx, fmt.Sprintf(
		"CREATE USER '%s'@'%%' IDENTIFIED BY '%s'",
		username,
		password,
	)); err != nil {
		return "", fmt.Errorf("failed to create user: %w", err)
	}

	if _, err := tx.ExecContext(ctx, fmt.Sprintf(
		"GRANT SELECT, SHOW VIEW, LOCK TABLES, TRIGGER, EVENT ON `%s`.* TO '%s'@'%%'",
		spec.DatabaseName,
		username,
	)); err != nil {
		return "", fmt.Errorf("failed to grant database privileges: %w", err)
	}

	if _, err := tx.ExecContext(ctx, "FLUSH PRIVILEGES"); err != nil {
		return "", fmt.Errorf("failed to flush privileges: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	isCommitted = true

	return password, nil
}
