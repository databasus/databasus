package backups_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func insertUsageDatabase(t *testing.T, backupTeam team, engine string) uuid.UUID {
	t.Helper()

	if engine == "POSTGRES_LOGICAL" {
		return backupTeam.insertLogicalDatabase(t)
	}

	// The creation API requires a reachable source; catalog reads must work without one.
	databaseID := uuid.New()
	backupTeam.app.ExecuteSQL(
		t,
		"INSERT INTO databases (id, workspace_id, name, type, health_status) VALUES (?, ?, ?, ?, 'AVAILABLE')",
		databaseID,
		backupTeam.workspace.Id,
		"Usage source",
		engine,
	)
	query := fmt.Sprintf(
		"INSERT INTO databases_%s (database_id, version, host, port, username, password, database) VALUES (?, ?, 'unreachable.invalid', 1234, 'backup', '', 'source')",
		strings.ToLower(engine),
	)
	version := map[string]string{"MYSQL": "8.0", "MARIADB": "11.4", "MONGODB": "8.0"}[engine]
	backupTeam.app.ExecuteSQL(t, query, databaseID, version)

	return databaseID
}

type usageBackupSpec struct {
	DatabaseID uuid.UUID
	StorageID  uuid.UUID
	Status     apiclient.LogicalBackupStatus
	SizeMb     float64
}

func insertUsageBackup(t *testing.T, backupTeam team, spec usageBackupSpec) uuid.UUID {
	t.Helper()
	backupID := insertLogicalBackupRecord(t, backupTeam, spec.DatabaseID, spec.StorageID, spec.Status, time.Now().UTC())
	backupTeam.app.ExecuteSQL(t, "UPDATE backups_logical SET backup_size_mb = ? WHERE id = ?", spec.SizeMb, backupID)

	return backupID
}

func getStorageUsage(t *testing.T, session apptest.Session, storageID uuid.UUID) float64 {
	t.Helper()
	response, err := session.Client.GetStorageBackupUsageWithResponse(t.Context(), storageID)
	require.NoError(t, err)
	require.NotNil(t, response.JSON200, "%s", response.Body)

	return response.JSON200.TotalUsageMb
}
