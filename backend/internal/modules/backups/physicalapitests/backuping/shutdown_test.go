package backuping_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/testwait"
)

const shutdownFailMessage = "backup interrupted by application shutdown"

func TestPhysicalBackupShutdown(t *testing.T) {
	t.Parallel()

	for _, kind := range []string{"full", "incremental"} {
		for _, scenario := range []struct {
			name               string
			shouldMetadataFail bool
		}{
			{name: "keeps a completed " + kind + " backup when shutdown reaches it before publication", shouldMetadataFail: false},
			{name: "fails a " + kind + " backup when metadata fails during shutdown", shouldMetadataFail: true},
		} {
			t.Run(scenario.name, func(t *testing.T) {
				t.Parallel()

				started := make(chan context.Context, 2)
				observer := backupRunObserver{Handler: slog.DiscardHandler, started: started}
				team := newBackupTeam(t, backupTeamSpec{}, apptest.WithLogSink(observer))

				if kind == "incremental" {
					require.Equal(t, "COMPLETED", team.runBackup(t, "full").Status)
					observer.WaitForStart(t)
				}

				failures, successes := &webhookInbox{}, &webhookInbox{}
				team.attachWebhook(t, failures, apiclient.NotificationCategoryBACKUPFAILED)
				team.attachWebhook(t, successes, apiclient.NotificationCategoryBACKUPSUCCESS)
				team.setNotifications(
					t,
					apiclient.NotificationTriggerBACKUPFAILED,
					apiclient.NotificationTriggerBACKUPSUCCESS,
				)

				metadata := holdBackupMetadata(t, team)
				team.trigger(t, kind)
				team.App.RunJobPass(t, "backups_physical_scheduler")
				metadata.Wait(t)
				backupCtx := observer.WaitForStart(t)

				var runningBackup backupRow
				for _, backup := range team.listBackups(t) {
					if backup.Status == "IN_PROGRESS" {
						runningBackup = backup
					}
				}

				require.NotEmpty(t, runningBackup.ID)

				var copiedBackup apptest.PhysicalFullBackupSeed
				team.App.QuerySQL(t, &copiedBackup, "SELECT * FROM backups_physical WHERE id=?", runningBackup.ID)

				require.NotNil(t, copiedBackup.FileName)
				require.True(t, team.App.IsLocalStorageFilePresent(t, *copiedBackup.FileName))
				require.True(t, team.App.IsLocalStorageFilePresent(t, *copiedBackup.FileName+".manifest"))

				team.App.StopModules()
				testwait.Until(t, "shutdown cancels the original backup run context", func(context.Context) error {
					if backupCtx.Err() == nil {
						return fmt.Errorf("backup run context is still active")
					}

					return nil
				})
				require.Equal(t, shutdownFailMessage, context.Cause(backupCtx).Error())

				if scenario.shouldMetadataFail {
					metadata.CancelWrite(t)
				}

				metadata.Release(t)
				team.App.RestartModules(t)

				if scenario.shouldMetadataFail {
					failedBackup := findPhysicalBackup(t, team, runningBackup)

					require.Equal(t, "ERROR", failedBackup.Status)
					require.NotNil(t, failedBackup.FailMessage)
					assert.Equal(t, shutdownFailMessage, *failedBackup.FailMessage)
					team.assertTerminalDetails(t, failedBackup, "APPLICATION_RESTART")
					team.assertNoRunningBackup(t)

					team.App.Clock.Advance(3 * time.Second)
					team.App.RunJobPass(t, "storage_file_deletion")

					for _, suffix := range []string{"", ".manifest", ".metadata"} {
						assert.False(t, team.App.IsLocalStorageFilePresent(t, *copiedBackup.FileName+suffix))
					}

					team.App.WaitForNotificationDeliveries(t)

					require.Len(t, failures.getMessages(), 1)
					assert.Contains(t, failures.getMessages()[0], failedBackup.ID.String())
					assert.Empty(t, successes.getMessages())

					return
				}

				completedBackup := findPhysicalBackup(t, team, runningBackup)

				require.Equal(t, "COMPLETED", completedBackup.Status)
				assert.Nil(t, completedBackup.FailMessage)
				team.assertNoRunningBackup(t)

				team.App.Clock.Advance(3 * time.Second)
				team.App.RunJobPass(t, "storage_file_deletion")

				team.assertArtifacts(t, completedBackup)

				var persistedBackup apptest.PhysicalFullBackupSeed
				team.App.QuerySQL(t, &persistedBackup, "SELECT * FROM backups_physical WHERE id=?", completedBackup.ID)

				assert.Nil(t, persistedBackup.ErrorReason)
				require.NotNil(t, persistedBackup.CompletedAt)
				require.NotNil(t, persistedBackup.FileName)
				assert.True(t, team.App.IsLocalStorageFilePresent(t, *persistedBackup.FileName+".metadata"))

				team.App.WaitForNotificationDeliveries(t)

				require.Len(t, successes.getMessages(), 1)
				assert.Contains(t, successes.getMessages()[0], completedBackup.ID.String())
				assert.Empty(t, failures.getMessages())
			})
		}

		t.Run(
			"records failure and notifies subscribers when shutdown interrupts a "+kind+" stream",
			func(t *testing.T) {
				t.Parallel()

				team := newBackupTeam(t, backupTeamSpec{})
				if kind == "incremental" {
					require.Equal(t, "COMPLETED", team.runBackup(t, "full").Status)
				}

				failures, successes := &webhookInbox{}, &webhookInbox{}
				team.attachWebhook(t, failures, apiclient.NotificationCategoryBACKUPFAILED)
				team.attachWebhook(t, successes, apiclient.NotificationCategoryBACKUPSUCCESS)
				team.setNotifications(t, apiclient.NotificationTriggerBACKUPFAILED)

				backup, releaseStorage := startHeldPhysicalStream(t, team, kind)
				team.App.RestartModules(t)
				releaseStorage()

				listed := findPhysicalBackup(t, team, backup)
				assert.Equal(t, "ERROR", listed.Status)
				require.NotNil(t, listed.FailMessage)
				assert.Equal(t, shutdownFailMessage, *listed.FailMessage)
				team.assertTerminalDetails(t, listed, "APPLICATION_RESTART")
				team.assertNoRunningBackup(t)

				team.App.WaitForNotificationDeliveries(t)
				require.Len(t, failures.getMessages(), 1)
				assert.Contains(t, failures.getMessages()[0], shutdownFailMessage)
				assert.Empty(t, successes.getMessages())

				team.App.RunJobPass(t, "storage_file_deletion")

				var artifacts []struct {
					FileName         *string
					ManifestFileName *string
				}
				table := "backups_physical"
				team.App.QuerySQL(
					t,
					&artifacts,
					"SELECT file_name, manifest_file_name FROM "+table+" WHERE id=?",
					backup.ID,
				)

				require.Len(t, artifacts, 1)
				require.NotNil(t, artifacts[0].FileName)
				assert.False(t, team.App.IsLocalStorageFilePresent(t, *artifacts[0].FileName))
				assert.False(t, team.App.IsLocalStorageFilePresent(t, *artifacts[0].FileName+".metadata"))
				if artifacts[0].ManifestFileName != nil {
					assert.False(t, team.App.IsLocalStorageFilePresent(t, *artifacts[0].ManifestFileName))
				}
			},
		)
		t.Run("keeps the user cancellation outcome when a user cancels a "+kind+" stream", func(t *testing.T) {
			t.Parallel()

			team := newBackupTeam(t, backupTeamSpec{})
			if kind == "incremental" {
				require.Equal(t, "COMPLETED", team.runBackup(t, "full").Status)
			}

			backup, releaseStorage := startHeldPhysicalStream(t, team, kind)
			cancelPhysicalBackup(t, team, backup)
			releaseStorage()
			team.App.WaitForPhysicalBackups(t)

			listed := findPhysicalBackup(t, team, backup)
			assert.Equal(t, "CANCELED", listed.Status)
			require.NotNil(t, listed.FailMessage)
			assert.Equal(t, "context canceled", *listed.FailMessage)
			team.assertNoRunningBackup(t)
		})
	}
}

