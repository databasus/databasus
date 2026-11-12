package patroni_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/clienttools"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

func runPatroniTransferLifecycle(t *testing.T, fixture *containers.Patroni, mode string) {
	t.Helper()

	proxy := containers.StartPostgresProxy(t, fixture.Endpoint)
	gate := proxy.HoldTransfer(t)
	workingDirectory, err := os.Getwd()
	require.NoError(t, err)
	bundleRoot, err := clienttools.FindBundleRoot(workingDirectory)
	require.NoError(t, err)
	tool := (pgshared.ClientTools{Root: bundleRoot}).GetExecutable(pgshared.Version17, pgshared.ExecutablePgBasebackup)
	ctx, cancel := context.WithCancel(t.Context())
	command := exec.CommandContext(
		ctx,
		tool,
		"--host",
		proxy.Endpoint.Host,
		"--port",
		strconv.Itoa(proxy.Endpoint.Port),
		"--username",
		containers.PostgresUsername,
		"--no-password",
		"--pgdata",
		t.TempDir(),
		"--format=tar",
		"--wal-method=none",
		"--checkpoint=fast",
	)
	command.Env = append(command.Environ(), "PGPASSWORD="+containers.PostgresPassword, "PGSSLMODE=disable")
	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &output
	require.NoError(t, command.Start())
	hasWaited := false
	t.Cleanup(func() {
		cancel()
		if !hasWaited {
			_ = command.Wait()
		}
	})

	testwait.Until(t, "the lifecycle backup reaches its transfer gate", func(context.Context) error {
		select {
		case event := <-gate.Reached:
			require.Positive(t, event.TransferredBytes)
			return nil
		default:
			return fmt.Errorf("no backup transfer observed")
		}
	})

	switch mode {
	case "transfer-failure":
		t.Fatal("intentional fixture transfer-failure")
	case "transfer-cancellation":
		cancel()
		err = command.Wait()
		hasWaited = true
		require.Error(t, err)
		testwait.Until(t, "the canceled transfer closes without releasing the gate", func(context.Context) error {
			for _, event := range proxy.GetEvents() {
				if event.Kind == "closed" {
					return nil
				}
			}

			return fmt.Errorf("canceled transfer remains connected")
		})
	case "transfer-release":
		gate.Release()
		gate.Release()
		err = command.Wait()
		hasWaited = true
		require.NoError(t, err, "%s", output.String())
	default:
		t.Fatalf("unknown transfer lifecycle mode %s", mode)
	}
}
