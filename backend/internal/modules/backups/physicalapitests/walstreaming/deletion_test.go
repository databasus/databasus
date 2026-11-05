package walstreaming_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/lib/testwait"
)

func TestDeleteStreamingPhysicalDatabase(t *testing.T) {
	t.Parallel()

	t.Run(
		"cancels the real backup and streamer and removes both slots and their local queue after database deletion",
		func(t *testing.T) {
			t.Parallel()

			team := newWalTeam(t, walTeamSpec{})
			team.enable(t)
			team.waitForReceiver(t)
			source := team.openSource(t)
			rotateWal(t, source)
			team.waitForWal(t, 1)
			team.App.WaitForPhysicalBackups(t)

			_, err := source.Exec(
				t.Context(),
				"CREATE TABLE deletion_backup_payload AS SELECT string_agg(md5(random()::text),'') AS payload FROM generate_series(1,1000000) entry GROUP BY entry/32",
			)
			require.NoError(t, err)

			team.App.InstallStorageWriteBarrier(t)
			storageLock := team.App.BeginSQLTransaction(t)
			var storageIDs []string
			storageLock.QuerySQL(t, &storageIDs, "SELECT id FROM storages WHERE id=? FOR NO KEY UPDATE", team.StorageID)
			require.Len(t, storageIDs, 1)

			team.call(
				t,
				apptest.PhysicalStartRequest(team.Database.Id, "FULL"),
				http.StatusAccepted,
				nil,
			)
			team.App.RunJobPass(t, "backups_physical_scheduler")

			testwait.Until(
				t,
				"a real full and WAL receiver both hold their replication slots",
				func(ctx context.Context) error {
					var count int
					err := source.QueryRow(ctx, "SELECT count(*) FROM pg_replication_slots WHERE starts_with(slot_name,'databasus_')").
						Scan(&count)
					if err != nil {
						return err
					}
					if count != 2 {
						return fmt.Errorf("%d slots instead of two", count)
					}
					return nil
				},
			)

			var runningBackup struct{ ID uuid.UUID }
			team.App.QuerySQL(
				t,
				&runningBackup,
				"SELECT id FROM backups_physical WHERE database_id=? AND status='IN_PROGRESS'",
				team.Database.Id,
			)
			resumeBackup := apptest.SuspendPhysicalBackupProcess(t, runningBackup.ID)

			deletionLock := team.App.BeginSQLTransaction(t)
			var databaseIDs []string
			deletionLock.QuerySQL(t, &databaseIDs, "SELECT id FROM databases WHERE id=? FOR UPDATE", team.Database.Id)
			require.Len(t, databaseIDs, 1)
			type deletionResponse struct {
				Status int
				Body   []byte
				Err    error
			}
			finished := make(chan deletionResponse, 1)
			go func() {
				response, err := team.Owner.Client.DeleteDatabaseWithResponse(t.Context(), team.Database.Id)
				if err != nil {
					finished <- deletionResponse{Err: err}
					return
				}
				finished <- deletionResponse{Status: response.StatusCode(), Body: response.Body}
			}()

			testwait.Until(t, "deletion waits on the database row before committing", func(context.Context) error {
				var waiting int
				team.App.QuerySQL(
					t,
					&waiting,
					`SELECT count(*) FROM pg_locks waiting
					JOIN pg_stat_activity activity ON activity.pid=waiting.pid
					WHERE NOT waiting.granted AND waiting.locktype='transactionid'
					AND activity.datname=current_database()`,
				)
				if waiting == 0 {
					return fmt.Errorf("deletion has not reached the held transaction")
				}
				return nil
			})

			assert.True(t, team.live(t).IsStreamerRunning)
			var runningSlots int
			require.NoError(
				t,
				source.QueryRow(t.Context(), "SELECT count(*) FROM pg_replication_slots WHERE starts_with(slot_name,'databasus_')").
					Scan(&runningSlots),
			)
			assert.Equal(t, 2, runningSlots)

			deletionLock.Commit(t)
			var deletion deletionResponse
			select {
			case deletion = <-finished:
			case <-t.Context().Done():
				t.Fatal("database deletion did not complete")
			}

			require.NoError(t, deletion.Err)
			require.Equal(t, http.StatusNoContent, deletion.Status, "%s", deletion.Body)
			resumeBackup()
			storageLock.Rollback(t)
			team.App.WaitForPhysicalBackups(t)

			testwait.Until(t, "the deleted database has no slots, streamer or queue", func(ctx context.Context) error {
				var slots int
				err := source.QueryRow(ctx, "SELECT count(*) FROM pg_replication_slots WHERE starts_with(slot_name,'databasus_')").
					Scan(&slots)
				if err != nil {
					return err
				}
				var rows int
				team.App.QuerySQL(
					t,
					&rows,
					"SELECT count(*) FROM backups_physical_wal_streamers WHERE database_id=?",
					team.Database.Id,
				)
				if slots != 0 || rows != 0 || team.App.IsWalQueueDirectoryPresent(t, team.Database.Id) {
					return fmt.Errorf("cleanup still has %d slots and %d streamers", slots, rows)
				}
				return nil
			})

			var runningBackups int
			team.App.QuerySQL(
				t,
				&runningBackups,
				"SELECT count(*) FROM backups_physical WHERE status = 'IN_PROGRESS' AND database_id=?",
				team.Database.Id,
			)
			assert.Zero(t, runningBackups)
		},
	)
}
