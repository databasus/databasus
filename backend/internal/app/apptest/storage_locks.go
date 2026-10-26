package apptest

import (
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

// StorageRowLock holds the row of a storage the way a transfer holds it, in a transaction the test
// commits when it chooses.
type StorageRowLock struct {
	transaction *gorm.DB
}

// LockStorageRow takes the row of the storage FOR UPDATE in a transaction of its own and leaves it
// open, so a test can show what waits for a transfer of the storage.
func (a *App) LockStorageRow(tb testing.TB, storageID uuid.UUID) *StorageRowLock {
	tb.Helper()

	db, err := database.Open(tb.Context(), a.databaseDsn)
	if err != nil {
		tb.Fatalf("open the database of the test application: %v", err)
	}

	transaction := db.WithContext(tb.Context()).Begin()
	tb.Cleanup(func() {
		transaction.Rollback()
		closeConnections(db)
	})

	var lockedIDs []uuid.UUID

	err = transaction.Raw("SELECT id FROM storages WHERE id = ? FOR UPDATE", storageID).Scan(&lockedIDs).Error
	if err != nil {
		tb.Fatalf("lock the storage: %v", err)
	}

	if len(lockedIDs) != 1 {
		tb.Fatalf("lock the storage: no storage %s", storageID)
	}

	return &StorageRowLock{transaction: transaction}
}

// MoveToWorkspace changes the storage's workspace in the held transaction, as a transfer does.
func (l *StorageRowLock) MoveToWorkspace(tb testing.TB, storageID, workspaceID uuid.UUID) {
	tb.Helper()

	err := l.transaction.Exec("UPDATE storages SET workspace_id = ? WHERE id = ?", workspaceID, storageID).Error
	if err != nil {
		tb.Fatalf("move the storage: %v", err)
	}
}

func (l *StorageRowLock) Commit(tb testing.TB) {
	tb.Helper()

	err := l.transaction.Commit().Error
	if err != nil {
		tb.Fatalf("commit the storage lock: %v", err)
	}
}
