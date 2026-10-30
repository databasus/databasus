package containers

import (
	"testing"

	"github.com/testcontainers/testcontainers-go"
)

// StartTimescaleDB boots a PostgreSQL server with the timescaledb extension from a
// timescale/timescaledb image (e.g. "timescale/timescaledb:2.17.0-pg17").
func StartTimescaleDB(t *testing.T, image string) Endpoint {
	t.Helper()

	return start(t, timescaleDBRequest(image), postgresPort)
}

// timescaleDBRequest mirrors postgresRequest but pins the data dir to the pre-18 layout: every
// published timescaledb image is pre-18, and postgresDataDir's tag parser would misread a
// "<ts>-pg<major>" tag. The timescaledb shared library is loaded by the image's own tuning, so no
// shared_preload_libraries override is needed here.
func timescaleDBRequest(image string) testcontainers.ContainerRequest {
	return testcontainers.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{postgresPort},
		Env:          postgresEnv(),
		Cmd:          []string{"-c", "fsync=off", "-c", "full_page_writes=off", "-c", "synchronous_commit=off"},
		Tmpfs:        map[string]string{"/var/lib/postgresql/data": dataDirTmpfsOptions},
		WaitingFor:   postgresReady(),
	}
}
