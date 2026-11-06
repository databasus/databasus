package logical

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"databasus-backend/internal/lib/namelist"
)

// has_table_privilege is asked by oid rather than by name so it reaches pg_class_aclcheck
// directly: the name form resolves through the schema and raises insufficient_privilege for
// a role that lacks USAGE, which would hide the very gap this looks for. One function covers
// sequences too, unlike has_sequence_privilege, which errors on a non-sequence.
const unreadableRelationsQuery = `
	SELECT n.nspname,
	       c.relname,
	       pg_catalog.pg_table_is_visible(c.oid),
	       NOT COALESCE(has_schema_privilege(current_user, n.oid, 'USAGE'), false)
` + dumpedRelationScope + `
	  AND c.relkind IN ('r', 'p', 'S')
	  AND NOT (
	    COALESCE(has_schema_privilege(current_user, n.oid, 'USAGE'), false)
	    AND COALESCE(has_table_privilege(current_user, c.oid, 'SELECT'), false)
	  )
	ORDER BY 1, 2
`

type unreadableRelation struct {
	relationName
	IsSchemaUsageMissing bool
}

func checkDumpReadPrivileges(ctx context.Context, conn *pgx.Conn, filter DumpFilter) error {
	unreadableRelations, err := getUnreadableDumpedRelations(ctx, conn, filter)
	if err != nil {
		return err
	}

	if len(unreadableRelations) == 0 {
		return nil
	}

	var schemasMissingUsage []string
	var relationsMissingSelect []string
	seenSchemas := make(map[string]struct{})

	for _, relation := range unreadableRelations {
		if !relation.IsSchemaUsageMissing {
			relationsMissingSelect = append(
				relationsMissingSelect,
				relation.SchemaName+"."+relation.TableName,
			)

			continue
		}

		if _, isSeen := seenSchemas[relation.SchemaName]; isSeen {
			continue
		}

		seenSchemas[relation.SchemaName] = struct{}{}
		schemasMissingUsage = append(schemasMissingUsage, relation.SchemaName)
	}

	return newInsufficientDumpPrivilegesError(ctx, conn, missingDumpPrivileges{
		SchemasMissingUsage:    schemasMissingUsage,
		RelationsMissingSelect: relationsMissingSelect,
	})
}

type missingDumpPrivileges struct {
	SchemasMissingUsage    []string
	RelationsMissingSelect []string
}

func getUnreadableDumpedRelations(
	ctx context.Context,
	conn *pgx.Conn,
	filter DumpFilter,
) ([]unreadableRelation, error) {
	rows, err := queryDumpedRelations(ctx, conn, unreadableRelationsQuery, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to check dump read privileges: %w", err)
	}

	defer rows.Close()

	excludeTablePatterns := filter.getExcludeTablePatterns()

	var unreadableRelations []unreadableRelation

	for rows.Next() {
		var relation unreadableRelation

		if err := rows.Scan(
			&relation.SchemaName,
			&relation.TableName,
			&relation.IsVisibleOnSearchPath,
			&relation.IsSchemaUsageMissing,
		); err != nil {
			return nil, fmt.Errorf("failed to scan unreadable relation: %w", err)
		}

		if isRelationExcluded(relation.relationName, excludeTablePatterns) {
			continue
		}

		unreadableRelations = append(unreadableRelations, relation)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read unreadable relations: %w", err)
	}

	return unreadableRelations, nil
}

