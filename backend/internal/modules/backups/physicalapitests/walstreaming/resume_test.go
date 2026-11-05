package walstreaming_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

func TestPhysicalWalResumeAfterSlotAdvance(t *testing.T) {
	t.Parallel()

	t.Run("archives older queued WAL and resumes above the rebuilt slot after modules restart", func(t *testing.T) {
		t.Parallel()

		team := newWalTeam(t, walTeamSpec{})
		roleName := team.useReplicationRole(t)
		team.useUnavailableStorage(t)
		team.enable(t)
		team.waitForReceiver(t)
		source := team.openSource(t)
		generateWal(t, source, 1)

		var queuedWalName string
		require.NoError(
			t,
			source.QueryRow(t.Context(), "SELECT pg_walfile_name(pg_current_wal_lsn())").Scan(&queuedWalName),
		)
		_, err := source.Exec(t.Context(), "SELECT pg_switch_wal()")
		require.NoError(t, err)

		testwait.Until(t, "the selected WAL segment reaches the failed upload queue", func(context.Context) error {
			var failedWrites int
			team.App.QuerySQL(
				t,
				&failedWrites,
				"SELECT count(*) FROM storages_pending_file_deletions WHERE storage_id=? AND file_name LIKE ? AND generation>0",
				team.StorageID,
				team.Database.Id.String()+"-WAL-tl%-"+queuedWalName+"-%.zst",
			)
			if failedWrites == 0 {
				return fmt.Errorf("the selected WAL segment has not reached the failed upload queue")
			}
			return nil
		})

		var slotName string
		team.App.QuerySQL(
			t,
			&slotName,
			"SELECT replication_slot_name FROM databases_postgresql_physical WHERE database_id=?",
			team.Database.Id,
		)
		_, err = source.Exec(t.Context(), "ALTER ROLE "+pgx.Identifier{roleName}.Sanitize()+" NOLOGIN")
		require.NoError(t, err)
		_, err = source.Exec(
			t.Context(),
			"SELECT pg_terminate_backend(active_pid) FROM pg_replication_slots WHERE slot_name=$1 AND active_pid IS NOT NULL",
			slotName,
		)
		require.NoError(t, err)

		testwait.Until(
			t,
			"the refused receiver ends its session without removing the queue",
			func(context.Context) error {
				if team.live(t).IsStreamerRunning {
					return fmt.Errorf("the first session is still running")
				}
				return nil
			},
		)
		assert.True(t, team.App.IsWalQueueDirectoryPresent(t, team.Database.Id))
		testwait.Until(t, "the unused replication slot can be dropped", func(ctx context.Context) error {
			_, err := source.Exec(
				ctx,
				"SELECT pg_drop_replication_slot(slot_name) FROM pg_replication_slots WHERE slot_name=$1",
				slotName,
			)
			return err
		})

		generateWal(t, source, 4*16*1024*1024)
		_, err = source.Exec(t.Context(), "CHECKPOINT")
		require.NoError(t, err)
		_, err = source.Exec(t.Context(), "SELECT pg_create_physical_replication_slot($1,true)", slotName)
		require.NoError(t, err)

		var isSlotAhead bool
		require.NoError(
			t,
			source.QueryRow(t.Context(), "SELECT pg_walfile_name(restart_lsn)>$2 FROM pg_replication_slots WHERE slot_name=$1", slotName, queuedWalName).
				Scan(&isSlotAhead),
		)
		require.True(t, isSlotAhead)

		_, err = source.Exec(t.Context(), "ALTER ROLE "+pgx.Identifier{roleName}.Sanitize()+" LOGIN")
		require.NoError(t, err)
		team.App.RestartModules(t)
		team.waitForReceiver(t)
		team.recoverStorage(t)

		testwait.Until(
			t,
			"the old queued segment is archived despite the newer resume floor",
			func(context.Context) error {
				var count int
				team.App.QuerySQL(
					t,
					&count,
					"SELECT count(*) FROM backups_physical_wal_segments WHERE database_id=? AND wal_filename=? AND file_name IS NOT NULL",
					team.Database.Id,
					queuedWalName,
				)
				if count != 1 {
					return fmt.Errorf("the queued old WAL has not committed")
				}
				return nil
			},
		)

		rotateWal(t, source)
		team.waitForWal(t, 2)
	})
}
