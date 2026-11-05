package mysql

import (
	"context"
	"database/sql"
	"log/slog"

	"databasus-backend/internal/engines/mysqlfamily"
)

const noActiveRolesMarker = "NONE"

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

// SHOW GRANTS reports role membership without expanding it, so a role's privileges have to be
// asked for separately or a role-based backup account looks like it holds nothing.
func readGrantLines(ctx context.Context, query backupPrivilegesQuery) ([]string, error) {
	grantLines, err := mysqlfamily.QueryGrantLines(
		ctx,
		query.DB,
		"SHOW GRANTS FOR CURRENT_USER()",
	)
	if err != nil {
		return nil, err
	}

	if query.Version == Version57 {
		return grantLines, nil
	}

	// Only the roles activated for this session count: mysqldump logs in the same way and, with
	// activate_all_roles_on_login off, a granted but inactive role gives it nothing.
	activeRoles, err := readActiveRoles(ctx, query.DB)
	if err != nil {
		query.Logger.WarnContext(ctx, "failed to read active roles", slog.Any("error", err))

		return grantLines, nil
	}

	if activeRoles == "" {
		return grantLines, nil
	}

	// The server returns the roles already quoted, so re-quoting them here would corrupt a role
	// name that contains a backquote. The value never comes from user input.
	roleGrantLines, err := mysqlfamily.QueryGrantLines(
		ctx,
		query.DB,
		"SHOW GRANTS FOR CURRENT_USER() USING "+activeRoles,
	)
	if err != nil {
		query.Logger.WarnContext(ctx, "failed to expand active roles", slog.Any("error", err))

		return grantLines, nil
	}

	return append(grantLines, roleGrantLines...), nil
}

func readActiveRoles(ctx context.Context, db *sql.DB) (string, error) {
	var activeRoles sql.NullString
	if err := db.QueryRowContext(ctx, "SELECT CURRENT_ROLE()").Scan(&activeRoles); err != nil {
		return "", err
	}

	if !activeRoles.Valid || activeRoles.String == noActiveRolesMarker {
		return "", nil
	}

	return activeRoles.String, nil
}
