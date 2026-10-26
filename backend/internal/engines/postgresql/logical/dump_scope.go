package logical

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/jackc/pgx/v5"

	"databasus-backend/internal/lib/namelist"
)

// pg_dump emits no data for a relation that belongs to an extension, so such a relation
// needs no read privilege and adds nothing to the restored size. The exception is a table
// the extension registered through pg_extension_config_dump: its contents are dumped like
// any other, which is how TimescaleDB's catalog reaches the archive.
const dumpedRelationScope = `
	FROM pg_class c
	JOIN pg_namespace n ON n.oid = c.relnamespace
	WHERE n.nspname NOT IN ('pg_catalog', 'information_schema')
	  AND n.nspname NOT LIKE 'pg\_toast%'
	  AND n.nspname NOT LIKE 'pg\_temp%'
	  AND (cardinality($1::text[]) = 0 OR n.nspname = ANY($1::text[]))
	  AND (
	    NOT EXISTS (
	      SELECT 1
	      FROM pg_depend d
	      WHERE d.classid = 'pg_class'::regclass
	        AND d.objid = c.oid
	        AND d.objsubid = 0
	        AND d.deptype = 'e'
	    )
	    OR EXISTS (SELECT 1 FROM pg_extension e WHERE c.oid = ANY(e.extconfig))
	  )
`

type DumpFilter struct {
	IncludeSchemas           []string
	ExcludeTablePatterns     []string
	ExcludeTableDataPatterns []string
}

type relationName struct {
	SchemaName            string
	TableName             string
	IsVisibleOnSearchPath bool
}

// pg_dump resolves table patterns right after RESET search_path, so whether an unqualified
// pattern reaches a table depends on the role's default search path, not on this session's.
func queryDumpedRelations(
	ctx context.Context,
	conn *pgx.Conn,
	query string,
	filter DumpFilter,
) (pgx.Rows, error) {
	if _, err := conn.Exec(ctx, "RESET search_path"); err != nil {
		return nil, fmt.Errorf("failed to reset search path: %w", err)
	}

	return conn.Query(ctx, query, filter.getIncludedSchemaNames())
}

// The dump arguments are built from normalized names, so a pasted multiline value must be
// normalized here too or the scope query would silently match nothing.
func (f DumpFilter) getIncludedSchemaNames() []string {
	return namelist.NormalizeUniqueNames(f.IncludeSchemas)
}

// Same parity requirement as getIncludedSchemaNames: pg_dump receives the exclude patterns
// normalized, so an unnormalized pattern here would demand SELECT on a table the dump skips.
func (f DumpFilter) getExcludeTablePatterns() []string {
	return namelist.NormalizeUniqueNames(f.ExcludeTablePatterns)
}

func (f DumpFilter) getExcludeTableDataPatterns() []string {
	return namelist.NormalizeUniqueNames(f.ExcludeTableDataPatterns)
}

func isRelationExcluded(relation relationName, excludeTablePatterns []string) bool {
	for _, excludeTablePattern := range excludeTablePatterns {
		if parseDumpTablePattern(excludeTablePattern).isMatchingRelation(relation) {
			return true
		}
	}

	return false
}

type dumpTablePattern struct {
	SchemaGlob        globPattern
	TableGlob         globPattern
	IsSchemaQualified bool
}

func (p dumpTablePattern) isMatchingRelation(relation relationName) bool {
	if p.IsSchemaQualified && !p.SchemaGlob.isMatchingName(relation.SchemaName) {
		return false
	}

	if !p.IsSchemaQualified && !relation.IsVisibleOnSearchPath {
		return false
	}

	return p.TableGlob.isMatchingName(relation.TableName)
}

// Mirrors pg_dump's table patterns: an unqualified pattern hits only tables visible on the
// default search path, as in psql, and the split happens on the first dot outside double quotes.
func parseDumpTablePattern(excludeTablePattern string) dumpTablePattern {
	isInsideQuotes := false

	for index, character := range excludeTablePattern {
		switch character {
		case '"':
			isInsideQuotes = !isInsideQuotes
		case '.':
			if !isInsideQuotes {
				return dumpTablePattern{
					SchemaGlob:        parseGlobPattern(excludeTablePattern[:index]),
					TableGlob:         parseGlobPattern(excludeTablePattern[index+1:]),
					IsSchemaQualified: true,
				}
			}
		}
	}

	return dumpTablePattern{TableGlob: parseGlobPattern(excludeTablePattern)}
}

type globPattern struct {
	Value     string
	IsLiteral bool
}

func parseGlobPattern(patternPart string) globPattern {
	if len(patternPart) >= 2 &&
		strings.HasPrefix(patternPart, `"`) &&
		strings.HasSuffix(patternPart, `"`) {
		unquoted := strings.ReplaceAll(patternPart[1:len(patternPart)-1], `""`, `"`)

		return globPattern{Value: unquoted, IsLiteral: true}
	}

	return globPattern{Value: patternPart}
}

func (g globPattern) isMatchingName(name string) bool {
	if g.IsLiteral {
		return g.Value == name
	}

	isMatching, err := path.Match(g.Value, name)
	if err != nil {
		return g.Value == name
	}

	return isMatching
}
