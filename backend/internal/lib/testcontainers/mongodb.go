package containers

import (
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Credentials baked into every test MongoDB container.
const (
	MongodbUsername     = "root"
	MongodbPassword     = "rootpassword"
	MongodbDatabase     = "testdb"
	MongodbAuthDatabase = "admin"
)

const mongodbPort = "27017/tcp"

// mongodbStartupTimeout is generous because go test -p=N starts many mongod containers at once;
// under CPU contention a cold boot runs much longer than its uncontended time. A fast host returns
// as soon as mongod is listening, so the ceiling is free there.
const mongodbStartupTimeout = 240 * time.Second

// The image entrypoint boots mongod twice: on 127.0.0.1 to create the root user, then on 0.0.0.0.
// Docker's proxy accepts on the mapped port during the first boot, so a port check alone hands out
// a connection that is immediately reset.
func mongodbReady() wait.Strategy {
	return wait.ForAll(
		wait.ForLog("Waiting for connections").
			WithOccurrence(2).WithStartupTimeout(mongodbStartupTimeout),
		wait.ForListeningPort(mongodbPort),
	)
}

func mongodbEnv() map[string]string {
	return map[string]string{
		"MONGO_INITDB_ROOT_USERNAME": MongodbUsername,
		"MONGO_INITDB_ROOT_PASSWORD": MongodbPassword,
		"MONGO_INITDB_DATABASE":      MongodbDatabase,
	}
}

// mongodbRequest builds the container request for an auth-enabled mongod from image
// (e.g. "mongo:7.0").
func mongodbRequest(image string) testcontainers.ContainerRequest {
	return testcontainers.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{mongodbPort},
		Env:          mongodbEnv(),
		Cmd:          []string{"mongod", "--auth"},
		Tmpfs:        map[string]string{"/data/db": dataDirTmpfsOptions},
		WaitingFor:   mongodbReady(),
	}
}

// StartMongodb boots an auth-enabled mongod from image (e.g. "mongo:7.0").
func StartMongodb(t *testing.T, image string) Endpoint {
	t.Helper()

	return start(t, mongodbRequest(image), mongodbPort)
}

func StartMongodbOnNetwork(t *testing.T, spec OnNetworkSpec) Endpoint {
	t.Helper()

	req := mongodbRequest(spec.Image)
	// wait.ForListeningPort resolves the published host port, and this container has none.
	req.WaitingFor = wait.ForLog("Waiting for connections").
		WithOccurrence(2).WithStartupTimeout(mongodbStartupTimeout)

	startUnpublished(t, req, spec.Placement)

	return Endpoint{Host: spec.Placement.Alias, Port: getPortNumber(mongodbPort)}
}

// StartMongodbSSL boots a requireTLS mongod, copying the server key/cert from pemPath and crtPath
// into the container. tlsAllowConnectionsWithoutCertificates lets clients connect without a client
// cert (the SSL test uses tlsInsecure).
func StartMongodbSSL(t *testing.T, pemPath, crtPath string) Endpoint {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        "mongo:8.2.3-noble",
		ExposedPorts: []string{mongodbPort},
		Env:          mongodbEnv(),
		Files: []testcontainers.ContainerFile{
			{HostFilePath: pemPath, ContainerFilePath: "/etc/ssl-test/server.pem", FileMode: 0o644},
			{HostFilePath: crtPath, ContainerFilePath: "/etc/ssl-test/server.crt", FileMode: 0o644},
		},
		Cmd: []string{
			"mongod", "--auth",
			"--tlsMode", "requireTLS",
			"--tlsCertificateKeyFile", "/etc/ssl-test/server.pem",
			"--tlsCAFile", "/etc/ssl-test/server.crt",
			"--tlsAllowConnectionsWithoutCertificates",
		},
		WaitingFor: mongodbReady(),
	}

	return start(t, req, mongodbPort)
}
