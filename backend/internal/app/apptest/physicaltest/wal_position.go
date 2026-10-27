package physicaltest

import (
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func parseLSN(t *testing.T, value string) uint64 {
	t.Helper()

	high, low, hasSeparator := strings.Cut(value, "/")
	require.True(t, hasSeparator, "malformed LSN %q", value)
	highBits, err := strconv.ParseUint(high, 16, 32)
	require.NoError(t, err)
	lowBits, err := strconv.ParseUint(low, 16, 32)
	require.NoError(t, err)
	return highBits<<32 | lowBits
}

func currentLSN(t *testing.T, conn *pgx.Conn) uint64 {
	t.Helper()

	var position string
	require.NoError(t, conn.QueryRow(t.Context(), "SELECT pg_current_wal_lsn()::text").Scan(&position))
	return parseLSN(t, position)
}
