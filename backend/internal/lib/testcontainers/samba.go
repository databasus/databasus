package containers

import (
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Credentials and share baked into the test Samba container (the NAS backend).
const (
	SambaUsername = "testuser"
	SambaPassword = "testpassword"
	SambaShare    = "backups"
)

const sambaPort = "445/tcp"

// StartSamba's image entrypoint creates the share directory, so a throwaway server needs no host volume.
func StartSamba(t *testing.T) Endpoint {
	t.Helper()

	return StartSambaContainer(t).Endpoint
}

func StartSambaContainer(t *testing.T) ContainerHandle {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        "dperson/samba:latest",
		ExposedPorts: []string{sambaPort},
		Env:          map[string]string{"USERID": "1000", "GROUPID": "1000"},
		Cmd: []string{
			"-u", SambaUsername + ";" + SambaPassword,
			"-s", SambaShare + ";/shared;yes;no;no;" + SambaUsername,
			"-p",
		},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort(sambaPort).WithStartupTimeout(120*time.Second),
			forSmbShare(sambaPort, 120*time.Second, SambaUsername, SambaPassword, SambaShare),
		),
	}

	return startContainer(t, req, sambaPort)
}
