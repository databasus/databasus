package apptest

import (
	"testing"

	"github.com/google/uuid"
)

func (a *App) InsertPhysicalDatabase(tb testing.TB, workspaceID uuid.UUID) uuid.UUID {
	tb.Helper()
	databaseID := uuid.New()
	a.ExecuteSQL(tb, `INSERT INTO databases (id, workspace_id, name, type, health_status)
  VALUES (?, ?, ?, 'POSTGRES_PHYSICAL', 'AVAILABLE')`, databaseID, workspaceID, "Physical database "+databaseID.String()[:8])
	a.ExecuteSQL(tb, `INSERT INTO databases_postgresql_physical
  (database_id, version, host, port, username, password, backup_type, replication_slot_name, backup_slot_name)
  VALUES (?, '17', '127.0.0.1', 1, 'postgres', '', 'FULL_INCREMENTAL_WAL_STREAM', ?, ?)`,
		databaseID, "databasus_slot_"+databaseID.String()[:8], "databasus_basebackup_"+databaseID.String()[:8])
	a.ExecuteSQL(
		tb,
		`INSERT INTO healthcheck_configs (database_id, interval_minutes, failed_attempts_before_unavailable, attempt_retention_days)
 VALUES (?,1,3,7)`,
		databaseID,
	)
	return databaseID
}
