package physical_service_test

import (
	"os"
	"testing"

	backuping_physical "databasus-backend/internal/features/backups/backups/backuping/physical"
)

func TestMain(m *testing.M) {
	backuping_physical.SubscribeToBackupConfigEventsWithoutApp()
	backuping_physical.SubscribeToDatabaseDeletingWithoutApp()

	os.Exit(m.Run())
}