func newInsufficientDumpPrivilegesError(
	ctx context.Context,
	conn *pgx.Conn,
	missing missingDumpPrivileges,
) error {
	// A pooled Supabase login carries a "postgres.<project>" username that no GRANT can name,
	// so the role has to come from the server rather than from the stored credentials.
	roleName, err := getCurrentRoleName(ctx, conn)
	if err != nil {
		return err
	}

	messageParts := []string{
		fmt.Sprintf(
			"insufficient permissions for backup: role %q cannot read everything pg_dump would dump.",
			roleName,
		),
	}

	if len(missing.SchemasMissingUsage) > 0 {
		messageParts = append(messageParts, fmt.Sprintf(
			"Missing USAGE on schemas: %s.",
			namelist.FormatTruncatedNames(missing.SchemasMissingUsage),
		))
	}

	if len(missing.RelationsMissingSelect) > 0 {
		messageParts = append(messageParts, fmt.Sprintf(
			"Missing SELECT on: %s.",
			namelist.FormatTruncatedNames(missing.RelationsMissingSelect),
		))
	}

	messageParts = append(messageParts, fmt.Sprintf(
		"Grant USAGE ON SCHEMA, SELECT ON ALL TABLES IN SCHEMA and SELECT ON ALL SEQUENCES IN SCHEMA "+
			"to %q for every schema listed above, or narrow the backup with include schemas and whole-table exclusion. "+
			"Note that excluding a table does not exclude the sequences it owns, and excluding only "+
			"a table's data still requires SELECT on that table.",
		roleName,
	))

	return fmt.Errorf("%s", strings.Join(messageParts, " "))
}

// PostgreSQL masks pg_user_mappings.umoptions (credentials) from any role that is not a superuser,
// the foreign server owner, or the mapping's own user. Such a role's pg_dump emits a bare CREATE
// USER MAPPING that loses the credentials and breaks restore for FDWs that require them (e.g.
// oracle_fdw), so refuse the backup when any mapping's options are hidden.
func checkUserMappingsReadable(ctx context.Context, conn *pgx.Conn) error {
	var unreadableCount int
	err := conn.QueryRow(ctx, "SELECT count(*) FROM pg_user_mappings WHERE umoptions IS NULL").
		Scan(&unreadableCount)
	if err != nil {
		return fmt.Errorf("cannot check user mapping options: %w", err)
	}

	if unreadableCount > 0 {
		return fmt.Errorf(
			"database has %d user mapping(s) whose options this role cannot read; their "+
				"credentials would be lost on restore — back up as a superuser or the foreign "+
				"server/mapping owner, or enable 'skip user mappings'",
			unreadableCount,
		)
	}

	return nil
}

// Generated read-only roles must not receive BYPASSRLS or ownership of source tables.
func refuseRowLevelSecurityTables(ctx context.Context, conn *pgx.Conn, includeSchemas []string) error {
	rowLevelSecurityTables, err := getRowLevelSecurityTables(ctx, conn, includeSchemas)
	if err != nil {
		return err
	}

	if len(rowLevelSecurityTables) > 0 {
		return fmt.Errorf(
			"%w on %s; use a role with BYPASSRLS, a superuser, or an effective table owner when RLS is not forced",
			ErrRowLevelSecurity,
			namelist.FormatTruncatedNames(rowLevelSecurityTables),
		)
	}

	return nil
}

// Generated read-only roles neither own the source tables nor receive BYPASSRLS.
func getRowLevelSecurityTables(
	ctx context.Context,
	conn *pgx.Conn,
	includeSchemas []string,
) ([]string, error) {
	query := `
		SELECT n.nspname || '.' || c.relname
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relrowsecurity
		AND c.relkind = 'r'
		AND n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		AND n.nspname NOT LIKE 'pg_temp_%'
		AND n.nspname NOT LIKE 'pg_toast_temp_%'
	`

	var rows pgx.Rows
	var err error

	if len(includeSchemas) > 0 {
		rows, err = conn.Query(ctx, query+" AND n.nspname = ANY($1::text[]) ORDER BY 1", includeSchemas)
	} else {
		rows, err = conn.Query(ctx, query+" ORDER BY 1")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to check row-level security: %w", err)
	}

	defer rows.Close()

	var rowLevelSecurityTables []string

	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, fmt.Errorf("failed to scan row-level security table: %w", err)
		}

		rowLevelSecurityTables = append(rowLevelSecurityTables, table)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating row-level security tables: %w", err)
	}

	return rowLevelSecurityTables, nil
}
