package mysqlfamily

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"

	"databasus-backend/internal/lib/namelist"
)

type BackupPrivilegesQuery struct {
	DB            *sql.DB
	SchemaName    string
	ExcludeTables []string

	// Queried by the caller, because that is where the engines differ: MySQL expands its active
	// roles with a USING clause, MariaDB walks a single active role instead.
	GrantLines []string
}

func ReadBackupPrivileges(
	ctx context.Context,
	query BackupPrivilegesQuery,
) (BackupPrivileges, error) {
	grantee, err := readGrantee(ctx, query.DB)
	if err != nil {
		return BackupPrivileges{}, err
	}

	dumpedTables, err := readDumpedTables(ctx, query)
	if err != nil {
		return BackupPrivileges{}, err
	}

	return GetBackupPrivileges(BackupPrivilegesSpec{
		SchemaName:   query.SchemaName,
		Grantee:      grantee,
		DumpedTables: dumpedTables,
		Grants:       CollectSchemaGrants(query.GrantLines, query.SchemaName),
	}), nil
}

// CURRENT_USER() returns "name@host". A username may itself contain an @ while a hostname cannot,
// so the account splits on the last one.
func readGrantee(ctx context.Context, db *sql.DB) (string, error) {
	var currentUser string
	if err := db.QueryRowContext(ctx, "SELECT CURRENT_USER()").Scan(&currentUser); err != nil {
		return "", fmt.Errorf("failed to read the current user: %w", err)
	}

	separatorIndex := strings.LastIndex(currentUser, "@")
	if separatorIndex < 0 {
		return fmt.Sprintf("'%s'", currentUser), nil
	}

	return fmt.Sprintf(
		"'%s'@'%s'",
		currentUser[:separatorIndex],
		currentUser[separatorIndex+1:],
	), nil
}

func QueryGrantLines(ctx context.Context, db *sql.DB, query string) ([]string, error) {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to check grants: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var grantLines []string

	for rows.Next() {
		var grantLine string
		if err := rows.Scan(&grantLine); err != nil {
			return nil, fmt.Errorf("failed to scan grant: %w", err)
		}

		grantLines = append(grantLines, grantLine)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating grants: %w", err)
	}

	return grantLines, nil
}

// information_schema.TABLES is itself filtered by privilege, so this is the set the dump tool will
// enumerate: a table the account cannot see at all is skipped by both it and us.
func readDumpedTables(ctx context.Context, query BackupPrivilegesQuery) ([]DumpedTable, error) {
	rows, err := query.DB.QueryContext(ctx, `
		SELECT TABLE_NAME, TABLE_TYPE
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = ? AND TABLE_TYPE IN ('BASE TABLE', 'VIEW')
	`, query.SchemaName)
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}
	defer func() { _ = rows.Close() }()

	excludedTableNames := namelist.NormalizeUniqueNames(query.ExcludeTables)
	var dumpedTables []DumpedTable

	for rows.Next() {
		var tableName string
		var tableType string
		if err := rows.Scan(&tableName, &tableType); err != nil {
			return nil, fmt.Errorf("failed to scan dumped table: %w", err)
		}

		// --ignore-table matches the name exactly, and table names are case sensitive on Linux.
		if slices.Contains(excludedTableNames, tableName) {
			continue
		}

		dumpedTables = append(dumpedTables, DumpedTable{
			Name:   tableName,
			IsView: tableType == viewTableType,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating table names: %w", err)
	}

	return dumpedTables, nil
}
