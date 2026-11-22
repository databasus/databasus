package neon

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
)

const containerImage = "postgres:18"

// startContainerServer starts the container twin of the Neon project: a PostgreSQL server of the
// harness reached through a connection URL, like the Neon test reaches its project.
func startContainerServer(t *testing.T) (containers.Endpoint, string) {
	t.Helper()

	endpoint := containers.StartPostgres(t, containerImage)

	return endpoint, newContainerConnectionURL(endpoint, containers.PostgresPassword)
}

func newContainerConnectionURL(endpoint containers.Endpoint, password string) string {
	return fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=disable",
		url.PathEscape(containers.PostgresUsername), url.PathEscape(password),
		endpoint.Host, endpoint.Port, containers.PostgresDatabase)
}

func parseContainerServer(t *testing.T, connectionURL string) managedServer {
	t.Helper()

	server, err := parseManagedServer(connectionURL)
	require.NoError(t, err)

	return server
}
