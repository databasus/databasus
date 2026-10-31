package containers

import (
	"strings"
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

	return StartSftpWithCredentials(t, SftpCredentials{Password: SftpPassword, PublicKey: ""})
}

type SftpCredentials struct {
	Password  string
	PublicKey string
}

func StartSftpWithCredentials(t *testing.T, credentials SftpCredentials) ContainerHandle {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        "atmoz/sftp:latest",
		ExposedPorts: []string{sftpPort},
		Cmd:          []string{SftpUsername + ":" + credentials.Password + ":1001::" + SftpUploadDir},
		WaitingFor:   sftpReady(),
	}

	if credentials.PublicKey != "" {
		req.Files = []testcontainers.ContainerFile{{
			Reader:            strings.NewReader(credentials.PublicKey),
			ContainerFilePath: "/home/" + SftpUsername + "/.ssh/keys/backup.pub",
			FileMode:          0o644,
		}}
	}

	return startContainer(t, req, sftpPort)
}
