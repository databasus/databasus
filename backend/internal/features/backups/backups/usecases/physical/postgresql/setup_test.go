package usecases_physical_postgresql_test

import (
	"os"
	"testing"

	backuping_physical "databasus-backend/internal/features/backups/backups/backuping/physical"
)

// The deletion handlers drop the per-backup and WAL streamer slots when RemoveTestDatabase tears a
// test database down.
func TestMain(m *testing.M) {
	backuping_physical.SubscribeToBackupConfigEventsWithoutApp()
	backuping_physical.SubscribeToDatabaseDeletingWithoutApp()

	os.Exit(m.Run())
}