func TestCancelDispatchedPhysicalBackup(t *testing.T) {
	t.Parallel()

	t.Run(
		"waits for process cleanup when cancellation records a terminal status before the process stops",
		func(t *testing.T) {
			t.Parallel()

			team := newBackupTeam(t, backupTeamSpec{})
			backup, releaseStorage := startHeldPhysicalStream(t, team, "full")
			resume := apptest.SuspendPhysicalBackupProcess(t, backup.ID)
			releaseStorage()
			cancelPhysicalBackup(t, team, backup)
			require.Equal(t, "CANCELED", findPhysicalBackup(t, team, backup).Status)

			team.App.WaitForPhysicalBackups(t)

			team.assertNoRunningBackup(t)
			resume()
		},
	)

	t.Run("ends the backup when cancellation immediately follows a scheduler pass", func(t *testing.T) {
		t.Parallel()

		team := newBackupTeam(t, backupTeamSpec{})
		releaseStorage := holdStorageWrites(t, team)
		team.trigger(t, "full")
		team.App.RunJobPass(t, "backups_physical_scheduler")

		backups := team.listBackups(t)
		require.Len(t, backups, 1)

		cancelPhysicalBackup(t, team, backups[0])
		releaseStorage()
		team.App.WaitForPhysicalBackups(t)

		listed := findPhysicalBackup(t, team, backups[0])
		assert.Contains(t, []string{"ERROR", "CANCELED"}, listed.Status)
		team.assertNoRunningBackup(t)
	})
}

