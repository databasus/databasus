package physical

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/stretchr/testify/require"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

const connCloseTimeout = 5 * time.Second

func getSupportedVersionImages() []string {
	return []string{"postgres:17", "postgres:18"}
}

func getAdminRole() testRole {
	return testRole{Name: containers.PostgresUsername, Password: containers.PostgresPassword}
}

func getFullOnlyCheck() ReplicationCheck {
	return ReplicationCheck{
		IsWalSummaryRequired:     false,
		IsWalRotationRequired:    false,
		ExpectedSystemIdentifier: "",
	}
}

func getIncrementalCheck() ReplicationCheck {
	return ReplicationCheck{
		IsWalSummaryRequired:     true,
		IsWalRotationRequired:    false,
		ExpectedSystemIdentifier: "",
	}
}

func getWalStreamCheck() ReplicationCheck {
	return ReplicationCheck{
		IsWalSummaryRequired:     true,
		IsWalRotationRequired:    true,
		ExpectedSystemIdentifier: "",
	}
}

type testRole struct {
	Name     string
	Password string
}

// testServer is the container of one PostgreSQL version, shared by the scenarios of one test
// function. Roles and slots are cluster-wide, so every scenario names its own with a UUID suffix.
type testServer struct {
	endpoint containers.Endpoint
	image    string
	version  pgshared.Version
}

// runOnEverySupportedVersion boots the versions one after another, so only one server of the
// function is alive at a time (docs/testing.md, "Database servers for engines").
func runOnEverySupportedVersion(t *testing.T, run func(t *testing.T, server testServer)) {
	t.Helper()

	for _, image := range getSupportedVersionImages() {
		t.Run(image, func(t *testing.T) {
			run(t, startTestServer(t, image))
		})
	}
}

func startTestServer(t *testing.T, image string, options ...containers.PhysicalPostgresOption) testServer {
	t.Helper()

	return testServer{
		endpoint: containers.StartPhysicalPostgres(t, image, options...),
		image:    image,
		version:  pgshared.Version(strconv.Itoa(containers.PostgresMajorVersion(image))),
	}
}

func (s testServer) getSettings(role testRole) Settings {
	return Settings{
		Host:     s.endpoint.Host,
		HostAddr: "",
		Port:     s.endpoint.Port,
		Username: role.Name,
		Password: role.Password,
		TLS: pgshared.TLS{
			Mode:       pgshared.SslModeDisable,
			ClientCert: "",
			ClientKey:  "",
			RootCert:   "",
		},
		SSHTunnel: nil,
	}
}

func (s testServer) newSource(role testRole) *Source {
	return NewSource(s.getSettings(role))
}

func (s testServer) connectAs(t *testing.T, role testRole) *pgx.Conn {
	t.Helper()

	conn, err := pgx.Connect(t.Context(), s.getConnString(role, false))
	require.NoError(t, err)

	t.Cleanup(func() {
		closeTestConn(t, conn.Close)
	})

	return conn
}

func (s testServer) openInspection(t *testing.T, role testRole) *Inspection {
	t.Helper()
	inspection, err := s.newSource(role).OpenInspection(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { closeTestConn(t, inspection.Close) })
	return inspection
}

func (s testServer) openReplicationConn(t *testing.T, role testRole) (*pgconn.PgConn, error) {
	t.Helper()

	conn, err := pgconn.Connect(t.Context(), s.getConnString(role, true))
	if err != nil {
		return nil, err
	}

	t.Cleanup(func() {
		closeTestConn(t, conn.Close)
	})

	return conn, nil
}

func (s testServer) getConnString(role testRole, isReplication bool) string {
	connString := fmt.Sprintf(
		"host=%s port=%d user='%s' password='%s' dbname=postgres sslmode=disable",
		s.endpoint.Host, s.endpoint.Port, role.Name, role.Password,
	)

	if isReplication {
		connString += " replication=true"
	}

	return connString
}

func (s testServer) readSystemIdentifier(t *testing.T) string {
	t.Helper()

	var systemIdentifier string

	err := s.connectAs(t, getAdminRole()).
		QueryRow(t.Context(), "SELECT system_identifier::text FROM pg_control_system()").
		Scan(&systemIdentifier)
	require.NoError(t, err)

	return systemIdentifier
}

// createLoginRole creates a role that may log in, with the given role attributes and nothing more.
func (s testServer) createLoginRole(t *testing.T, attributes string) testRole {
	t.Helper()

	role := testRole{Name: "test_role_" + newNameSuffix(), Password: "role password 123"}

	execStatements(t, s.connectAs(t, getAdminRole()),
		fmt.Sprintf(`CREATE USER "%s" WITH PASSWORD '%s' LOGIN %s`, role.Name, role.Password, attributes),
	)

	return role
}

func (s testServer) isRolePresent(t *testing.T, roleName string) bool {
	t.Helper()

	var isPresent bool

	err := s.connectAs(t, getAdminRole()).
		QueryRow(t.Context(), `SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname = $1)`, roleName).
		Scan(&isPresent)
	require.NoError(t, err)

	return isPresent
}

