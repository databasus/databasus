package apptest

import (
	"path/filepath"
	"testing"

	"github.com/google/uuid"

	old_config "databasus-backend/internal/config"
)

func (a *App) IsWalQueueDirectoryPresent(tb testing.TB, databaseID uuid.UUID) bool {
	tb.Helper()
	return isFilePresent(tb, filepath.Join(old_config.GetEnv().DataFolder, "wal-queue", databaseID.String()))
}
