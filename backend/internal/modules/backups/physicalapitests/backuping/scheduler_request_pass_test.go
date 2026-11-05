package backuping_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/testwait"
)

const backupRequestJob = "backups_physical_request_scheduler"

func TestPhysicalBackupRequestPass(t *testing.T) {
	t.Parallel()

	t.Run("starts a requested full backup without a full scheduler pass", func(t *testing.T) {
		t.Parallel()

		team := newBackupState(t)
		team.seedFull(t, schedulerFullSpec{Age: time.Minute})
		team.setSchedules(t, false)
		listedBefore := team.listBackups(t)

		team.trigger(t, "full")
		team.App.RunJobPass(t, backupRequestJob)

		startedBackups := team.listNewBackups(t, listedBefore)
		require.Len(t, startedBackups, 1)
		assert.Equal(t, "FULL", startedBackups[0].Type)
	})

	t.Run("starts a requested incremental backup on the extendable chain without a full scheduler pass",
		func(t *testing.T) {
			t.Parallel()

			team := newBackupState(t)
			full := team.seedFull(t, schedulerFullSpec{Age: time.Minute, ManifestFileName: new("root.manifest")})
			team.setSchedules(t, false)
			listedBefore := team.listBackups(t)

			team.trigger(t, "incremental")
			team.App.RunJobPass(t, backupRequestJob)

			startedBackups := team.listNewBackups(t, listedBefore)
			require.Len(t, startedBackups, 1)
			assert.Equal(t, "INCREMENTAL", startedBackups[0].Type)

			var rootFullBackupID string
			team.App.QuerySQL(
				t,
				&rootFullBackupID,
				"SELECT root_full_backup_id FROM backups_physical WHERE id=?",
				startedBackups[0].ID,
			)
			assert.Equal(t, full.ID.String(), rootFullBackupID)
		},
	)

	t.Run("starts a backup requested during a running backup after that backup ends", func(t *testing.T) {
		t.Parallel()

		team := newBackupTeam(t, backupTeamSpec{})
		releaseStorage := holdStorageWrites(t, team)

		team.trigger(t, "full")
		team.App.RunJobPass(t, backupRequestJob)

		runningBackups := team.listBackups(t)
		require.Len(t, runningBackups, 1)
		require.Equal(t, "IN_PROGRESS", runningBackups[0].Status)

		team.trigger(t, "full")
		team.App.RunJobPass(t, backupRequestJob)

		assert.Len(t, team.listBackups(t), 1)
		assert.True(t, team.hasFullBackupRequest(t))

		releaseStorage()
		testwait.Until(t, "the requested full starts after the running full ends", func(context.Context) error {
			team.App.RunJobPass(t, backupRequestJob)

			if listedBackups := team.listBackups(t); len(listedBackups) != 2 {
				return fmt.Errorf("%d backups are listed", len(listedBackups))
			}

			return nil
		})

		assert.False(t, team.hasFullBackupRequest(t))
	})

	t.Run("leaves a due schedule to the full scheduler pass when the database has no request", func(t *testing.T) {
		t.Parallel()

		team := newBackupState(t)

		team.App.RunJobPass(t, backupRequestJob)

		assert.Empty(t, team.listBackups(t))

		team.App.RunJobPass(t, "backups_physical_scheduler")

		startedBackups := team.listBackups(t)
		require.Len(t, startedBackups, 1)
		assert.Equal(t, "FULL", startedBackups[0].Type)
	})

	t.Run("keeps an incremental request pending and starts a requested full backup when a mismatch is unresolved",
		func(t *testing.T) {
			t.Parallel()

			team := newBackupState(t)
			team.seedFull(t, schedulerFullSpec{Age: 96 * time.Hour, ManifestFileName: new("root.manifest")})
			team.setSchedules(t, false)
			team.trigger(t, "incremental")
			team.seedFull(
				t,
				schedulerFullSpec{
					Status: "CHAIN_BROKEN",
					Reason: new("SYSTEM_IDENTIFIER_MISMATCH"),
					Age:    time.Minute,
				},
			)
			listedBefore := team.listBackups(t)

			team.App.RunJobPass(t, backupRequestJob)

			assert.Empty(t, team.listNewBackups(t, listedBefore))

			var pendingRequests int64
			team.App.QuerySQL(
				t,
				&pendingRequests,
				"SELECT count(*) FROM backups_configs_physical WHERE database_id=? AND force_incremental_requested_at IS NOT NULL",
				team.Database.Id,
			)
			assert.Equal(t, int64(1), pendingRequests)

			team.trigger(t, "full")
			team.App.RunJobPass(t, backupRequestJob)

			startedBackups := team.listNewBackups(t, listedBefore)
			require.Len(t, startedBackups, 1)
			assert.Equal(t, "FULL", startedBackups[0].Type)
		},
	)

	t.Run("leaves a replacement full to the full scheduler pass after a timeline failure when no request is pending",
		func(t *testing.T) {
			t.Parallel()

			team := newBackupState(t)
			full := team.seedFull(t, schedulerFullSpec{Age: 3 * time.Hour})
			team.seedIncremental(t, full.ID, "TIMELINE_SWITCH_DETECTED")
			team.setSchedules(t, false)
			listedBefore := team.listBackups(t)

			team.App.RunJobPass(t, backupRequestJob)

			assert.Empty(t, team.listNewBackups(t, listedBefore))

			team.App.RunJobPass(t, "backups_physical_scheduler")

			startedBackups := team.listNewBackups(t, listedBefore)
			require.Len(t, startedBackups, 1)
			assert.Equal(t, "FULL", startedBackups[0].Type)
		},
	)

	t.Run("starts a replacement full instead of a pending incremental request after a timeline failure",
		func(t *testing.T) {
			t.Parallel()

			team := newBackupState(t)
			full := team.seedFull(t, schedulerFullSpec{Age: 3 * time.Hour})
			team.seedIncremental(t, full.ID, "TIMELINE_SWITCH_DETECTED")
			team.setSchedules(t, false)
			team.App.ExecuteSQL(
				t,
				"UPDATE backups_configs_physical SET force_incremental_requested_at=? WHERE database_id=?",
				time.Now().UTC(),
				team.Database.Id,
			)
			listedBefore := team.listBackups(t)

			team.App.RunJobPass(t, backupRequestJob)

			startedBackups := team.listNewBackups(t, listedBefore)
			require.Len(t, startedBackups, 1)
			assert.Equal(t, "FULL", startedBackups[0].Type)
		},
	)

	t.Run("starts one full and clears a pending full request after a timeline failure", func(t *testing.T) {
		t.Parallel()

		team := newBackupState(t)
		full := team.seedFull(t, schedulerFullSpec{Age: 3 * time.Hour})
		team.seedIncremental(t, full.ID, "TIMELINE_SWITCH_DETECTED")
		team.setSchedules(t, false)
		team.requestFullWithoutStart(t)
		listedBefore := team.listBackups(t)

		team.App.RunJobPass(t, backupRequestJob)

		startedBackups := team.listNewBackups(t, listedBefore)
		require.Len(t, startedBackups, 1)
		assert.Equal(t, "FULL", startedBackups[0].Type)
		assert.False(t, team.hasFullBackupRequest(t))
	})

	t.Run("leaves a replacement full to the full scheduler pass after an interrupted full when no request is pending",
		func(t *testing.T) {
			t.Parallel()

			team := newBackupState(t)
			team.seedFull(
				t,
				schedulerFullSpec{Status: "ERROR", Reason: new("FAILOVER_DURING_BACKUP"), Age: time.Minute},
			)
			team.setSchedules(t, false)
			listedBefore := team.listBackups(t)

			team.App.RunJobPass(t, backupRequestJob)

			assert.Empty(t, team.listNewBackups(t, listedBefore))

			team.App.RunJobPass(t, "backups_physical_scheduler")

			startedBackups := team.listNewBackups(t, listedBefore)
			require.Len(t, startedBackups, 1)
			assert.Equal(t, "FULL", startedBackups[0].Type)
		},
	)

	t.Run("starts no replacement full after an interrupted full when a mismatch is unresolved", func(t *testing.T) {
		t.Parallel()

		team := newBackupState(t)
		team.seedUnresolvedMismatch(t)
		team.seedFull(
			t,
			schedulerFullSpec{Status: "ERROR", Reason: new("FAILOVER_DURING_BACKUP"), Age: time.Minute, Segment: 2},
		)
		team.setSchedules(t, false)
		team.App.ExecuteSQL(
			t,
			"UPDATE backups_configs_physical SET force_incremental_requested_at=? WHERE database_id=?",
			time.Now().UTC(),
			team.Database.Id,
		)
		listedBefore := team.listBackups(t)

		team.App.RunJobPass(t, backupRequestJob)
		team.App.RunJobPass(t, "backups_physical_scheduler")

		assert.Empty(t, team.listNewBackups(t, listedBefore))
	})

	t.Run("clears an incremental request and starts nothing when incrementals are disabled", func(t *testing.T) {
		t.Parallel()

		team := newBackupState(t)
		team.setSchedules(t, true)
		team.seedFull(t, schedulerFullSpec{Age: 2 * time.Hour})
		team.App.ExecuteSQL(
			t,
			"UPDATE backups_configs_physical SET force_incremental_requested_at=? WHERE database_id=?",
			time.Now().UTC(),
			team.Database.Id,
		)
		listedBefore := team.listBackups(t)

		team.App.RunJobPass(t, backupRequestJob)

		assert.Empty(t, team.listNewBackups(t, listedBefore))

		var pendingRequests int64
		team.App.QuerySQL(
			t,
			&pendingRequests,
			"SELECT count(*) FROM backups_configs_physical WHERE database_id=? AND force_incremental_requested_at IS NOT NULL",
			team.Database.Id,
		)
		assert.Zero(t, pendingRequests)
	})

	t.Run("starts a full backup in the new storage when a storage switch recorded the request", func(t *testing.T) {
		t.Parallel()

		team := newBackupState(t)
		team.seedFull(t, schedulerFullSpec{Age: time.Minute})
		team.setSchedules(t, false)

		targetStorageID := team.App.CreateLocalStorage(t, team.Owner, team.Database.WorkspaceId)
		switched := team.saveStorage(t, storageConfigSave{StorageID: &targetStorageID, IsEnabled: true})
		require.Equal(t, http.StatusOK, switched.StatusCode(), "%s", switched.Body)

		listedBefore := team.listBackups(t)

		team.App.RunJobPass(t, backupRequestJob)

		startedBackups := team.listNewBackups(t, listedBefore)
		require.Len(t, startedBackups, 1)
		assert.Equal(t, "FULL", startedBackups[0].Type)

		var storageID string
		team.App.QuerySQL(t, &storageID, "SELECT storage_id FROM backups_physical WHERE id=?", startedBackups[0].ID)
		assert.Equal(t, targetStorageID.String(), storageID)
	})

	t.Run("starts a full backup when a WAL slot rebuild recorded the request", func(t *testing.T) {
		t.Parallel()

		team := newBackupState(t)
		team.seedFull(t, schedulerFullSpec{Age: time.Minute})
		team.setSchedules(t, false)
		team.requestFullWithoutStart(t)
		listedBefore := team.listBackups(t)

		team.App.RunJobPass(t, backupRequestJob)

		startedBackups := team.listNewBackups(t, listedBefore)
		require.Len(t, startedBackups, 1)
		assert.Equal(t, "FULL", startedBackups[0].Type)
	})

	t.Run("starts a requested full backup after the backups of the database are turned back on", func(t *testing.T) {
		t.Parallel()

		team := newBackupState(t)
		team.trigger(t, "full")
		setBackupsEnabled(t, team, false)

		team.App.RunJobPass(t, backupRequestJob)

		assert.Empty(t, team.listBackups(t))
		assert.True(t, team.isFullBackupPending(t))

		setBackupsEnabled(t, team, true)
		team.App.RunJobPass(t, backupRequestJob)

		startedBackups := team.listBackups(t)
		require.Len(t, startedBackups, 1)
		assert.Equal(t, "FULL", startedBackups[0].Type)
	})

	t.Run("starts nothing and keeps the request when the backup config names no storage", func(t *testing.T) {
		t.Parallel()

		team := newBackupState(t)
		team.trigger(t, "full")
		team.App.ExecuteSQL(t, "UPDATE backups_configs SET storage_id=NULL WHERE database_id=?", team.Database.Id)

		team.App.RunJobPass(t, backupRequestJob)

		assert.Empty(t, team.listBackups(t))
		assert.True(t, team.isFullBackupPending(t))
	})
}