// createClusterRole creates a role whose name is fixed by a cloud platform, so it is visible to
// every scenario on the server. Only scenarios that do not run in parallel create one, and the role
// is gone before the parallel scenarios start.
func (s testServer) createClusterRole(t *testing.T, roleName string, isGrantedToAdmin bool) {
	t.Helper()

	adminConn := s.connectAs(t, getAdminRole())
	execStatements(t, adminConn, fmt.Sprintf(`CREATE ROLE "%s" NOLOGIN`, roleName))

	if isGrantedToAdmin {
		execStatements(t, adminConn, fmt.Sprintf(`GRANT "%s" TO "%s"`, roleName, getAdminRole().Name))
	}

	t.Cleanup(func() {
		cleanupConn, err := pgx.Connect(context.WithoutCancel(t.Context()), s.getConnString(getAdminRole(), false))
		require.NoError(t, err)

		defer closeTestConn(t, cleanupConn.Close)

		_, err = cleanupConn.Exec(context.WithoutCancel(t.Context()), fmt.Sprintf(`DROP ROLE "%s"`, roleName))
		require.NoError(t, err, "a cloud platform role left behind changes the platform the other scenarios see")
	})
}

func (s testServer) createLogicalSlot(t *testing.T, slotName string) {
	t.Helper()

	execStatements(t, s.connectAs(t, getAdminRole()),
		fmt.Sprintf(`SELECT pg_create_logical_replication_slot('%s', 'test_decoding')`, slotName))
}

func (s testServer) createReservedPhysicalSlot(t *testing.T, slotName string) {
	t.Helper()

	execStatements(t, s.connectAs(t, getAdminRole()),
		fmt.Sprintf(`SELECT pg_create_physical_replication_slot('%s', true)`, slotName))
}

func (s testServer) readSlotType(t *testing.T, slotName string) (string, bool) {
	t.Helper()

	slot, err := (&Inspection{conn: s.connectAs(t, getAdminRole())}).GetSlot(t.Context(), slotName)
	require.NoError(t, err)

	if slot == nil {
		return "", false
	}
	return slot.Type, true
}

// attachSlotConsumer streams from the slot over a physical replication connection, as
// pg_receivewal does, so the server reports the slot active until the connection ends.
func (s testServer) attachSlotConsumer(t *testing.T, slotName string) {
	t.Helper()

	var restartLsn string

	err := s.connectAs(t, getAdminRole()).
		QueryRow(t.Context(), "SELECT restart_lsn::text FROM pg_replication_slots WHERE slot_name = $1", slotName).
		Scan(&restartLsn)
	require.NoError(t, err)

	consumerConn, err := s.openReplicationConn(t, getAdminRole())
	require.NoError(t, err)

	frontend := consumerConn.Frontend()
	frontend.Send(&pgproto3.Query{String: fmt.Sprintf("START_REPLICATION SLOT %s PHYSICAL %s", slotName, restartLsn)})
	require.NoError(t, frontend.Flush())

	for {
		message, err := frontend.Receive()
		require.NoError(t, err)

		switch typedMessage := message.(type) {
		case *pgproto3.CopyBothResponse:
			s.waitForSlotActive(t, slotName)

			return
		case *pgproto3.ErrorResponse:
			t.Fatalf("failed to start streaming from slot %s: %s", slotName, typedMessage.Message)
		}
	}
}

func (s testServer) waitForSlotActive(t *testing.T, slotName string) {
	t.Helper()

	adminConn := s.connectAs(t, getAdminRole())

	testwait.Until(t, "replication slot "+slotName+" becomes active", func(ctx context.Context) error {
		slot, err := (&Inspection{conn: adminConn}).GetSlot(ctx, slotName)
		if err != nil {
			return err
		}

		if slot == nil || !slot.IsActive {
			return fmt.Errorf("slot is absent or inactive")
		}

		return nil
	})
}

func execStatements(t *testing.T, conn *pgx.Conn, statements ...string) {
	t.Helper()

	for _, statement := range statements {
		_, err := conn.Exec(t.Context(), statement)
		require.NoError(t, err, statement)
	}
}

func closeTestConn(t *testing.T, closeConn func(ctx context.Context) error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), connCloseTimeout)
	defer cancel()

	_ = closeConn(ctx)
}

func getConnectionTestErrorCode(t *testing.T, err error) pgshared.ConnectionErrorCode {
	t.Helper()

	var connectionTestErr *pgshared.ConnectionTestError
	require.ErrorAs(t, err, &connectionTestErr)

	return connectionTestErr.Code
}

func newNameSuffix() string {
	return uuid.New().String()[:8]
}

func newSlotName() string {
	return "test_slot_" + newNameSuffix()
}

func newTestLogger(t *testing.T) *slog.Logger {
	t.Helper()

	return slog.New(slog.NewTextHandler(t.Output(), nil))
}
