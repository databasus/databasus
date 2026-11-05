package enginetest

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"testing"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	containers "databasus-backend/internal/lib/testcontainers"
)

// SkipVerifyMysqlTLSConfigName names the TLS config of the MySQL driver that the TLS tests of MySQL
// and MariaDB connect with, since their servers present self-signed certificates.
const SkipVerifyMysqlTLSConfigName = "engine-tls-skip-verify"

// RegisterSkipVerifyMysqlTLSConfig registers the config once per process, because the driver panics
// when a name is registered twice.
//
//nolint:gochecknoglobals // The external SQL driver registers TLS configurations process-wide; one registration is shared across test-owned connections.
var RegisterSkipVerifyMysqlTLSConfig = sync.OnceValue(func() error {
	// The disposable TLS servers present test-owned self-signed certificates.
	// nosemgrep: problem-based-packs.insecure-transport.go-stdlib.bypass-tls-verification.bypass-tls-verification
	return mysqldriver.RegisterTLSConfig(SkipVerifyMysqlTLSConfigName, &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: true,
	})
})

// MysqlFamilyServer is a MySQL or MariaDB server of a test, reached as root.
type MysqlFamilyServer struct {
	Endpoint     containers.Endpoint
	RootPassword string
	// DSNOptions are extra driver options of the test's own connections, such as the TLS config.
	DSNOptions string
	Admin      *sqlx.DB
}

type mysqlFamilyTestRow struct {
	ID        int       `db:"id"`
	Name      string    `db:"name"`
	Value     int       `db:"value"`
	CreatedAt time.Time `db:"created_at"`
}

func ConnectToMysqlFamily(
	t *testing.T,
	endpoint containers.Endpoint,
	rootPassword string,
	dsnOptions string,
) MysqlFamilyServer {
	t.Helper()

	server := MysqlFamilyServer{Endpoint: endpoint, RootPassword: rootPassword, DSNOptions: dsnOptions, Admin: nil}
	server.Admin = server.Open(t, containers.MysqlDatabase)

	return server
}

// Open connects to the database as root until the test ends.
func (s MysqlFamilyServer) Open(t *testing.T, databaseName string) *sqlx.DB {
	t.Helper()

	dsn := fmt.Sprintf("root:%s@tcp(%s:%d)/%s?parseTime=true",
		s.RootPassword, s.Endpoint.Host, s.Endpoint.Port, databaseName)
	if s.DSNOptions != "" {
		dsn += "&" + s.DSNOptions
	}

	connection, err := sqlx.ConnectContext(t.Context(), "mysql", dsn)
	if err != nil {
		t.Fatalf("connect to %s: %v", databaseName, err)
	}

	t.Cleanup(func() { _ = connection.Close() })

	return connection
}

func (s MysqlFamilyServer) Execute(t *testing.T, statement string) {
	t.Helper()

	_, err := s.Admin.ExecContext(t.Context(), statement)
	if err != nil {
		t.Fatalf("run %q: %v", statement, err)
	}
}

// CleanUp runs the statement on the shared server when the test ends, so the next scenario starts from
// the same database.
func (s MysqlFamilyServer) CleanUp(t *testing.T, statement string) {
	t.Helper()

	admin := s.Admin

	t.Cleanup(func() {
		_, _ = admin.ExecContext(context.WithoutCancel(t.Context()), statement)
	})
}

// CreateTestDataTable creates test_data with three rows, replacing the one an earlier scenario left.
func (s MysqlFamilyServer) CreateTestDataTable(t *testing.T) {
	t.Helper()

	s.Execute(t, "DROP TABLE IF EXISTS test_data")
	s.Execute(t, `
		CREATE TABLE test_data (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			value INT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`)
	s.Execute(t, "INSERT INTO test_data (name, value) VALUES ('test1', 100), ('test2', 200), ('test3', 300)")
}

// CreateExtraTable creates extra_table with one row and drops it when the test ends.
func (s MysqlFamilyServer) CreateExtraTable(t *testing.T) {
	t.Helper()

	s.Execute(t, "DROP TABLE IF EXISTS extra_table")
	s.Execute(t, "CREATE TABLE extra_table (id INT AUTO_INCREMENT PRIMARY KEY, data VARCHAR(255) NOT NULL)")
	s.Execute(t, "INSERT INTO extra_table (data) VALUES ('drop_me')")
	s.CleanUp(t, "DROP TABLE IF EXISTS extra_table")
}

// CreateRestoreDatabase creates an empty database the test restores into and drops it when the test
// ends.
func (s MysqlFamilyServer) CreateRestoreDatabase(t *testing.T, prefix string) (string, *sqlx.DB) {
	t.Helper()

	databaseName := prefix + "_" + NewUniqueSuffix()
	s.Execute(t, "CREATE DATABASE "+databaseName)
	s.CleanUp(t, "DROP DATABASE IF EXISTS "+databaseName)

	return databaseName, s.Open(t, databaseName)
}

func CountMysqlFamilyTables(t *testing.T, db *sqlx.DB, databaseName, tableName string) int {
	t.Helper()

	var tableCount int

	err := db.GetContext(t.Context(), &tableCount,
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ? AND table_name = ?",
		databaseName, tableName)
	if err != nil {
		t.Fatalf("count the tables named %s: %v", tableName, err)
	}

	return tableCount
}

// VerifyMysqlFamilyRows checks that the restored test_data holds the source's rows.
func VerifyMysqlFamilyRows(t *testing.T, sourceDB, restoredDB *sqlx.DB) {
	t.Helper()

	sourceRows := selectMysqlFamilyRows(t, sourceDB)
	restoredRows := selectMysqlFamilyRows(t, restoredDB)

	if len(restoredRows) != len(sourceRows) {
		t.Fatalf("the restored table has %d rows and the source %d", len(restoredRows), len(sourceRows))
	}

	for rowIndex, sourceRow := range sourceRows {
		restoredRow := restoredRows[rowIndex]

		if restoredRow.ID != sourceRow.ID || restoredRow.Name != sourceRow.Name ||
			restoredRow.Value != sourceRow.Value {
			t.Errorf("restored row %d is (%d, %s, %d), the source row is (%d, %s, %d)", rowIndex,
				restoredRow.ID, restoredRow.Name, restoredRow.Value, sourceRow.ID, sourceRow.Name, sourceRow.Value)
		}
	}
}

func selectMysqlFamilyRows(t *testing.T, db *sqlx.DB) []mysqlFamilyTestRow {
	t.Helper()

	var rows []mysqlFamilyTestRow

	err := db.SelectContext(t.Context(), &rows, "SELECT id, name, value, created_at FROM test_data ORDER BY id")
	if err != nil {
		t.Fatalf("read test_data: %v", err)
	}

	return rows
}
