package backups_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

const logicalBackupRetentionJobName = "backups_logical_retention"

//nolint:tparallel // The subtests share one source server.
func TestLogicalBackupRetention(t *testing.T) {
	t.Parallel()

	server := startRestoreServer(t, restoreServerImage)

	t.Run("keeps historical GFS points and their files after daily cleanup across months", func(t *testing.T) {
		tm := newTeam(t)
		source := newBackupSourceDatabase(t, tm, server)
		settings := newLogicalBackupSettings()
		settings.Retention = apiclient.LogicalRetention{
			Policy:    apiclient.LogicalRetentionPolicyGFS,
			Period:    apiclient.RetentionPeriodFOREVER,
			GfsDays:   2,
			GfsWeeks:  1,
			GfsMonths: 1,
		}

		saveLogicalBackupSettings(
			t,
			tm.owner,
			source.databaseID,
			source.storageID,
			settings,
			apiclient.BackupEncryptionNONE,
		)
		start := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
		backupIDs := make(map[string]uuid.UUID)
		backupFiles := make(map[string]string)

		for day := range 53 {
			backupTime := start.AddDate(0, 0, day)
			tm.app.Clock.Set(backupTime)
			backup := completeRestorableBackup(t, tm.owner, source.databaseID)
			date := backupTime.Format(time.DateOnly)
			backupIDs[date] = backup.Id
			backupFiles[date] = getBackupFileName(t, tm, backup.Id)

			tm.app.Clock.Advance(2 * time.Hour)
			tm.app.RunJobPass(t, logicalBackupRetentionJobName)
			tm.app.RunJobPass(t, storageFileDeletionJobName)

			if day == 24 {
				assert.ElementsMatch(
					t,
					[]uuid.UUID{
						backupIDs["2026-11-05"],
						backupIDs["2026-11-04"],
						backupIDs["2026-11-01"],
						backupIDs["2026-10-31"],
					},
					listLogicalBackupIDs(t, tm.owner, source.databaseID),
				)
				for _, keptDate := range []string{"2026-11-05", "2026-11-04", "2026-11-01", "2026-10-31"} {
					assert.True(t, tm.app.IsLocalStorageFilePresent(t, backupFiles[keptDate]))
					assert.True(t, tm.app.IsLocalStorageFilePresent(t, backupFiles[keptDate]+".metadata"))
				}
			}
		}

		assert.ElementsMatch(
			t,
			[]uuid.UUID{
				backupIDs["2026-12-03"],
				backupIDs["2026-12-02"],
				backupIDs["2026-11-30"],
				backupIDs["2026-11-29"],
			},
			listLogicalBackupIDs(t, tm.owner, source.databaseID),
		)
		assert.False(t, tm.app.IsLocalStorageFilePresent(t, backupFiles["2026-10-31"]))
		assert.False(t, tm.app.IsLocalStorageFilePresent(t, backupFiles["2026-10-31"]+".metadata"))
		assert.True(t, tm.app.IsLocalStorageFilePresent(t, backupFiles["2026-11-30"]))
		assert.True(t, tm.app.IsLocalStorageFilePresent(t, backupFiles["2026-11-30"]+".metadata"))
	})

	t.Run("keeps the successful GFS candidate after newer cancelled and failed attempts", func(t *testing.T) {
		tm := newTeam(t)
		source := newBackupSourceDatabase(t, tm, server)
		settings := newLogicalBackupSettings()
		settings.Retention = apiclient.LogicalRetention{
			Policy:    apiclient.LogicalRetentionPolicyGFS,
			Period:    apiclient.RetentionPeriodFOREVER,
			GfsMonths: 1,
		}
		saveLogicalBackupSettings(
			t,
			tm.owner,
			source.databaseID,
			source.storageID,
			settings,
			apiclient.BackupEncryptionNONE,
		)

		tm.app.Clock.Set(time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC))
		septemberBackup := completeRestorableBackup(t, tm.owner, source.databaseID)

		tm.app.Clock.Set(time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC))
		octoberBackup := completeRestorableBackup(t, tm.owner, source.databaseID)

		tm.app.Clock.Set(time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC))
		cancelledBackup, releaseLock := startBlockedSourceBackup(t, tm, server, source)
		response := requestLogicalBackupCancel(t, tm.owner, cancelledBackup.Id)
		require.Equal(t, http.StatusAccepted, response.StatusCode())
		waitForRestorableBackupStatus(t, tm.owner, cancelledBackup.Id, apiclient.LogicalBackupStatusCANCELED)
		releaseLock()
		tm.app.WaitForBackupRuns(t)

		routeThroughBastionPort(t, tm, source, getClosedPort(t))
		failedBackup := startRestorableBackup(t, tm.owner, source.databaseID)
		waitForRestorableBackupStatus(t, tm.owner, failedBackup.Id, apiclient.LogicalBackupStatusFAILED)
		tm.app.Clock.Advance(2 * time.Hour)

		tm.app.RunJobPass(t, logicalBackupRetentionJobName)

		assert.ElementsMatch(
			t,
			[]uuid.UUID{septemberBackup.Id, octoberBackup.Id, cancelledBackup.Id, failedBackup.Id},
			listLogicalBackupIDs(t, tm.owner, source.databaseID),
		)

		tm.app.Clock.Advance(400 * 24 * time.Hour)
		tm.app.RunJobPass(t, logicalBackupRetentionJobName)

		assert.ElementsMatch(
			t,
			[]uuid.UUID{septemberBackup.Id, octoberBackup.Id, cancelledBackup.Id, failedBackup.Id},
			listLogicalBackupIDs(t, tm.owner, source.databaseID),
		)
	})

	t.Run("removes the two oldest of seven backups and deletes their files when the config keeps five",
		func(t *testing.T) {
			tm := newTeam(t)
			source := newBackupSourceDatabase(t, tm, server)
			keepingFiveSettings := newLogicalBackupSettings()
			keepingFiveSettings.Retention.Policy = apiclient.LogicalRetentionPolicyCOUNT
			keepingFiveSettings.Retention.Count = 5
			saveLogicalBackupSettings(t, tm.owner, source.databaseID, source.storageID, keepingFiveSettings,
				apiclient.BackupEncryptionNONE)

			backupIDs := make([]uuid.UUID, 0, 7)
			for range 7 {
				backupIDs = append(backupIDs, completeRestorableBackup(t, tm.owner, source.databaseID).Id)
				tm.app.Clock.Advance(time.Hour)
			}

			removedFileNames := []string{getBackupFileName(t, tm, backupIDs[0]), getBackupFileName(t, tm, backupIDs[1])}
			keptFileName := getBackupFileName(t, tm, backupIDs[2])

			tm.app.RunJobPass(t, logicalBackupRetentionJobName)

			assert.Equal(t, []uuid.UUID{backupIDs[6], backupIDs[5], backupIDs[4], backupIDs[3], backupIDs[2]},
				listLogicalBackupIDs(t, tm.owner, source.databaseID))

			tm.app.RunJobPass(t, storageFileDeletionJobName)

			for _, removedFileName := range removedFileNames {
				assert.False(t, tm.app.IsLocalStorageFilePresent(t, removedFileName))
				assert.False(t, tm.app.IsLocalStorageFilePresent(t, removedFileName+".metadata"))
			}

			assert.True(t, tm.app.IsLocalStorageFilePresent(t, keptFileName))
		})

	t.Run("removes the other expired backups when one removal fails", func(t *testing.T) {
		logSink := &errorLogSink{}
		testApp := apptest.New(t, apptest.WithLogSink(logSink))
		admin := testApp.CreateAdmin(t)
		tm := team{
			app:       testApp,
			admin:     admin,
			owner:     testApp.CreateNonAdminUser(t, admin),
			workspace: apiclient.Workspace{},
		}
		tm.workspace = apptest.CreateWorkspace(t, tm.owner, "Retention failure")
		source := newBackupSourceDatabase(t, tm, server)
		keepingOneSettings := newLogicalBackupSettings()
		keepingOneSettings.Retention.Policy = apiclient.LogicalRetentionPolicyCOUNT
		keepingOneSettings.Retention.Count = 1
		saveLogicalBackupSettings(t, tm.owner, source.databaseID, source.storageID, keepingOneSettings,
			apiclient.BackupEncryptionNONE)

		backupIDs := make([]uuid.UUID, 0, 4)
		for range 4 {
			backupIDs = append(backupIDs, completeRestorableBackup(t, tm.owner, source.databaseID).Id)
			tm.app.Clock.Advance(time.Hour)
		}

		refusedBackupID := backupIDs[2]
		tm.app.ExecuteSQL(t, fmt.Sprintf(`CREATE FUNCTION refuse_backup_removal() RETURNS trigger AS $$ BEGIN
			IF OLD.id = '%s' THEN RAISE EXCEPTION 'removal refused'; END IF;
			RETURN OLD; END $$ LANGUAGE plpgsql`, refusedBackupID))
		tm.app.ExecuteSQL(t, `CREATE TRIGGER refuse_backup_removal BEFORE DELETE ON backups_logical
			FOR EACH ROW EXECUTE FUNCTION refuse_backup_removal()`)

		tm.app.RunJobPass(t, logicalBackupRetentionJobName)

		assert.Equal(
			t,
			[]uuid.UUID{backupIDs[3], refusedBackupID},
			listLogicalBackupIDs(t, tm.owner, source.databaseID),
		)
		assert.True(t, logSink.hasErrorNaming(refusedBackupID.String()))

		tm.app.ExecuteSQL(t, "DROP TRIGGER refuse_backup_removal ON backups_logical")
		tm.app.RunJobPass(t, logicalBackupRetentionJobName)

		assert.Equal(t, []uuid.UUID{backupIDs[3]}, listLogicalBackupIDs(t, tm.owner, source.databaseID))
	})

	t.Run("keeps an expired backup while it runs and removes it at the first check after it ends",
		func(t *testing.T) {
			tm := newTeam(t)
			source := newBackupSourceDatabase(t, tm, server)
			keepingOneDaySettings := newLogicalBackupSettings()
			keepingOneDaySettings.Retention.Policy = apiclient.LogicalRetentionPolicyTIMEPERIOD
			keepingOneDaySettings.Retention.Period = apiclient.RetentionPeriodDAY
			saveLogicalBackupSettings(t, tm.owner, source.databaseID, source.storageID, keepingOneDaySettings,
				apiclient.BackupEncryptionNONE)
			runningBackup, releaseLock := startBlockedSourceBackup(t, tm, server, source)
			tm.app.Clock.Advance(25 * time.Hour)

			tm.app.RunJobPass(t, logicalBackupRetentionJobName)

			require.Equal(t, []uuid.UUID{runningBackup.Id}, listLogicalBackupIDs(t, tm.owner, source.databaseID))

			releaseLock()
			waitForNoRunningLogicalBackup(t, tm, source.databaseID)
			tm.app.RunJobPass(t, logicalBackupRetentionJobName)

			assert.Empty(t, listLogicalBackupIDs(t, tm.owner, source.databaseID))
		})

	t.Run("keeps a backup younger than one hour that the count no longer keeps", func(t *testing.T) {
		tm := newTeam(t)
		source := newBackupSourceDatabase(t, tm, server)
		keepingOneSettings := newLogicalBackupSettings()
		keepingOneSettings.Retention.Policy = apiclient.LogicalRetentionPolicyCOUNT
		keepingOneSettings.Retention.Count = 1
		saveLogicalBackupSettings(t, tm.owner, source.databaseID, source.storageID, keepingOneSettings,
			apiclient.BackupEncryptionNONE)
		olderBackup := completeRestorableBackup(t, tm.owner, source.databaseID)
		tm.app.Clock.Advance(time.Minute)
		newerBackup := completeRestorableBackup(t, tm.owner, source.databaseID)

		tm.app.RunJobPass(t, logicalBackupRetentionJobName)

		assert.Equal(
			t,
			[]uuid.UUID{newerBackup.Id, olderBackup.Id},
			listLogicalBackupIDs(t, tm.owner, source.databaseID),
		)
	})
}

// errorLogSink keeps the attributes of the error records the application logged.
type errorLogSink struct {
	mutex         sync.Mutex
	errorMessages []string
}

func (s *errorLogSink) Enabled(_ context.Context, level slog.Level) bool {
	return level >= slog.LevelError
}

func (s *errorLogSink) Handle(_ context.Context, record slog.Record) error {
	var message strings.Builder

	message.WriteString(record.Message)
	record.Attrs(func(attribute slog.Attr) bool {
		message.WriteString(" " + attribute.String())

		return true
	})

	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.errorMessages = append(s.errorMessages, message.String())

	return nil
}

func (s *errorLogSink) WithAttrs([]slog.Attr) slog.Handler { return s }

func (s *errorLogSink) WithGroup(string) slog.Handler { return s }

func (s *errorLogSink) hasErrorNaming(text string) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return slices.ContainsFunc(s.errorMessages, func(message string) bool {
		return strings.Contains(message, text)
	})
}
