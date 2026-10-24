//go:build testhelpers

package databases

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"databasus-backend/internal/config"
	"databasus-backend/internal/features/databases/databases/mariadb"
	"databasus-backend/internal/features/databases/databases/mongodb"
	"databasus-backend/internal/features/databases/databases/mysql"
	postgresql_logical "databasus-backend/internal/features/databases/databases/postgresql/logical"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	postgresql_shared "databasus-backend/internal/features/databases/databases/postgresql/shared"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/sshtunnel"
	"databasus-backend/internal/features/storages"
	"databasus-backend/internal/util/tools"
)

var PhysicalPostgresVersionTags = []string{"17", "18"}

func GetTestPostgresConfig() *postgresql_logical.PostgresqlLogicalDatabase {
	env := config.GetEnv()
	port, err := strconv.Atoi(env.TestLogicalPostgres16Port)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse TEST_LOGICAL_POSTGRES_16_PORT: %v", err))
	}

	testDbName := "testdb"
	return &postgresql_logical.PostgresqlLogicalDatabase{
		Version:  tools.PostgresqlVersion16,
		Host:     config.GetEnv().TestLocalhost,
		Port:     port,
		Username: "testuser",
		Password: "testpassword",
		Database: &testDbName,
		CpuCount: 1,
	}
}

// physicalPostgresVersion maps a "17"/"18" tag to the typed version enum.
func physicalPostgresVersion(versionTag string) tools.PostgresqlVersion {
	switch versionTag {
	case "17":
		return tools.PostgresqlVersion17
	case "18":
		return tools.PostgresqlVersion18
	default:
		panic(fmt.Sprintf(
			"unsupported physical postgres version tag: %s (supported: %v)",
			versionTag, PhysicalPostgresVersionTags,
		))
	}
}

// physicalPrimaryEndpoint resolves the shared compose primary source's host and port for versionTag.
// The ports are validated non-empty in config.go at startup (it os.Exit(1)s with the exact env var
// when one is unset), so a missing DSN fails the run there, never here and never as a skip. Only the
// shared-fixture callers use this — the throwaway testcontainers sources pass their own endpoint.
func physicalPrimaryEndpoint(versionTag string) (string, int) {
	env := config.GetEnv()

	var portStr string

	switch versionTag {
	case "17":
		portStr = env.TestPhysicalPostgres17Port
	case "18":
		portStr = env.TestPhysicalPostgres18Port
	default:
		panic(fmt.Sprintf(
			"unsupported physical postgres version tag: %s (supported: %v)",
			versionTag, PhysicalPostgresVersionTags,
		))
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse physical postgres %s port: %v", versionTag, err))
	}

	return env.TestLocalhost, port
}

// GetTestPhysicalPostgresConfigWithType builds a physical config at host:port with an explicit
// BackupType, so scheduler-driven tests can build chains (FULL_AND_INCREMENTAL) or stream WAL
// (FULL_INCREMENTAL_AND_WAL_STREAM) — the scheduler's incremental decision keys off this DB-level
// field, which the backup-config API cannot change. The caller owns the container lifecycle.
func GetTestPhysicalPostgresConfigWithType(
	host string,
	port int,
	versionTag string,
	backupType postgresql_physical.BackupType,
) *postgresql_physical.PostgresqlPhysicalDatabase {
	return &postgresql_physical.PostgresqlPhysicalDatabase{
		Version:    physicalPostgresVersion(versionTag),
		Host:       host,
		Port:       port,
		Username:   "testuser",
		Password:   "testpassword",
		BackupType: backupType,
	}
}

// GetTestPhysicalPostgresConfigNoSummary builds a FULL_ONLY config at host:port pointed at a
// summarize_wal=off cluster, so an incremental pre-check reaches the SUMMARIZER_OFF branch.
func GetTestPhysicalPostgresConfigNoSummary(
	host string,
	port int,
	versionTag string,
) *postgresql_physical.PostgresqlPhysicalDatabase {
	return &postgresql_physical.PostgresqlPhysicalDatabase{
		Version:    physicalPostgresVersion(versionTag),
		Host:       host,
		Port:       port,
		Username:   "testuser",
		Password:   "testpassword",
		BackupType: postgresql_physical.BackupTypeFullOnly,
	}
}

