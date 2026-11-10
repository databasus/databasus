package physicaltest

import (
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type SchemaRecoveryStage int

const (
	SchemaBeforeWal SchemaRecoveryStage = iota
	SchemaAfterWal
	SchemaLatest
)

type SchemaState struct {
	Definitions []string
	Rows        []string
}

type recoveryQuery struct {
	Statement string
	Arguments []any
}

func (schema SchemaRecovery) GetState(t *testing.T, conn *pgx.Conn) SchemaState {
	t.Helper()

	definitions := getRecoveryText(t, conn, recoveryQuery{Statement: `
		SELECT 'schema|' || nspname FROM pg_namespace WHERE nspname IN ($1, $2)
		UNION ALL
		SELECT concat_ws('|', 'relation', namespace.nspname || '.' || relation.relname, relation.relkind, relation.relpersistence)
		FROM pg_class relation JOIN pg_namespace namespace ON namespace.oid = relation.relnamespace
		WHERE namespace.nspname IN ($1, $2)
		UNION ALL
		SELECT concat_ws('|', 'column', namespace.nspname || '.' || relation.relname,
			attribute.attname, format_type(attribute.atttypid, attribute.atttypmod), attribute.attnotnull::text,
			COALESCE(pg_get_expr(column_default.adbin, column_default.adrelid), ''), attribute.attidentity)
		FROM pg_attribute attribute JOIN pg_class relation ON relation.oid = attribute.attrelid
		JOIN pg_namespace namespace ON namespace.oid = relation.relnamespace
		LEFT JOIN pg_attrdef column_default ON column_default.adrelid = relation.oid AND column_default.adnum = attribute.attnum
		WHERE namespace.nspname IN ($1, $2) AND attribute.attnum > 0 AND NOT attribute.attisdropped AND relation.relkind = 'r'
		UNION ALL
		SELECT concat_ws('|', 'index', namespace.nspname || '.' || relation.relname,
			pg_get_indexdef(index.indexrelid), index.indisvalid, index.indisready)
		FROM pg_index index JOIN pg_class relation ON relation.oid = index.indexrelid
		JOIN pg_namespace namespace ON namespace.oid = relation.relnamespace
		WHERE namespace.nspname IN ($1, $2)
		UNION ALL
		SELECT concat_ws('|', 'constraint', namespace.nspname || '.' || relation.relname,
			constraint_definition.conname, pg_get_constraintdef(constraint_definition.oid), constraint_definition.convalidated)
		FROM pg_constraint constraint_definition JOIN pg_class relation ON relation.oid = constraint_definition.conrelid
		JOIN pg_namespace namespace ON namespace.oid = relation.relnamespace
		WHERE namespace.nspname IN ($1, $2)
		ORDER BY 1`, Arguments: []any{schema.Namespace, schema.AddedNamespace}})

	tables := getRecoveryText(t, conn, recoveryQuery{Statement: `SELECT namespace.nspname || '.' || relation.relname
		FROM pg_class relation JOIN pg_namespace namespace ON namespace.oid = relation.relnamespace
		WHERE namespace.nspname IN ($1, $2) AND relation.relkind = 'r' ORDER BY 1`, Arguments: []any{schema.Namespace, schema.AddedNamespace}})

	var storedRows []string
	for _, table := range tables {
		identifiers := strings.Split(table, ".")
		rows := getRecoveryText(t, conn, recoveryQuery{Statement: `SELECT to_jsonb(stored_row)::text FROM ` +
			pgx.Identifier(identifiers).Sanitize() + ` stored_row ORDER BY 1`})

		for _, row := range rows {
			storedRows = append(storedRows, table+"|"+row)
		}
	}

	return SchemaState{Definitions: definitions, Rows: storedRows}
}

func (schema SchemaRecovery) sql(statement string) string {
	return strings.NewReplacer("{base}", pgx.Identifier{schema.Namespace}.Sanitize(),
		"{added}", pgx.Identifier{schema.AddedNamespace}.Sanitize()).Replace(statement)
}

func (schema SchemaRecovery) execute(t *testing.T, conn *pgx.Conn, statement string) {
	t.Helper()

	_, err := conn.Exec(t.Context(), schema.sql(statement))
	require.NoError(t, err)
}

func getRecoveryText(t *testing.T, conn *pgx.Conn, query recoveryQuery) []string {
	t.Helper()

	rows, err := conn.Query(t.Context(), query.Statement, query.Arguments...)
	require.NoError(t, err)

	values, err := pgx.CollectRows(rows, pgx.RowTo[string])
	require.NoError(t, err)

	return values
}
