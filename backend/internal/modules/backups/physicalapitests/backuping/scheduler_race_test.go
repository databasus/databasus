package backuping_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

func TestSchedulePhysicalBackupDuringOrphanCleanup(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name                           string
		isFullCompletedWhileWaiting    bool
		isMismatchRecordedWhileWaiting bool
	}{
		{name: "inserts the full after the orphan cleanup transaction releases its lock"},
		{name: "skips the full when another full completes while the scheduler waits on the lock", isFullCompletedWhileWaiting: true},
		{name: "skips the full when a cluster mismatch is recorded while the scheduler waits on the lock", isMismatchRecordedWhileWaiting: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			team := newBackupState(t)
			team.setSchedules(t, true)

			transaction := team.App.BeginSQLTransaction(t)
			transaction.ExecuteSQL(
				t,
				"SELECT pg_advisory_xact_lock(hashtextextended('physical-backup:' || ?::text,0))",
				team.Database.Id,
			)

			finished := make(chan struct{}, 1)
			go func() { team.App.RunJobPass(t, "backups_physical_scheduler"); finished <- struct{}{} }()

			testwait.Until(t, "the scheduler waits on the held orphan cleanup lock", func(context.Context) error {
				var waiting int64
				team.App.QuerySQL(
					t,
					&waiting,
					`SELECT count(*) FROM pg_locks waiting JOIN pg_locks held ON waiting.locktype=held.locktype AND waiting.database=held.database AND waiting.classid=held.classid AND waiting.objid=held.objid AND waiting.objsubid=held.objsubid WHERE NOT waiting.granted AND held.granted AND held.pid IN (SELECT pid FROM pg_stat_activity WHERE datname=current_database()) AND waiting.locktype='advisory'`,
				)
				if waiting == 0 {
					return fmt.Errorf("the scheduler has not reached the lock")
				}
				return nil
			})

			assert.Empty(t, team.listBackups(t))
			var runningBackups int64
			team.App.QuerySQL(
				t,
				&runningBackups,
				"SELECT count(*) FROM backups_physical WHERE status = 'IN_PROGRESS' AND database_id=?",
				team.Database.Id,
			)
			assert.Zero(t, runningBackups)

			concurrentFullID := uuid.New()
			if testCase.isFullCompletedWhileWaiting || testCase.isMismatchRecordedWhileWaiting {
				status := "COMPLETED"
				var reason *string
				if testCase.isMismatchRecordedWhileWaiting {
					status, reason = "CHAIN_BROKEN", new("SYSTEM_IDENTIFIER_MISMATCH")
				}
				transaction.ExecuteSQL(
					t,
					`INSERT INTO backups_physical
				(id,database_id,storage_id,timeline_id,kind,status,error_reason,file_name,start_lsn,stop_lsn,created_at,completed_at)
				VALUES (?,?,?,1,'FULL',?,?,'concurrent-full','0/1000000','0/2000000',?,?)`,
					concurrentFullID,
					team.Database.Id,
					team.StorageID,
					status,
					reason,
					team.App.Clock.Now(),
					team.App.Clock.Now(),
				)
			}

			transaction.Commit(t)
			select {
			case <-finished:
			case <-t.Context().Done():
				t.Fatal("scheduler did not finish after orphan cleanup committed")
			}

			if testCase.isFullCompletedWhileWaiting || testCase.isMismatchRecordedWhileWaiting {
				backups := team.listBackups(t)
				require.Len(t, backups, 1)
				assert.Equal(t, concurrentFullID, backups[0].ID)
				team.assertNoRunningBackup(t)
				return
			}

			testwait.Until(t, "the dispatched full has a terminal outcome", func(context.Context) error {
				backups := team.listBackups(t)
				if len(backups) != 1 || backups[0].Status == "IN_PROGRESS" {
					return fmt.Errorf("no terminal full")
				}
				assert.Equal(t, "FULL", backups[0].Type)
				return nil
			})

			team.assertNoRunningBackup(t)
		})
	}
}
