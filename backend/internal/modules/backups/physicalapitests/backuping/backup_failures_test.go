package backuping_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/modules/storages"
)

func TestPhysicalBackupFailures(t *testing.T) {
	t.Parallel()

	t.Run(
		"records the tool exit and server rejection when physical replication is denied by pg_hba",
		func(t *testing.T) {
			t.Parallel()

			team := newBackupTeam(t, backupTeamSpec{IsPhysicalReplicationDenied: true})
			backup := team.runBackup(t, "full")

			require.Equal(t, "ERROR", backup.Status)
			require.NotNil(t, backup.FailMessage)
			assert.Contains(t, *backup.FailMessage, "exit status 1")
			assert.Contains(t, *backup.FailMessage, "FATAL")
			assert.NotContains(t, *backup.FailMessage, "assets/tools")

			team.assertNoRunningBackup(t)
			team.assertTerminalDetails(t, backup, "PG_BASEBACKUP_FAILED")
		},
	)
	t.Run("records a failed full when the source becomes unreachable", func(t *testing.T) {
		t.Parallel()

		team := newBackupTeam(t, backupTeamSpec{})

		team.App.ExecuteSQL(
			t,
			"UPDATE databases_postgresql_physical SET host='127.0.0.1',port=1 WHERE database_id=?",
			team.Database.Id,
		)

		backup := team.runBackup(t, "full")
		require.Equal(t, "ERROR", backup.Status)
		require.NotNil(t, backup.FailMessage)
		assert.NotEmpty(t, *backup.FailMessage)
		team.assertNoRunningBackup(t)
		team.assertTerminalDetails(t, backup, "")
	})
	t.Run(
		"records a failed incremental when the source becomes unreachable",
		func(t *testing.T) {
			t.Parallel()

			team := newBackupTeam(t, backupTeamSpec{})
			full := team.runBackup(t, "full")
			require.Equal(t, "COMPLETED", full.Status)

			team.App.ExecuteSQL(
				t,
				"UPDATE databases_postgresql_physical SET host='127.0.0.1',port=1 WHERE database_id=?",
				team.Database.Id,
			)

			backup := team.runBackup(t, "incremental")
			require.Equal(t, "ERROR", backup.Status)
			team.assertNoRunningBackup(t)
			team.assertTerminalDetails(t, backup, "")
		},
	)
	t.Run("breaks the incremental chain without an artifact when summarization was disabled", func(t *testing.T) {
		t.Parallel()

		team := newBackupTeam(t, backupTeamSpec{})
		full := team.runBackup(t, "full")
		require.Equal(t, "COMPLETED", full.Status)

		source := team.openSource(t)
		_, err := source.Exec(t.Context(), "ALTER SYSTEM SET summarize_wal=off")
		require.NoError(t, err)
		_, err = source.Exec(t.Context(), "SELECT pg_reload_conf()")
		require.NoError(t, err)

		backup := team.runBackup(t, "incremental")
		require.Equal(t, "CHAIN_BROKEN", backup.Status)
		team.assertTerminalDetails(t, backup, "SUMMARIZER_OFF")
		team.assertNoRunningBackup(t)

		var incremental apptest.PhysicalIncrementalBackupSeed
		team.App.QuerySQL(t, &incremental, "SELECT * FROM backups_physical WHERE id=?", backup.ID)
		assert.Nil(t, incremental.FileName)
	})
	t.Run("breaks the full chain when the saved system identifier belongs to another cluster", func(t *testing.T) {
		t.Parallel()

		team := newBackupTeam(t, backupTeamSpec{})

		team.App.ExecuteSQL(
			t,
			"UPDATE databases_postgresql_physical SET system_identifier='1' WHERE database_id=?",
			team.Database.Id,
		)

		backup := team.runBackup(t, "full")
		require.Equal(t, "CHAIN_BROKEN", backup.Status)
		team.assertTerminalDetails(t, backup, "SYSTEM_IDENTIFIER_MISMATCH")
		team.assertNoRunningBackup(t)
	})
	t.Run("breaks an incremental before running the tool when its parent manifest is missing", func(t *testing.T) {
		t.Parallel()

		team := newBackupTeam(t, backupTeamSpec{})
		full := team.runBackup(t, "full")
		require.Equal(t, "COMPLETED", full.Status)

		team.App.ExecuteSQL(t, "UPDATE backups_physical SET manifest_file_name=NULL WHERE id=?", full.ID)
		backup := team.runBackup(t, "incremental")
		require.Equal(t, "CHAIN_BROKEN", backup.Status)
		team.assertTerminalDetails(t, backup, "PARENT_MANIFEST_MISSING")
		team.assertNoRunningBackup(t)
	})
	t.Run("breaks the incremental chain when its stored parent manifest is corrupted", func(t *testing.T) {
		t.Parallel()

		team := newBackupTeam(t, backupTeamSpec{})
		full := team.runBackup(t, "full")
		require.Equal(t, "COMPLETED", full.Status)

		var manifest string
		team.App.QuerySQL(t, &manifest, "SELECT manifest_file_name FROM backups_physical WHERE id=?", full.ID)
		team.App.PublishStorageFile(
			t,
			storages.StoredFileReference{StorageID: team.StorageID, FileName: manifest},
			strings.NewReader("corrupted manifest"),
		)

		incremental := team.runBackup(t, "incremental")
		require.Equal(t, "CHAIN_BROKEN", incremental.Status)
		team.assertTerminalDetails(t, incremental, "MANIFEST_CORRUPTED")
		team.assertNoRunningBackup(t)
	})
}

func (team *backupTeam) assertTerminalDetails(t *testing.T, backup backupRow, reason string) {
	t.Helper()
	table := "backups_physical"
	var details []struct {
		ErrorReason *string
		CompletedAt *time.Time
	}
	team.App.QuerySQL(t, &details, "SELECT error_reason,completed_at FROM "+table+" WHERE id=?", backup.ID)
	require.Len(t, details, 1)
	require.NotNil(t, details[0].CompletedAt)
	if reason != "" {
		require.NotNil(t, details[0].ErrorReason)
		assert.Equal(t, reason, *details[0].ErrorReason)
	}
}