// GetTestPhysicalPostgresConfigMtls builds a physical config at host:port pointed at the
// replication-capable mTLS cluster, with client certs read from the physical testdata/mtls fixtures.
// BackupType is FULL_ONLY; the WAL-stream-over-mTLS test constructs the streamer spec directly.
func GetTestPhysicalPostgresConfigMtls(host string, port int) *postgresql_physical.PostgresqlPhysicalDatabase {
	clientCert, clientKey, rootCert := readPhysicalMtlsCerts()

	// FULL_ONLY keeps the shared fixture's config-save valid (WAL_STREAM would
	// require CHAINS retention + a lag threshold). The WAL-stream-over-mTLS test
	// constructs the streamer spec directly against this DB, so the declared
	// backup type does not gate it.
	return &postgresql_physical.PostgresqlPhysicalDatabase{
		Version:       tools.PostgresqlVersion17,
		Host:          host,
		Port:          port,
		Username:      "testuser",
		Password:      "testpassword",
		BackupType:    postgresql_physical.BackupTypeFullOnly,
		SslMode:       postgresql_shared.PostgresSslModeVerifyCA,
		SslClientCert: clientCert,
		SslClientKey:  clientKey,
		SslRootCert:   rootCert,
	}
}

// readPhysicalMtlsCerts reads the committed client cert/key + CA from the
// physical mTLS testdata. Paths are resolved relative to this package so the
// helper works regardless of the calling test's directory.
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

// GetPhysicalMtlsTestdataDir resolves the committed physical mTLS testdata directory (server +
// client certs, pg_hba.conf, Dockerfile) from this source file's location, so it is independent of
// the caller's working directory. It is the single resolver for that directory: both the client-cert
// reader here and the mTLS server-image build context in the backup tests use it.
func GetPhysicalMtlsTestdataDir() string {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("cannot resolve physical mTLS testdata dir: runtime.Caller failed")
	}

	return filepath.Join(
		filepath.Dir(thisFile), "..", "tests", "physical", "testdata", "mtls",
	)
}

// CreateTestPhysicalPostgresDatabaseMtls persists a physical DB row pointed at
// the mTLS replication cluster at host:port, for FULL- and WAL-stream-over-mTLS tests.
func CreateTestPhysicalPostgresDatabaseMtls(
	host string,
	port int,
	workspaceID uuid.UUID,
	notifier *notifiers.Notifier,
) *Database {
	database := &Database{
		WorkspaceID:        &workspaceID,
		Name:               "test-physical-pg-mtls " + uuid.New().String(),
		Type:               DatabaseTypePostgresPhysical,
		PostgresqlPhysical: GetTestPhysicalPostgresConfigMtls(host, port),
		Notifiers: []notifiers.Notifier{
			*notifier,
		},
	}

	database, err := databaseRepository.Save(database)
	if err != nil {
		panic(err)
	}

	return database
}

// GetTestMariadbConfig builds a MariaDB 10.11 database config pointing at host:port. The
// caller owns the container lifecycle (host/port come from a testcontainers endpoint),
// keeping this file free of the testcontainers dependency.
func GetTestMariadbConfig(host string, port int) *mariadb.MariadbDatabase {
	testDbName := "testdb"
	return &mariadb.MariadbDatabase{
		Version:  tools.MariadbVersion1011,
		Host:     host,
		Port:     port,
		Username: "testuser",
		Password: "testpassword",
		Database: &testDbName,
	}
}

// The caller owns the container lifecycle (host/port come from a testcontainers endpoint),
// which keeps this file free of the testcontainers dependency.
func GetTestMysqlConfig(host string, port int) *mysql.MysqlDatabase {
	testDbName := "testdb"
	return &mysql.MysqlDatabase{
		Version:  tools.MysqlVersion80,
		Host:     host,
		Port:     port,
		Username: "testuser",
		Password: "testpassword",
		Database: &testDbName,
	}
}

