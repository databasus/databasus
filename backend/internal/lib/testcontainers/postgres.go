package containers

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Credentials baked into every test PostgreSQL container.
const (
	PostgresUsername = "testuser"
	PostgresPassword = "testpassword"
	PostgresDatabase = "testdb"
)

const postgresPort = "5432/tcp"

// postgresStartupTimeout is generous because go test -p=N boots many containers at once; under CPU
// contention a cold start runs far longer than its uncontended time, and a fast host returns as
// soon as the server is ready, so the high ceiling costs nothing there.
const postgresStartupTimeout = 240 * time.Second

func postgresEnv() map[string]string {
	return map[string]string{
		"POSTGRES_DB":       PostgresDatabase,
		"POSTGRES_USER":     PostgresUsername,
		"POSTGRES_PASSWORD": PostgresPassword,
	}
}

// postgresReady waits for the network-listening server. The entrypoint starts a temporary
// socket-only server for initdb and then restarts the real one, so "ready to accept connections"
// is logged twice; waiting for the second occurrence avoids racing the socket-only server.
func postgresReady() wait.Strategy {
	return wait.ForAll(
		wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).WithStartupTimeout(postgresStartupTimeout),
		wait.ForListeningPort(postgresPort),
	)
}

// postgresRequest builds the container request for image (e.g. "postgres:16"). The durability
// flags are safe for a throwaway server and a large speedup for the write-heavy e2e restore.
func postgresRequest(image string) testcontainers.ContainerRequest {
	return testcontainers.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{postgresPort},
		Env:          postgresEnv(),
		Cmd:          []string{"-c", "fsync=off", "-c", "full_page_writes=off", "-c", "synchronous_commit=off"},
		Tmpfs:        map[string]string{postgresDataDir(image): dataDirTmpfsOptions},
		WaitingFor:   postgresReady(),
	}
}

// postgresDataDir returns the data directory to mount on tmpfs for image. postgres 14–17 keep
// PGDATA and the data VOLUME at /var/lib/postgresql/data. postgres:18 relocated PGDATA and moved
// the volume up to /var/lib/postgresql; mounting tmpfs at the old path there leaves a stray empty
// dir the entrypoint's layout detection rejects, so the server exits before becoming ready.
// Mounting at the parent volume avoids that. An unparseable tag falls back to the pre-18 path.
func postgresDataDir(image string) string {
	if postgresMajorVersion(image) >= 18 {
		return "/var/lib/postgresql"
	}

	return "/var/lib/postgresql/data"
}

// PostgresMajorVersion returns the major version from an image like "postgres:16" or
// "postgres:17.2", or 0 when the tag is missing or not a leading integer (treated as pre-18).
func PostgresMajorVersion(image string) int {
	return postgresMajorVersion(image)
}

// postgresMajorVersion returns the major version from an image like "postgres:16" or
// "postgres:17.2", or 0 when the tag is missing or not a leading integer (treated as pre-18).
func postgresMajorVersion(image string) int {
	_, tag, hasTag := strings.Cut(image, ":")
	if !hasTag {
		return 0
	}

	major, _, _ := strings.Cut(tag, ".")

	version, err := strconv.Atoi(major)
	if err != nil {
		return 0
	}

	return version
}

// StartPostgres boots a PostgreSQL server from image (e.g. "postgres:16").
func StartPostgres(t *testing.T, image string) Endpoint {
	t.Helper()

	return start(t, postgresRequest(image), postgresPort)
}

func StartPostgresOnNetwork(t *testing.T, spec OnNetworkSpec) Endpoint {
	t.Helper()

	req := postgresRequest(spec.Image)
	// wait.ForListeningPort resolves the published host port, and this container has none.
	req.WaitingFor = wait.ForLog("database system is ready to accept connections").
		WithOccurrence(2).WithStartupTimeout(postgresStartupTimeout)

	startUnpublished(t, req, spec.Placement)

	return Endpoint{Host: spec.Placement.Alias, Port: getPortNumber(postgresPort)}
}

// StartPostgresSSL boots an ssl=on PostgreSQL whose pg_hba.conf rejects non-TLS TCP. The image is
// built from contextDir (Dockerfile, server cert/key, pg_hba.conf) because PostgreSQL refuses a key
// file that is group/world-readable or not owned by postgres — the Dockerfile chowns+chmods it,
// which a copied-in file cannot. The test connects with sslmode=require.
func StartPostgresSSL(t *testing.T, contextDir string) Endpoint {
	t.Helper()

	return startPostgresBuild(t, contextDir, "databasus-test-postgres-ssl")
}

// StartPostgresMtls boots a PostgreSQL that also requires a client certificate signed by its ca.crt
// (pg_hba clientcert=verify-ca). Same build rationale as StartPostgresSSL; the test connects with
// sslmode=verify-ca and a client cert/key.
func StartPostgresMtls(t *testing.T, contextDir string) Endpoint {
	t.Helper()

	return startPostgresBuild(t, contextDir, "databasus-test-postgres-mtls")
}

// startPostgresBuild builds the image from contextDir/Dockerfile and boots it. The stable repo tag
// plus KeepImage lets Docker's layer cache skip the rebuild on repeat runs; the readiness flags
// live in the Dockerfile's CMD, so the request only sets the env and wait strategy.
func startPostgresBuild(t *testing.T, contextDir, repo string) Endpoint {
	t.Helper()

	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:    contextDir,
			Dockerfile: "Dockerfile",
			Repo:       repo,
			Tag:        "latest",
			KeepImage:  true,
		},
		ExposedPorts: []string{postgresPort},
		Env:          postgresEnv(),
		WaitingFor:   postgresReady(),
	}

	return start(t, req, postgresPort)
}
