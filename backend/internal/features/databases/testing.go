//go:build testhelpers

package databases_bridge

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"databasus-backend/internal/config"
	"databasus-backend/internal/lib/sshtunnel"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/notifiers"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
	"databasus-backend/internal/storage"
)

var PhysicalPostgresVersionTags = []string{"17", "18"}

// PhysicalTestDatabaseSpec describes a physical PostgreSQL database for the helpers. Host and Port
// address a bastioned cluster from inside the bastion's network when SshTunnel is set, so they are
// unreachable from the host running the test: only the tunnel gets there.
type PhysicalTestDatabaseSpec struct {
	Host       string
	Port       int
	VersionTag string
	BackupType databases.PhysicalBackupType
	SshTunnel  *sshtunnel.Settings
	TLS        *PhysicalTestTLS
}

// PhysicalTestTLS holds the certificates of a physical database reached over mutual TLS.
type PhysicalTestTLS struct {
	ClientCert string
	ClientKey  string
	RootCert   string
}

// CreateTestDatabase inserts a logical PostgreSQL database that points at the shared PostgreSQL 16
// test server, as the old tests need a database without creating it through the contract.
func CreateTestDatabase(workspaceID uuid.UUID, notifier *notifiers.Notifier) databases.Database {
	env := config.GetEnv()

	port, err := strconv.Atoi(env.TestLogicalPostgres16Port)
	if err != nil {
		panic(fmt.Sprintf("failed to parse TEST_LOGICAL_POSTGRES_16_PORT: %v", err))
	}

	databaseID := insertTestDatabase("test "+uuid.NewString(), databases.TypePostgresqlLogical, workspaceID, notifier)
	execute(`INSERT INTO databases_postgresql_logical (database_id, version, host, port, username, password, database)
		VALUES (?, '16', ?, ?, 'testuser', 'testpassword', 'testdb')`,
		databaseID, env.TestLocalhost, port)

	return getTestDatabase(databaseID)
}

// CreateTestPhysicalPostgresDatabase inserts a FULL physical database that points at the shared
// primary test server of the version.
func CreateTestPhysicalPostgresDatabase(
	workspaceID uuid.UUID,
	notifier *notifiers.Notifier,
	versionTag string,
) databases.Database {
	host, port := getPhysicalPrimaryEndpoint(versionTag)

	return CreateTestPhysicalPostgresDatabaseWithType(
		host, port, workspaceID, notifier, versionTag, databases.PhysicalBackupTypeFull,
	)
}

// CreateTestPhysicalPostgresDatabaseWithType inserts a physical database at host:port with the backup
// type, for scheduler-driven incremental and WAL stream chains whose eligibility the scheduler reads
// from it. The caller owns the server.
func CreateTestPhysicalPostgresDatabaseWithType(
	host string,
	port int,
	workspaceID uuid.UUID,
	notifier *notifiers.Notifier,
	versionTag string,
	backupType databases.PhysicalBackupType,
) databases.Database {
	return CreateTestPhysicalPostgresDatabaseFromSpec(PhysicalTestDatabaseSpec{
		Host:       host,
		Port:       port,
		VersionTag: versionTag,
		BackupType: backupType,
		SshTunnel:  nil,
		TLS:        nil,
	}, workspaceID, notifier)
}

// CreateTestPhysicalPostgresDatabaseNoSummary inserts a FULL physical database at host:port that
// points at a server with summarize_wal off, so an incremental pre-check reaches the SUMMARIZER_OFF
// branch.
func CreateTestPhysicalPostgresDatabaseNoSummary(
	host string,
	port int,
	workspaceID uuid.UUID,
	notifier *notifiers.Notifier,
	versionTag string,
) databases.Database {
	return CreateTestPhysicalPostgresDatabaseWithType(
		host, port, workspaceID, notifier, versionTag, databases.PhysicalBackupTypeFull,
	)
}

// CreateTestPhysicalPostgresDatabaseMtls inserts a FULL physical PostgreSQL 17 database at host:port
// that reaches the replication server over mutual TLS with the committed test certificates.
func CreateTestPhysicalPostgresDatabaseMtls(
	host string,
	port int,
	workspaceID uuid.UUID,
	notifier *notifiers.Notifier,
) databases.Database {
	clientCert, clientKey, rootCert := readPhysicalMtlsCerts()

	return CreateTestPhysicalPostgresDatabaseFromSpec(PhysicalTestDatabaseSpec{
		Host:       host,
		Port:       port,
		VersionTag: "17",
		BackupType: databases.PhysicalBackupTypeFull,
		SshTunnel:  nil,
		TLS:        &PhysicalTestTLS{ClientCert: clientCert, ClientKey: clientKey, RootCert: rootCert},
	}, workspaceID, notifier)
}

