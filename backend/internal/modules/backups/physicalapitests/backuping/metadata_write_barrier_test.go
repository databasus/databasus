package backuping_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/lib/testwait"
)

type metadataWriteBarrier struct {
	team        *backupTeam
	transaction *apptest.SQLTransaction
	blockingPID int
	releaseOnce sync.Once
}

func holdBackupMetadata(t *testing.T, team *backupTeam) *metadataWriteBarrier {
	t.Helper()
	team.App.ExecuteSQL(t, `CREATE FUNCTION hold_backup_metadata_write() RETURNS trigger AS $$ BEGIN
		IF right(NEW.file_name, 9) = '.metadata' AND NEW.not_before > NEW.created_at THEN
		PERFORM pg_advisory_xact_lock(1);
		END IF; RETURN NEW; END $$ LANGUAGE plpgsql`)
	team.App.ExecuteSQL(t, `CREATE TRIGGER hold_backup_metadata_write BEFORE INSERT ON storages_pending_file_deletions
		FOR EACH ROW EXECUTE FUNCTION hold_backup_metadata_write()`)

	transaction := team.App.BeginSQLTransaction(t)
	transaction.ExecuteSQL(t, "SELECT pg_advisory_xact_lock(1)")
	var blockingPID int
	transaction.QuerySQL(t, &blockingPID, "SELECT pg_backend_pid()")

	barrier := &metadataWriteBarrier{team: team, transaction: transaction, blockingPID: blockingPID}
	t.Cleanup(func() { barrier.Release(t) })

	return barrier
}

func (barrier *metadataWriteBarrier) Wait(t *testing.T) {
	t.Helper()
	testwait.Until(t, "the completed archive waits to write metadata", func(context.Context) error {
		if !barrier.isWriteHeld(t) {
			return fmt.Errorf("metadata write has not reached the barrier")
		}

		return nil
	})
}

func (barrier *metadataWriteBarrier) Release(t *testing.T) {
	t.Helper()
	barrier.releaseOnce.Do(func() { barrier.transaction.Rollback(t) })
}

func (barrier *metadataWriteBarrier) CancelWrite(t *testing.T) {
	t.Helper()
	var canceledWrites []bool
	barrier.team.App.QuerySQL(t, &canceledWrites, `SELECT pg_cancel_backend(pid) FROM pg_stat_activity
		WHERE datname = current_database() AND ? = ANY(pg_blocking_pids(pid))`, barrier.blockingPID)

	require.Len(t, canceledWrites, 1)
	require.True(t, canceledWrites[0])

	testwait.Until(t, "the metadata write exits with a cancellation error", func(context.Context) error {
		if barrier.isWriteHeld(t) {
			return fmt.Errorf("metadata write is still waiting at the barrier")
		}

		return nil
	})
}

func (barrier *metadataWriteBarrier) isWriteHeld(t *testing.T) bool {
	t.Helper()
	var isMetadataHeld bool
	barrier.team.App.QuerySQL(t, &isMetadataHeld, `SELECT EXISTS (
		SELECT 1 FROM pg_stat_activity
		WHERE datname = current_database() AND ? = ANY(pg_blocking_pids(pid))
	)`, barrier.blockingPID)

	return isMetadataHeld
}
