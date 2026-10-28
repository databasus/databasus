package backuping_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/lib/testwait"
)

func TestSchedulePhysicalBackupDuringOrphanCleanup(t *testing.T) {
	t.Run("inserts the full and its claim after the orphan cleanup transaction releases its lock", func(t *testing.T) {
		team := newBackupState(t)
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
		var claims int64
		team.App.QuerySQL(
			t,
			&claims,
			"SELECT count(*) FROM backups_physical_in_flight WHERE database_id=?",
			team.Database.Id,
		)
		assert.Zero(t, claims)
		transaction.Commit(t)
		select {
		case <-finished:
		case <-t.Context().Done():
			t.Fatal("scheduler did not finish after orphan cleanup committed")
		}
		testwait.Until(t, "the dispatched full has a terminal outcome", func(context.Context) error {
			backups := team.listBackups(t)
			if len(backups) != 1 || backups[0].Status == "IN_PROGRESS" {
				return fmt.Errorf("no terminal full")
			}
			assert.Equal(t, "FULL", backups[0].Type)
			return nil
		})
		team.assertClaimReleased(t)
	})
}
