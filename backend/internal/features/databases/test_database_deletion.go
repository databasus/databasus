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

// Pausing after the database lock lets a test race publication against deletion of the dependent
// backup rows.
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

// Finish removes the referencing backup records before committing the database deletion.
func (d *TestDatabaseDeletion) Finish() error {
	for _, table := range []string{"backups_physical_wal_stream_samples", "backups_physical_wal_streamers", "backups_physical_in_flight", "backups_physical_wal_segments", "backups_physical_wal_history_files", "backups_physical_incremental", "backups_physical_full"} {
		if err := d.tx.Exec("DELETE FROM "+table+" WHERE database_id = ?", d.databaseID).Error; err != nil {
			return err
		}
	}

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
