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

// InstallStorageWriteBarrier makes a storage write wait while a test holds the storage's row. A row
// lock alone does not stop a backup's upload, which can finish before the test acts on the running
// backup.
func (a *App) InstallStorageWriteBarrier(tb testing.TB) {
	tb.Helper()
	a.ExecuteSQL(tb, `CREATE FUNCTION hold_physical_stream_write() RETURNS trigger AS $$ BEGIN
		IF NEW.not_before>NEW.created_at THEN
		PERFORM id FROM storages WHERE id=NEW.storage_id FOR SHARE;
		END IF; RETURN NEW; END $$ LANGUAGE plpgsql`)
	a.ExecuteSQL(
		tb,
		`CREATE TRIGGER hold_physical_stream_write BEFORE INSERT ON storages_pending_file_deletions FOR EACH ROW EXECUTE FUNCTION hold_physical_stream_write()`,
	)
}