// CreateTestPhysicalPostgresDatabaseFromSpec inserts a physical database as the spec describes. Its
// replication slots get names of their own, as the provider gives a new database.
func CreateTestPhysicalPostgresDatabaseFromSpec(
	spec PhysicalTestDatabaseSpec,
	workspaceID uuid.UUID,
	notifier *notifiers.Notifier,
) databases.Database {
	databaseID := insertTestDatabase(
		"test-physical-pg "+uuid.NewString(), databases.TypePostgresqlPhysical, workspaceID, notifier,
	)

	sslMode, clientCert, clientKey, rootCert := "disable", "", "", ""
	if spec.TLS != nil {
		sslMode, clientCert, clientKey, rootCert = "verify-ca", spec.TLS.ClientCert, spec.TLS.ClientKey, spec.TLS.RootCert
	}

	execute(`INSERT INTO databases_postgresql_physical (database_id, version, host, port, username, password,
		backup_type, replication_slot_name, backup_slot_name, ssl_mode, ssl_client_cert, ssl_client_key, ssl_root_cert)
		VALUES (?, ?, ?, ?, 'testuser', 'testpassword', ?, ?, ?, ?, ?, ?, ?)`,
		databaseID, getPhysicalPostgresVersion(spec.VersionTag), spec.Host, spec.Port, spec.BackupType,
		"databasus_slot_"+newSlotSuffix(), "databasus_basebackup_"+newSlotSuffix(),
		sslMode, clientCert, clientKey, rootCert)

	if spec.SshTunnel != nil {
		execute(`UPDATE databases_postgresql_physical SET ssh_is_enabled = TRUE, ssh_host = ?, ssh_port = ?,
			ssh_username = ?, ssh_auth_type = ?, ssh_password = ?, ssh_private_key = ?, ssh_private_key_passphrase = ?
			WHERE database_id = ?`,
			spec.SshTunnel.Host, spec.SshTunnel.Port, spec.SshTunnel.Username, spec.SshTunnel.AuthType,
			spec.SshTunnel.Password, spec.SshTunnel.PrivateKey, spec.SshTunnel.PrivateKeyPassphrase, databaseID)
	}

	return getTestDatabase(databaseID)
}

// CreateTestMariadbDatabase inserts a MariaDB 10.11 database at host:port. The caller owns the server.
func CreateTestMariadbDatabase(
	host string,
	port int,
	workspaceID uuid.UUID,
	notifier *notifiers.Notifier,
) databases.Database {
	databaseID := insertTestDatabase("test-mariadb "+uuid.NewString(), databases.TypeMariadb, workspaceID, notifier)
	execute(`INSERT INTO databases_mariadb (database_id, version, host, port, username, password, database)
		VALUES (?, '10.11', ?, ?, 'testuser', 'testpassword', 'testdb')`,
		databaseID, host, port)

	return getTestDatabase(databaseID)
}

// CreateTestMongodbDatabase inserts a MongoDB 7 database at host:port. The caller owns the server.
func CreateTestMongodbDatabase(
	host string,
	port int,
	workspaceID uuid.UUID,
	notifier *notifiers.Notifier,
) databases.Database {
	databaseID := insertTestDatabase("test-mongodb "+uuid.NewString(), databases.TypeMongodb, workspaceID, notifier)
	execute(`INSERT INTO databases_mongodb (database_id, version, host, port, username, password, database)
		VALUES (?, '7', ?, ?, 'root', 'rootpassword', 'testdb')`,
		databaseID, host, port)

	return getTestDatabase(databaseID)
}

// GetPhysicalMtlsTestdataDir resolves the committed physical mTLS testdata directory (server and
// client certificates, pg_hba.conf, Dockerfile) from this source file's location, so it does not
// depend on the caller's working directory.
func GetPhysicalMtlsTestdataDir() string {
	_, thisFile, _, isResolved := runtime.Caller(0)
	if !isResolved {
		panic("cannot resolve physical mTLS testdata dir: runtime.Caller failed")
	}

	return filepath.Join(filepath.Dir(thisFile), "..", "tests", "physical", "testdata", "mtls")
}

