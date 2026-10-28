package apptest

import (
	"testing"

	"github.com/google/uuid"
)

type StorageRowLock struct {
	*SQLTransaction
}

func (a *App) LockStorageRow(tb testing.TB, storageID uuid.UUID) *StorageRowLock {
	tb.Helper()
	transaction := a.BeginSQLTransaction(tb)
	var lockedIDs []uuid.UUID
	transaction.QuerySQL(tb, &lockedIDs, "SELECT id FROM storages WHERE id = ? FOR UPDATE", storageID)
	if len(lockedIDs) != 1 {
		tb.Fatalf("lock the storage: no storage %s", storageID)
	}
	return &StorageRowLock{SQLTransaction: transaction}
}

func (l *StorageRowLock) MoveToWorkspace(tb testing.TB, storageID, workspaceID uuid.UUID) {
	tb.Helper()
	l.ExecuteSQL(tb, "UPDATE storages SET workspace_id = ? WHERE id = ?", workspaceID, storageID)
}
