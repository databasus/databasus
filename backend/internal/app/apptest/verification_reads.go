package apptest

import (
	"testing"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/backups"
)

func (a *App) GetPostgresqlLogicalVersions(tb testing.TB, databaseIDs []uuid.UUID) map[uuid.UUID]string {
	tb.Helper()
	versions, err := a.app.GetPostgresqlLogicalVersions(tb.Context(), databaseIDs)
	if err != nil {
		tb.Fatal(err)
	}
	return versions
}

func (a *App) GetLatestCompletedLogicalBackups(
	tb testing.TB,
	databaseIDs []uuid.UUID,
) map[uuid.UUID]backups.LogicalBackup {
	tb.Helper()
	completedBackups, err := a.app.GetLatestCompletedLogicalBackups(tb.Context(), databaseIDs)
	if err != nil {
		tb.Fatal(err)
	}
	return completedBackups
}
