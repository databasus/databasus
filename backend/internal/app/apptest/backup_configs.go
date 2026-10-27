package apptest

import (
	"net/http"
	"testing"

	"github.com/google/uuid"

	apiclient "databasus-backend/internal/lib/apiclient"
)

// GetBackupConfig reads the backup config of the database through the API and fails the test unless
// the read succeeds.
func GetBackupConfig(tb testing.TB, member Session, databaseID uuid.UUID) apiclient.BackupConfig {
	tb.Helper()

	response, err := member.Client.GetBackupConfigWithResponse(tb.Context(), databaseID)
	if err != nil {
		tb.Fatalf("get the backup config: %v", err)
	}

	if response.StatusCode() != http.StatusOK {
		tb.Fatalf("get the backup config answered %d: %s", response.StatusCode(), response.Body)
	}

	return *response.JSON200
}

// SaveBackupConfigWithStorage saves the database's backup config as read, with backups on and the
// storage named, through the API, and fails the test unless the save succeeds.
func SaveBackupConfigWithStorage(tb testing.TB, member Session, use StorageUse) apiclient.BackupConfig {
	tb.Helper()

	readConfig := GetBackupConfig(tb, member, use.DatabaseID)

	response, err := member.Client.UpdateBackupConfigWithResponse(tb.Context(), use.DatabaseID,
		apiclient.BackupConfigInput{
			IsBackupsEnabled: true,
			StorageId:        &use.StorageID,
			Encryption:       readConfig.Encryption,
			Settings:         readConfig.Settings,
		})
	if err != nil {
		tb.Fatalf("save the backup config: %v", err)
	}

	if response.StatusCode() != http.StatusOK {
		tb.Fatalf("save the backup config answered %d: %s", response.StatusCode(), response.Body)
	}

	return *response.JSON200
}