// RemoveTestDatabase deletes the database through the module, so the subscribed handlers remove its
// backups and drop its replication slots, as its workspace owner. A database the test already deleted
// counts as removed. The context is stripped of cancellation because callers pass t.Context() from a
// t.Cleanup, and the test context is already cancelled by the time cleanup runs.
func RemoveTestDatabase(ctx context.Context, database databases.Database) {
	ctx = context.WithoutCancel(ctx)

	owner, isFound := findTestDatabaseOwner(ctx, database.ID)
	if !isFound {
		return
	}

	err := GetModule().DeleteDatabase(ctx, owner, database.ID)
	if err != nil && !errors.Is(err, databases.ErrDatabaseNotFound) {
		panic(fmt.Sprintf("failed to delete test database: %v", err))
	}
}

func insertTestDatabase(
	name string,
	databaseType databases.Type,
	workspaceID uuid.UUID,
	notifier *notifiers.Notifier,
) uuid.UUID {
	databaseID := uuid.New()
	execute("INSERT INTO databases (id, workspace_id, name, type) VALUES (?, ?, ?, ?)",
		databaseID, workspaceID, name, databaseType)

	if notifier != nil {
		execute("INSERT INTO databases_notifiers (database_id, notifier_id) VALUES (?, ?)", databaseID, notifier.ID)
	}

	return databaseID
}

func getTestDatabase(databaseID uuid.UUID) databases.Database {
	foundDatabases, err := GetModule().GetDatabases(context.Background(), []uuid.UUID{databaseID})
	if err != nil {
		panic(err)
	}

	if len(foundDatabases) != 1 {
		panic(fmt.Sprintf("the inserted test database %s is not found", databaseID))
	}

	return foundDatabases[0]
}

func findTestDatabaseOwner(ctx context.Context, databaseID uuid.UUID) (users.Actor, bool) {
	var ownerIDs []uuid.UUID

	err := storage.GetDb().WithContext(ctx).Raw(`SELECT m.user_id
		FROM databases d
		JOIN workspaces_members m ON m.workspace_id = d.workspace_id AND m.role = ?
		WHERE d.id = ?`, workspaces.RoleOwner, databaseID).Scan(&ownerIDs).Error
	if err != nil {
		panic(err)
	}

	if len(ownerIDs) == 0 {
		return users.Actor{}, false
	}

	return users.Actor{UserID: ownerIDs[0], Role: users.RoleMember}, true
}

func getPhysicalPrimaryEndpoint(versionTag string) (string, int) {
	env := config.GetEnv()

	var portText string

	switch versionTag {
	case "17":
		portText = env.TestPhysicalPostgres17Port
	case "18":
		portText = env.TestPhysicalPostgres18Port
	default:
		panic(fmt.Sprintf("unsupported physical postgres version tag: %s (supported: %v)",
			versionTag, PhysicalPostgresVersionTags))
	}

	port, err := strconv.Atoi(portText)
	if err != nil {
		panic(fmt.Sprintf("failed to parse physical postgres %s port: %v", versionTag, err))
	}

	return env.TestLocalhost, port
}

func getPhysicalPostgresVersion(versionTag string) string {
	if versionTag != "17" && versionTag != "18" {
		panic(fmt.Sprintf("unsupported physical postgres version tag: %s (supported: %v)",
			versionTag, PhysicalPostgresVersionTags))
	}

	return versionTag
}

// readPhysicalMtlsCerts reads the committed client certificate, key and CA from the physical mTLS
// testdata.
func readPhysicalMtlsCerts() (clientCert, clientKey, rootCert string) {
	read := func(name string) string {
		content, err := os.ReadFile(filepath.Join(GetPhysicalMtlsTestdataDir(), name))
		if err != nil {
			panic(fmt.Sprintf("failed to read physical mTLS test certificate %s: %v", name, err))
		}

		return string(content)
	}

	return read("client.crt"), read("client.key"), read("ca.crt")
}

func newSlotSuffix() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")
}

func execute(statement string, values ...any) {
	err := storage.GetDb().Exec(statement, values...).Error
	if err != nil {
		panic(err)
	}
}
