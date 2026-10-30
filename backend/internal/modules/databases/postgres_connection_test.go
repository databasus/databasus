package databases_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
)

// connectAsAdmin opens a connection to a database of a PostgreSQL container as its superuser.
func connectAsAdmin(t *testing.T, server containers.Endpoint, databaseName string) *pgx.Conn {
	t.Helper()

	return connectToPostgres(t, server, databaseName, containers.PostgresUsername, containers.PostgresPassword)
}

func connectToPostgres(t *testing.T, server containers.Endpoint, databaseName, username, password string) *pgx.Conn {
	t.Helper()

	conn, err := pgx.Connect(t.Context(), fmt.Sprintf(
		"host=%s port=%d user='%s' password='%s' dbname=%s sslmode=disable",
		server.Host, server.Port, username, password, databaseName,
	))
	require.NoError(t, err)

	t.Cleanup(func() { _ = conn.Close(t.Context()) })

	return conn
}

func newUniqueName(prefix string) string {
	return prefix + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
}
