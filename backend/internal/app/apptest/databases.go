package apptest

import (
	"net/http"
	"testing"

	"github.com/google/uuid"

	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/modules/databases"
)

// CreateDatabase creates a database in the workspace through the API, which connects to the server
// the settings name, and fails the test unless it succeeds.
func CreateDatabase(
	tb testing.TB,
	member Session,
	workspaceID uuid.UUID,
	settings apiclient.DatabaseSettings,
) apiclient.Database {
	tb.Helper()

	response, err := member.Client.CreateDatabaseWithResponse(tb.Context(), apiclient.DatabaseCreate{
		WorkspaceId: workspaceID,
		Name:        "Database " + uuid.NewString()[:8],
		NotifierIds: []uuid.UUID{},
		Settings:    settings,
	})
	if err != nil {
		tb.Fatalf("create the database: %v", err)
	}

	if response.StatusCode() != http.StatusCreated {
		tb.Fatalf("create the database answered %d: %s", response.StatusCode(), response.Body)
	}

	return *response.JSON201
}

// InsertDatabase writes a logical PostgreSQL database of the workspace directly, with settings that
// point at no server, because creating a database through the API connects to a real one.
func (a *App) InsertDatabase(tb testing.TB, workspaceID uuid.UUID) uuid.UUID {
	tb.Helper()

	databaseID := uuid.New()
	a.ExecuteSQL(tb,
		"INSERT INTO databases (id, workspace_id, name, type, health_status) VALUES (?, ?, ?, ?, ?)",
		databaseID, workspaceID, "Database "+databaseID.String()[:8], "POSTGRES_LOGICAL", "AVAILABLE")
	a.ExecuteSQL(tb,
		`INSERT INTO databases_postgresql_logical (database_id, version, host, port, username, password)
		VALUES (?, '17', 'localhost', 5432, 'postgres', 'password')`,
		databaseID)

	a.ExecuteSQL(
		tb,
		`INSERT INTO healthcheck_configs (database_id, interval_minutes, failed_attempts_before_unavailable, attempt_retention_days)
 VALUES (?,1,3,7)`,
		databaseID,
	)
	return databaseID
}

// GetPhysicalBackupTypes reads the backup types of physical databases through the databases module and
// fails the test unless the read succeeds.
func (a *App) GetPhysicalBackupTypes(
	tb testing.TB,
	databaseIDs []uuid.UUID,
) map[uuid.UUID]databases.PhysicalBackupType {
	tb.Helper()

	backupTypes, err := a.app.GetPhysicalBackupTypes(tb.Context(), databaseIDs)
	if err != nil {
		tb.Fatalf("get the physical backup types: %v", err)
	}

	return backupTypes
}
