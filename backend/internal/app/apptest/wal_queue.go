package apptest

import (
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

func (a *App) IsWalQueueDirectoryPresent(tb testing.TB, databaseID uuid.UUID) bool {
	tb.Helper()
	return isFilePresent(tb, filepath.Join(a.dataFolder, "wal-queue", databaseID.String()))
}
