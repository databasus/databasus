package apptest

import (
	"net/http"
	"testing"

	"github.com/google/uuid"

	apiclient "databasus-backend/internal/lib/apiclient"
)

// StorageUse names the database whose backup config uses the storage.
type StorageUse struct {
	DatabaseID uuid.UUID
	StorageID  uuid.UUID
}

// CreateStorage creates a storage in the workspace through the API and fails the test unless it
// succeeds.
func CreateStorage(
	tb testing.TB,
	member Session,
	workspaceID uuid.UUID,
	settings apiclient.StorageSettings,
) apiclient.Storage {
	tb.Helper()

	response, err := member.Client.CreateStorageWithResponse(tb.Context(), apiclient.StorageCreate{
		WorkspaceId: workspaceID,
		Name:        "Storage " + uuid.NewString()[:8],
		Settings:    settings,
	})
	if err != nil {
		tb.Fatalf("create the storage: %v", err)
	}

	if response.StatusCode() != http.StatusCreated {
		tb.Fatalf("create the storage answered %d: %s", response.StatusCode(), response.Body)
	}

	return *response.JSON201
}

// CreateLocalStorage creates a local storage in the workspace through the API and returns its ID.
func (a *App) CreateLocalStorage(tb testing.TB, member Session, workspaceID uuid.UUID) uuid.UUID {
	tb.Helper()

	var settings apiclient.StorageSettings

	err := settings.FromLocalStorageSettings(apiclient.LocalStorageSettings{
		Type: apiclient.LocalStorageSettingsTypeLOCAL,
	})
	if err != nil {
		tb.Fatalf("encode the local storage settings: %v", err)
	}

	return CreateStorage(tb, member, workspaceID, settings).Id
}

// InsertBackupConfigUsingStorage writes a logical backup config of the database directly, so a test
// of another module needs no backup config operation.
func (a *App) InsertBackupConfigUsingStorage(tb testing.TB, use StorageUse) {
	tb.Helper()

	a.ExecuteSQL(tb, "INSERT INTO backups_configs (database_id, method, storage_id) VALUES (?, 'LOGICAL', ?)",
		use.DatabaseID, use.StorageID)
	a.ExecuteSQL(tb, `INSERT INTO backups_configs_logical (database_id, interval_type, time_of_day)
		VALUES (?, 'DAILY', '04:00')`,
		use.DatabaseID)
}

// InsertPhysicalBackupConfigUsingStorage writes a physical backup config of the database directly, so a
// test of another module needs no physical database with a backup config.
func (a *App) InsertPhysicalBackupConfigUsingStorage(tb testing.TB, use StorageUse) {
	tb.Helper()

	a.ExecuteSQL(tb, "INSERT INTO backups_configs (database_id, method, storage_id) VALUES (?, 'PHYSICAL', ?)",
		use.DatabaseID, use.StorageID)
	a.ExecuteSQL(tb, `INSERT INTO backups_configs_physical (database_id, full_interval_type, full_time_of_day)
		VALUES (?, 'DAILY', '04:00')`,
		use.DatabaseID)
}
