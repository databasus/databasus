package logical

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	containers "databasus-backend/internal/lib/testcontainers"
)

func getAdminRole() testRole {
	return testRole{Name: containers.PostgresUsername, Password: containers.PostgresPassword}
}

type testRole struct {
	Name     string
	Password string
}

// testServer is the container of one PostgreSQL version, shared by every scenario of that version.
type testServer struct {
	endpoint containers.Endpoint
	version  pgshared.Version
}

// testDatabase is the database of one scenario. Each scenario takes its own, so scenarios that
// drop the public schema, enable row-level security or change what PUBLIC may do run in parallel
// without seeing each other.
type testDatabase struct {
	server testServer
	name   string
	admin  *pgx.Conn
	// roles are cluster-wide, so they are dropped after the database that granted them anything.
	createdRoleNames *[]string
}

func startTestServer(t *testing.T, image string) testServer {
	t.Helper()

	return testServer{
		endpoint: containers.StartPostgres(t, image),
		version:  pgshared.Version(strconv.Itoa(containers.PostgresMajorVersion(image))),
	}
}

func (s testServer) createDatabase(t *testing.T) testDatabase {
	t.Helper()

	return s.createNamedDatabase(t, "db_"+newNameSuffix())
}

func (s testServer) createNamedDatabase(t *testing.T, databaseName string) testDatabase {
	t.Helper()

	serverConn, err := s.openConn(t.Context(), getAdminRole(), containers.PostgresDatabase)
	require.NoError(t, err)

	_, err = serverConn.Exec(t.Context(), fmt.Sprintf(`CREATE DATABASE "%s"`, databaseName))
	require.NoError(t, serverConn.Close(t.Context()))
	require.NoError(t, err)

	createdRoleNames := &[]string{}

	t.Cleanup(func() {
		s.dropDatabaseAndRoles(t, databaseName, *createdRoleNames)
	})

	return testDatabase{
		server:           s,
		name:             databaseName,
		admin:            s.connectAs(t, getAdminRole(), databaseName),
		createdRoleNames: createdRoleNames,
	}
}

func (s testServer) dropDatabaseAndRoles(t *testing.T, databaseName string, roleNames []string) {
	t.Helper()

	ctx := context.WithoutCancel(t.Context())

	serverConn, err := s.openConn(ctx, getAdminRole(), containers.PostgresDatabase)
	if err != nil {
		t.Logf("failed to connect to drop database %s: %v", databaseName, err)

		return
	}

	defer func() { _ = serverConn.Close(ctx) }()

	// PostgreSQL 12 has no DROP DATABASE ... WITH (FORCE).
	statements := []string{
		fmt.Sprintf(
			`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '%s' AND pid <> pg_backend_pid()`,
			databaseName,
		),
		fmt.Sprintf(`DROP DATABASE IF EXISTS "%s"`, databaseName),
		// The data directory is a small tmpfs, and a checkpoint lets the server recycle the WAL
		// the dropped database wrote.
		`CHECKPOINT`,
	}

	for _, roleName := range slices.Backward(roleNames) {
		statements = append(statements, fmt.Sprintf(`DROP ROLE IF EXISTS "%s"`, roleName))
	}

	for _, statement := range statements {
		if _, err := serverConn.Exec(ctx, statement); err != nil {
			t.Logf("failed to clean up after the test: %s: %v", statement, err)
		}
	}
}

func (s testServer) connectAs(t *testing.T, role testRole, databaseName string) *pgx.Conn {
	t.Helper()

	conn, err := s.openConn(t.Context(), role, databaseName)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = conn.Close(context.WithoutCancel(t.Context()))
	})

	return conn
}

func (s testServer) openConn(ctx context.Context, role testRole, databaseName string) (*pgx.Conn, error) {
	return pgx.Connect(ctx, fmt.Sprintf(
		"host=%s port=%d user='%s' password='%s' dbname='%s' sslmode=disable",
		s.endpoint.Host, s.endpoint.Port, role.Name, role.Password, databaseName,
	))
}

func (s testServer) newSource(role testRole, databaseName string) *Source {
	return NewSource(Settings{
		Host:      s.endpoint.Host,
		HostAddr:  "",
		Port:      s.endpoint.Port,
		Username:  role.Name,
		Password:  role.Password,
		Database:  databaseName,
		TLS:       pgshared.TLS{Mode: pgshared.SslModeDisable},
		SSHTunnel: nil,
	})
}

func (d testDatabase) exec(t *testing.T, statements ...string) {
	t.Helper()

	for _, statement := range statements {
		_, err := d.admin.Exec(t.Context(), statement)
		require.NoError(t, err, statement)
	}
}

// createLoginRole creates a role that may log in and connect to this database, and nothing more.
func (d testDatabase) createLoginRole(t *testing.T, namePrefix string) testRole {
	t.Helper()

	role := testRole{Name: namePrefix + "_" + newNameSuffix(), Password: "role password 123"}

	d.exec(t, fmt.Sprintf(`CREATE USER "%s" WITH PASSWORD '%s' LOGIN`, role.Name, role.Password))
	d.dropRoleOnCleanup(role.Name)
	d.exec(t, fmt.Sprintf(`GRANT CONNECT ON DATABASE "%s" TO "%s"`, d.name, role.Name))

	return role
}

func (d testDatabase) dropRoleOnCleanup(roleName string) {
	*d.createdRoleNames = append(*d.createdRoleNames, roleName)
}

func (d testDatabase) newSource(role testRole) *Source {
	return d.server.newSource(role, d.name)
}

func (d testDatabase) newAdminSource() *Source {
	return d.newSource(getAdminRole())
}

func (d testDatabase) connectAs(t *testing.T, role testRole) *pgx.Conn {
	t.Helper()

	return d.server.connectAs(t, role, d.name)
}

func (d testDatabase) isRolePresent(t *testing.T, roleName string) bool {
	t.Helper()

	var isPresent bool

	err := d.admin.QueryRow(t.Context(), `SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname = $1)`, roleName).
		Scan(&isPresent)
	require.NoError(t, err)

	return isPresent
}

func newNameSuffix() string {
	return strings.ReplaceAll(uuid.New().String()[:8], "-", "")
}

func newTestLogger(t *testing.T) *slog.Logger {
	t.Helper()

	return slog.New(slog.NewTextHandler(t.Output(), nil))
}
