package mariadb

import (
	"context"
	"database/sql"
	"log/slog"
	"strings"

	"databasus-backend/internal/engines/mysqlfamily"
)

type backupPrivilegesQuery struct {
	DB            *sql.DB
	Logger        *slog.Logger
	Version       Version
	SchemaName    string
	ExcludeTables []string
}

func readBackupPrivileges(
	ctx context.Context,
	query backupPrivilegesQuery,
) (mysqlfamily.BackupPrivileges, error) {
	grantLines, err := readGrantLines(ctx, query)
	if err != nil {
		return mysqlfamily.BackupPrivileges{}, err
	}

	return mysqlfamily.ReadBackupPrivileges(ctx, mysqlfamily.BackupPrivilegesQuery{
		DB:            query.DB,
		SchemaName:    query.SchemaName,
		ExcludeTables: query.ExcludeTables,
		GrantLines:    grantLines,
	})
}

func readGrantLines(ctx context.Context, query backupPrivilegesQuery) ([]string, error) {
	grantLines, err := mysqlfamily.QueryGrantLines(ctx, query.DB, "SHOW GRANTS FOR CURRENT_USER")
	if err != nil {
		return nil, err
	}

	if query.Version == Version55 {
		return grantLines, nil
	}

	activeRole, err := readActiveRole(ctx, query.DB)
	if err != nil {
		query.Logger.WarnContext(ctx, "failed to read the active role", slog.Any("error", err))

		return grantLines, nil
	}

	if activeRole == "" {
		return grantLines, nil
	}

	// SHOW GRANTS FOR a role already includes every role granted to it, transitively, so one
	// query covers the whole hierarchy behind the active role.
	activeRoleGrantLines, err := mysqlfamily.QueryGrantLines(
		ctx,
		query.DB,
		"SHOW GRANTS FOR "+quoteRoleName(activeRole),
	)
	if err != nil {
		query.Logger.WarnContext(ctx, "failed to read role grants", slog.Any("error", err))

		return grantLines, nil
	}

	return append(grantLines, activeRoleGrantLines...), nil
}

// MariaDB activates a single role at a time and reports it without a host part, unlike MySQL,
// which can hold several and needs them spliced into a USING clause.
func readActiveRole(ctx context.Context, db *sql.DB) (string, error) {
	var activeRole sql.NullString
	if err := db.QueryRowContext(ctx, "SELECT CURRENT_ROLE()").Scan(&activeRole); err != nil {
		return "", err
	}

	if !activeRole.Valid {
		return "", nil
	}

	return activeRole.String, nil
}

func quoteRoleName(roleName string) string {
	return "`" + strings.ReplaceAll(roleName, "`", "``") + "`"
}
