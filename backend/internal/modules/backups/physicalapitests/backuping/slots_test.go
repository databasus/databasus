package backuping_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

func (team *backupTeam) slotNames(t *testing.T) (string, string) {
	t.Helper()
	var names []struct {
		BackupSlotName    string
		WalStreamSlotName string
	}
	team.App.QuerySQL(
		t,
		&names,
		"SELECT backup_slot_name,replication_slot_name AS wal_stream_slot_name FROM databases_postgresql_physical WHERE database_id=?",
		team.Database.Id,
	)
	require.Len(t, names, 1)
	return names[0].BackupSlotName, names[0].WalStreamSlotName
}

func TestPhysicalBackupSlots(t *testing.T) {
	t.Parallel()

	for _, version := range []string{"17", "18"} {
		for _, kind := range []string{"full", "incremental"} {
			t.Run(
				"keeps a permanent physical slot during "+kind+" on PostgreSQL "+version+" and removes it after completion",
				func(t *testing.T) {
					t.Parallel()

					team := newBackupTeam(t, backupTeamSpec{Image: "postgres:" + version})
					if kind == "incremental" {
						require.Equal(t, "COMPLETED", team.runBackup(t, "full").Status)
					}

					source := team.openSource(t)
					slotName, _ := team.slotNames(t)
					team.App.InstallStorageWriteBarrier(t)
					lock := team.App.BeginSQLTransaction(t)
					var storageIDs []string
					lock.QuerySQL(
						t,
						&storageIDs,
						"SELECT id FROM storages WHERE id=? FOR NO KEY UPDATE",
						team.StorageID,
					)
					require.Len(t, storageIDs, 1)

					team.trigger(t, kind)
					team.App.RunJobPass(t, "backups_physical_scheduler")

					testwait.Until(
						t,
						"the running backup reserves WAL with its physical slot",
						func(ctx context.Context) error {
							var count int
							err := source.QueryRow(ctx, "SELECT count(*) FROM pg_replication_slots WHERE slot_name=$1 AND slot_type='physical' AND NOT temporary AND restart_lsn IS NOT NULL", slotName).
								Scan(&count)
							if err != nil {
								return err
							}
							if count != 1 {
								return fmt.Errorf("slot is not ready")
							}
							return nil
						},
					)

					lock.Rollback(t)
					team.App.WaitForPhysicalBackups(t)
					rows := team.listBackups(t)
					require.Equal(t, "COMPLETED", rows[0].Status)

					var count int
					require.NoError(
						t,
						source.QueryRow(t.Context(), "SELECT count(*) FROM pg_replication_slots WHERE slot_name=$1", slotName).
							Scan(&count),
					)
					assert.Zero(t, count)

					team.assertNoRunningBackup(t)
				},
			)
		}
	}
	t.Run("recovers an orphan before a full and reuses its slot name for the next full", func(t *testing.T) {
		t.Parallel()

		team := newBackupTeam(t, backupTeamSpec{})
		source := team.openSource(t)
		slotName, _ := team.slotNames(t)
		_, err := source.Exec(t.Context(), "SELECT pg_create_physical_replication_slot($1,true)", slotName)
		require.NoError(t, err)

		for range 2 {
			require.Equal(t, "COMPLETED", team.runBackup(t, "full").Status)
			var count int
			require.NoError(
				t,
				source.QueryRow(t.Context(), "SELECT count(*) FROM pg_replication_slots WHERE slot_name=$1", slotName).
					Scan(&count),
			)
			assert.Zero(t, count)
		}
	})
	t.Run("reuses the slot after a failed full when its source becomes reachable", func(t *testing.T) {
		t.Parallel()

		team := newBackupTeam(t, backupTeamSpec{})

		team.App.ExecuteSQL(
			t,
			"UPDATE databases_postgresql_physical SET host='127.0.0.1',port=1 WHERE database_id=?",
			team.Database.Id,
		)
		require.Equal(t, "ERROR", team.runBackup(t, "full").Status)

		team.App.ExecuteSQL(
			t,
			"UPDATE databases_postgresql_physical SET host=?,port=? WHERE database_id=?",
			team.Source.Host,
			team.Source.Port,
			team.Database.Id,
		)

		require.Equal(t, "COMPLETED", team.runBackup(t, "full").Status)
		team.assertNoRunningBackup(t)
	})
}

func TestPhysicalBackupSlotStartupCleanup(t *testing.T) {
	t.Parallel()

	for _, isBastioned := range []bool{false, true} {
		scenario := "removes an orphan and preserves the streamer and unrelated slots when modules restart"
		if isBastioned {
			scenario = "removes the orphan through its SSH bastion when modules restart"
		}
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()

			team := newBackupTeam(t, backupTeamSpec{IsBastioned: isBastioned})
			source := team.openSource(t)
			backupSlot, streamerSlot := team.slotNames(t)

			for _, slot := range []string{backupSlot, streamerSlot, "unrelated_physical_test_slot"} {
				_, err := source.Exec(t.Context(), "SELECT pg_create_physical_replication_slot($1,true)", slot)
				require.NoError(t, err)
			}

			team.App.RestartModules(t)

			var slots []string
			rows, err := source.Query(t.Context(), "SELECT slot_name FROM pg_replication_slots")
			require.NoError(t, err)
			defer rows.Close()
			for rows.Next() {
				var slot string
				require.NoError(t, rows.Scan(&slot))
				slots = append(slots, slot)
			}
			require.NoError(t, rows.Err())

			assert.ElementsMatch(t, []string{streamerSlot, "unrelated_physical_test_slot"}, slots)
		})
	}
	t.Run("skips an unreachable source and removes its orphan after the next reachable restart", func(t *testing.T) {
		t.Parallel()

		team := newBackupTeam(t, backupTeamSpec{})
		source := team.openSource(t)
		slotName, _ := team.slotNames(t)
		_, err := source.Exec(t.Context(), "SELECT pg_create_physical_replication_slot($1,true)", slotName)
		require.NoError(t, err)

		team.App.ExecuteSQL(
			t,
			"UPDATE databases_postgresql_physical SET host='127.0.0.1',port=1 WHERE database_id=?",
			team.Database.Id,
		)

		team.App.RestartModules(t)

		var count int
		require.NoError(
			t,
			source.QueryRow(t.Context(), "SELECT count(*) FROM pg_replication_slots WHERE slot_name=$1", slotName).
				Scan(&count),
		)
		assert.Equal(t, 1, count)

		team.App.ExecuteSQL(
			t,
			"UPDATE databases_postgresql_physical SET host=?,port=? WHERE database_id=?",
			team.Source.Host,
			team.Source.Port,
			team.Database.Id,
		)

		team.App.RestartModules(t)
		require.NoError(
			t,
			source.QueryRow(t.Context(), "SELECT count(*) FROM pg_replication_slots WHERE slot_name=$1", slotName).
				Scan(&count),
		)
		assert.Zero(t, count)
	})
}
