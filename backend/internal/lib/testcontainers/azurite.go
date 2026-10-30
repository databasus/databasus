package containers

import (
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	AzuriteAccountName = "devstoreaccount1"
	// AzuriteAccountKey is Azurite's well-known development account key — a fixed public constant
	// shipped with the emulator for everyone, not a secret.
	AzuriteAccountKey = "Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw=="
)

const azuriteBlobPort = "10000/tcp"

// StartAzurite's caller creates its blob containers against the returned endpoint using the fixed
// Azurite dev account above.
func StartAzurite(t *testing.T) Endpoint {
	t.Helper()

	return StartAzuriteContainer(t).Endpoint
}

func StartAzuriteContainer(t *testing.T) ContainerHandle {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        "mcr.microsoft.com/azure-storage/azurite",
		ExposedPorts: []string{azuriteBlobPort},
		Cmd:          []string{"azurite-blob", "--blobHost", "0.0.0.0", "--skipApiVersionCheck"},
		WaitingFor:   wait.ForListeningPort(azuriteBlobPort).WithStartupTimeout(120 * time.Second),
	}

	return startContainer(t, req, azuriteBlobPort)
}