// requestFullWithoutStart records what a WAL slot rebuild records. Causing a real rebuild needs a lost
// replication slot on a running source, and the rebuild writes only this column.
func (team *backupTeam) requestFullWithoutStart(t *testing.T) {
	t.Helper()
	team.App.ExecuteSQL(
		t,
		"UPDATE backups_configs_physical SET force_full_requested_at=? WHERE database_id=?",
		time.Now().UTC(),
		team.Database.Id,
	)
}

// hasFullBackupRequest reads the config, because the WAL stream statistics also report a full backup
// as pending while one is running.
func (team *backupTeam) hasFullBackupRequest(t *testing.T) bool {
	t.Helper()

	var requestCount int64
	team.App.QuerySQL(
		t,
		&requestCount,
		"SELECT count(*) FROM backups_configs_physical WHERE database_id=? AND force_full_requested_at IS NOT NULL",
		team.Database.Id,
	)

	return requestCount == 1
}

func (team *backupTeam) isFullBackupPending(t *testing.T) bool {
	t.Helper()
	response, err := team.Owner.Client.GetWalStreamStatisticsWithResponse(
		t.Context(),
		team.Database.Id,
		&apiclient.GetWalStreamStatisticsParams{Period: apiclient.WalStreamStatisticsPeriodHOUR},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), "%s", response.Body)

	return response.JSON200.Health.IsFullBackupPending
}
