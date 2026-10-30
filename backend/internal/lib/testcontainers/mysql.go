package containers

import (
	"fmt"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql" // Registers the MySQL driver for readiness probes.
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Credentials baked into every test MySQL container.
const (
	MysqlRootPassword = "rootpassword"
	MysqlUsername     = "testuser"
	MysqlPassword     = "testpassword"
	MysqlDatabase     = "testdb"
)

const mysqlPort = "3306/tcp"

// mysqlStartupTimeout is generous because go test -p=N boots many containers at once; the two-phase
// MySQL/MariaDB cold init (temp server for initdb, then a restart of the real one) runs under CPU
// contention and needs far more than its uncontended ~20s. A fast host returns as soon as the
// server is ready, so the ceiling is free there.
const mysqlStartupTimeout = 300 * time.Second

func mysqlEnv() map[string]string {
	return map[string]string{
		"MYSQL_ROOT_PASSWORD": MysqlRootPassword,
		"MYSQL_DATABASE":      MysqlDatabase,
		"MYSQL_USER":          MysqlUsername,
		"MYSQL_PASSWORD":      MysqlPassword,
	}
}

// MariaDB speaks the MySQL protocol but provisions itself from its own credential constants, so
// each engine supplies the ones its entrypoint actually applied.
type mysqlFamilyReadinessSpec struct {
	Port          string
	RootPassword  string
	Database      string
	IsSslRequired bool
}

// mysqlFamilyReadyOnNetwork is the unpublished-container counterpart of mysqlFamilyReady, which
// cannot be used there because wait.ForSQL resolves a published host port. Forcing TCP is what
// keeps the probe honest: the entrypoint's temporary initdb server runs with --skip-networking, so
// it cannot answer, and the probe only passes once the real server is listening.
func mysqlFamilyReadyOnNetwork(adminBinary, rootPassword string) wait.Strategy {
	return wait.ForExec([]string{
		adminBinary, "ping",
		"--protocol=TCP",
		"--host=127.0.0.1",
		"--user=root",
		"--password=" + rootPassword,
	}).WithExitCode(0).WithStartupTimeout(mysqlStartupTimeout)
}

// mysqlFamilyReady completes a real handshake rather than counting "ready for connections" log
// lines: the entrypoint's temporary socket-only initdb server logs that phrase too, and MySQL 8.x
// adds two more lines from X Plugin, so no fixed occurrence identifies the real server across the
// family. Waiting on the published port alone is not enough either — docker-proxy accepts TCP
// before mysqld binds, which surfaces as an EOF mid-handshake. Selecting the entrypoint-created
// database also proves initdb finished, not merely that the port answers.
func mysqlFamilyReady(spec mysqlFamilyReadinessSpec) wait.Strategy {
	// Servers under require_secure_transport reject the plaintext handshake, and the certificates
	// their images auto-generate are self-signed.
	dsnParams := ""
	if spec.IsSslRequired {
		dsnParams = "?tls=skip-verify"
	}

	return wait.ForSQL(spec.Port, "mysql", func(host, mappedPort string) string {
		portNumber, _, _ := strings.Cut(mappedPort, "/")

		return fmt.Sprintf("root:%s@tcp(%s:%s)/%s%s",
			spec.RootPassword, host, portNumber, spec.Database, dsnParams)
	}).WithStartupTimeout(mysqlStartupTimeout)
}

// mysqlFamilyCmd returns the server flags shared by MySQL and MariaDB: the utf8mb4 charset
// defaults, plus durability-off flags (no fsync, doublewrite or binary log) that are safe for
// throwaway tmpfs containers and make the cold init and the e2e restore RAM-fast.
func mysqlFamilyCmd() []string {
	return []string{
		"--character-set-server=utf8mb4",
		"--collation-server=utf8mb4_unicode_ci",
		"--innodb-flush-log-at-trx-commit=0",
		"--innodb-doublewrite=0",
		"--sync-binlog=0",
		"--skip-log-bin",
	}
}

// mysqlCmd appends the native password plugin for mysql:8.0 (removed in 8.4+) to the shared family
// flags; later versions use the family flags unchanged.
func mysqlCmd(image string) []string {
	cmd := mysqlFamilyCmd()

	if image == "mysql:8.0" {
		cmd = append(cmd, "--default-authentication-plugin=mysql_native_password")
	}

	return cmd
}

func mysqlReadinessSpec(isSslRequired bool) mysqlFamilyReadinessSpec {
	return mysqlFamilyReadinessSpec{
		Port:          mysqlPort,
		RootPassword:  MysqlRootPassword,
		Database:      MysqlDatabase,
		IsSslRequired: isSslRequired,
	}
}

// mysqlRequest builds the container request for image (e.g. "mysql:8.0").
func mysqlRequest(image string) testcontainers.ContainerRequest {
	return testcontainers.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{mysqlPort},
		Env:          mysqlEnv(),
		Cmd:          mysqlCmd(image),
		Tmpfs:        map[string]string{"/var/lib/mysql": dataDirTmpfsOptions},
		WaitingFor:   mysqlFamilyReady(mysqlReadinessSpec(false)),
	}
}

// StartMysql boots a MySQL server from image (e.g. "mysql:8.0").
func StartMysql(t *testing.T, image string) Endpoint {
	t.Helper()

	return start(t, mysqlRequest(image), mysqlPort)
}

func StartMysqlOnNetwork(t *testing.T, spec OnNetworkSpec) Endpoint {
	t.Helper()

	req := mysqlRequest(spec.Image)
	req.WaitingFor = mysqlFamilyReadyOnNetwork("mysqladmin", MysqlRootPassword)

	startUnpublished(t, req, spec.Placement)

	return Endpoint{Host: spec.Placement.Alias, Port: getPortNumber(mysqlPort)}
}

// StartMysqlWithoutCompression stands in for managed endpoints whose proxy cannot negotiate
// compression even though the server advertises it (issue #687).
func StartMysqlWithoutCompression(t *testing.T, image string) Endpoint {
	t.Helper()

	req := mysqlRequest(image)
	req.Cmd = append(req.Cmd, "--protocol_compression_algorithms=uncompressed")

	return start(t, req, mysqlPort)
}

// StartMysqlSSL boots a MySQL server that rejects unencrypted connections. The image auto-generates
// its server certificates, so the test connects with tlsInsecure and no client cert.
func StartMysqlSSL(t *testing.T, image string) Endpoint {
	t.Helper()

	req := mysqlRequest(image)
	req.Cmd = append(req.Cmd, "--require_secure_transport=ON")
	req.WaitingFor = mysqlFamilyReady(mysqlReadinessSpec(true))

	return start(t, req, mysqlPort)
}