// GetTestMongodbConfig builds a Mongo 7.0 database config pointing at host:port. The
// caller owns the container lifecycle (host/port come from a testcontainers endpoint),
// keeping this file free of the testcontainers dependency.
func GetTestMongodbConfig(host string, port int) *mongodb.MongodbDatabase {
	return &mongodb.MongodbDatabase{
		Version:      tools.MongodbVersion7,
		Host:         host,
		Port:         &port,
		Username:     "root",
		Password:     "rootpassword",
		Database:     "testdb",
		AuthDatabase: "admin",
		IsHttps:      false,
		IsSrv:        false,
		CpuCount:     1,
	}
}

func CreateTestDatabase(
	workspaceID uuid.UUID,
	storage *storages.Storage,
	notifier *notifiers.Notifier,
) *Database {
	database := &Database{
		WorkspaceID:       &workspaceID,
		Name:              "test " + uuid.New().String(),
		Type:              DatabaseTypePostgresLogical,
		PostgresqlLogical: GetTestPostgresConfig(),
		Notifiers: []notifiers.Notifier{
			*notifier,
		},
	}

	database, err := databaseRepository.Save(database)
	if err != nil {
		panic(err)
	}

	return database
}

// CreateTestPhysicalPostgresDatabase persists a FULL_ONLY physical DB row pointed at the shared
// compose primary source for versionTag (the cross-suite fixture, like the logical PG-16 fixture).
func CreateTestPhysicalPostgresDatabase(
	workspaceID uuid.UUID,
	notifier *notifiers.Notifier,
	versionTag string,
) *Database {
	host, port := physicalPrimaryEndpoint(versionTag)

	return CreateTestPhysicalPostgresDatabaseWithType(
		host, port, workspaceID, notifier, versionTag, postgresql_physical.BackupTypeFullOnly)
}

// CreateTestPhysicalPostgresDatabaseWithType persists a physical DB row at host:port with an explicit
// BackupType, for scheduler-driven incremental / WAL-stream chains whose eligibility the scheduler
// reads from this DB-level field. The caller owns the container lifecycle (host/port may be a
// throwaway testcontainers endpoint), keeping this file free of the testcontainers dependency.
func CreateTestPhysicalPostgresDatabaseWithType(
	host string,
	port int,
	workspaceID uuid.UUID,
	notifier *notifiers.Notifier,
	versionTag string,
	backupType postgresql_physical.BackupType,
) *Database {
	database := &Database{
		WorkspaceID:        &workspaceID,
		Name:               "test-physical-pg " + uuid.New().String(),
		Type:               DatabaseTypePostgresPhysical,
		PostgresqlPhysical: GetTestPhysicalPostgresConfigWithType(host, port, versionTag, backupType),
		Notifiers: []notifiers.Notifier{
			*notifier,
		},
	}

	database, err := databaseRepository.Save(database)
	if err != nil {
		panic(err)
	}

	return database
}

// Host and Port address a bastioned cluster from inside the bastion's network, so they are
// unreachable from the host running the test: only the tunnel gets there.
type PhysicalTestDatabaseSpec struct {
	Host       string
	Port       int
	VersionTag string
	BackupType postgresql_physical.BackupType
	SshTunnel  sshtunnel.Config
}

func CreateTestPhysicalPostgresDatabaseWithTunnel(
	spec PhysicalTestDatabaseSpec,
	workspaceID uuid.UUID,
	notifier *notifiers.Notifier,
) *Database {
	physicalConfig := GetTestPhysicalPostgresConfigWithType(
		spec.Host, spec.Port, spec.VersionTag, spec.BackupType,
	)
	physicalConfig.SshTunnel = spec.SshTunnel

	database := &Database{
		WorkspaceID:        &workspaceID,
		Name:               "test-physical-pg-bastioned " + uuid.New().String(),
		Type:               DatabaseTypePostgresPhysical,
		PostgresqlPhysical: physicalConfig,
		Notifiers: []notifiers.Notifier{
			*notifier,
		},
	}

	database, err := databaseRepository.Save(database)
	if err != nil {
		panic(err)
	}

	return database
}

