//go:build testhelpers

package databases_bridge

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/storage"
)

var errNoDatabaseWriteWaiting = errors.New("no write of the databases table waits for a lock yet")

// TestDatabaseDeletion deletes a database in the lock order of DeleteDatabase, the database row first
// and then its children by cascade, but stops after the first lock, so a test can race a writer of
// the database's children against the deletion.
type TestDatabaseDeletion struct {
	tx         *gorm.DB
	databaseID uuid.UUID
}

// BeginTestDatabaseDeletion locks the row of the database as DeleteDatabase does first.
func BeginTestDatabaseDeletion(t *testing.T, databaseID uuid.UUID) *TestDatabaseDeletion {
	t.Helper()

	tx := storage.GetDb().Begin()
	require.NoError(t, tx.Error)
	t.Cleanup(func() { tx.Rollback() })

	require.NoError(t, tx.Exec("SELECT id FROM databases WHERE id = ? FOR UPDATE", databaseID).Error)

	return &TestDatabaseDeletion{tx: tx, databaseID: databaseID}
}

// WaitForBlockedDatabaseWrite waits until another transaction waits to write the databases table,
// which the held row lock blocks.
func (d *TestDatabaseDeletion) WaitForBlockedDatabaseWrite(t *testing.T) {
	t.Helper()

	testwait.Until(t, "a write of the databases table waits for the deleted row", func(context.Context) error {
		var waitingWriteCount int64

		err := storage.GetDb().Raw(`SELECT COUNT(*) FROM pg_stat_activity
			WHERE datname = current_database()
			AND wait_event_type = 'Lock'
			AND query LIKE 'UPDATE databases%'`).
			Scan(&waitingWriteCount).
			Error
		if err != nil {
			return err
		}

		if waitingWriteCount == 0 {
			return errNoDatabaseWriteWaiting
		}

		return nil
	})
}

// Finish deletes the database and commits. The backup configs reference the database with RESTRICT,
// so they go first, as the DatabaseDeleting handler of the backups module removes them.
func (d *TestDatabaseDeletion) Finish() error {
	err := d.tx.Exec("DELETE FROM backups_configs WHERE database_id = ?", d.databaseID).Error
	if err != nil {
		return err
	}

	err = d.tx.Exec("DELETE FROM databases WHERE id = ?", d.databaseID).Error
	if err != nil {
		return err
	}

	return d.tx.Commit().Error
}
