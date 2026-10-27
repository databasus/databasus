package usecases_physical_postgresql_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/testcontainers/testcontainers-go/network"

	backuping_physical "databasus-backend/internal/features/backups/backups/backuping/physical"
)

// The deletion handlers drop the per-backup and WAL streamer slots when RemoveTestDatabase tears a
// test database down.
func TestMain(m *testing.M) {
	// Other packages share Ryuk with this process. Keeping a network through the run prevents
	// Ryuk from retiring between this package's late container tests and the earlier packages.
	ctx := context.Background()
	sessionNetwork, err := network.New(ctx)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "create test session network:", err)
		os.Exit(1)
	}

	backuping_physical.SubscribeToBackupConfigEventsWithoutApp()
	backuping_physical.SubscribeToDatabaseDeletingWithoutApp()

	exitCode := m.Run()
	if err := sessionNetwork.Remove(ctx); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "remove test session network:", err)
		exitCode = 1
	}
	os.Exit(exitCode)
}