// CreateTestPhysicalPostgresDatabaseNoSummary persists a physical DB row at host:port pointed at a
// summarize_wal=off cluster, so incremental pre-checks reach the SUMMARIZER_OFF branch
// deterministically (ALTER SYSTEM cannot override a command-line GUC on the standard fixture).
func CreateTestPhysicalPostgresDatabaseNoSummary(
	host string,
	port int,
	workspaceID uuid.UUID,
	notifier *notifiers.Notifier,
	versionTag string,
) *Database {
	database := &Database{
		WorkspaceID:        &workspaceID,
		Name:               "test-physical-pg-no-summary " + uuid.New().String(),
		Type:               DatabaseTypePostgresPhysical,
		PostgresqlPhysical: GetTestPhysicalPostgresConfigNoSummary(host, port, versionTag),
		Notifiers: []notifiers.Notifier{
			*notifier,
		},
	}

	database, err := databaseRepository.Save(database)
	if err != nil {
		panic(err)
	}

	return database
}

func CreateTestMariadbDatabase(
	host string,
	port int,
	workspaceID uuid.UUID,
	notifier *notifiers.Notifier,
) *Database {
	database := &Database{
		WorkspaceID: &workspaceID,
		Name:        "test-mariadb " + uuid.New().String(),
		Type:        DatabaseTypeMariadb,
		Mariadb:     GetTestMariadbConfig(host, port),
		Notifiers: []notifiers.Notifier{
			*notifier,
		},
	}

	database, err := databaseRepository.Save(database)
	if err != nil {
		panic(err)
	}

	return database
}

func CreateTestMongodbDatabase(
	host string,
	port int,
	workspaceID uuid.UUID,
	notifier *notifiers.Notifier,
) *Database {
	database := &Database{
		WorkspaceID: &workspaceID,
		Name:        "test-mongodb " + uuid.New().String(),
		Type:        DatabaseTypeMongodb,
		Mongodb:     GetTestMongodbConfig(host, port),
		Notifiers: []notifiers.Notifier{
			*notifier,
		},
	}

	database, err := databaseRepository.Save(database)
	if err != nil {
		panic(err)
	}

	return database
}

// The context is stripped of cancellation because callers pass t.Context() from a t.Cleanup, and
// the test context is already cancelled by the time cleanup runs.
func RemoveTestDatabase(ctx context.Context, database *Database) {
	if err := databaseService.DeleteForTest(context.WithoutCancel(ctx), database.ID); err != nil {
		panic(fmt.Sprintf("failed to delete test database: %v", err))
	}
}

// DeleteForTest removes a database through the listener chain, which owns external cleanup such as
// backup rows and replication slots, without permission checks or audit logging. A missing row
// counts as success, so a fixture's teardown can re-delete a database the test already deleted.
// Under the
// parallel test suite (go test -p=N) the cascade can deadlock against a concurrent writer in
// the same process (a backup scheduler still settling), surfacing as SQLSTATE 40P01. The
// aborted transaction is transient, so the whole delete — idempotent on a re-run — is retried.
func (s *DatabaseService) DeleteForTest(ctx context.Context, id uuid.UUID) error {
	const maxAttempts = 5

	deleteOnce := func() error {
		for _, listener := range s.dbRemoveListener {
			if err := listener.OnBeforeDatabaseRemove(ctx, id); err != nil {
				return err
			}
		}

		if err := s.dbRepository.Delete(id); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		return nil
	}

	var err error
	for attempt := range maxAttempts {
		err = deleteOnce()
		if err == nil || !isTransientSerializationError(err) {
			return err
		}

		time.Sleep(time.Duration(attempt+1) * 20 * time.Millisecond)
	}

	return err
}

// isTransientSerializationError reports whether err is a PostgreSQL deadlock (40P01) or
// serialization failure (40001) — both transient, where retrying the aborted transaction is
// the correct response rather than failing.
func isTransientSerializationError(err error) bool {
	var pgErr *pgconn.PgError

	return errors.As(err, &pgErr) && (pgErr.Code == "40P01" || pgErr.Code == "40001")
}
