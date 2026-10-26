package apptest

import (
	"testing"

	"github.com/google/uuid"
)

// InsertDatabase writes a logical PostgreSQL database of the workspace directly, with settings that
// point at no server, because creating a database through the API connects to a real one.
func (a *App) InsertDatabase(tb testing.TB, workspaceID uuid.UUID) uuid.UUID {
	tb.Helper()

	databaseID := uuid.New()
	a.ExecuteSQL(tb,
		"INSERT INTO databases (id, workspace_id, name, type, health_status) VALUES (?, ?, ?, ?, ?)",
		databaseID, workspaceID, "Database "+databaseID.String()[:8], "POSTGRES_LOGICAL", "AVAILABLE")
	a.ExecuteSQL(tb,
		`INSERT INTO postgresql_logical_databases (database_id, version, host, port, username, password)
		VALUES (?, '17', 'localhost', 5432, 'postgres', 'password')`,
		databaseID)

	return databaseID
}
