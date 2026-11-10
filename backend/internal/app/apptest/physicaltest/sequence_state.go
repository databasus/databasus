package physicaltest

import (
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type SequenceState struct {
	Name       string
	DataType   string
	Start      int64
	Increment  int64
	Minimum    int64
	Maximum    int64
	Cache      int64
	IsCycle    bool
	Dependency string
}

func (sequences SequenceRecovery) GetStates(t *testing.T, conn *pgx.Conn) []SequenceState {
	t.Helper()

	rows, err := conn.Query(t.Context(), `SELECT namespace.nspname || '.' || relation.relname,
		format_type(sequence.seqtypid, NULL), sequence.seqstart, sequence.seqincrement,
		sequence.seqmin, sequence.seqmax, sequence.seqcache, sequence.seqcycle,
		COALESCE((SELECT owner_namespace.nspname || '.' || owner_table.relname || '.' || attribute.attname || '|' || dependency.deptype::text
			FROM pg_depend dependency JOIN pg_class owner_table ON owner_table.oid = dependency.refobjid
			JOIN pg_namespace owner_namespace ON owner_namespace.oid = owner_table.relnamespace
			JOIN pg_attribute attribute ON attribute.attrelid = owner_table.oid AND attribute.attnum = dependency.refobjsubid
			WHERE dependency.objid = relation.oid AND dependency.classid = 'pg_class'::regclass
			AND dependency.refclassid = 'pg_class'::regclass AND dependency.deptype IN ('a', 'i')), '')
		FROM pg_sequence sequence JOIN pg_class relation ON relation.oid = sequence.seqrelid
		JOIN pg_namespace namespace ON namespace.oid = relation.relnamespace
		WHERE namespace.nspname IN ($1, $2) ORDER BY 1`, sequences.Schema.Namespace, sequences.Schema.AddedNamespace)
	require.NoError(t, err)

	states, err := pgx.CollectRows(rows, pgx.RowToStructByPos[SequenceState])
	require.NoError(t, err)

	return states
}
