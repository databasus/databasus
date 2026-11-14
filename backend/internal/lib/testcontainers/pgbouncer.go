package containers

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"databasus-backend/internal/lib/testwait"
)

type PgBouncer struct {
	ContainerHandle
	users string
}

type PgBouncerCredential struct {
	Username string
	Password string
}

func StartPostgresThroughPgBouncer(t *testing.T, image string) *PgBouncer {
	t.Helper()
	return startPgBouncer(t, postgresRequest(image))
}

func StartPhysicalPostgresThroughPgBouncer(t *testing.T, image string) *PgBouncer {
	t.Helper()
	return startPgBouncer(t, physicalPostgresRequest(image, resolvePhysicalPostgresOptions(nil)))
}

func startPgBouncer(t *testing.T, source testcontainers.ContainerRequest) *PgBouncer {
	t.Helper()
	networkName := StartNetwork(t)
	source.WaitingFor = wait.ForLog("database system is ready to accept connections").
		WithOccurrence(2).
		WithStartupTimeout(postgresStartupTimeout)
	startUnpublished(t, source, NetworkPlacement{NetworkName: networkName, Alias: "postgres"})

	users := fmt.Sprintf("%q %q\n", PostgresUsername, PostgresPassword)
	configuration := `[databases]
* = host=postgres port=5432
[pgbouncer]
listen_addr = 0.0.0.0
listen_port = 5432
unix_socket_dir = /tmp
pool_mode = session
ignore_startup_parameters = extra_float_digits
auth_type = scram-sha-256
auth_file = /tmp/pgbouncer-users.txt
admin_users = testuser
default_pool_size = 20
max_client_conn = 100
server_tls_sslmode = disable
`
	handle := startContainer(t, testcontainers.ContainerRequest{
		Image:      "edoburu/pgbouncer:v1.26.1-p0",
		Entrypoint: []string{"pgbouncer"}, Cmd: []string{"/tmp/pgbouncer.ini"},
		ExposedPorts: []string{postgresPort}, Networks: []string{networkName},
		Files: []testcontainers.ContainerFile{
			{Reader: strings.NewReader(configuration), ContainerFilePath: "/tmp/pgbouncer.ini", FileMode: 0o644},
			{Reader: strings.NewReader(users), ContainerFilePath: "/tmp/pgbouncer-users.txt", FileMode: 0o644},
		},
		WaitingFor: wait.ForListeningPort(postgresPort).WithStartupTimeout(postgresStartupTimeout),
	}, postgresPort)
	pooler := &PgBouncer{ContainerHandle: handle, users: users}
	testwait.Until(t, "PgBouncer accepts authenticated SQL", func(ctx context.Context) error {
		connection, err := pgx.Connect(ctx, pooler.buildConnectionString(PostgresDatabase))
		if err != nil {
			return err
		}

		defer func() { _ = connection.Close(ctx) }()
		return connection.Ping(ctx)
	})

	return pooler
}

func (p *PgBouncer) AddCredential(t *testing.T, credential PgBouncerCredential) {
	t.Helper()
	p.users += fmt.Sprintf("%q %q\n", credential.Username, credential.Password)
	require.NoError(t, p.Container.CopyToContainer(t.Context(), []byte(p.users), "/tmp/pgbouncer-users.txt", 0o644))

	connection, err := pgx.Connect(t.Context(), p.buildConnectionString("pgbouncer"))
	require.NoError(t, err)
	defer func() { _ = connection.Close(context.WithoutCancel(t.Context())) }()

	_, err = connection.Exec(t.Context(), "RELOAD")
	require.NoError(t, err)
}

func (p *PgBouncer) buildConnectionString(databaseName string) string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		p.Host,
		p.Port,
		PostgresUsername,
		PostgresPassword,
		databaseName,
	)
}
