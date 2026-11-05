package backuping_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

func TestFailedPhysicalBackupStream(t *testing.T) {
	t.Parallel()

	t.Run(
		"discards a partial artifact and drops the slot when the real pg_basebackup walsender is terminated",
		func(t *testing.T) {
			t.Parallel()
			terminateFullBackupStream(t, false)
		},
	)
	t.Run(
		"drops the slot over a fresh connection when the backup role loses all its source connections",
		func(t *testing.T) {
			t.Parallel()
			terminateFullBackupStream(t, true)
		},
	)
}

func terminateFullBackupStream(t *testing.T, isHoldingConnectionTerminated bool) {
	t.Helper()
	team := newBackupTeam(t, backupTeamSpec{})
	roleName := ""
	if isHoldingConnectionTerminated {
		roleName = team.useReplicationRole(t)
	}

	source := team.openSource(t)
	_, seedErr := source.Exec(
		t.Context(),
		"CREATE TABLE interrupted_backup_payload AS SELECT string_agg(md5(random()::text),'') AS payload FROM generate_series(1,1000000) entry GROUP BY entry/32",
	)
	require.NoError(t, seedErr)

	team.App.InstallStorageWriteBarrier(t)
	storageLock := team.App.BeginSQLTransaction(t)
	var storageIDs []string
	storageLock.QuerySQL(t, &storageIDs, "SELECT id FROM storages WHERE id=? FOR NO KEY UPDATE", team.StorageID)
	require.Len(t, storageIDs, 1)

	team.trigger(t, "full")
	team.App.RunJobPass(t, "backups_physical_scheduler")

	var walsenderPID int
	testwait.Until(
		t,
		"pg_basebackup streams from the source while file publication waits",
		func(context.Context) error {
			var pids []int
			rows, err := source.Query(
				t.Context(),
				"SELECT pid FROM pg_stat_activity WHERE application_name='pg_basebackup' AND backend_type='walsender'",
			)
			if err != nil {
				return err
			}
			defer rows.Close()
			for rows.Next() {
				var pid int
				if err := rows.Scan(&pid); err != nil {
					return err
				}
				pids = append(pids, pid)
			}
			if len(pids) == 0 {
				return fmt.Errorf("no pg_basebackup walsender")
			}
			walsenderPID = pids[0]
			return rows.Err()
		},
	)

	var fileNames []string
	team.App.QuerySQL(
		t,
		&fileNames,
		"SELECT file_name FROM backups_physical WHERE database_id=? AND file_name IS NOT NULL",
		team.Database.Id,
	)
	require.Len(t, fileNames, 1)

	var err error
	if isHoldingConnectionTerminated {
		_, err = source.Exec(
			t.Context(),
			"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE usename=$1",
			roleName,
		)
	} else {
		_, err = source.Exec(t.Context(), "SELECT pg_terminate_backend($1)", walsenderPID)
	}

	require.NoError(t, err)
	storageLock.Rollback(t)
	team.App.WaitForPhysicalBackups(t)

	backups := team.listBackups(t)
	require.Len(t, backups, 1)
	assert.Equal(t, "ERROR", backups[0].Status)
	team.assertTerminalDetails(t, backups[0], "PG_BASEBACKUP_FAILED")
	team.assertNoRunningBackup(t)

	team.App.RunJobPass(t, "storage_file_deletion")
	for _, fileName := range fileNames {
		assert.False(t, team.App.IsLocalStorageFilePresent(t, fileName))
		assert.False(t, team.App.IsLocalStorageFilePresent(t, fileName+".manifest"))
	}

	var slots int
	require.NoError(
		t,
		source.QueryRow(t.Context(), "SELECT count(*) FROM pg_replication_slots WHERE slot_name LIKE 'databasus_basebackup_%'").
			Scan(&slots),
	)
	assert.Zero(t, slots)
}
