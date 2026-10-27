package physicaltest

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

const (
	restoredPgUser     = "testuser"
	restoredPgPassword = "testpassword"
	restoredPgDatabase = "testdb"
)

func openConnAt(t *testing.T, host string, port int) *pgx.Conn {
	t.Helper()

	conn, err := pgx.Connect(
		t.Context(),
		fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			host,
			port,
			restoredPgUser,
			restoredPgPassword,
			restoredPgDatabase,
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close(context.WithoutCancel(t.Context())) })
	return conn
}
