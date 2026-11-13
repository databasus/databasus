package physicaltest

import (
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

// requireMatchingCollationVersions fails when the restored server's C library sorts differently from
// the source's: indexes built by the source would then return wrong results.
func requireMatchingCollationVersions(t *testing.T, connection *pgx.Conn) {
	t.Helper()

	var mismatches []string
	require.NoError(t, connection.QueryRow(t.Context(), `SELECT COALESCE(array_agg(name::text), ARRAY[]::text[]) FROM (
		SELECT datname AS name FROM pg_database
		WHERE datcollversion IS NOT NULL AND datcollversion IS DISTINCT FROM pg_database_collation_actual_version(oid)
		UNION ALL
		SELECT collname FROM pg_collation
		WHERE collversion IS NOT NULL AND collversion IS DISTINCT FROM pg_collation_actual_version(oid)
	) mismatched`).Scan(&mismatches))

	require.Empty(t, mismatches, "collation versions differ from the source")
}
