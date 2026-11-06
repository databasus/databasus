package logical

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"databasus-backend/internal/lib/namelist"
)

const rlsRestrictedRelationsQuery = `
	SELECT n.nspname, c.relname, pg_catalog.pg_table_is_visible(c.oid)
` + dumpedRelationScope + `
	  AND c.relkind = 'r'
	  AND pg_catalog.row_security_active(c.oid)
	ORDER BY 1, 2
`

func checkDumpRLSPrivileges(ctx context.Context, conn *pgx.Conn, filter DumpFilter) error {
	relations, err := getRLSRestrictedDumpedRelations(ctx, conn, filter)
	if err != nil {
		return err
	}

	if len(relations) == 0 {
		return nil
	}

	roleName, err := getCurrentRoleName(ctx, conn)
	if err != nil {
		return err
	}

	return fmt.Errorf(
		"%w: role %q is subject to row-level security (RLS) on %s, preventing a complete logical backup; "+
			"use a backup role authorized for complete row access or exclude the affected table data",
		ErrRowLevelSecurity, roleName, namelist.FormatTruncatedNames(relations),
	)
}

func getRLSRestrictedDumpedRelations(ctx context.Context, conn *pgx.Conn, filter DumpFilter) ([]string, error) {
	rows, err := queryDumpedRelations(ctx, conn, rlsRestrictedRelationsQuery, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to check dump RLS privileges: %w", err)
	}
	defer rows.Close()

	excludePatterns := append(filter.getExcludeTablePatterns(), filter.getExcludeTableDataPatterns()...)
	var relations []string

	for rows.Next() {
		var relation relationName
		if err := rows.Scan(&relation.SchemaName, &relation.TableName, &relation.IsVisibleOnSearchPath); err != nil {
			return nil, fmt.Errorf("failed to scan RLS-restricted relation: %w", err)
		}

		if !isRelationExcluded(relation, excludePatterns) {
			relations = append(relations, relation.SchemaName+"."+relation.TableName)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read RLS-restricted relations: %w", err)
	}

	return relations, nil
}
