package containers

import (
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Credentials and the writable directory baked into the test atmoz/sftp container.
const (
	SftpUsername  = "testuser"
	SftpPassword  = "testpassword"
	SftpUploadDir = "upload"
)

const (
	sftpPort           = "22/tcp"
	sftpStartupTimeout = 120 * time.Second
)

// The entrypoint generates host keys before it starts sshd, and the 4096-bit RSA key takes
// seconds. Docker's proxy accepts on the mapped port meanwhile, so a port check alone hands out
// a connection that is reset during the SSH handshake.
func sftpReady() wait.Strategy {
	return wait.ForAll(
		wait.ForLog("Server listening on 0.0.0.0 port 22").WithStartupTimeout(sftpStartupTimeout),
		wait.ForListeningPort(sftpPort).WithStartupTimeout(sftpStartupTimeout),
	)
}

func StartSftp(t *testing.T) Endpoint {
	t.Helper()

	return StartSftpContainer(t).Endpoint
}

func StartSftpContainer(t *testing.T) ContainerHandle {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        "atmoz/sftp:latest",
		ExposedPorts: []string{sftpPort},
		Cmd:          []string{SftpUsername + ":" + SftpPassword + ":1001::" + SftpUploadDir},
		WaitingFor:   sftpReady(),
	}

	return startContainer(t, req, sftpPort)
}