func startHeldPhysicalStream(t *testing.T, team *backupTeam, kind string) (backupRow, func()) {
	t.Helper()
	source := team.openSource(t)
	_, err := source.Exec(
		t.Context(),
		"CREATE TABLE shutdown_backup_payload AS SELECT string_agg(md5(random()::text),'') AS payload FROM generate_series(1,1000000) entry GROUP BY entry/32",
	)
	require.NoError(t, err)

	releaseStorage := holdStorageWrites(t, team)
	team.trigger(t, kind)
	team.App.RunJobPass(t, "backups_physical_scheduler")

	var backup backupRow
	testwait.Until(t, "pg_basebackup streams the held physical backup", func(ctx context.Context) error {
		var activeCount int
		if err := source.QueryRow(ctx, "SELECT count(*) FROM pg_stat_activity WHERE application_name='pg_basebackup' AND backend_type='walsender'").
			Scan(&activeCount); err != nil {
			return err
		}
		if activeCount == 0 {
			return fmt.Errorf("pg_basebackup has not started streaming")
		}
		for _, listed := range team.listBackups(t) {
			if listed.Status == "IN_PROGRESS" {
				backup = listed
				return nil
			}
		}
		return fmt.Errorf("no in-progress physical backup")
	})

	return backup, releaseStorage
}

func cancelPhysicalBackup(t *testing.T, team *backupTeam, backup backupRow) {
	t.Helper()
	status, body := team.App.CallPhysicalAPI(
		t,
		team.Owner,
		apptest.PhysicalCancelRequest(backup.ID),
	)
	require.Equal(t, http.StatusOK, status, "%s", body)
}

func findPhysicalBackup(t *testing.T, team *backupTeam, backup backupRow) backupRow {
	t.Helper()
	for _, listed := range team.listBackups(t) {
		if listed.ID == backup.ID {
			return listed
		}
	}
	t.Fatalf("physical backup %s is absent from the list", backup.ID)
	return backupRow{}
}
